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
		// Rounding: 35% of a non-round cent value rounds half-up on the final cent.
		// salary = 3 cents, 1 year: 3 × 35 / 100 = 1.05 → 1 cent.
		{"round down", 3, 1, false, false, 1},
		// salary = 10 cents, 1 year: 10 × 35 / 100 = 3.5 → 4 cents (half-up).
		{"round half up", 10, 1, false, false, 4},
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
