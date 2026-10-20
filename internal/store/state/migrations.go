package state

import (
	"context"
	"database/sql"
	"fmt"
)

// stateOwner is this package's namespace in schema_migrations; each store owns only its own rows.
const stateOwner = "state"

// maxKnownStateVersion is the newest migration this binary knows, taken from the registry. A DB
// stamped newer than this was touched by a newer binary, and opening it is refused: there are no
// down-migrations.
func maxKnownStateVersion() int {
	migs := stateMigrations()
	return migs[len(migs)-1].version
}

// schema_migrations holds one marker per (owner, version). method is 'migrated' for a step that
// ran, or 'reconciled' for one found already in place on an older DB and only stamped.
const schemaMigrationsDDL = `
CREATE TABLE IF NOT EXISTS schema_migrations (
	owner       TEXT    NOT NULL,
	version     INTEGER NOT NULL,
	applied_at  TEXT    NOT NULL DEFAULT (datetime('now')),
	method      TEXT    NOT NULL DEFAULT 'migrated',
	PRIMARY KEY (owner, version)
);`

// migration is one forward-only step. apply is idempotent. isAlreadyApplied inspects the data, so
// an older DB can be stamped without re-running destructive work.
type migration struct {
	version          int
	apply            func(context.Context, *sql.Tx) error
	isAlreadyApplied func(context.Context, *sql.Tx) (bool, error)
}

// stateMigrations is the forward-only registry. Versions are dense and ascending; never renumber or
// reuse one. Append only.
func stateMigrations() []migration {
	return []migration{
		{version: 1, apply: migrateMoneyCents, isAlreadyApplied: moneyCentsApplied},
		{version: 2, apply: dropLegacyMoneyColumns, isAlreadyApplied: legacyColumnsDropped},
		{version: 3, apply: dropContractYearsColumn, isAlreadyApplied: contractYearsColumnDropped},
		{version: 4, apply: dropFeedCaches, isAlreadyApplied: feedCachesDropped},
	}
}

// runMigrations retains the old owner/version history inside the file adoption step.
func runMigrations(ctx context.Context, tx *sql.Tx) error {
	applied, err := appliedStateVersions(ctx, tx)
	if err != nil {
		return err
	}
	for _, m := range stateMigrations() {
		if applied[m.version] {
			continue
		}
		already, err := m.isAlreadyApplied(ctx, tx)
		if err != nil {
			return err
		}
		method := "reconciled"
		if !already {
			if err := m.apply(ctx, tx); err != nil {
				return err
			}
			method = "migrated"
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (owner,version,method) VALUES (?,?,?)`, stateOwner, m.version, method); err != nil {
			return fmt.Errorf("state: stamp migration v%d: %w", m.version, err)
		}
	}
	return nil
}

type schemaQuerier interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func appliedStateVersions(ctx context.Context, tx *sql.Tx) (map[int]bool, error) {
	rows, err := tx.QueryContext(ctx, `SELECT version FROM schema_migrations WHERE owner = ?`, stateOwner)
	if err != nil {
		return nil, fmt.Errorf("state: read schema_migrations: %w", err)
	}
	defer func() { _ = rows.Close() }()
	applied := make(map[int]bool)
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return nil, fmt.Errorf("state: scan schema_migrations: %w", err)
		}
		applied[version] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("state: schema_migrations rows: %w", err)
	}
	return applied, nil
}

func checkNoDowngrade(ctx context.Context, query schemaQuerier) error {
	var exists int
	if err := query.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='schema_migrations'`).Scan(&exists); err != nil {
		return fmt.Errorf("state: look for schema_migrations: %w", err)
	}
	if exists == 0 {
		return nil
	}
	var maxV int
	if err := query.QueryRowContext(ctx, `SELECT COALESCE(MAX(version),0) FROM schema_migrations WHERE owner=?`, stateOwner).Scan(&maxV); err != nil {
		return fmt.Errorf("state: read max migration version: %w", err)
	}
	if known := maxKnownStateVersion(); maxV > known {
		return fmt.Errorf("state: this database was last written by a NEWER version of TheWarRoom (schema state v%d; this build understands up to v%d) — update the app to open it", maxV, known)
	}
	return nil
}

// moneyCentsApplied is v1's check. No cents column: not applied. Cents present and the REAL column
// gone: applied (the migration is atomic). Both present: verify every cent round-trips before
// calling it applied.
func moneyCentsApplied(ctx context.Context, tx *sql.Tx) (bool, error) {
	haveCents, err := columnExists(ctx, tx, "contracts", "annual_salary_cents")
	if err != nil {
		return false, err
	}
	if !haveCents {
		return false, nil
	}
	haveLegacy, err := columnExists(ctx, tx, "contracts", "annual_salary")
	if err != nil {
		return false, err
	}
	if !haveLegacy {
		return true, nil
	}
	var bad int
	if err := tx.QueryRowContext(ctx, `
SELECT COUNT(1) FROM contracts
WHERE ABS(annual_salary_cents / 100000000.0 - annual_salary) > 0.000000005`).Scan(&bad); err != nil {
		return false, fmt.Errorf("state: money-cents reconcile verify: %w", err)
	}
	return bad == 0, nil
}

// legacyColumnsDropped is v2's check: applied once none of the old money columns remain.
func legacyColumnsDropped(ctx context.Context, tx *sql.Tx) (bool, error) {
	for _, col := range []string{"annual_salary", "adjusted_salary", "adjusted_salary_cents"} {
		have, err := columnExists(ctx, tx, "contracts", col)
		if err != nil {
			return false, err
		}
		if have {
			return false, nil
		}
	}
	return true, nil
}
