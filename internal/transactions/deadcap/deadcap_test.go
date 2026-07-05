package deadcap

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// TestCharge pins the §8 formula: 35% × annual salary × remaining years, 50% if
// restructured, 0 when claimed / no years / no salary. Salaries are exact cents
// ($1M = 100_000_000 cents).
func TestCharge(t *testing.T) {
	const m = 100_000_000 // one $1M in cents
	for _, tc := range []struct {
		name           string
		salary         domain.Money
		remaining      int
		isRestructured bool
		claimed        bool
		want           domain.Money
	}{
		// $10M, 2 years left, standard: 35% × 10M × 2 = $7M.
		{"standard 35pct", 10 * m, 2, false, false, 7 * m},
		// $10M, 2 years, restructured: 50% × 10M × 2 = $10M.
		{"restructured 50pct", 10 * m, 2, true, false, 10 * m},
		// One remaining year: 35% × $8M × 1 = $2.8M.
		{"one year", 8 * m, 1, false, false, 280_000_000},
		// Claimed off waivers → obligation ends, $0 regardless of terms.
		{"claimed is zero", 10 * m, 3, true, true, 0},
		// Expiring/UFA deal (0 remaining) → $0 even though salaried.
		{"no remaining years", 10 * m, 0, false, false, 0},
		{"negative remaining clamps to zero", 10 * m, -1, false, false, 0},
		{"zero salary", 0, 3, false, false, 0},
		// $10k snap (universal grid, governs over the old cent rounding): the charge lands on
		// the nearest $10,000. 35% × $6,010,000 × 1 = $2,103,500 → snaps DOWN to $2,100,000.
		{"snaps down to $10k grid", 601_000_000, 1, false, false, 210_000_000},
		// 35% × $6,100,000 × 1 = $2,135,000 → snaps half-up to $2,140,000.
		{"snaps half-up to $10k grid", 610_000_000, 1, false, false, 214_000_000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := Charge(tc.salary, tc.remaining, tc.isRestructured, tc.claimed)
			if got != tc.want {
				t.Errorf("Charge(%d, %d, restructured=%v, claimed=%v) = %d, want %d",
					tc.salary, tc.remaining, tc.isRestructured, tc.claimed, got, tc.want)
			}
		})
	}
}
