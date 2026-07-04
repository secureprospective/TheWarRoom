package contracts_test

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/transactions/contracts"
)

const m = domain.Money(100_000_000) // $1M in cents

// TestMaxMove pins the §11 restructure tier table exactly, including the boundaries: the
// tier is inclusive at its threshold, and a salary below $3M is ineligible (ok=false).
func TestMaxMove(t *testing.T) {
	cases := []struct {
		name     string
		salary   domain.Money
		wantMax  domain.Money
		wantElig bool
	}{
		{"below floor", 2999999 + 1, 0, false}, // $2,999,999.99+1c … still < $3M
		{"just under $3M", 3*m - 1, 0, false},
		{"exactly $3M", 3 * m, 1 * m, true},
		{"between 3 and 6", 5 * m, 1 * m, true},
		{"just under $6M", 6*m - 1, 1 * m, true},
		{"exactly $6M", 6 * m, 2 * m, true},
		{"just under $12M", 12*m - 1, 2 * m, true},
		{"exactly $12M", 12 * m, 3 * m, true},
		{"well above $12M", 40 * m, 3 * m, true},
		{"zero salary", 0, 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gotMax, gotElig := contracts.MaxMove(c.salary)
			if gotMax != c.wantMax || gotElig != c.wantElig {
				t.Fatalf("MaxMove(%s) = (%s, %v), want (%s, %v)",
					c.salary, gotMax, gotElig, c.wantMax, c.wantElig)
			}
		})
	}
}
