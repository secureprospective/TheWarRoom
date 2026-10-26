package db

import (
	"context"
	"database/sql"
	"fmt"
)

// CreateLeagueRecovery belongs to the history-file migration list. Restore points
// stay inside that file so the existing encrypted offsite snapshot includes them.
func CreateLeagueRecovery(ctx context.Context, tx *sql.Tx) error {
	const ddl = `CREATE TABLE league_roster_restore (
 league TEXT PRIMARY KEY,
 revision INTEGER NOT NULL CHECK(revision > 0),
 payload TEXT NOT NULL CHECK(json_valid(payload) AND json_extract(payload,'$.version')=1)
) STRICT;
CREATE TABLE league_recovery_log (
 sequence INTEGER PRIMARY KEY,
 league TEXT NOT NULL,
 revision INTEGER NOT NULL CHECK(revision > 0),
 recorded_at TEXT NOT NULL,
 message TEXT NOT NULL CHECK(length(message)>0)
) STRICT;
CREATE INDEX recovery_log_by_league ON league_recovery_log(league,sequence);
CREATE TRIGGER recovery_log_no_update BEFORE UPDATE ON league_recovery_log
 BEGIN SELECT RAISE(ABORT,'league recovery log is append-only'); END;
CREATE TRIGGER recovery_log_no_delete BEFORE DELETE ON league_recovery_log
 BEGIN SELECT RAISE(ABORT,'league recovery log is append-only'); END;`
	if _, err := tx.ExecContext(ctx, ddl); err != nil {
		return fmt.Errorf("create league recovery schema: %w", err)
	}
	return nil
}

// SaveLeagueRestorePoint is called only after domain validation, in the league
// change's transaction. A failed point or log write must fail that transaction.
func SaveLeagueRestorePoint(ctx context.Context, tx *sql.Tx, league string, revision int64, at, payload string) error {
	if _, err := tx.ExecContext(ctx, `INSERT INTO league_roster_restore(league,revision,payload) VALUES(?,?,?)
 ON CONFLICT(league) DO UPDATE SET revision=excluded.revision,payload=excluded.payload`, league, revision, payload); err != nil {
		return fmt.Errorf("save league roster restore point: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO league_recovery_log(league,revision,recorded_at,message) VALUES(?,?,?,?)`,
		league, revision, at, "Saved rosters and contracts together. Historical results are not needed to read this restore point."); err != nil {
		return fmt.Errorf("record league recovery log: %w", err)
	}
	return nil
}
