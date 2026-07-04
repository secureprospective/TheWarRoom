package state

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// legacyContractsDDL is the pre-B7b contracts schema (REAL money, NO cents columns) —
// exactly what a live Beelink DB carries before the migration runs.
const legacyContractsDDL = `
CREATE TABLE rosters (
	id TEXT PRIMARY KEY, league_id TEXT NOT NULL, mfl_id TEXT NOT NULL,
	franchise_id TEXT NOT NULL, roster_status TEXT NOT NULL, season INTEGER NOT NULL,
	as_of TEXT NOT NULL, UNIQUE (league_id, season, mfl_id));
CREATE TABLE contracts (
	id TEXT PRIMARY KEY, league_id TEXT NOT NULL, mfl_id TEXT NOT NULL,
	franchise_id TEXT NOT NULL, annual_salary REAL NOT NULL DEFAULT 0,
	adjusted_salary REAL NOT NULL DEFAULT 0, contract_years INTEGER NOT NULL DEFAULT 0,
	expiration_year INTEGER NOT NULL DEFAULT 0, contract_status TEXT NOT NULL DEFAULT '',
	is_restructured INTEGER NOT NULL DEFAULT 0, is_tagged INTEGER NOT NULL DEFAULT 0,
	season INTEGER NOT NULL, last_updated TEXT NOT NULL,
	UNIQUE (league_id, season, mfl_id));`

// TestMigrateMoneyCents_BackfillsExactly proves the additive REAL→cents migration: a
// legacy DB with only the REAL money columns gets the cents columns added and backfilled
// exactly ($millions × 1e8), verified with zero mismatches, and served as domain.Money.
func TestMigrateMoneyCents_BackfillsExactly(t *testing.T) {
	ctx := context.Background()
	pools, err := db.Open(ctx, filepath.Join(t.TempDir(), "legacy.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { _ = pools.Close() })

	// Stand up the legacy schema and seed one franchise's player with REAL money only.
	if _, err := pools.Write().ExecContext(ctx, legacyContractsDDL); err != nil {
		t.Fatalf("legacy ddl: %v", err)
	}
	const league, season = testLeague, testSeason
	if _, err := pools.Write().ExecContext(ctx,
		`INSERT INTO rosters (id, league_id, mfl_id, franchise_id, roster_status, season, as_of)
		 VALUES ('r1', ?, '0001', '0001', 'ROSTER', ?, 'now')`, league, season); err != nil {
		t.Fatalf("seed legacy roster: %v", err)
	}
	// 7.0M → 700,000,000 cents; 1.30M adjusted → 130,000,000 cents.
	if _, err := pools.Write().ExecContext(ctx,
		`INSERT INTO contracts (id, league_id, mfl_id, franchise_id, annual_salary,
		   adjusted_salary, contract_years, expiration_year, contract_status, season, last_updated)
		 VALUES ('c1', ?, '0001', '0001', 7.0, 1.30, 2, 2028, 'UFA', ?, 'now')`, league, season); err != nil {
		t.Fatalf("seed legacy contract: %v", err)
	}

	// Initialize must run the migration (initSchema → migrateMoneyCents), skip the seed
	// (state already present), then load the cents columns into memory.
	s := New(pools, league, season)
	if err := s.Initialize(ctx, &fakeSource{}); err != nil {
		t.Fatalf("Initialize (with migration): %v", err)
	}

	p, ok := s.Player("0001")
	if !ok {
		t.Fatal("player 0001 missing after migration")
	}
	if p.Salary != domain.Money(700_000_000) {
		t.Errorf("annual salary = %d cents, want 700000000", p.Salary)
	}
	if p.AdjustedSalary != domain.Money(130_000_000) {
		t.Errorf("adjusted salary = %d cents, want 130000000", p.AdjustedSalary)
	}
	// EffectiveSalary picks the adjusted value once set.
	if got := EffectiveSalary(p); got != domain.Money(130_000_000) {
		t.Errorf("effective salary = %d, want 130000000", got)
	}

	// Idempotent: a second Initialize on the migrated DB sees the cents columns present
	// and is a no-op (no error, no double-migration).
	s2 := New(pools, league, season)
	if err := s2.Initialize(ctx, &fakeSource{}); err != nil {
		t.Fatalf("second Initialize (already migrated): %v", err)
	}
}
