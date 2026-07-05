package state

import (
	"bytes"
	"context"
	"log"
	"path/filepath"
	"strings"
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
	// 7.0M → 700,000,000 cents; the 1.30M legacy adjusted column is dead (Ship 4 drops it).
	if _, err := pools.Write().ExecContext(ctx,
		`INSERT INTO contracts (id, league_id, mfl_id, franchise_id, annual_salary,
		   adjusted_salary, contract_years, expiration_year, contract_status, season, last_updated)
		 VALUES ('c1', ?, '0001', '0001', 7.0, 1.30, 2, 2028, 'UFA', ?, 'now')`, league, season); err != nil {
		t.Fatalf("seed legacy contract: %v", err)
	}
	// A Ship-1-seeded ledger cell: the current-season PAID cell is the KING (Ship 3). Stand it
	// up (via the REAL ledger schema, so the fixture can't drift from production) so load()'s M1
	// drift guard sees a cell for this rostered salaried player. (A pre-ledger DB with no cells
	// is an unsupported upgrade path — carry-forward, see the project CLAUDE.md.)
	s := New(pools, league, season)
	if err := s.initLedgerSchema(ctx); err != nil {
		t.Fatalf("init ledger schema: %v", err)
	}
	if _, err := pools.Write().ExecContext(ctx,
		`INSERT INTO contract_years (league_id, mfl_id, league_year, salary_cents, year_status,
		   contract_id, source, last_updated)
		 VALUES (?, '0001', ?, 700000000, 'PAID', 'c1', 'seed', 'now')`, league, season); err != nil {
		t.Fatalf("seed legacy ledger cell: %v", err)
	}

	// Initialize must run the migration (initSchema → migrateMoneyCents), skip the seed
	// (state already present), then load the cents columns into memory.
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
	// Ship 4: the dead money columns are DROPPED by dropLegacyMoneyColumns — the two legacy
	// REAL columns and the now-unread adjusted_salary_cents — while annual_salary_cents (the
	// base §9/§11 rule salary) survives. Prove the drop landed and the survivor remains.
	for _, dropped := range []string{"annual_salary", "adjusted_salary", "adjusted_salary_cents"} {
		if have, err := s.columnExists(ctx, "contracts", dropped); err != nil {
			t.Fatalf("columnExists(%s): %v", dropped, err)
		} else if have {
			t.Errorf("column %s still present after Ship 4 drop", dropped)
		}
	}
	if have, err := s.columnExists(ctx, "contracts", "annual_salary_cents"); err != nil {
		t.Fatalf("columnExists(annual_salary_cents): %v", err)
	} else if !have {
		t.Error("annual_salary_cents was dropped — it is the base rule salary and must survive")
	}

	// Idempotent: a second Initialize on the migrated DB sees the cents columns present
	// and is a no-op (no error, no double-migration).
	s2 := New(pools, league, season)
	if err := s2.Initialize(ctx, &fakeSource{}); err != nil {
		t.Fatalf("second Initialize (already migrated): %v", err)
	}
}

// ship3ContractsDDL is the pre-Ship-4 contracts schema: BOTH legacy REAL columns AND both
// cents columns present (a DB that already ran the B7b cents migration). Ship 4 must drop the
// two REAL columns AND the now-dead adjusted_salary_cents, keeping only annual_salary_cents.
const ship3ContractsDDL = `
CREATE TABLE rosters (
	id TEXT PRIMARY KEY, league_id TEXT NOT NULL, mfl_id TEXT NOT NULL,
	franchise_id TEXT NOT NULL, roster_status TEXT NOT NULL, season INTEGER NOT NULL,
	as_of TEXT NOT NULL, UNIQUE (league_id, season, mfl_id));
CREATE TABLE contracts (
	id TEXT PRIMARY KEY, league_id TEXT NOT NULL, mfl_id TEXT NOT NULL,
	franchise_id TEXT NOT NULL, annual_salary REAL NOT NULL DEFAULT 0,
	adjusted_salary REAL NOT NULL DEFAULT 0,
	annual_salary_cents INTEGER NOT NULL DEFAULT 0,
	adjusted_salary_cents INTEGER NOT NULL DEFAULT 0,
	contract_years INTEGER NOT NULL DEFAULT 0, expiration_year INTEGER NOT NULL DEFAULT 0,
	contract_status TEXT NOT NULL DEFAULT '', is_restructured INTEGER NOT NULL DEFAULT 0,
	is_tagged INTEGER NOT NULL DEFAULT 0, season INTEGER NOT NULL, last_updated TEXT NOT NULL,
	UNIQUE (league_id, season, mfl_id));`

// TestDropLegacyMoneyColumns_Ship3Era proves the Ship 4 column drop on a DB where
// adjusted_salary_cents is actually PRESENT (unlike the pre-cents legacy DB, where the guard
// skips it) — so the drop branch is exercised, not silently no-op'd. A $9M-cap player with a
// current cell must survive the drop with his base salary intact and cap derived from the cell.
func TestDropLegacyMoneyColumns_Ship3Era(t *testing.T) {
	ctx := context.Background()
	pools, err := db.Open(ctx, filepath.Join(t.TempDir(), "ship3.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { _ = pools.Close() })

	const league, season = testLeague, testSeason
	if _, err := pools.Write().ExecContext(ctx, ship3ContractsDDL); err != nil {
		t.Fatalf("ship3 ddl: %v", err)
	}
	if _, err := pools.Write().ExecContext(ctx,
		`INSERT INTO rosters (id, league_id, mfl_id, franchise_id, roster_status, season, as_of)
		 VALUES ('r1', ?, '0001', '0001', 'ROSTER', ?, 'now')`, league, season); err != nil {
		t.Fatalf("seed ship3 roster: %v", err)
	}
	if _, err := pools.Write().ExecContext(ctx,
		`INSERT INTO contracts (id, league_id, mfl_id, franchise_id, annual_salary,
		   adjusted_salary, annual_salary_cents, adjusted_salary_cents, contract_years,
		   expiration_year, contract_status, season, last_updated)
		 VALUES ('c1', ?, '0001', '0001', 10.0, 9.0, 1000000000, 900000000, 2, 2028, 'UFA', ?, 'now')`,
		league, season); err != nil {
		t.Fatalf("seed ship3 contract: %v", err)
	}
	s := New(pools, league, season)
	if err := s.initLedgerSchema(ctx); err != nil {
		t.Fatalf("init ledger schema: %v", err)
	}
	if _, err := pools.Write().ExecContext(ctx,
		`INSERT INTO contract_years (league_id, mfl_id, league_year, salary_cents, year_status,
		   contract_id, source, last_updated)
		 VALUES (?, '0001', ?, 900000000, 'PAID', 'c1', 'seed', 'now')`, league, season); err != nil {
		t.Fatalf("seed ship3 ledger cell: %v", err)
	}

	if err := s.Initialize(ctx, &fakeSource{}); err != nil {
		t.Fatalf("Initialize (Ship 4 drop): %v", err)
	}

	for _, dropped := range []string{"annual_salary", "adjusted_salary", "adjusted_salary_cents"} {
		if have, err := s.columnExists(ctx, "contracts", dropped); err != nil {
			t.Fatalf("columnExists(%s): %v", dropped, err)
		} else if have {
			t.Errorf("column %s still present after Ship 4 drop", dropped)
		}
	}
	// Base salary survives (annual_salary_cents kept); cap derives from the $9M cell.
	p, ok := s.Player("0001")
	if !ok || p.Salary != domain.Money(1_000_000_000) {
		t.Fatalf("base salary lost after drop: %+v ok=%v", p, ok)
	}
	if used, _ := s.CapUsed("0001"); used != domain.Money(900_000_000) {
		t.Fatalf("cap after drop = %d, want 900000000 (cell-derived)", used)
	}
}

// TestLoadWarnsAndContinuesOnMissingCell pins the M1 drift guard's contract (GLM-5.2 Ship-4
// review): a rostered player who carries a base salary but has NO PAID current-season ledger
// cell must NOT brick startup — load() surfaces the drift with a loud WARNING and serves him at
// $0 cap (degraded but open). Stands up the same Ship-3-era DB but WITHOUT the player's cell.
func TestLoadWarnsAndContinuesOnMissingCell(t *testing.T) {
	ctx := context.Background()
	pools, err := db.Open(ctx, filepath.Join(t.TempDir(), "nocell.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { _ = pools.Close() })

	const league, season = testLeague, testSeason
	if _, err := pools.Write().ExecContext(ctx, ship3ContractsDDL); err != nil {
		t.Fatalf("ship3 ddl: %v", err)
	}
	if _, err := pools.Write().ExecContext(ctx,
		`INSERT INTO rosters (id, league_id, mfl_id, franchise_id, roster_status, season, as_of)
		 VALUES ('r1', ?, '0001', '0001', 'ROSTER', ?, 'now')`, league, season); err != nil {
		t.Fatalf("seed roster: %v", err)
	}
	if _, err := pools.Write().ExecContext(ctx,
		`INSERT INTO contracts (id, league_id, mfl_id, franchise_id, annual_salary_cents,
		   contract_years, expiration_year, contract_status, season, last_updated)
		 VALUES ('c1', ?, '0001', '0001', 1000000000, 2, 2028, 'UFA', ?, 'now')`, league, season); err != nil {
		t.Fatalf("seed contract: %v", err)
	}
	// The ledger table exists (real schema) but holds NO cell for 0001 — the planted drift.
	s := New(pools, league, season)
	if err := s.initLedgerSchema(ctx); err != nil {
		t.Fatalf("init ledger schema: %v", err)
	}

	// Capture the warning: the guard must surface the drift loudly even though it does not fail.
	var logbuf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&logbuf)
	defer log.SetOutput(prev)

	if err := s.Initialize(ctx, &fakeSource{}); err != nil {
		t.Fatalf("load hard-failed on a drift row — must warn and continue, not brick: %v", err)
	}
	if !strings.Contains(logbuf.String(), `state: load: WARNING rostered player "0001" has base salary`) {
		t.Fatalf("drift not surfaced — the M1 guard logged no warning; got: %q", logbuf.String())
	}
	// Degraded-but-open: the drift player is served at $0 cap until reconciled.
	players, ok := s.Roster("0001")
	if !ok || len(players) != 1 {
		t.Fatalf("franchise 0001 roster missing after drift-warn: ok=%v players=%d", ok, len(players))
	}
	if players[0].CapSalary != 0 {
		t.Fatalf("drift player CapSalary = %s, want $0 (no PAID cell)", players[0].CapSalary)
	}
}
