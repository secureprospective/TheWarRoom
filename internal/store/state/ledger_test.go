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

// applyCells commits a ledger cell primitive through a RAW write tx, deliberately BYPASSING
// WriteTx's parity gate. The gate rejects a ledger-only change (legacy/ledger disagree) at
// commit — correct for production, where every money op dual-writes both sides — but these
// white-box tests exercise ONE cell primitive in isolation, so they need a committed
// ledger-only write to inspect. The full dual-write ops (restructure/tag/waiver) are covered
// by the transactions integration tests, where parity DOES hold end-to-end. It does not reload
// memory: these tests read cells straight from the DB via readCells.
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

// TestParityGateRollsBackDrift gives the in-tx parity gate teeth (Ship 2, 4/4): a WriteTx that
// moves money between a player's cells WITHOUT a matching legacy change drifts the two models,
// so the gate must REJECT the whole transaction before commit — a stronger guarantee than the
// diagnostic CheckLedgerParity (which only observes committed state). After the rejection
// nothing is committed: the cells are unchanged and parity still holds. (In real ops the two
// sides are always written together; this plants a ledger-only drift.)
func TestParityGateRollsBackDrift(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: ledgerRosters(t)})
	ctx := context.Background()

	// Seed parity holds.
	if err := s.CheckLedgerParity(ctx); err != nil {
		t.Fatalf("seed parity should hold: %v", err)
	}
	// Move $10k out of the current-season (2026) cell into 2028 — legacy columns untouched.
	// The gate sees the drift on the uncommitted tx and rolls the WHOLE transaction back.
	err := s.WriteTx(ctx, func(w TxWriter) error {
		return w.MoveCellMoney(ctx, "0001", 2026, 2028, 1_000_000, "planted drift")
	})
	if err == nil {
		t.Fatal("WriteTx committed a ledger-only drift — the parity gate has no teeth")
	}
	// Nothing committed: the 2026 cell is still its seeded $20k, and parity still holds.
	if got := readCells(t, s, "0001")[2026].salary; got != 2_000_000 {
		t.Fatalf("rolled-back drift still moved the 2026 cell to %d, want 2000000 (untouched)", got)
	}
	if err := s.CheckLedgerParity(ctx); err != nil {
		t.Fatalf("parity broke after a rejected+rolled-back drift: %v", err)
	}
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
	s := New(pools, testLeague, testSeason)
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
