package commissioner

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// A 32-Team Default league records its start from its own franchises: each gets one O-Line card on a
// 1-year contract at the league minimum (DEFAULT-32 spec §3), and the snapshot reads back as its assets.
func TestDefault32LeagueStartsWithOneCardEach(t *testing.T) {
	b := newLedger(t, PresetDefault32)
	raw, err := Default32SeedCommand(b.records, "native.test", "start", "Start the league", 1, seasonMoment(2))
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckNativeAppend(raw, b.records); err != nil {
		t.Fatalf("the start is refused: %v", err)
	}
	b.add(raw, seasonMoment(2))
	assets, err := NativeAssetsReading("native.test", b.records, seasonMoment(3))
	if err != nil || !assets.Seeded {
		t.Fatalf("assets: %v %+v", err, assets)
	}
	rules, err := RulesNow(b.records, seasonMoment(3))
	if err != nil {
		t.Fatal(err)
	}
	teams := nativecheck.LineCardTeams()
	byFranchise := map[string][]nativecheck.RosterPlayer{}
	for i, p := range assets.Snapshot.Players {
		want, _ := nativecheck.LineCardID(teams[i])
		if p.ID != want || p.Franchise != fmt.Sprintf("franchise.%02d", i+1) || p.Salary != 33000000 ||
			p.ExpirationYear != 2026 || p.Status != domain.RosterActive || p.ContractStatus != domain.CStatusUFA {
			t.Fatalf("player %d: %+v", i, p)
		}
		byFranchise[p.Franchise] = append(byFranchise[p.Franchise], nativecheck.RosterPlayer{ID: p.ID})
	}
	if len(byFranchise) != 32 {
		t.Fatalf("franchises holding a card: %d", len(byFranchise))
	}
	for f, held := range byFranchise {
		if err := nativecheck.LineCardsHeld(rules.Roster, held); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
	}
	again, err := Default32SeedCommand(b.records, "native.test", "start", "Start the league", 1, seasonMoment(4))
	if err != nil || again.Args.Native.RosterAt != raw.Args.Native.RosterAt {
		t.Fatalf("a retry reuses the recorded body: %v", err)
	}
	if _, err := Default32SeedCommand(b.records, "native.test", "start", "Other", 1, seasonMoment(4)); err == nil {
		t.Fatal("a reused request with another reason must be refused")
	}
	if err := CheckNativeAppend(again, b.records); err == nil {
		t.Fatal("a started league must not start again")
	}
}

// Only one card per franchise, and nothing else, starts a 32-Team Default league; Legacy cannot start this way.
func TestDefault32StartRefusesAnythingElse(t *testing.T) {
	b := newLedger(t, PresetDefault32)
	raw, err := Default32SeedCommand(b.records, "native.test", "start", "Start", 1, seasonMoment(2))
	if err != nil {
		t.Fatal(err)
	}
	for name, change := range map[string]func(s *RawNativeSnapshot){
		"two cards on one franchise": func(s *RawNativeSnapshot) { s.Players[1].Franchise = s.Players[0].Franchise },
		"a player, not a card":       func(s *RawNativeSnapshot) { s.Players[0].ID = "1000" },
		"a card on taxi":             func(s *RawNativeSnapshot) { s.Players[0].Status = domain.RosterTaxi },
		"a 2-year card":              func(s *RawNativeSnapshot) { s.Players[0].ExpirationYear = 2027 },
		"a pick": func(s *RawNativeSnapshot) {
			s.Picks = append(s.Picks, NativePick{Season: 2027, Round: 1, Original: "franchise.01", Owner: "franchise.01"})
		},
		"another season": func(s *RawNativeSnapshot) { s.Season = 2027 },
	} {
		snap := *raw.Args.Native
		snap.Players = append([]NativePlayer{}, snap.Players...)
		change(&snap)
		bad := raw
		bad.Args.Native = &snap
		if err := CheckNativeAppend(bad, b.records); err == nil {
			t.Errorf("%s: recorded", name)
		}
	}
	legacy := newLedger(t, PresetLegacy)
	if _, err := Default32SeedCommand(legacy.records, "native.test", "start", "Start", 1, seasonMoment(2)); err == nil {
		t.Fatal("a Legacy league starts from held data, not from cards")
	}
	if err := CheckNativeAppend(raw, legacy.records); err == nil || !strings.Contains(err.Error(), "held") {
		t.Fatalf("a Legacy league must refuse a card start: %v", err)
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
