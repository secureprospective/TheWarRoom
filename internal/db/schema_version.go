package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// MigrateSchema applies one named, forward-only migration list in a file whose user_version is
// owned by another list. Its version lives in schema_version under name; each step and its marker
// commit together and a newer version is refused, exactly as Migrate does for user_version. Version
// zero is adoption: the first step must be idempotent DDL so a file already holding the tables keeps
// its rows. Steps after adoption are preceded by the same pre-migration backup Migrate takes.
func (p *Pools) MigrateSchema(ctx context.Context, name string, steps ...Migration) error {
	if name == "" || len(steps) == 0 {
		return fmt.Errorf("db: schema migration needs a name and steps")
	}
	p.migrationMu.Lock()
	defer p.migrationMu.Unlock()
	version, err := p.schemaVersion(ctx, name)
	if err != nil {
		return err
	}
	if err := refuseNewer(version, len(steps)); err != nil {
		return fmt.Errorf("%s schema: %w", name, err)
	}
	if version > 0 && version < len(steps) && p.scratchDir == "" {
		if _, err := p.backupMigration(ctx, version); err != nil {
			return err
		}
	}
	for next := version + 1; next <= len(steps); next++ {
		if err := p.migrateSchemaStep(ctx, name, next, steps[next-1]); err != nil {
			return err
		}
	}
	return nil
}

func (p *Pools) schemaVersion(ctx context.Context, name string) (int, error) {
	var version int
	err := p.read.QueryRowContext(ctx, `SELECT version FROM schema_version WHERE name=?`, name).Scan(&version)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		var held int
		if probe := p.read.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_schema WHERE type='table' AND name='schema_version'`).Scan(&held); probe == nil && held == 0 {
			return 0, nil
		}
		return 0, fmt.Errorf("db: %s schema version: %w", name, err)
	}
	return version, nil
}

func (p *Pools) migrateSchemaStep(ctx context.Context, name string, version int, step Migration) error {
	tx, err := p.write.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("db: %s migration v%d begin: %w", name, version, err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_version (
 name TEXT PRIMARY KEY NOT NULL, version INTEGER NOT NULL CHECK(version > 0)) STRICT`); err != nil {
		return fmt.Errorf("db: %s migration v%d marker table: %w", name, version, err)
	}
	if err := step(ctx, tx); err != nil {
		return fmt.Errorf("db: %s migration v%d: %w", name, version, err)
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO schema_version(name, version) VALUES (?, ?)
 ON CONFLICT(name) DO UPDATE SET version=excluded.version WHERE version=excluded.version-1`, name, version)
	if err == nil {
		var n int64
		if n, err = result.RowsAffected(); err == nil && n != 1 {
			err = fmt.Errorf("version moved underneath")
		}
	}
	if err != nil {
		return fmt.Errorf("db: %s migration v%d stamp: %w", name, version, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("db: %s migration v%d commit: %w", name, version, err)
	}
	return nil
}
