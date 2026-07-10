package state

import (
	"context"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// readCellSources returns a player's cells keyed by league_year → source column (white-box:
// ledger_test's readCells omits source, which the §10 extension tag lives in).
func readCellSources(t *testing.T, s *Store, mflID string) map[int]string {
	t.Helper()
	rows, err := s.pools.Read().QueryContext(context.Background(),
		`SELECT league_year, source FROM contract_years WHERE league_id = ? AND mfl_id = ?`,
		testLeague, mflID)
	if err != nil {
		t.Fatalf("read sources: %v", err)
	}
	defer func() { _ = rows.Close() }()
	out := map[int]string{}
	for rows.Next() {
		var year int
		var src string
		if err := rows.Scan(&year, &src); err != nil {
			t.Fatalf("scan source: %v", err)
		}
		out[year] = src
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate sources: %v", err)
	}
	return out
}

// TestAppendExtensionYearsPromotesAndAppends proves the §10 primitive on player 0001
// (seeded PAID 2026/2027/2028 @ $2M + UFA 2029): a 2-year extension at $5M promotes the UFA
// slot to the first paid extension year, appends the second, and lays a fresh UFA slot after —
// leaving contiguous PAID cells followed by exactly one UFA slot, the seed cells untouched, and
// every new/promoted cell tagged source "extension".
func TestAppendExtensionYearsPromotesAndAppends(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: ledgerRosters(t)})
	ctx := context.Background()

	if err := applyCells(t, s, func(w *txWriter) error {
		return w.AppendExtensionYears(ctx, "0001", 2, 5_000_000, "§10 extension test")
	}); err != nil {
		t.Fatalf("AppendExtensionYears: %v", err)
	}

	cells := readCells(t, s, "0001")
	if len(cells) != 6 {
		t.Fatalf("got %d cells, want 6 (3 seed PAID + 2 ext PAID + 1 UFA): %+v", len(cells), cells)
	}
	// Seed PAID cells: untouched value, status, and their original source.
	srcs := readCellSources(t, s, "0001")
	for _, y := range []int{2026, 2027, 2028} {
		if cells[y].status != "PAID" || cells[y].salary != 2_000_000 {
			t.Fatalf("seed cell %d changed: %+v", y, cells[y])
		}
		if srcs[y] != "seed" {
			t.Fatalf("seed cell %d source = %q, want \"seed\" (a restructure/tag/extension must not retag seed cells)", y, srcs[y])
		}
	}
	// Extension PAID cells: promoted 2029 + appended 2030, both $5M, both source "extension",
	// sharing the seed contract_id (one contract term).
	for _, y := range []int{2029, 2030} {
		if cells[y].status != "PAID" || cells[y].salary != 5_000_000 {
			t.Fatalf("extension cell %d = %+v, want PAID $5M", y, cells[y])
		}
		if srcs[y] != "extension" {
			t.Fatalf("extension cell %d source = %q, want \"extension\" (the prior-extension marker)", y, srcs[y])
		}
		if cells[y].contractID != cells[2028].contractID {
			t.Fatalf("extension cell %d contract_id %q != seed contract_id %q", y, cells[y].contractID, cells[2028].contractID)
		}
	}
	// The fresh trailing UFA slot: $0, status UFA, after the new last paid year.
	if cells[2031].status != "UFA" || cells[2031].salary != 0 {
		t.Fatalf("trailing slot 2031 = %+v, want UFA $0", cells[2031])
	}
	// Change log: exactly 3 new rows beyond the 4 seed rows (promote + 1 append + 1 UFA).
	var n int
	if err := s.pools.Read().QueryRowContext(ctx,
		`SELECT COUNT(1) FROM contract_year_changes WHERE league_id = ? AND mfl_id = ?`,
		testLeague, "0001").Scan(&n); err != nil {
		t.Fatalf("count changes: %v", err)
	}
	if n != 7 {
		t.Fatalf("got %d change rows, want 7 (4 seed + 3 extension)", n)
	}
}

// TestPaidCellsReturnsSourceAndOrder proves PaidCells (the §10 read half) returns PAID cells
// in year order with their source, omits the UFA slot, and surfaces the facts the handler
// derives: the highest-paid remaining year, the total paid count, and a prior-extension marker.
func TestPaidCellsReturnsSourceAndOrder(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: ledgerRosters(t)})
	ctx := context.Background()

	var got []LedgerCell
	if err := applyCells(t, s, func(w *txWriter) error {
		if err := w.AppendExtensionYears(ctx, "0001", 1, 5_000_000, "ext"); err != nil {
			return err
		}
		cells, err := w.PaidCells(ctx, "0001")
		got = cells
		return err
	}); err != nil {
		t.Fatalf("apply: %v", err)
	}

	// 4 PAID cells (2026/2027/2028 seed + 2029 promoted), UFA at 2030 omitted, in year order.
	if len(got) != 4 {
		t.Fatalf("got %d paid cells, want 4: %+v", len(got), got)
	}
	wantYears := []int{2026, 2027, 2028, 2029}
	hasExtension := false
	for i, c := range got {
		if c.Year != wantYears[i] {
			t.Fatalf("cell %d year = %d, want %d (must be year-ordered)", i, c.Year, wantYears[i])
		}
		if c.Source == "extension" {
			hasExtension = true
		}
	}
	if !hasExtension {
		t.Fatalf("no PAID cell tagged \"extension\" — the no-2nd-extension guard would miss a prior extension")
	}
	// Highest-paid remaining year = the $5M extension year (the §10 pricing base).
	var highest domain.Money
	for _, c := range got {
		if c.Salary > highest {
			highest = c.Salary
		}
	}
	if highest != 5_000_000 {
		t.Fatalf("highest paid cell = %s, want $5M", highest)
	}
}

// TestAppendExtensionYearsGuards proves the primitive fails loud on bad arguments and an
// unknown player, before touching any cell.
func TestAppendExtensionYearsGuards(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: ledgerRosters(t)})
	ctx := context.Background()

	cases := []struct {
		name       string
		mflID      string
		addedYears int
		price      domain.Money
	}{
		{"zero years", "0001", 0, 5_000_000},
		{"negative years", "0001", -1, 5_000_000},
		{"zero price", "0001", 2, 0},
		{"negative price", "0001", 2, -5_000_000},
		{"unknown player", "9999", 2, 5_000_000},
	}
	for _, tc := range cases {
		err := applyCells(t, s, func(w *txWriter) error {
			return w.AppendExtensionYears(ctx, tc.mflID, tc.addedYears, tc.price, "guard")
		})
		if err == nil {
			t.Errorf("AppendExtensionYears(%s) returned nil, want an error", tc.name)
		}
	}
	// No cell was written by any rejected call — 0001 still has its 4 seed cells.
	if cells := readCells(t, s, "0001"); len(cells) != 4 {
		t.Fatalf("guard cases mutated cells: got %d, want 4", len(cells))
	}
}
