package state

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
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
	apply            func(*Store, context.Context) error
	isAlreadyApplied func(*Store, context.Context) (bool, error)
}

// stateMigrations is the forward-only registry. Versions are dense and ascending; never renumber or
// reuse one. Append only.
func stateMigrations() []migration {
	return []migration{
		{version: 1, apply: (*Store).migrateMoneyCents, isAlreadyApplied: (*Store).moneyCentsApplied},
		{version: 2, apply: (*Store).dropLegacyMoneyColumns, isAlreadyApplied: (*Store).legacyColumnsDropped},
		{version: 3, apply: (*Store).dropContractYearsColumn, isAlreadyApplied: (*Store).contractYearsColumnDropped},
	}
}

// runMigrations refuses a downgrade, then for each unstamped version either stamps it (already in
// place) or runs it, taking one VACUUM INTO backup first if any real work is pending. The backup is
// the rollback. It runs after all DDL, so every table the checks read exists.
func (s *Store) runMigrations(ctx context.Context) error {
	if _, err := s.pools.Write().ExecContext(ctx, schemaMigrationsDDL); err != nil {
		return fmt.Errorf("state: create schema_migrations: %w", err)
	}
	if err := s.checkNoDowngrade(ctx); err != nil {
		return err
	}
	applied, err := s.appliedStateVersions(ctx)
	if err != nil {
		return err
	}

	type step struct {
		m       migration
		already bool
	}
	var todo []step
	needBackup := false
	for _, m := range stateMigrations() {
		if applied[m.version] {
			continue
		}
		already, aerr := m.isAlreadyApplied(s, ctx)
		if aerr != nil {
			return aerr
		}
		if !already {
			needBackup = true
		}
		todo = append(todo, step{m: m, already: already})
	}
	if len(todo) == 0 {
		return nil
	}
	if needBackup {
		if err := s.backupBeforeMigration(ctx); err != nil {
			return err
		}
	}
	for _, st := range todo {
		// Already in place: stamp without running apply.
		method := "reconciled"
		if !st.already {
			if err := st.m.apply(s, ctx); err != nil {
				return err
			}
			method = "migrated"
		}
		if err := s.stampMigration(ctx, st.m.version, method); err != nil {
			return err
		}
	}
	return nil
}

// appliedStateVersions returns the stamped versions.
func (s *Store) appliedStateVersions(ctx context.Context) (map[int]bool, error) {
	rows, err := s.pools.Read().QueryContext(ctx,
		`SELECT version FROM schema_migrations WHERE owner = ?`, stateOwner)
	if err != nil {
		return nil, fmt.Errorf("state: read schema_migrations: %w", err)
	}
	defer func() { _ = rows.Close() }()
	applied := make(map[int]bool)
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("state: scan schema_migrations: %w", err)
		}
		applied[v] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("state: schema_migrations rows: %w", err)
	}
	return applied, nil
}

// checkNoDowngrade refuses a DB stamped newer than this binary; the fix is to update the app.
func (s *Store) checkNoDowngrade(ctx context.Context) error {
	var maxV int // NULL on a fresh DB
	if err := s.pools.Read().QueryRowContext(ctx,
		`SELECT COALESCE(MAX(version), 0) FROM schema_migrations WHERE owner = ?`,
		stateOwner).Scan(&maxV); err != nil {
		return fmt.Errorf("state: read max migration version: %w", err)
	}
	if known := maxKnownStateVersion(); maxV > known {
		return fmt.Errorf(
			"state: this database was last written by a NEWER version of TheWarRoom "+
				"(schema state v%d; this build understands up to v%d) — update the app to open it",
			maxV, known)
	}
	return nil
}

// stampMigration records a version as applied.
func (s *Store) stampMigration(ctx context.Context, version int, method string) error {
	if _, err := s.pools.Write().ExecContext(ctx,
		`INSERT INTO schema_migrations (owner, version, method) VALUES (?, ?, ?)`,
		stateOwner, version, method); err != nil {
		return fmt.Errorf("state: stamp migration v%d: %w", version, err)
	}
	return nil
}

// backupBeforeMigration checkpoints the WAL and VACUUMs INTO a sibling file. Never an OS copy:
// WAL state spans three files and a copy can tear. Keeps the newest 3; a no-op for an in-memory
// DB.
func (s *Store) backupBeforeMigration(ctx context.Context) error {
	if _, err := s.pools.Write().ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		return fmt.Errorf("state: pre-migration checkpoint: %w", err)
	}
	path, err := s.mainDBPath(ctx)
	if err != nil {
		return err
	}
	if path == "" {
		return nil // in-memory: nothing to back up
	}
	dest := fmt.Sprintf("%s.premigration-%s", path, time.Now().UTC().Format("20060102T150405Z"))
	// VACUUM INTO takes a literal, not a parameter; dest is our own path, quote-escaped.
	lit := "'" + strings.ReplaceAll(dest, "'", "''") + "'"
	if _, err := s.pools.Write().ExecContext(ctx, "VACUUM INTO "+lit); err != nil {
		return fmt.Errorf("state: pre-migration backup: %w", err)
	}
	return pruneBackups(path)
}

// mainDBPath returns the file behind the main schema (PRAGMA database_list), or "" in memory.
func (s *Store) mainDBPath(ctx context.Context) (string, error) {
	rows, err := s.pools.Read().QueryContext(ctx, `PRAGMA database_list`)
	if err != nil {
		return "", fmt.Errorf("state: database_list: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var seq int
		var name, file string
		if err := rows.Scan(&seq, &name, &file); err != nil {
			return "", fmt.Errorf("state: database_list scan: %w", err)
		}
		if name == "main" {
			return file, nil
		}
	}
	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("state: database_list rows: %w", err)
	}
	return "", nil
}

// pruneBackups keeps the newest 3 snapshots; fixed-width UTC suffixes sort chronologically.
func pruneBackups(dbPath string) error {
	matches, err := filepath.Glob(dbPath + ".premigration-*")
	if err != nil {
		return fmt.Errorf("state: list pre-migration backups: %w", err)
	}
	if len(matches) <= 3 {
		return nil
	}
	sort.Strings(matches)
	for _, old := range matches[:len(matches)-3] {
		if err := os.Remove(old); err != nil {
			return fmt.Errorf("state: prune old backup %s: %w", old, err)
		}
	}
	return nil
}

// moneyCentsApplied is v1's check. No cents column: not applied. Cents present and the REAL column
// gone: applied (the migration is atomic). Both present: verify every cent round-trips before
// calling it applied.
func (s *Store) moneyCentsApplied(ctx context.Context) (bool, error) {
	haveCents, err := s.columnExists(ctx, "contracts", "annual_salary_cents")
	if err != nil {
		return false, err
	}
	if !haveCents {
		return false, nil
	}
	haveLegacy, err := s.columnExists(ctx, "contracts", "annual_salary")
	if err != nil {
		return false, err
	}
	if !haveLegacy {
		return true, nil
	}
	var bad int
	if err := s.pools.Read().QueryRowContext(ctx, `
SELECT COUNT(1) FROM contracts
WHERE ABS(annual_salary_cents / 100000000.0 - annual_salary) > 0.000000005`).Scan(&bad); err != nil {
		return false, fmt.Errorf("state: money-cents reconcile verify: %w", err)
	}
	return bad == 0, nil
}

// legacyColumnsDropped is v2's check: applied once none of the old money columns remain.
func (s *Store) legacyColumnsDropped(ctx context.Context) (bool, error) {
	for _, col := range []string{"annual_salary", "adjusted_salary", "adjusted_salary_cents"} {
		have, err := s.columnExists(ctx, "contracts", col)
		if err != nil {
			return false, err
		}
		if have {
			return false, nil
		}
	}
	return true, nil
}
