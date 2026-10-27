package commissioner

import (
	"fmt"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// Starting cards belong to the auction, never to franchise creation order.
func TestDefault32LeagueStartsWithUnassignedAuctionLots(t *testing.T) {
	b := newLedger(t, PresetDefault32)
	raw, err := Default32SeedCommand(b.records, "native.test", "start", "Start", 1, seasonMoment(2))
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckNativeAppend(raw, b.records); err != nil {
		t.Fatal(err)
	}
	b.add(raw, seasonMoment(2))
	assets, err := NativeAssetsReading("native.test", b.records, seasonMoment(3))
	if err != nil || !assets.Seeded || len(assets.Snapshot.Players) != 0 || len(assets.Snapshot.Pool) != 32 {
		t.Fatalf("%+v %v", assets, err)
	}
	again, err := Default32SeedCommand(b.records, "native.test", "start", "Start", 1, seasonMoment(4))
	if err != nil || again.Args.Native.RosterAt != raw.Args.Native.RosterAt {
		t.Fatalf("retry: %v", err)
	}
	b.add(RawCommand{League: "native.test", Command: "commish.seat.replace", Args: RawArgs{People: &RawPeople{Franchise: "franchise.01", Principal: "creator"}}}, seasonMoment(3))
	lots, err := NativeBidsReading("native.test", "franchise.01", "creator", b.records, seasonMoment(3), time.UTC, nativecheck.Facts{})
	if err != nil || len(lots.Players) != 32 {
		t.Fatalf("auction lots: %+v %v", lots, err)
	}
	for _, lot := range lots.Players {
		if lot.Mode != string(leaguerules.Open) || lot.Leader != nil {
			t.Fatalf("not unassigned open lot: %+v", lot)
		}
	}
}

func TestDefault32StartRefusesAssignedOrIncompleteCards(t *testing.T) {
	b := newLedger(t, PresetDefault32)
	raw, err := Default32SeedCommand(b.records, "native.test", "start", "Start", 1, seasonMoment(2))
	if err != nil {
		t.Fatal(err)
	}
	for name, change := range map[string]func(s *RawNativeSnapshot){
		"assigned card": func(s *RawNativeSnapshot) {
			s.Players = []NativePlayer{{ID: balCard, Franchise: "franchise.01", Status: domain.RosterActive, ExpirationYear: 2026}}
		},
		"missing lot":  func(s *RawNativeSnapshot) { s.Pool = s.Pool[:31] },
		"duplicate":    func(s *RawNativeSnapshot) { s.Pool[1] = s.Pool[0] },
		"not card":     func(s *RawNativeSnapshot) { s.Pool[0].Player = "1000" },
		"wrong season": func(s *RawNativeSnapshot) { s.Season = 2027 },
	} {
		snap := *raw.Args.Native
		snap.Pool = append([]NativePoolEntry{}, snap.Pool...)
		change(&snap)
		bad := raw
		bad.Args.Native = &snap
		if err := CheckNativeAppend(bad, b.records); err == nil {
			t.Errorf("%s admitted", name)
		}
	}
	legacy := newLedger(t, PresetLegacy)
	if _, err := Default32Start("native.test", legacy.records, seasonMoment(2)); err == nil {
		t.Fatal("Legacy accepted card start")
	}
}

// A card trade that leaves the receiver over the 44 active limit is refused until the owner cuts.
func TestLineCardTradeMustCutToReceive(t *testing.T) {
	rules := leaguerules.Default32()
	rules.Season = 2026
	facts := nativecheck.Facts{}
	seed := &RawNativeSnapshot{Players: []NativePlayer{
		{ID: balCard, Franchise: "a", Status: domain.RosterActive, ExpirationYear: 2026},
		{ID: pitCard, Franchise: "b", Status: domain.RosterActive, ExpirationYear: 2026}}}
	for i := range 43 {
		for _, f := range []string{"a", "b"} {
			id := fmt.Sprintf("%d", map[string]int{"a": 1000, "b": 2000}[f]+i)
			seed.Players = append(seed.Players, NativePlayer{ID: id, Franchise: f, Status: domain.RosterActive, ExpirationYear: 2026})
			facts[id] = nativecheck.PlayerFacts{Position: []leaguerules.Position{leaguerules.WR, leaguerules.DE}[i%2]}
		}
	}
	state, err := governance.Replay([]governance.Event{governance.LeagueCreated{Creator: "creator",
		Franchises: []governance.FranchiseID{"a", "b"}}})
	if err != nil {
		t.Fatal(err)
	}
	trade := TradeState{Parties: []string{"a", "b"}, Policy: TradePolicy{RosterGate: true}, Assets: []TradeAsset{
		{From: "a", To: "b", Player: balCard}, {From: "a", To: "b", Player: "1000"}, {From: "b", To: "a", Player: pitCard}}}
	full := tradeExecution{seed: seed, state: state, rules: rules, facts: facts, zone: time.UTC}
	if err := full.legal(trade); err == nil {
		t.Fatal("b at 44 must cut before receiving a card and a player for its card")
	}
	cut := withPlayers(seed, func(p []NativePlayer) []NativePlayer { return p[:len(p)-1] })
	if last := seed.Players[len(seed.Players)-1]; last.Franchise != "b" {
		t.Fatalf("fixture: %+v", last)
	}
	after := tradeExecution{seed: cut, state: state, rules: rules, facts: facts, zone: time.UTC}
	if err := after.legal(trade); err != nil {
		t.Fatalf("after the cut the trade is legal: %v", err)
	}
}
