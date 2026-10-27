package market

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

const (
	balCard = "990003"
	pitCard = "990027"
	ariCard = "990001"
)

// lineCardLeague is a 32-Team Default league cut to two franchises: a holds BAL's card and a receiver,
// b holds PIT's card and a receiver. Size limits are opened so only the card rule is judged.
func lineCardLeague() (leaguerules.RuleSet, *commissioner.RawNativeSnapshot, nativecheck.Facts) {
	rules := leaguerules.Default32()
	rules.Season = 2026
	rules.Roster.Limits = leaguerules.Known(leaguerules.Bounds{Min: leaguerules.Known(0), Max: leaguerules.Known(44)})
	seed := &commissioner.RawNativeSnapshot{Players: []commissioner.NativePlayer{
		{ID: balCard, Franchise: "a", Status: domain.RosterActive, ExpirationYear: 2026},
		{ID: "1000", Franchise: "a", Status: domain.RosterActive, ExpirationYear: 2026},
		{ID: pitCard, Franchise: "b", Status: domain.RosterActive, ExpirationYear: 2026},
		{ID: "2000", Franchise: "b", Status: domain.RosterActive, ExpirationYear: 2026},
	}}
	facts := nativecheck.Facts{"1000": {Position: leaguerules.WR}, "2000": {Position: leaguerules.WR},
		"3000": {Position: leaguerules.WR}}
	return rules, seed, facts
}

func withPlayers(seed *commissioner.RawNativeSnapshot, change func([]commissioner.NativePlayer) []commissioner.NativePlayer) *commissioner.RawNativeSnapshot {
	out := *seed
	out.Players = change(append([]commissioner.NativePlayer{}, seed.Players...))
	return &out
}

func TestLineCardTradeOnlyOneForOne(t *testing.T) {
	rules, seed, facts := lineCardLeague()
	state, err := governance.Replay([]governance.Event{governance.LeagueCreated{Creator: "creator",
		Franchises: []governance.FranchiseID{"a", "b"}}})
	if err != nil {
		t.Fatal(err)
	}
	swap := []commissioner.TradeAsset{{From: "a", To: "b", Player: balCard}, {From: "b", To: "a", Player: pitCard}}
	forPlayer := []commissioner.TradeAsset{{From: "a", To: "b", Player: balCard}, {From: "b", To: "a", Player: "2000"}}
	alone := []commissioner.TradeAsset{{From: "a", To: "b", Player: balCard}}
	players := []commissioner.TradeAsset{{From: "a", To: "b", Player: "1000"}, {From: "b", To: "a", Player: "2000"}}
	for _, gate := range []bool{true, false} {
		for name, c := range map[string]struct {
			assets []commissioner.TradeAsset
			legal  bool
		}{"card for card": {swap, true}, "card for player": {forPlayer, false}, "card alone": {alone, false},
			"players only": {players, true}} {
			x := tradeExecution{seed: seed, state: state, rules: rules, facts: facts, zone: time.UTC}
			err := x.legal(commissioner.TradeState{Parties: []string{"a", "b"}, Assets: c.assets, Policy: commissioner.TradePolicy{RosterGate: gate}})
			if c.legal != (err == nil) {
				t.Errorf("%s (roster gate %v): legal %v, got %v", name, gate, c.legal, err)
			}
			if !c.legal && err != nil && !strings.Contains(err.Error(), "O-Line card") {
				t.Errorf("%s (roster gate %v): refused for another reason: %v", name, gate, err)
			}
		}
	}
}

// Cut, IR/taxi placement, waiver claims, awards and bids all judge the roster after the move with
// rosterCapAllows; a franchise left with zero or two cards is refused on each.
func TestLineCardHeldOnRosterMoves(t *testing.T) {
	rules, seed, facts := lineCardLeague()
	for name, c := range map[string]struct {
		after *commissioner.RawNativeSnapshot
		legal bool
	}{
		"as seeded":    {seed, true},
		"cut the card": {withPlayers(seed, func(p []commissioner.NativePlayer) []commissioner.NativePlayer { return p[1:] }), false},
		"cut a player": {withPlayers(seed, func(p []commissioner.NativePlayer) []commissioner.NativePlayer { return append(p[:1], p[2:]...) }), true},
		"card to IR": {withPlayers(seed, func(p []commissioner.NativePlayer) []commissioner.NativePlayer {
			p[0].Status = domain.RosterIR
			return p
		}), false},
		"card to taxi": {withPlayers(seed, func(p []commissioner.NativePlayer) []commissioner.NativePlayer {
			p[0].Status = domain.RosterTaxi
			return p
		}), false},
		"claim a second card": {withPlayers(seed, func(p []commissioner.NativePlayer) []commissioner.NativePlayer {
			return append(p, commissioner.NativePlayer{ID: ariCard, Franchise: "a", Status: domain.RosterActive})
		}), false},
		"claim a player": {withPlayers(seed, func(p []commissioner.NativePlayer) []commissioner.NativePlayer {
			return append(p, commissioner.NativePlayer{ID: "3000", Franchise: "a", Status: domain.RosterActive})
		}), true},
	} {
		err := rosterCapAllows(c.after, rules, "a", facts, time.UTC)
		if c.legal != (err == nil) {
			t.Errorf("%s: legal %v, got %v", name, c.legal, err)
		}
	}
	// A waiver claim on another team's card is judged on the claimed roster.
	rules.Contracts.ReleaseReSignSameSeason = leaguerules.Known(true)
	rules.Contracts.ClaimClearsDeadCap = leaguerules.Known(true)
	w := &commissioner.WaiverWindow{ID: "w", Season: 2026, Player: commissioner.NativePlayer{ID: ariCard, Franchise: "c", ExpirationYear: 2026}}
	if err := waiverClaimAllows(seed, rules, w, "a", facts, time.UTC); err == nil || !strings.Contains(err.Error(), "O-Line card") {
		t.Fatalf("second card claimed: %v", err)
	}
}

// A card trade that leaves the receiver over the 44 active limit is refused until the owner cuts.
func TestLineCardTradeMustCutToReceive(t *testing.T) {
	rules := leaguerules.Default32()
	rules.Season = 2026
	facts := nativecheck.Facts{}
	seed := &commissioner.RawNativeSnapshot{Players: []commissioner.NativePlayer{
		{ID: balCard, Franchise: "a", Status: domain.RosterActive, ExpirationYear: 2026},
		{ID: pitCard, Franchise: "b", Status: domain.RosterActive, ExpirationYear: 2026}}}
	for i := range 43 {
		for _, f := range []string{"a", "b"} {
			id := fmt.Sprintf("%d", map[string]int{"a": 1000, "b": 2000}[f]+i)
			seed.Players = append(seed.Players, commissioner.NativePlayer{ID: id, Franchise: f, Status: domain.RosterActive, ExpirationYear: 2026})
			facts[id] = nativecheck.PlayerFacts{Position: []leaguerules.Position{leaguerules.WR, leaguerules.DE}[i%2]}
		}
	}
	state, err := governance.Replay([]governance.Event{governance.LeagueCreated{Creator: "creator",
		Franchises: []governance.FranchiseID{"a", "b"}}})
	if err != nil {
		t.Fatal(err)
	}
	trade := commissioner.TradeState{Parties: []string{"a", "b"}, Policy: commissioner.TradePolicy{RosterGate: true}, Assets: []commissioner.TradeAsset{
		{From: "a", To: "b", Player: balCard}, {From: "a", To: "b", Player: "1000"}, {From: "b", To: "a", Player: pitCard}}}
	full := tradeExecution{seed: seed, state: state, rules: rules, facts: facts, zone: time.UTC}
	if err := full.legal(trade); err == nil {
		t.Fatal("b at 44 must cut before receiving a card and a player for its card")
	}
	cut := withPlayers(seed, func(p []commissioner.NativePlayer) []commissioner.NativePlayer { return p[:len(p)-1] })
	if last := seed.Players[len(seed.Players)-1]; last.Franchise != "b" {
		t.Fatalf("fixture: %+v", last)
	}
	after := tradeExecution{seed: cut, state: state, rules: rules, facts: facts, zone: time.UTC}
	if err := after.legal(trade); err != nil {
		t.Fatalf("after the cut the trade is legal: %v", err)
	}
}
