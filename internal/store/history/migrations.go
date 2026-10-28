package history

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/db"
)

// SchemaVersion is the newest file migration this build can read.
func SchemaVersion() int { return len(historyMigrations()) }

func historyMigrations() []db.Migration {
	return []db.Migration{adoptHistory, quarantineOrphans, guardHistoryJSON, addClosedBooks, initializeLedgerAudit, addInactiveImports, addTransferAuthority, db.CreateServerRelease, addRosterRestorePoints, addPlayerIdentity, addPlayerRecord, db.AddPlayerCareer}
}

func addPlayerRecord(ctx context.Context, tx *sql.Tx) error {
	if err := db.CreatePlayerRecord(ctx, tx); err != nil {
		return fmt.Errorf("history: %w", err)
	}
	return nil
}

func adoptHistory(ctx context.Context, tx *sql.Tx) error {
	var ddl strings.Builder
	ddl.WriteString(historyDDL)
	ddl.WriteString(eventsDDL)
	for _, table := range appendOnly() {
		ddl.WriteString(appendOnlyDDL(table))
	}
	ddl.WriteString(registryDDL)
	ddl.WriteString(featureViewsDDL())
	if _, err := tx.ExecContext(ctx, ddl.String()); err != nil {
		return fmt.Errorf("history: adopt schema: %w", err)
	}
	return nil
}

func appendOnlyDDL(table string) string {
	var ddl strings.Builder
	for _, op := range []string{"UPDATE", "DELETE"} {
		fmt.Fprintf(&ddl, `
CREATE TRIGGER IF NOT EXISTS %[1]s_no_%[2]s BEFORE %[2]s ON %[1]s
BEGIN SELECT RAISE(ABORT, '%[1]s is append-only: %[2]s forbidden'); END;`, table, strings.ToLower(op))
	}
	return ddl.String()
}

const quarantineDDL = `
CREATE TABLE quarantine_model_runs (
 run_id INTEGER PRIMARY KEY,
 kind TEXT NOT NULL CHECK (kind='model'),
 season INTEGER NOT NULL,
 as_of TEXT NOT NULL,
 param_set_id TEXT NOT NULL,
 engine TEXT NOT NULL,
 inputs_hash TEXT NOT NULL,
 scores_hash TEXT NOT NULL,
 missing_measures TEXT NOT NULL,
 created_at TEXT NOT NULL,
 reason TEXT NOT NULL,
 found_at TEXT NOT NULL
);
CREATE TABLE quarantine_model_scores (
 run_id INTEGER NOT NULL,
 mfl_id TEXT NOT NULL,
 position TEXT NOT NULL,
 now REAL NOT NULL,
 now_ppg REAL NOT NULL,
 dynasty REAL NOT NULL,
 dynasty_ppg REAL NOT NULL,
 prior REAL NOT NULL,
 past_games REAL NOT NULL,
 season_games REAL NOT NULL,
 z_past REAL NOT NULL,
 z_now REAL NOT NULL,
 on_field REAL NOT NULL,
 inputs TEXT NOT NULL,
 reason TEXT NOT NULL,
 found_at TEXT NOT NULL,
 PRIMARY KEY (run_id,mfl_id)
);`

func quarantineOrphans(ctx context.Context, tx *sql.Tx) error {
	findings, err := db.CheckForeignKeys(ctx, tx)
	if err != nil {
		return fmt.Errorf("history: find orphan models: %w", err)
	}
	if _, err := tx.ExecContext(ctx, quarantineDDL); err != nil {
		return fmt.Errorf("history: quarantine schema: %w", err)
	}
	// Only the versioned repair can remove evidence; rollback restores these triggers too.
	if _, err := tx.ExecContext(ctx, `DROP TRIGGER model_runs_no_delete; DROP TRIGGER model_scores_no_delete`); err != nil {
		return fmt.Errorf("history: prepare quarantine: %w", err)
	}
	for _, finding := range findings {
		if !finding.RowID.Valid {
			continue
		}
		switch {
		case finding.Table == "model_runs" && finding.Parent == "param_sets":
			if err := quarantineRun(ctx, tx, finding.RowID.Int64); err != nil {
				return err
			}
		case finding.Table == "model_scores" && finding.Parent == "model_runs":
			if err := quarantineScore(ctx, tx, finding.RowID.Int64); err != nil {
				return err
			}
		}
	}
	if _, err := tx.ExecContext(ctx, appendOnlyDDL("model_runs")+appendOnlyDDL("model_scores")+appendOnlyDDL("quarantine_model_runs")+appendOnlyDDL("quarantine_model_scores")); err != nil {
		return fmt.Errorf("history: protect quarantine: %w", err)
	}
	return nil
}

func quarantineRun(ctx context.Context, tx *sql.Tx, rowID int64) error {
	const reason = "model_runs.param_set_id references missing param_sets"
	if _, err := tx.ExecContext(ctx, `INSERT INTO quarantine_model_runs SELECT *, ?, datetime('now') FROM model_runs WHERE rowid=?`, reason, rowID); err != nil {
		return fmt.Errorf("history: quarantine model run: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO quarantine_model_scores SELECT *, ?, datetime('now') FROM model_scores WHERE run_id=?`, reason, rowID); err != nil {
		return fmt.Errorf("history: quarantine dependent scores: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM model_scores WHERE run_id=?`, rowID); err != nil {
		return fmt.Errorf("history: remove quarantined scores: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM model_runs WHERE rowid=?`, rowID); err != nil {
		return fmt.Errorf("history: remove quarantined run: %w", err)
	}
	return nil
}

func quarantineScore(ctx context.Context, tx *sql.Tx, rowID int64) error {
	const reason = "model_scores.run_id references missing model_runs"
	if _, err := tx.ExecContext(ctx, `INSERT INTO quarantine_model_scores SELECT *, ?, datetime('now') FROM model_scores WHERE rowid=?`, reason, rowID); err != nil {
		return fmt.Errorf("history: quarantine orphan score: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM model_scores WHERE rowid=?`, rowID); err != nil {
		return fmt.Errorf("history: remove orphan score: %w", err)
	}
	return nil
}

func guardHistoryJSON(ctx context.Context, tx *sql.Tx) error {
	if err := db.GuardJSON(ctx, tx,
		db.JSONColumn{Table: "league_event", Column: "payload", Versioned: true},
		db.JSONColumn{Table: "param_sets", Column: "measures"},
		db.JSONColumn{Table: "param_sets", Column: "params"},
		db.JSONColumn{Table: "scoring_runs", Column: "missing_measures"},
		db.JSONColumn{Table: "model_runs", Column: "missing_measures"},
		db.JSONColumn{Table: "model_scores", Column: "inputs"},
		db.JSONColumn{Table: "input_manifests", Column: "entries"},
		db.JSONColumn{Table: "snapshot_checks", Column: "checks"},
	); err != nil {
		return fmt.Errorf("history: JSON guards: %w", err)
	}
	return nil
}

func addPlayerIdentity(ctx context.Context, tx *sql.Tx) error {
	if err := db.CreatePlayerIdentity(ctx, tx); err != nil {
		return fmt.Errorf("history: migrate player identities: %w", err)
	}
	if err := db.GuardPlayerEvidence(ctx, tx); err != nil {
		return fmt.Errorf("history: guard player evidence: %w", err)
	}
	rows, err := tx.QueryContext(ctx, `SELECT league FROM league_roster_restore ORDER BY league`)
	if err != nil {
		return fmt.Errorf("find identity rosters: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var leagues []string
	for rows.Next() {
		var league string
		if err := rows.Scan(&league); err != nil {
			return fmt.Errorf("read identity roster: %w", err)
		}
		leagues = append(leagues, league)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read identity rosters: %w", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close identity rosters: %w", err)
	}
	for _, league := range leagues {
		if err := (&Store{}).captureRosterRestorePoint(ctx, tx, league); err != nil {
			return err
		}
	}
	return nil
}
