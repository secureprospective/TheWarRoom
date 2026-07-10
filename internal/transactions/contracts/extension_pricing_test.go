package contracts

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// m is $1,000,000 in exact cents, the unit of the §10 tables.
const m = domain.Money(100_000_000)

// TestExtensionYearPrice pins the §10 per-year extension price: 150% of the highest-paid
// remaining year, snapped to $10k, then raised to the position floor if that is greater. Every
// fencepost — floor-loses, floor-wins, exact-tie, and the $10k half-up snap — is nailed.
func TestExtensionYearPrice(t *testing.T) {
	cases := []struct {
		name             string
		highestRemaining domain.Money
		floor            domain.Money
		want             domain.Money
	}{
		{"150% beats floor", 6 * m, 3 * m, 9 * m},      // 1.5×$6M = $9M > $3M floor
		{"floor beats 150%", 2 * m, 8 * m, 8 * m},      // 1.5×$2M = $3M < $8M floor
		{"150% exactly at floor", 4 * m, 6 * m, 6 * m}, // 1.5×$4M = $6M == floor
		{"snap half-up", 1_000_000, 1, 2_000_000},      // 1.5×$10k = $15k → snaps up to $20k
		{"snap keeps grid clean", 10 * m, 1, 15 * m},   // 1.5×$10M = $15M, already on grid
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := extensionYearPrice(tc.highestRemaining, tc.floor); got != tc.want {
				t.Fatalf("extensionYearPrice(%s, %s) = %s, want %s", tc.highestRemaining, tc.floor, got, tc.want)
			}
			// Every result must be a clean $10k multiple (both inputs are, §1).
			if got := extensionYearPrice(tc.highestRemaining, tc.floor); domain.RoundToNearest10k(got) != got {
				t.Fatalf("extensionYearPrice(%s, %s) = %s is off the $10k grid", tc.highestRemaining, tc.floor, got)
			}
		})
	}
}

// TestScanExtensionCells pins the two semantics the §10 handler depends on: the pricing base is
// the highest-paid REMAINING year EXCLUSIVE of the current season (GLM M1 — a front-loaded
// current-season cell must NOT set the price), and a prior extension is detected from any PAID
// cell tagged source "extension" (GLM M2 — the no-second-extension guard's input, load-bearing
// only across seasons where the per-season counter has reset).
func TestScanExtensionCells(t *testing.T) {
	const season = 2026
	lc := func(year int, sal domain.Money, src string) state.LedgerCell {
		return state.LedgerCell{Year: year, Salary: sal, Source: src}
	}

	// Front-loaded: current season is the single highest-paid year. "Remaining" excludes it, so
	// the base is the highest FUTURE year ($6M), NOT the $8M current — the M1 fix.
	hi, ext, last := scanExtensionCells([]state.LedgerCell{
		lc(2026, 8*m, "seed"), lc(2027, 6*m, "seed"), lc(2028, 6*m, "seed"),
	}, season)
	if hi != 6*m {
		t.Fatalf("front-loaded: highestRemaining = %s, want $6M (current season excluded)", hi)
	}
	if ext {
		t.Fatal("front-loaded: alreadyExtended true on a seed-only contract")
	}
	if last != 2028 {
		t.Fatalf("front-loaded: lastPaidYear = %d, want 2028", last)
	}

	// A prior extension cell is detected regardless of price ordering.
	_, ext2, last2 := scanExtensionCells([]state.LedgerCell{
		lc(2026, 6*m, "seed"), lc(2027, 6*m, "seed"), lc(2028, 6*m, "seed"),
		lc(2029, 9*m, state.SourceExtension), lc(2030, 9*m, state.SourceExtension),
	}, season)
	if !ext2 {
		t.Fatal("did not detect a prior extension cell (source=extension)")
	}
	if last2 != 2030 {
		t.Fatalf("extended: lastPaidYear = %d, want 2030", last2)
	}

	// No future paid year (final-year contract): highestRemaining is 0 — the eligibility gate
	// rejects this upstream, but the scan must not price off the current season.
	hi3, _, last3 := scanExtensionCells([]state.LedgerCell{lc(2026, 6*m, "seed")}, season)
	if hi3 != 0 {
		t.Fatalf("final-year: highestRemaining = %s, want 0 (no year > season)", hi3)
	}
	if last3 != 2026 {
		t.Fatalf("final-year: lastPaidYear = %d, want 2026", last3)
	}
}
