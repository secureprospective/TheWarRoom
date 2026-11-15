package history

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/db"
)

// A hosted league lives in its own file (Christopher, 2026-11-14: "Each league needs its own dedicated,
// segregated database that doesn't mix from one league to the next league"). The file names its league
// once, in league_file, and every table with a league column refuses a row for any other league, so a
// wrong-file write fails in SQLite itself, not only in the resolver. league_file also keeps the final
// head of the one shared audit chain the server had before the split, closed once (ruling 2).

// leagueTables are the league file's tables that carry a league column, in the order a split copies
// them: events before the import receipt and transfer state that guard later events, and the transfer
// state before authority settings, which those triggers also guard.
func leagueTables() []string {
	return []string{"league_event", "league_season_summary", "league_recovery_log", "league_roster_restore",
		"player_ownership", "league_import", "transfer_transition", "authority_policy"}
}

// sharedTables are the league file's tables without a league column that a split copies whole: the
// permanent ids of players any league has owned. (transfer_identity is not copied: each league file
// names itself as a move destination when it is created, so a league move in flight at the split must
// be started again.)
func sharedTables() []string { return []string{"league_player"} }

func addLeagueIdentity(ctx context.Context, tx *sql.Tx) error {
	var ddl strings.Builder
	ddl.WriteString(`CREATE TABLE league_file (
 singleton INTEGER PRIMARY KEY CHECK(singleton=1),
 league TEXT NOT NULL CHECK(length(league)>0),
 shared TEXT NOT NULL CHECK(json_valid(shared))
) STRICT;
CREATE TRIGGER league_file_no_update BEFORE UPDATE ON league_file BEGIN SELECT RAISE(ABORT,'a league file belongs to one league for good'); END;
CREATE TRIGGER league_file_no_delete BEFORE DELETE ON league_file BEGIN SELECT RAISE(ABORT,'a league file belongs to one league for good'); END;
`)
	for _, table := range leagueTables() {
		for _, op := range []string{"INSERT", "UPDATE"} {
			fmt.Fprintf(&ddl, `CREATE TRIGGER %[1]s_own_league_%[2]s BEFORE %[2]s ON %[1]s
 WHEN EXISTS(SELECT 1 FROM league_file WHERE league!=NEW.league)
 BEGIN SELECT RAISE(ABORT,'this database belongs to another league'); END;
`, table, strings.ToLower(op))
		}
	}
	if _, err := tx.ExecContext(ctx, ddl.String()); err != nil {
		return fmt.Errorf("create league identity: %w", err)
	}
	return nil
}

// LeagueFile is what a league file says about itself.
type LeagueFile struct {
	League string
	// Shared is the final head of the shared audit chain the server closed when leagues got their own
	// files (empty on a server that never had one).
	Shared db.AuditHead
}

// ErrNotLeagueFile is a file that does not name its league (the closed shared file, or an unclaimed one).
var ErrNotLeagueFile = errors.New("this file does not belong to a league")

// ReadLeagueFile reads the file's league and the shared head it keeps.
func ReadLeagueFile(ctx context.Context, pools *db.Pools) (LeagueFile, error) {
	found, err := db.HasTable(ctx, pools, "league_file")
	if err != nil {
		return LeagueFile{}, fmt.Errorf("history: %w", err)
	}
	if !found {
		return LeagueFile{}, ErrNotLeagueFile
	}
	var out LeagueFile
	var shared string
	err = pools.Read().QueryRowContext(ctx, `SELECT league, shared FROM main.league_file WHERE singleton=1`).Scan(&out.League, &shared)
	if errors.Is(err, sql.ErrNoRows) {
		return LeagueFile{}, ErrNotLeagueFile
	}
	if err != nil {
		return LeagueFile{}, fmt.Errorf("history: read the league file's league: %w", err)
	}
	if err := json.Unmarshal([]byte(shared), &out.Shared); err != nil {
		return LeagueFile{}, fmt.Errorf("history: read the closed shared audit head: %w", err)
	}
	return out, nil
}

// ClaimLeagueFile names a new, empty league file's league, in tx.
func claimLeagueFile(ctx context.Context, tx *sql.Tx, file LeagueFile) error {
	shared, err := json.Marshal(file.Shared)
	if err != nil {
		return fmt.Errorf("history: encode the closed shared audit head: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO main.league_file(singleton, league, shared) VALUES(1,?,?)`, file.League, string(shared)); err != nil {
		return fmt.Errorf("history: name the league file's league: %w", err)
	}
	return nil
}

// CreateLeagueFile prepares a new league's empty file: the league schema, then its league named.
func CreateLeagueFile(ctx context.Context, pools *db.Pools, file LeagueFile) error {
	if err := migrateLeagueFile(ctx, pools); err != nil {
		return err
	}
	tx, err := pools.BeginWrite(ctx)
	if err != nil {
		return fmt.Errorf("history: begin the new league file: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := claimLeagueFile(ctx, tx, file); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("history: commit the new league file: %w", err)
	}
	return nil
}
