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

// stateOwner is this package's migration namespace in schema_migrations. The three schema
// owners are disjoint (D-V6) — state is the only one with migrations, so it owns exactly
// its own rows and never reads or writes another owner's. Adding a fourth store package
// later costs one new owner string and zero coupling here.
const stateOwner = "state"

// maxKnownStateVersion is the highest migration version THIS binary understands. If a DB
// carries a state row newer than this, the binary is older than the data that touched it
// and refuses to open (checkNoDowngrade) — the two machines (CT105 + Beelink gate clone)
// will drift, and a forward-only ledger has no down-migration to recover with. Bump this
// in lockstep with every migration appended to stateMigrations.
const maxKnownStateVersion = 2

// schemaMigrationsDDL is the per-(owner, version) marker table (D-V6). No central
// coordinator, no PRAGMA user_version. Each owner creates it idempotently, reads its own
// rows, runs its own pending migrations. method distinguishes a migration we actually RAN
// ('migrated') from one we found already in its target state on a pre-marker DB and merely
// stamped ('reconciled').
const schemaMigrationsDDL = `
CREATE TABLE IF NOT EXISTS schema_migrations (
	owner       TEXT    NOT NULL,
	version     INTEGER NOT NULL,
	applied_at  TEXT    NOT NULL DEFAULT (datetime('now')),
	method      TEXT    NOT NULL DEFAULT 'migrated',
	PRIMARY KEY (owner, version)
);`

// migration is one forward-only schema step. apply performs it (idempotent — a no-op if the
// DB is already in the target state, so it is safe to invoke in the reconcile path too).
// isAlreadyApplied is a DATA predicate: it inspects the DB's actual state to decide whether
// this migration's effect is already present, so a pre-marker DB can be stamped without
// re-running work that would otherwise be destructive or wasteful.
type migration struct {
	version          int
	apply            func(*Store, context.Context) error
	isAlreadyApplied func(*Store, context.Context) (bool, error)
}

// stateMigrations is the ordered, forward-only migration registry. Versions are dense and
// ascending; NEVER renumber or reuse a version — the marker rows are permanent. Append only.
// v1/v2 wrap the two pre-existing migrations (money-cents, drop-legacy-columns), whose bodies
// are unchanged in schema.go; the registry adds marker tracking + a pre-migration backup.
func stateMigrations() []migration {
	return []migration{
		{version: 1, apply: (*Store).migrateMoneyCents, isAlreadyApplied: (*Store).moneyCentsApplied},
		{version: 2, apply: (*Store).dropLegacyMoneyColumns, isAlreadyApplied: (*Store).legacyColumnsDropped},
	}
}

// runMigrations brings the state schema forward: create the marker table, refuse a
// downgrade, then for every not-yet-stamped version decide reconcile-vs-run from a data
// predicate, take ONE VACUUM INTO backup if any real work is pending, apply in order, and
// stamp each. Forward-only — the backup is the rollback (D-V6). Called from initSchema
// AFTER the base + per-feature DDL, so every table the predicates inspect already exists.
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
			needBackup = true // this version will do real, irreversible work
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
		if err := st.m.apply(s, ctx); err != nil {
			return err
		}
		method := "migrated"
		if st.already {
			method = "reconciled"
		}
		if err := s.stampMigration(ctx, st.m.version, method); err != nil {
			return err
		}
	}
	return nil
}

// appliedStateVersions returns the set of already-stamped state migration versions.
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

// checkNoDowngrade refuses to open a DB whose highest state migration exceeds what this
// binary knows how to run. Plain-language message: the fix is to update the app, not the DB.
func (s *Store) checkNoDowngrade(ctx context.Context) error {
	var maxV int // MAX over zero rows is NULL → coalesce to 0 (fresh DB)
	if err := s.pools.Read().QueryRowContext(ctx,
		`SELECT COALESCE(MAX(version), 0) FROM schema_migrations WHERE owner = ?`,
		stateOwner).Scan(&maxV); err != nil {
		return fmt.Errorf("state: read max migration version: %w", err)
	}
	if maxV > maxKnownStateVersion {
		return fmt.Errorf(
			"state: this database was last written by a NEWER version of TheWarRoom "+
				"(schema state v%d; this build understands up to v%d) — update the app to open it",
			maxV, maxKnownStateVersion)
	}
	return nil
}

// stampMigration records a version as applied. applied_at defaults to datetime('now').
func (s *Store) stampMigration(ctx context.Context, version int, method string) error {
	if _, err := s.pools.Write().ExecContext(ctx,
		`INSERT INTO schema_migrations (owner, version, method) VALUES (?, ?, ?)`,
		stateOwner, version, method); err != nil {
		return fmt.Errorf("state: stamp migration v%d: %w", version, err)
	}
	return nil
}

// backupBeforeMigration writes a fully self-contained snapshot of the live DB just before a
// migration runs. It checkpoints the WAL back into the main file, then VACUUM INTO a sibling
// file — NEVER an OS copy, because WAL state spans .db + .db-wal + .db-shm and a plain copy
// would capture a torn picture. Keeps the newest 3 snapshots. On an anonymous/in-memory DB
// (no backing file) there is nothing to snapshot and this is a no-op.
func (s *Store) backupBeforeMigration(ctx context.Context) error {
	if _, err := s.pools.Write().ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		return fmt.Errorf("state: pre-migration checkpoint: %w", err)
	}
	path, err := s.mainDBPath(ctx)
	if err != nil {
		return err
	}
	if path == "" {
		return nil // anonymous/in-memory DB — nothing to back up
	}
	dest := fmt.Sprintf("%s.premigration-%s", path, time.Now().UTC().Format("20060102T150405Z"))
	// dest is a filesystem path, not caller input; VACUUM INTO takes a string-literal filename
	// (not a bindable parameter), so it is embedded with standard SQL single-quote escaping.
	lit := "'" + strings.ReplaceAll(dest, "'", "''") + "'"
	if _, err := s.pools.Write().ExecContext(ctx, "VACUUM INTO "+lit); err != nil {
		return fmt.Errorf("state: pre-migration backup: %w", err)
	}
	return pruneBackups(path)
}

// mainDBPath returns the filesystem path backing the main schema, via PRAGMA database_list
// (the file actually attached to this connection — correct even through a symlink). Empty
// string for an anonymous or in-memory database.
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

// pruneBackups keeps only the newest 3 pre-migration snapshots for a given DB path. The
// timestamp suffix is fixed-width UTC (YYYYMMDDThhmmssZ), so lexical order == chronological.
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

// moneyCentsApplied is v1's data predicate. Absent cents column → not applied (run it). Cents
// present with the legacy REAL column already dropped → applied (migrateMoneyCents is atomic,
// so a present cents column implies a completed backfill — verified fact #1). Cents present
// AND legacy column still there → precisely verify every cent round-trips before declaring it
// applied, so a pre-marker DB is reconciled on real data, not on column presence alone.
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

// legacyColumnsDropped is v2's data predicate: applied iff none of the dead REAL/adjusted
// money columns remain (the exact set dropLegacyMoneyColumns removes).
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
