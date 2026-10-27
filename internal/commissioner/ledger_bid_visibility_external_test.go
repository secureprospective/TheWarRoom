package commissioner_test

import (
	"slices"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

type bidVisibilityAuth struct{}

func (bidVisibilityAuth) Allow(_, principal, _ string) (string, error) {
	if principal == "commish" {
		return "commissioner", nil
	}
	return "owner", nil
}

func TestLedgerSealedBidVisibility(t *testing.T) {
	f := newMarketFixture(t)
	f.setup = true
	bid := f.act(t, commissioner.BidPlace, commissioner.RawArgs{Bid: &commissioner.RawBidArgs{
		Player: "5001", Franchise: "franchise.01", Salary: 500000000, Years: 2,
	}}, "owner.one")
	check := func(stage string, other, commish bool) {
		t.Helper()
		for _, cell := range []struct {
			name, principal string
			want            bool
		}{
			{"other owner", "owner.two", other},
			{"commissioner", "commish", commish},
			{"bid owner", "owner.one", true},
		} {
			t.Run(stage+"/"+cell.name, func(t *testing.T) {
				ledger, err := commissioner.Reading(rolloverLeague, cell.principal, f.b.records, bidVisibilityAuth{})
				if err != nil {
					t.Fatal(err)
				}
				got := slices.ContainsFunc(ledger.Entries, func(entry commissioner.Entry) bool { return entry.ID == bid })
				if got != cell.want {
					t.Errorf("bid visible = %v, want %v", got, cell.want)
				}
			})
		}
	}
	check("undecided", false, false)
	f.act(t, commissioner.BidsProcess, commissioner.RawArgs{Bid: &commissioner.RawBidArgs{Player: "5001"}}, "commish")
	check("decided", false, true)
}
