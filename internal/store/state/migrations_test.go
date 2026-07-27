package state

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/db"
)

// readStateMigrations returns owner="state" schema_migrations rows as version→method.
func readStateMigrations(ctx context.Context, t *testing.T, s *Store) map[int]string {
	t.Helper()
	rows, err := s.pools.Read().QueryContext(ctx,
		`SELECT version, method FROM schema_migrations WHERE owner = ? ORDER BY version`, stateOwner)
	if err != nil {
		t.Fatalf("read schema_migrations: %v", err)
	}
	defer func() { _ = rows.Close() }()
	out := make(map[int]string)
	for rows.Next() {
		var v int
		var m string
		if err := rows.Scan(&v, &m); err != nil {
			t.Fatalf("scan schema_migrations: %v", err)
		}
		out[v] = m
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("schema_migrations rows: %v", err)
	}
	return out
}

func countBackups(t *testing.T, dbPath string) int {
	t.Helper()
	matches, err := filepath.Glob(dbPath + ".premigration-*")
	if err != nil {
		t.Fatalf("glob backups: %v", err)
	}
	return len(matches)
}

// TestMigrations_LegacyDBStampedMigratedAndBackedUp proves the Tier 2 marker + backup on the
// real upgrade path: a pre-cents legacy DB gets both migrations RUN (method='migrated'), a
// single VACUUM INTO snapshot is taken because real work was pending, and the data survives.
func TestMigrations_LegacyDBStampedMigratedAndBackedUp(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "legacy.db")
	pools, err := db.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { _ = pools.Close() })

	const league, season = testLeague, testSeason
	if _, err := pools.Write().ExecContext(ctx, legacyContractsDDL); err != nil {
		t.Fatalf("legacy ddl: %v", err)
	}
	if _, err := pools.Write().ExecContext(ctx,
		`INSERT INTO rosters (id, league_id, mfl_id, franchise_id, roster_status, season, as_of)
		 VALUES ('r1', ?, '0001', '0001', 'ROSTER', ?, 'now')`, league, season); err != nil {
		t.Fatalf("seed roster: %v", err)
	}
	if _, err := pools.Write().ExecContext(ctx,
		`INSERT INTO contracts (id, league_id, mfl_id, franchise_id, annual_salary,
		   adjusted_salary, contract_years, expiration_year, contract_status, season, last_updated)
		 VALUES ('c1', ?, '0001', '0001', 7.0, 1.30, 2, 2028, 'UFA', ?, 'now')`, league, season); err != nil {
		t.Fatalf("seed contract: %v", err)
	}
	s := New(pools, league, season, nil)
	if err := s.initLedgerSchema(ctx); err != nil {
		t.Fatalf("init ledger schema: %v", err)
	}
	if _, err := pools.Write().ExecContext(ctx,
		`INSERT INTO contract_years (league_id, mfl_id, league_year, salary_cents, year_status,
		   contract_id, source, last_updated)
		 VALUES (?, '0001', ?, 700000000, 'PAID', 'c1', 'seed', 'now')`, league, season); err != nil {
		t.Fatalf("seed ledger cell: %v", err)
	}

	if err := s.Initialize(ctx, &fakeSource{}); err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	got := readStateMigrations(ctx, t, s)
	if got[1] != "migrated" || got[2] != "migrated" {
		t.Fatalf("markers = %v, want v1+v2 method='migrated'", got)
	}
	if n := countBackups(t, dbPath); n != 1 {
		t.Fatalf("backup count = %d, want exactly 1 (pending work → one snapshot)", n)
	}
}

// TestMigrations_FreshDBReconciledNoBackup proves a brand-new DB (whose DDL already carries the
// cents column and never had the legacy columns) stamps BOTH migrations 'reconciled' from the
// data predicates, without doing real work — so no pre-migration backup is written.
func TestMigrations_FreshDBReconciledNoBackup(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "fresh.db")
	pools, err := db.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { _ = pools.Close() })

	s := New(pools, testLeague, testSeason, nil)
	if err := s.Initialize(ctx, &fakeSource{rosters: baseRosters(t)}); err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	got := readStateMigrations(ctx, t, s)
	if got[1] != "reconciled" || got[2] != "reconciled" {
		t.Fatalf("markers = %v, want v1+v2 method='reconciled'", got)
	}
	if n := countBackups(t, dbPath); n != 0 {
		t.Fatalf("backup count = %d, want 0 (nothing pending on a fresh DB)", n)
	}
}

// TestMigrations_SecondInitializeIsNoOp proves the marker short-circuit: once stamped, a second
// Initialize adds no markers and takes no new backup (the versions are skipped on the marker,
// never re-evaluated against data).
func TestMigrations_SecondInitializeIsNoOp(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "twice.db")
	pools, err := db.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { _ = pools.Close() })

	s1 := New(pools, testLeague, testSeason, nil)
	if err := s1.Initialize(ctx, &fakeSource{rosters: baseRosters(t)}); err != nil {
		t.Fatalf("first Initialize: %v", err)
	}
	before := readStateMigrations(ctx, t, s1)

	s2 := New(pools, testLeague, testSeason, nil)
	if err := s2.Initialize(ctx, &fakeSource{}); err != nil {
		t.Fatalf("second Initialize: %v", err)
	}
	after := readStateMigrations(ctx, t, s2)

	if len(after) != len(before) {
		t.Fatalf("marker count changed on re-init: before=%v after=%v", before, after)
	}
	if n := countBackups(t, dbPath); n != 0 {
		t.Fatalf("backup count = %d after no-op re-init, want 0", n)
	}
}

// TestMigrations_DowngradeRefused proves checkNoDowngrade: a DB stamped with a state version
// newer than this binary understands refuses to open with a plain-language message.
func TestMigrations_DowngradeRefused(t *testing.T) {
	ctx := context.Background()
	pools, err := db.Open(ctx, filepath.Join(t.TempDir(), "future.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { _ = pools.Close() })

	s1 := New(pools, testLeague, testSeason, nil)
	if err := s1.Initialize(ctx, &fakeSource{rosters: baseRosters(t)}); err != nil {
		t.Fatalf("first Initialize: %v", err)
	}
	// A future binary wrote a v99 state migration this build has never heard of.
	if _, err := pools.Write().ExecContext(ctx,
		`INSERT INTO schema_migrations (owner, version, method) VALUES (?, 99, 'migrated')`,
		stateOwner); err != nil {
		t.Fatalf("stamp future version: %v", err)
	}

	s2 := New(pools, testLeague, testSeason, nil)
	err = s2.Initialize(ctx, &fakeSource{})
	if err == nil {
		t.Fatal("Initialize opened a DB newer than the binary — want a downgrade refusal")
	}
	if !strings.Contains(err.Error(), "NEWER version") {
		t.Fatalf("downgrade error = %q, want a plain-language NEWER-version refusal", err)
	}
}
