package db

import (
	"context"
	"database/sql"
	"fmt"
)

// Migration is one forward-only file version. Its writes and version marker commit together.
type Migration func(context.Context, *sql.Tx) error

// ForeignKeyFinding is one row returned by SQLite's foreign_key_check.
type ForeignKeyFinding struct {
	Table      string
	RowID      sql.NullInt64
	Parent     string
	Constraint int
}

// Findings retains checks before and after migration, including orphans a repair moved aside.
type Findings struct {
	Before CheckFindings
	After  CheckFindings
	Backup string
}

// CheckFindings contains SQLite's findings, not a synthesized pass/fail verdict.
type CheckFindings struct {
	Integrity   []string
	ForeignKeys []ForeignKeyFinding
}

// Migrate backs up before any pending step, then commits each step and marker atomically.
// Findings are returned even when pre-existing foreign-key violations remain.
func (p *Pools) Migrate(ctx context.Context, steps ...Migration) (Findings, error) {
	p.migrationMu.Lock()
	defer p.migrationMu.Unlock()
	var result Findings
	var version int
	if err := p.write.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return result, fmt.Errorf("db: migration version: %w", err)
	}
	if err := refuseNewer(version, len(steps)); err != nil {
		return result, err
	}
	before, err := Check(ctx, p.write)
	result.Before = before
	if err != nil {
		return result, err
	}
	if version < len(steps) {
		result.Backup, err = p.backupMigration(ctx, version)
		if err != nil {
			return result, err
		}
	}
	for next := version + 1; next <= len(steps); next++ {
		if err := p.migrateStep(ctx, next, steps[next-1]); err != nil {
			return result, err
		}
	}
	result.After, err = Check(ctx, p.write)
	return result, err
}

func (p *Pools) migrateStep(ctx context.Context, version int, step Migration) error {
	tx, err := p.write.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("db: migration v%d begin: %w", version, err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := step(ctx, tx); err != nil {
		return fmt.Errorf("db: migration v%d: %w", version, err)
	}
	// The version is the registry index, never external SQL.
	if _, err := tx.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", version)); err != nil {
		return fmt.Errorf("db: migration v%d stamp: %w", version, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("db: migration v%d commit: %w", version, err)
	}
	return nil
}

func (p *Pools) backupMigration(ctx context.Context, version int) (string, error) {
	dest := fmt.Sprintf("%s.pre-v%d.bak", p.path, version)
	if _, err := p.write.ExecContext(ctx, "VACUUM INTO ?", dest); err != nil {
		return "", fmt.Errorf("db: pre-migration backup: %w", err)
	}
	return dest, nil
}

func refuseNewer(version, maxVersion int) error {
	if version > maxVersion {
		return fmt.Errorf("db: this database was last written by a NEWER version of TheWarRoom (schema v%d; this build understands up to v%d) — update the app to open it", version, maxVersion)
	}
	return nil
}
