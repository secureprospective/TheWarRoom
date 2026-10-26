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

// ReadLeagueRestoreFile reads an existing snapshot without migrating it or
// acquiring a writer. The source can be a decrypted R2 server.db or desktop history.db.
func ReadLeagueRestoreFile(ctx context.Context, path, league string, maxVersion int) (string, error) {
	source, err := sql.Open("sqlite", fileURI(path)+"?mode=ro")
	if err != nil {
		return "", fmt.Errorf("open roster recovery source: %w", err)
	}
	defer func() { _ = source.Close() }()
	tx, err := source.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("read roster recovery source: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var version int
	if err := tx.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&version); err != nil {
		return "", fmt.Errorf("read recovery schema version: %w", err)
	}
	if err := refuseNewer(version, maxVersion); err != nil {
		return "", err
	}
	var revision int64
	var body string
	if err := tx.QueryRowContext(ctx, `SELECT revision,payload FROM league_roster_restore WHERE league=? AND length(CAST(payload AS BLOB))<=33554432`, league).Scan(&revision, &body); err != nil {
		return "", fmt.Errorf("read league restore point (maximum 32 MiB): %w", err)
	}
	var matches bool
	if err := tx.QueryRowContext(ctx, `SELECT json_extract(?1,'$.league')=?2 AND json_extract(?1,'$.revision')=?3`, body, league, revision).Scan(&matches); err != nil {
		return "", fmt.Errorf("check recovery source identity: %w", err)
	}
	if !matches {
		return "", fmt.Errorf("restore point does not match its stored league or revision")
	}
	return body, nil
}
