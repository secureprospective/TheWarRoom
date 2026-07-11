package deadcap

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// TestBuyoutRatePct pins §12's rate table and its fail-loud edges: rates exist ONLY for 2/3/4
// remaining years (60/75/90%); everything else (1, 5, 6, 0, negative) has no rate.
func TestBuyoutRatePct(t *testing.T) {
	for _, tc := range []struct {
		remaining int
		wantPct   int64
		wantOK    bool
	}{
		{2, 60, true}, {3, 75, true}, {4, 90, true},
		{1, 0, false}, {5, 0, false}, {6, 0, false}, {0, 0, false}, {-1, 0, false},
	} {
		pct, ok := buyoutRatePct(tc.remaining)
		if pct != tc.wantPct || ok != tc.wantOK {
			t.Errorf("buyoutRatePct(%d) = (%d,%v), want (%d,%v)", tc.remaining, pct, ok, tc.wantPct, tc.wantOK)
		}
	}
}

// TestBuyoutCharge pins the charge = rate% × average remaining salary, snapped ONCE to $10k.
func TestBuyoutCharge(t *testing.T) {
	const m = 100_000_000 // $1M in cents
	for _, tc := range []struct {
		name      string
		avg       domain.Money
		remaining int
		want      domain.Money
		wantOK    bool
	}{
		{"2yr 60pct", 6 * m, 2, 36 * m / 10, true}, // $3.6M
		{"3yr 75pct", 6 * m, 3, 45 * m / 10, true}, // $4.5M
		{"4yr 90pct", 6 * m, 4, 54 * m / 10, true}, // $5.4M
		{"out of range 1", 6 * m, 1, 0, false},     // no §12 rate
		{"out of range 5", 6 * m, 5, 0, false},     // extended contract, no §12 rate
		{"zero average is zero", 0, 3, 0, true},    // degenerate but valid
		// $10k snap: 75% × $6,010,000 = $4,507,500 → snaps half-up to $4,510,000.
		{"snaps half-up to $10k", 601 * m / 100, 3, 451 * m / 100, true},
		// 60% × $6,005,000 = $3,603,000 → snaps down to $3,600,000.
		{"snaps down to $10k", 600_500_000, 2, 360 * m / 100, true},
	} {
		got, ok := buyoutCharge(tc.avg, tc.remaining)
		if got != tc.want || ok != tc.wantOK {
			t.Errorf("%s: buyoutCharge(%d,%d) = (%d,%v), want (%d,%v)", tc.name, tc.avg, tc.remaining, got, ok, tc.want, tc.wantOK)
		}
	}
}

// TestRemainingAfter pins the "remaining = years strictly after the current season" convention
// and the half-up average over those cells.
func TestRemainingAfter(t *testing.T) {
	const m = 100_000_000
	cells := []state.LedgerCell{
		{Year: 2026, Salary: 4 * m}, // current season — excluded
		{Year: 2027, Salary: 6 * m},
		{Year: 2028, Salary: 8 * m},
	}
	n, avg := remainingAfter(cells, 2026)
	if n != 2 || avg != 7*m { // (6M + 8M)/2 = 7M
		t.Fatalf("remainingAfter = (%d,%d), want (2,%d)", n, avg, 7*m)
	}
	// No cell after the season → (0,0).
	if n, avg := remainingAfter([]state.LedgerCell{{Year: 2026, Salary: 5 * m}}, 2026); n != 0 || avg != 0 {
		t.Fatalf("remainingAfter(no remaining) = (%d,%d), want (0,0)", n, avg)
	}
}
