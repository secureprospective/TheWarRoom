package market

import (
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

func TestMarketSinglePoolLaws(t *testing.T) {
	for _, row := range []struct {
		name string
		run  func(*testing.T)
	}{
		{"rostered bid refused", func(t *testing.T) {
			seed := ledgerSnapshot()
			x := bidExecution{assets: commissioner.NativeAssets{Snapshot: seed}, facts: nativecheck.Facts{"0003": {Name: "Starter"}}}
			err := x.place(&commissioner.RawCommand{Args: commissioner.RawArgs{Bid: &commissioner.RawBidArgs{Player: "0003", Franchise: "franchise.01"}}})
			if err == nil || err.Error() != "Starter is on a roster; only free agents can be bid on" {
				t.Fatalf("refusal: %v", err)
			}
		}},
		{"released player awarded", func(t *testing.T) {
			f := newMarketFixture(t)
			f.setup = true
			w := f.act(t, commissioner.RosterCut, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{Player: "0003", Franchise: "franchise.02"}}, "owner.two")
			f.act(t, commissioner.WaiverClaim, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{Window: w, Franchise: "franchise.01"}}, "owner.one")
			f.now = f.now.Add(7 * 24 * time.Hour)
			f.act(t, commissioner.WaiverProcess, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{Window: w}}, "creator")
			got, err := commissioner.NativeAssetsReading(rolloverLeague, f.b.records)
			if err != nil {
				t.Fatal(err)
			}
			for _, p := range got.Snapshot.Players {
				if p.ID == "0003" {
					if p.Franchise != "franchise.01" {
						t.Fatalf("owner: %s", p.Franchise)
					}
					return
				}
			}
			t.Fatal("player missing")
		}},
		{"accepted trade moves player", func(t *testing.T) {
			f := newMarketFixture(t)
			f.setup = true
			for _, who := range []string{"dot.a", "dot.b", "dot.c", "dot.d"} {
				f.b.add(commissioner.RawCommand{League: rolloverLeague, Command: roleGrant, Args: commissioner.RawArgs{People: &commissioner.RawPeople{Principal: who, Role: governance.DOT}}, Effective: "now", Visibility: "private"}, seasonMoment(5))
			}
			offer := f.act(t, commissioner.TradeOffer, commissioner.RawArgs{Trade: &commissioner.RawTradeArgs{Franchise: "franchise.02", Parties: []string{"franchise.02", "franchise.01"}, Assets: []commissioner.TradeAsset{{From: "franchise.02", To: "franchise.01", Player: "0003"}}}}, "owner.two")
			f.act(t, commissioner.TradeAccept, commissioner.RawArgs{Trade: &commissioner.RawTradeArgs{ID: offer, Franchise: "franchise.01"}}, "owner.one")
			for _, who := range []string{"dot.a", "dot.b", "dot.c"} {
				f.act(t, commissioner.TradeReview, commissioner.RawArgs{Trade: &commissioner.RawTradeArgs{ID: offer, Decision: "approve"}}, who)
			}
			got, err := commissioner.NativeAssetsReading(rolloverLeague, f.b.records)
			if err != nil {
				t.Fatal(err)
			}
			for _, p := range got.Snapshot.Players {
				if p.ID == "0003" {
					if p.Franchise != "franchise.01" {
						t.Fatalf("owner: %s", p.Franchise)
					}
					return
				}
			}
			t.Fatal("player missing")
		}},
	} {
		t.Run(row.name, row.run)
	}
}
