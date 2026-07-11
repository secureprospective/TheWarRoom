package deadcap

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// TestRetirementCharge pins §13's retirement charge = 30% of the SUM of every remaining cell
// (years strictly after the current season), snapped ONCE to $10k. It sums the actual cells (the
// ledger is king — no annual×years approximation), which diverges from a flat approximation when
// the remaining years carry different salaries.
func TestRetirementCharge(t *testing.T) {
	const m = 100_000_000 // $1M in cents
	for _, tc := range []struct {
		name   string
		cells  []state.LedgerCell
		season int
		want   domain.Money
	}{
		{
			name:   "flat 10M/yr, 3 remaining",
			cells:  []state.LedgerCell{{Year: 2026, Salary: 10 * m}, {Year: 2027, Salary: 10 * m}, {Year: 2028, Salary: 10 * m}, {Year: 2029, Salary: 10 * m}},
			season: 2026,
			want:   9 * m, // 30% × (10M+10M+10M for 2027/28/29) = 30% × 30M = $9M
		},
		{
			name:   "unequal cells summed, not approximated",
			cells:  []state.LedgerCell{{Year: 2026, Salary: 12 * m}, {Year: 2027, Salary: 8 * m}, {Year: 2028, Salary: 4 * m}},
			season: 2026,
			want:   36 * m / 10, // 30% × (8M + 4M) = 30% × 12M = $3.6M (current 2026 excluded)
		},
		{
			name:   "no remaining year is $0",
			cells:  []state.LedgerCell{{Year: 2026, Salary: 10 * m}},
			season: 2026,
			want:   0,
		},
		{
			name:   "snaps to the $10k grid",
			cells:  []state.LedgerCell{{Year: 2027, Salary: 601 * m / 100}}, // $6.01M → 30% = $1,803,000 → snaps to $1,800,000
			season: 2026,
			want:   180 * m / 100,
		},
	} {
		if got := retirementCharge(tc.cells, tc.season); got != tc.want {
			t.Errorf("%s: retirementCharge = %d, want %d", tc.name, got, tc.want)
		}
	}
}
