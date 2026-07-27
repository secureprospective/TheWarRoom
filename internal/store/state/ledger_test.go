package state

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// ledgerRosters is a single-player fixture with realistic cent values (the baseRosters
// salaries are single-digit cents that would all snap to $0). Player 0001: $20,000 flat,
// expires 2028; seeded in season 2026 → PAID 2026/2027/2028 + UFA 2029 (the locked
// fencepost). Salary is an exact $10k multiple so snapping is a no-op here.
func ledgerRosters(t *testing.T) []domain.Roster {
	t.Helper()
	return []domain.Roster{
		{FranchiseID: "0001", Players: []domain.PlayerRecord{
			{MFLID: mustID(t, "0001"), Salary: 2_000_000, ContractYear: 2028,
				RosterStatus: domain.RosterActive, ContractStatus: domain.CStatusUFA},
		}},
	}
}

type ledgerCell struct {
	salary     int64
	status     string
	contractID string
}

// readCells returns a player's ledger cells keyed by league_year (white-box: reads the DB
// straight, since Ship 1 exposes no cell-read API by design — nothing in prod reads cells
// yet).
func readCells(t *testing.T, s *Store, mflID string) map[int]ledgerCell {
	t.Helper()
	rows, err := s.pools.Read().QueryContext(context.Background(),
		`SELECT league_year, salary_cents, year_status, contract_id
		 FROM contract_years WHERE league_id = ? AND mfl_id = ? ORDER BY league_year`,
		testLeague, mflID)
	if err != nil {
		t.Fatalf("read cells: %v", err)
	}
	defer func() { _ = rows.Close() }()
	out := map[int]ledgerCell{}
	for rows.Next() {
		var year int
		var c ledgerCell
		if err := rows.Scan(&year, &c.salary, &c.status, &c.contractID); err != nil {
			t.Fatalf("scan cell: %v", err)
		}
		out[year] = c
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate cells: %v", err)
	}
	return out
}

func TestSeedLedgerFencepost(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: ledgerRosters(t)})
	cells := readCells(t, s, "0001")

	if len(cells) != 4 {
		t.Fatalf("got %d cells, want 4 (2026/2027/2028 PAID + 2029 UFA): %+v", len(cells), cells)
	}
	for _, y := range []int{2026, 2027, 2028} {
		c, ok := cells[y]
		if !ok {
			t.Fatalf("missing PAID cell for %d", y)
		}
		if c.status != yearStatusPaid || c.salary != 2_000_000 {
			t.Fatalf("cell %d = {%d, %s}, want {2000000, PAID}", y, c.salary, c.status)
		}
	}
	ufa, ok := cells[2029]
	if !ok || ufa.status != yearStatusUFA || ufa.salary != 0 {
		t.Fatalf("2029 = %+v, want {0, UFA}", ufa)
	}
	// All cells of the one term share the same contract_id.
	id := cells[2026].contractID
	if id == "" {
		t.Fatal("empty contract_id")
	}
	for y, c := range cells {
		if c.contractID != id {
			t.Fatalf("cell %d contract_id %q != %q", y, c.contractID, id)
		}
	}
	// One change-log row per seeded cell.
	var n int
	if err := s.pools.Read().QueryRowContext(context.Background(),
		`SELECT COUNT(1) FROM contract_year_changes WHERE league_id = ? AND mfl_id = ?`,
		testLeague, "0001").Scan(&n); err != nil {
		t.Fatalf("count changes: %v", err)
	}
	if n != 4 {
		t.Fatalf("got %d change rows, want 4 (one per cell)", n)
	}
}

func TestSeedLedgerSnapsOffGridSalary(t *testing.T) {
	rosters := []domain.Roster{
		{FranchiseID: "0001", Players: []domain.PlayerRecord{
			// $20,049.99 → snaps to $20,000 (2_000_000 cents) in every PAID cell.
			{MFLID: mustID(t, "0001"), Salary: 2_004_999, ContractYear: 2027,
				RosterStatus: domain.RosterActive, ContractStatus: domain.CStatusUFA},
		}},
	}
	s := newStore(t, &fakeSource{rosters: rosters})
	cells := readCells(t, s, "0001")
	for _, y := range []int{2026, 2027} {
		if cells[y].salary != 2_000_000 {
			t.Fatalf("cell %d salary = %d, want 2000000 (snapped)", y, cells[y].salary)
		}
	}
}

func TestSeedLedgerAbsentContractYearCoversSeason(t *testing.T) {
	rosters := []domain.Roster{
		{FranchiseID: "0001", Players: []domain.PlayerRecord{
			// ContractYear 0 (absent) → at least the live season is PAID, then UFA next.
			{MFLID: mustID(t, "0001"), Salary: 1_000_000, ContractYear: 0,
				RosterStatus: domain.RosterActive, ContractStatus: domain.CStatusUFA},
		}},
	}
	s := newStore(t, &fakeSource{rosters: rosters})
	cells := readCells(t, s, "0001")
	if len(cells) != 2 {
		t.Fatalf("got %d cells, want 2 (2026 PAID + 2027 UFA): %+v", len(cells), cells)
	}
	if cells[2026].status != yearStatusPaid || cells[2027].status != yearStatusUFA {
		t.Fatalf("cells = %+v, want 2026 PAID + 2027 UFA", cells)
	}
}

// applyCells commits a ledger cell primitive through a RAW write tx (no memory reload) so a
// white-box test can inspect ONE primitive's effect straight from the DB via readCells. It
// exercises the txWriter cell ops (SetCell/VoidCells) in isolation; the full ops that compose
// them (restructure/tag/waiver) are covered by the transactions integration tests.
func applyCells(t *testing.T, s *Store, fn func(*txWriter) error) error {
	t.Helper()
	ctx := context.Background()
	tx, err := s.pools.Write().BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	// Deferred rollback fires on an early error return OR a panic in fn (matching WriteTx's
	// own idiom); it is a harmless no-op once Commit has succeeded (database/sql returns
	// ErrTxDone, ignored). This keeps a panicking primitive test from leaking the write lock.
	defer func() { _ = tx.Rollback() }()
	if ferr := fn(&txWriter{s: s, tx: tx}); ferr != nil {
		return ferr
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
	return nil
}

// TestSetCellSetsAbsoluteAndLogs proves the §9 tag primitive: SetCell overwrites one cell
// with an absolute value (not a delta), logs the old→new change, and leaves other cells
// untouched. Unlike MoveCellMoney it does not conserve the contract total.
func TestSetCellSetsAbsoluteAndLogs(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: ledgerRosters(t)})
	ctx := context.Background()

	if err := applyCells(t, s, func(w *txWriter) error {
		return w.SetCell(ctx, "0001", 2026, 5_000_000, "tag test")
	}); err != nil {
		t.Fatalf("SetCell: %v", err)
	}
	cells := readCells(t, s, "0001")
	if cells[2026].salary != 5_000_000 {
		t.Fatalf("cell 2026 = %d, want 5000000 (set)", cells[2026].salary)
	}
	// Non-conserving: the other PAID cells are unchanged (money was not moved out of them).
	for _, y := range []int{2027, 2028} {
		if cells[y].salary != 2_000_000 {
			t.Fatalf("cell %d = %d, want 2000000 (untouched)", y, cells[y].salary)
		}
	}
	// One extra change-log row beyond the 4 seed rows.
	var n int
	if err := s.pools.Read().QueryRowContext(ctx,
		`SELECT COUNT(1) FROM contract_year_changes WHERE league_id = ? AND mfl_id = ?`,
		testLeague, "0001").Scan(&n); err != nil {
		t.Fatalf("count changes: %v", err)
	}
	if n != 5 {
		t.Fatalf("got %d change rows, want 5 (4 seed + 1 set)", n)
	}
	// The set's log row records the correct old→new transition and reason (content, not just
	// count — a swapped old/new or a wrong-value log would pass a count-only check). GLM lead.
	var oldC, newC int64
	var reason string
	if err := s.pools.Read().QueryRowContext(ctx,
		`SELECT old_cents, new_cents, reason FROM contract_year_changes
		 WHERE league_id = ? AND mfl_id = ? AND league_year = ? ORDER BY changed_at DESC, rowid DESC LIMIT 1`,
		testLeague, "0001", 2026).Scan(&oldC, &newC, &reason); err != nil {
		t.Fatalf("read set log row: %v", err)
	}
	if oldC != 2_000_000 || newC != 5_000_000 || reason != "tag test" {
		t.Fatalf("set log row = {old %d, new %d, reason %q}, want {2000000, 5000000, \"tag test\"}", oldC, newC, reason)
	}
}

// TestSetCellFailsLoud proves SetCell rejects a missing cell and a negative value rather
// than silently no-op'ing or corrupting the ledger.
func TestSetCellFailsLoud(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: ledgerRosters(t)})
	ctx := context.Background()

	if err := s.WriteTx(ctx, func(w TxWriter) error {
		return w.SetCell(ctx, "0001", 2099, 5_000_000, "no such cell")
	}); err == nil {
		t.Fatal("SetCell on a nonexistent cell succeeded, want fail-loud")
	}
	if err := s.WriteTx(ctx, func(w TxWriter) error {
		return w.SetCell(ctx, "0001", 2026, -1, "negative")
	}); err == nil {
		t.Fatal("SetCell with a negative value succeeded, want fail-loud")
	}
}

// TestVoidCellsVoidsAllPaidAndLogs proves the §8 waiver primitive: VoidCells flips EVERY
// PAID cell to $0/VOID (relieving the whole contract, not one year), logs each old→0 change,
// removes the cells from the cap-bearing read (LedgerCells) and keeps them for history (the
// rows are not deleted). Player 0001 has PAID 2026/2027/2028 → all three void.
func TestVoidCellsVoidsAllPaidAndLogs(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: ledgerRosters(t)})
	ctx := context.Background()

	if err := applyCells(t, s, func(w *txWriter) error {
		return w.VoidCells(ctx, "0001", "waiver test")
	}); err != nil {
		t.Fatalf("VoidCells: %v", err)
	}
	cells := readCells(t, s, "0001")
	for _, y := range []int{2026, 2027, 2028} {
		c, ok := cells[y]
		if !ok {
			t.Fatalf("cell %d was DELETED, want kept as VOID (history preserved)", y)
		}
		if c.status != yearStatusVoid || c.salary != 0 {
			t.Fatalf("cell %d = {%d, %s}, want {0, VOID}", y, c.salary, c.status)
		}
	}
	// The cap-bearing read now returns no PAID cell for the cut player.
	paid, err := s.LedgerCells(ctx, "0001")
	if err != nil {
		t.Fatalf("LedgerCells: %v", err)
	}
	if len(paid) != 0 {
		t.Fatalf("LedgerCells after void = %v, want empty (no cap-bearing cell)", paid)
	}
	// Three void change-log rows beyond the 4 seed rows, each old=2M→new=0.
	var n int
	if err := s.pools.Read().QueryRowContext(ctx,
		`SELECT COUNT(1) FROM contract_year_changes WHERE league_id = ? AND mfl_id = ? AND new_cents = 0 AND old_cents = 2000000`,
		testLeague, "0001").Scan(&n); err != nil {
		t.Fatalf("count void changes: %v", err)
	}
	if n != 3 {
		t.Fatalf("got %d void change rows (old 2M→new 0), want 3", n)
	}
}

// TestVoidCellsFailsLoudOnNoPaidCell proves VoidCells fails loud rather than silently no-op'ing
// when a player has no PAID cell (already voided) — the fail-loud house style.
func TestVoidCellsFailsLoudOnNoPaidCell(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: ledgerRosters(t)})
	ctx := context.Background()

	if err := applyCells(t, s, func(w *txWriter) error {
		return w.VoidCells(ctx, "0001", "first cut")
	}); err != nil {
		t.Fatalf("first VoidCells: %v", err)
	}
	// Second void: every PAID cell is already VOID → nothing to void → fail loud.
	if err := applyCells(t, s, func(w *txWriter) error {
		return w.VoidCells(ctx, "0001", "second cut")
	}); err == nil {
		t.Fatal("VoidCells with no PAID cell succeeded, want fail-loud")
	}
}

// TestContractYearChangesImmutable proves the double-immutability: the BEFORE UPDATE and
// BEFORE DELETE triggers abort a raw write that bypasses the (nonexistent) Go mutation API.
func TestContractYearChangesImmutable(t *testing.T) {
	pools, err := db.Open(context.Background(), filepath.Join(t.TempDir(), "imm.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	s := New(pools, testLeague, testSeason, nil)
	if err := s.Initialize(context.Background(), &fakeSource{rosters: ledgerRosters(t)}); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	if _, err := pools.Write().ExecContext(context.Background(),
		`UPDATE contract_year_changes SET reason = 'tamper' WHERE mfl_id = ?`, "0001"); err == nil {
		t.Fatal("raw UPDATE on contract_year_changes succeeded, want trigger abort")
	}
	if _, err := pools.Write().ExecContext(context.Background(),
		`DELETE FROM contract_year_changes WHERE mfl_id = ?`, "0001"); err == nil {
		t.Fatal("raw DELETE on contract_year_changes succeeded, want trigger abort")
	}
}

// discountRosters is a single franchise with one player on each roster status, so the
// franchise cap total isolates each status's discount contribution.
func discountRosters(t *testing.T) []domain.Roster {
	t.Helper()
	return []domain.Roster{
		{FranchiseID: "0001", Players: []domain.PlayerRecord{
			{MFLID: mustID(t, "0001"), Salary: 10 * capUnit, ContractYear: 2028,
				RosterStatus: domain.RosterActive, ContractStatus: domain.CStatusUFA},
			{MFLID: mustID(t, "0002"), Salary: 10 * capUnit, ContractYear: 2028,
				RosterStatus: domain.RosterTaxi, ContractStatus: domain.CStatusUFA},
			{MFLID: mustID(t, "0003"), Salary: 10 * capUnit, ContractYear: 2028,
				RosterStatus: domain.RosterIR, ContractStatus: domain.CStatusUFA},
		}},
	}
}

// TestLoadCellCapAppliesTaxiIRDiscount confirms Session 0's cap-math fix: a taxi/IR
// player's cap CONTRIBUTION is scaled by the discounts source, while the player's own
// raw CapSalary (the rule base other math reads) stays undiscounted.
func TestLoadCellCapAppliesTaxiIRDiscount(t *testing.T) {
	s := newStoreWithDiscounts(t, &fakeSource{rosters: discountRosters(t)}, fakeDiscounts{taxiPct: 50, irPct: 0})

	fs, ok := s.FranchiseState("0001")
	if !ok {
		t.Fatal("FranchiseState(0001) not found")
	}
	// 10 (ROSTER, 100%) + 5 (TAXI, 50%) + 0 (IR, 0%) = 15 units.
	if fs.CapUsed != 15*capUnit {
		t.Fatalf("CapUsed = %v, want %v", fs.CapUsed, 15*capUnit)
	}
	for _, p := range fs.Players {
		if p.CapSalary != 10*capUnit {
			t.Fatalf("player %s CapSalary = %v, want undiscounted %v", p.MFLID, p.CapSalary, 10*capUnit)
		}
	}
}

// TestLoadCellCapNilDiscountsIsNoDiscount confirms a nil CapDiscounts source (no
// rulebook wired, e.g. tests or a pre-Session-0 caller) preserves the historical
// 100%-for-everyone behavior.
func TestLoadCellCapNilDiscountsIsNoDiscount(t *testing.T) {
	s := newStoreWithDiscounts(t, &fakeSource{rosters: discountRosters(t)}, nil)

	fs, ok := s.FranchiseState("0001")
	if !ok {
		t.Fatal("FranchiseState(0001) not found")
	}
	if fs.CapUsed != 30*capUnit {
		t.Fatalf("CapUsed = %v, want %v (no discount)", fs.CapUsed, 30*capUnit)
	}
}
