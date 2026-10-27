package leagueservice

import (
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner/market"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

func TestStartAuctionAwardsCardsByOpenBidsNotCreationOrder(t *testing.T) {
	f := newDueFixture(t)
	// A second league shares the real worker/store, without importing any rosters or schedule.
	f.league = "native.cards"
	f.execute(t, "creator", commissioner.Create, commissioner.RawArgs{Name: "Cards", Preset: commissioner.PresetDefault32, Season: 2026})
	raw, err := commissioner.Default32SeedCommand(reserveRecords(t, f), f.league, "start", "Open auction", 1, f.clock.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.ExecuteCommissionerWithFacts(t.Context(), raw, "creator", nil, f.facts, time.UTC); err != nil {
		t.Fatal(err)
	}
	for _, seat := range []struct{ team, owner string }{{"franchise.01", "owner.one"}, {"franchise.02", "owner.two"}} {
		f.execute(t, "creator", "commish.seat.replace", commissioner.RawArgs{People: &commissioner.RawPeople{Franchise: seat.team, Principal: seat.owner}})
	}
	nativecheck.AddLineCards(f.facts)
	card, _ := nativecheck.LineCardID("PIT")
	// Even a later rule selecting blind UFA bids cannot hide a league-start card auction.
	reserveRule(t, f, "bids.ufa", `"blind"`)
	f.execute(t, "owner.one", commissioner.BidPlace, commissioner.RawArgs{Bid: &commissioner.RawBidArgs{Player: card, Franchise: "franchise.01", Salary: 40000000, Years: 1}})
	f.execute(t, "owner.two", commissioner.BidPlace, commissioner.RawArgs{Bid: &commissioner.RawBidArgs{Player: card, Franchise: "franchise.02", Salary: 50000000, Years: 1}})
	reading, err := market.NativeBidsReading(f.league, "franchise.01", "owner.one", reserveRecords(t, f), f.clock.Now(), time.UTC, f.facts)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, lot := range reading.Players {
		if lot.Player == card {
			found = true
			if lot.Leader == nil || lot.Leader.Franchise != "franchise.02" || lot.Mode != "open" {
				t.Fatalf("not open: %+v", lot)
			}
		}
	}
	if !found {
		t.Fatal("card lot missing")
	}
	f.clock.instant.Add(int64(24 * time.Hour))
	if _, err := f.worker.step(t.Context()); err != nil {
		t.Fatal(err)
	}
	assets, err := commissioner.NativeAssetsReading(f.league, reserveRecords(t, f), f.clock.Now())
	if err != nil || len(assets.Snapshot.Players) != 1 {
		t.Fatalf("award: %+v %v", assets, err)
	}
	p := assets.Snapshot.Players[0]
	if p.ID != card || p.Franchise != "franchise.02" || p.Salary != 50000000 || p.ExpirationYear != 2026 || p.Status != domain.RosterActive {
		t.Fatalf("wrong award: %+v", p)
	}
	another, _ := nativecheck.LineCardID("BAL")
	raw = f.raw(t, commissioner.BidPlace, commissioner.RawArgs{Bid: &commissioner.RawBidArgs{Player: another, Franchise: "franchise.02", Salary: 40000000, Years: 1}})
	if _, err := f.store.ExecuteCommissionerWithFacts(t.Context(), raw, "owner.two", nil, f.facts, time.UTC); err == nil {
		t.Fatal("one franchise could win two cards")
	}
	raw = f.raw(t, commissioner.BidPlace, commissioner.RawArgs{Bid: &commissioner.RawBidArgs{Player: another, Franchise: "franchise.01", Salary: 40000000, Years: 2}})
	if _, err := f.store.ExecuteCommissionerWithFacts(t.Context(), raw, "owner.one", nil, f.facts, time.UTC); err == nil {
		t.Fatal("initial card got multiple years")
	}
}
