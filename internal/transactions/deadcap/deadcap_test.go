package deadcap

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

func TestCharge(t *testing.T) {
	const m = 100_000_000
	unequal := []state.LedgerCell{{Year: 2026, Salary: 12 * m}, {Year: 2027, Salary: 8 * m}, {Year: 2028, Salary: 4 * m}}
	for _, tc := range []struct {
		name                  string
		cells                 []state.LedgerCell
		restructured, claimed bool
		want                  domain.Money
	}{
		{"own cells not current salary", unequal, false, false, 420_000_000},
		{"restructured rate", unequal, true, false, 600_000_000},
		{"claimed", unequal, true, true, 0},
		{"final year", unequal[:1], false, false, 0},
		{"empty", nil, false, false, 0},
		{"one snap", []state.LedgerCell{{Year: 2027, Salary: 601_000_000}, {Year: 2028, Salary: 601_000_000}}, false, false, 421_000_000},
		{"half up", []state.LedgerCell{{Year: 2027, Salary: 610_000_000}}, false, false, 214_000_000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := Charge(tc.cells, 2026, tc.restructured, tc.claimed); got != tc.want {
				t.Fatalf("Charge = %d, want %d", got, tc.want)
			}
		})
	}
}
