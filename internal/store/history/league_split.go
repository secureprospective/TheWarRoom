package history

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
)

// Before slice 6 every league on a hosted server shared one file (server.db). The split moves each
// league into its own file, through the same build-prove-commit steps as the sign-in and reference
// moves: each league's file is built and proved beside the shared file, then one commit in the shared
// file drops its league tables and records the split (the point of no return). The shared audit chain
// is closed there once, its final head kept in the record and in every league file.

// LeagueSplitTable is the shared file's record that its leagues moved to their own files.
const LeagueSplitTable = "league_split"

// LeagueSplit is that record.
type LeagueSplit struct {
	Leagues []string
	Shared  db.AuditHead
	At      time.Time
	// Proofs are each league table's rows and digest as the split copied and proved them (league →
	// table); a server put back from backups has none.
	Proofs map[string]map[string]db.RowProof
}

func sqlText(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

// SharedLeagues lists every league the shared file holds rows for, in any league table.
func SharedLeagues(ctx context.Context, shared *db.Pools) ([]string, error) {
	parts := make([]string, 0, len(leagueTables()))
	for _, table := range leagueTables() {
		parts = append(parts, `SELECT league FROM main.`+table)
	}
	return readStrings(ctx, shared, `SELECT DISTINCT league FROM (`+strings.Join(parts, ` UNION `)+`) ORDER BY league`)
}

// SharedAuditHead verifies the shared file's ledger audit chain and returns its head.
func SharedAuditHead(ctx context.Context, shared *db.Pools) (db.AuditHead, error) {
	tx, err := shared.Read().BeginTx(ctx, nil)
	if err != nil {
		return db.AuditHead{}, fmt.Errorf("history: read the shared audit chain: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	head, err := db.VerifyAudit(ctx, tx, ledgerAuditSource, ledgerAuditQuery)
	if err != nil {
		return db.AuditHead{}, fmt.Errorf("history: verify the shared audit chain: %w", err)
	}
	return head, nil
}

func leagueRows(league string) []db.MovedRows {
	var out []db.MovedRows
	for _, table := range sharedTables() {
		out = append(out, db.MovedRows{Table: table})
	}
	for _, table := range leagueTables() {
		rows := db.MovedRows{Table: table, Where: "league=" + sqlText(league)}
		if table == "league_event" {
			rows.Order = "rowid" // the audit chain and supersedes links follow the order events were written
		}
		out = append(out, rows)
	}
	return out
}

// CopyLeague fills target, a new league file, with league's rows from the shared file, in one
// transaction: its league named, the shared id tables whole, the league's rows as stored, and a new
// audit chain over its own events in the order they were written.
func CopyLeague(ctx context.Context, shared, target *db.Pools, file LeagueFile) error {
	tx, err := target.BeginWrite(ctx)
	if err != nil {
		return fmt.Errorf("history: begin league %s's file: %w", file.League, err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := claimLeagueFile(ctx, tx, file); err != nil {
		return err
	}
	for _, rows := range leagueRows(file.League) {
		if err := db.CopyRows(ctx, shared.Read(), tx, rows); err != nil {
			return fmt.Errorf("history: league %s: %w", file.League, err)
		}
	}
	if err := chainLedger(ctx, tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("history: commit league %s's file: %w", file.League, err)
	}
	return nil
}

// chainLedger appends every event of a freshly copied file to its (empty) audit chain, in order.
func chainLedger(ctx context.Context, tx *sql.Tx) error {
	rows, err := tx.QueryContext(ctx, ledgerAuditQuery)
	if err != nil {
		return fmt.Errorf("history: read events to chain: %w", err)
	}
	defer func() { _ = rows.Close() }()
	type entry struct{ key, payload string }
	var entries []entry
	for rows.Next() {
		var e entry
		if err := rows.Scan(&e.key, &e.payload); err != nil {
			return fmt.Errorf("history: read events to chain: %w", err)
		}
		entries = append(entries, e)
	}
	if err := errors.Join(rows.Err(), rows.Close()); err != nil {
		return fmt.Errorf("history: read events to chain: %w", err)
	}
	for _, e := range entries {
		if err := db.AppendAudit(ctx, tx, ledgerAuditSource, e.key, e.payload); err != nil {
			return fmt.Errorf("history: chain the league's events: %w", err)
		}
	}
	return nil
}

// ProveLeague checks target holds exactly league's rows of the shared file, table by table (count and
// ordered-row digest), names its league, and has a whole audit chain over all its events.
// It returns the proofs it checked, table by table.
func ProveLeague(ctx context.Context, shared, target *db.Pools, league string) (map[string]db.RowProof, error) {
	want, err := LeagueProofs(ctx, shared, league, true)
	if err != nil {
		return nil, err
	}
	got, err := LeagueProofs(ctx, target, league, false)
	if err != nil {
		return nil, err
	}
	for _, rows := range leagueRows(league) {
		if got[rows.Table] != want[rows.Table] {
			return nil, fmt.Errorf("history: league %s's %s does not match the shared file (%d rows, want %d)", league, rows.Table, got[rows.Table].Rows, want[rows.Table].Rows)
		}
	}
	file, err := ReadLeagueFile(ctx, target)
	if err != nil {
		return nil, err
	}
	if file.League != league {
		return nil, fmt.Errorf("history: the file built for league %s names league %s", league, file.League)
	}
	head, err := New(target, nil).VerifyAudit(ctx)
	if err != nil {
		return nil, err
	}
	var events int64
	if err := target.Read().QueryRowContext(ctx, `SELECT count(*) FROM main.league_event`).Scan(&events); err != nil {
		return nil, fmt.Errorf("history: count league %s's events: %w", league, err)
	}
	if head.Size != events || head.Baseline != 0 {
		return nil, fmt.Errorf("history: league %s's audit chain covers %d of %d events", league, head.Size-head.Baseline, events)
	}
	return got, nil
}

// LeagueProofs is each of league's tables' row count and ordered-row digest: in the shared file
// (inShared) only league's rows, in a league file the whole table.
func LeagueProofs(ctx context.Context, pools *db.Pools, league string, inShared bool) (map[string]db.RowProof, error) {
	out := map[string]db.RowProof{}
	for _, rows := range leagueRows(league) {
		m := db.MovedRows{Table: rows.Table}
		if inShared {
			m.Where = rows.Where
		}
		proof, err := db.ProveRows(ctx, pools.Read(), m)
		if err != nil {
			return nil, fmt.Errorf("history: %w", err)
		}
		out[rows.Table] = proof
	}
	return out, nil
}

// splitDrops are the shared file's league tables, dropped by the split commit (views first; dropping a
// table drops its triggers).
func splitDrops() []string {
	drops := []string{"DROP VIEW IF EXISTS main.transfer_current"}
	for _, table := range append(append(leagueTables(), sharedTables()...), "transfer_identity", "audit_chain", "audit_chain_head") {
		drops = append(drops, "DROP TABLE IF EXISTS main."+table)
	}
	return drops
}

// CommitLeagueSplit is the point of no return. In one transaction of the shared file it checks the
// shared chain still ends at split.Shared, drops the league tables and records the split.
func CommitLeagueSplit(ctx context.Context, shared *db.Pools, split LeagueSplit) error {
	return commitSplit(ctx, shared, split, true)
}

// RecordLeagueSplit records split in a new, empty shared file (a restored server's server.db), whose
// own chain is empty: the closed head it names comes from the restored league files.
func RecordLeagueSplit(ctx context.Context, shared *db.Pools, split LeagueSplit) error {
	return commitSplit(ctx, shared, split, false)
}

func commitSplit(ctx context.Context, shared *db.Pools, split LeagueSplit, verify bool) error {
	leagues, err := json.Marshal(split.Leagues)
	if err != nil {
		return fmt.Errorf("history: encode the split: %w", err)
	}
	head, err := json.Marshal(split.Shared)
	if err != nil {
		return fmt.Errorf("history: encode the split: %w", err)
	}
	proofs := []byte("{}")
	if len(split.Proofs) > 0 {
		if proofs, err = json.Marshal(split.Proofs); err != nil {
			return fmt.Errorf("history: encode the split: %w", err)
		}
	}
	tx, err := shared.BeginWrite(ctx)
	if err != nil {
		return fmt.Errorf("history: begin the split: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	now, err := db.VerifyAudit(ctx, tx, ledgerAuditSource, ledgerAuditQuery)
	if err != nil {
		return fmt.Errorf("history: verify the shared audit chain: %w", err)
	}
	if verify && now != split.Shared {
		return fmt.Errorf("history: the shared file changed while its leagues were being split; nothing was changed")
	}
	for _, drop := range splitDrops() {
		if _, err := tx.ExecContext(ctx, drop); err != nil {
			return fmt.Errorf("history: drop the shared league tables: %w", err)
		}
	}
	if _, err := tx.ExecContext(ctx, `CREATE TABLE main.`+LeagueSplitTable+` (
 singleton INTEGER PRIMARY KEY CHECK(singleton=1),
 leagues TEXT NOT NULL CHECK(json_valid(leagues)),
 shared TEXT NOT NULL CHECK(json_valid(shared)),
 split_at TEXT NOT NULL,
 proofs TEXT NOT NULL CHECK(json_valid(proofs))
) STRICT;
CREATE TRIGGER league_split_no_update BEFORE UPDATE ON `+LeagueSplitTable+` BEGIN SELECT RAISE(ABORT,'the split is recorded once'); END;
CREATE TRIGGER league_split_no_delete BEFORE DELETE ON `+LeagueSplitTable+` BEGIN SELECT RAISE(ABORT,'the split is recorded once'); END;`); err != nil {
		return fmt.Errorf("history: record the split: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO main.`+LeagueSplitTable+` VALUES(1,?,?,?,?)`, string(leagues), string(head), split.At.UTC().Format(time.RFC3339Nano), string(proofs)); err != nil {
		return fmt.Errorf("history: record the split: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("history: commit the split: %w", err)
	}
	return nil
}

// ReadLeagueSplit reads the shared file's split record; ok is false before the split.
func ReadLeagueSplit(ctx context.Context, shared *db.Pools) (LeagueSplit, bool, error) {
	found, err := db.HasTable(ctx, shared, LeagueSplitTable)
	if err != nil || !found {
		return LeagueSplit{}, false, err //nolint:wrapcheck // db names the file
	}
	var leagues, head, at, proofs string
	if err := shared.Read().QueryRowContext(ctx, `SELECT leagues, shared, split_at, proofs FROM main.`+LeagueSplitTable+` WHERE singleton=1`).Scan(&leagues, &head, &at, &proofs); err != nil {
		return LeagueSplit{}, true, fmt.Errorf("history: read the split: %w", err)
	}
	var out LeagueSplit
	if err := json.Unmarshal([]byte(leagues), &out.Leagues); err != nil {
		return out, true, fmt.Errorf("history: read the split: %w", err)
	}
	if err := json.Unmarshal([]byte(head), &out.Shared); err != nil {
		return out, true, fmt.Errorf("history: read the split: %w", err)
	}
	if err := json.Unmarshal([]byte(proofs), &out.Proofs); err != nil {
		return out, true, fmt.Errorf("history: read the split: %w", err)
	}
	when, err := time.Parse(time.RFC3339Nano, at)
	if err != nil {
		return out, true, fmt.Errorf("history: read the split: %w", err)
	}
	out.At = when
	return out, true, nil
}
