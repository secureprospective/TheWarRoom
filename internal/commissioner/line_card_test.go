package commissioner

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
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
func lineCardLeague() (leaguerules.RuleSet, *RawNativeSnapshot, nativecheck.Facts) {
	rules := leaguerules.Default32()
	rules.Season = 2026
	rules.Roster.Limits = leaguerules.Known(leaguerules.Bounds{Min: leaguerules.Known(0), Max: leaguerules.Known(44)})
	seed := &RawNativeSnapshot{Players: []NativePlayer{
		{ID: balCard, Franchise: "a", Status: domain.RosterActive, ExpirationYear: 2026},
		{ID: "1000", Franchise: "a", Status: domain.RosterActive, ExpirationYear: 2026},
		{ID: pitCard, Franchise: "b", Status: domain.RosterActive, ExpirationYear: 2026},
		{ID: "2000", Franchise: "b", Status: domain.RosterActive, ExpirationYear: 2026},
	}}
	facts := nativecheck.Facts{"1000": {Position: leaguerules.WR}, "2000": {Position: leaguerules.WR},
		"3000": {Position: leaguerules.WR}}
	return rules, seed, facts
}

func withPlayers(seed *RawNativeSnapshot, change func([]NativePlayer) []NativePlayer) *RawNativeSnapshot {
	out := *seed
	out.Players = change(append([]NativePlayer{}, seed.Players...))
	return &out
}

func TestLineCardTradeOnlyOneForOne(t *testing.T) {
	rules, seed, facts := lineCardLeague()
	state, err := governance.Replay([]governance.Event{governance.LeagueCreated{Creator: "creator",
		Franchises: []governance.FranchiseID{"a", "b"}}})
	if err != nil {
		t.Fatal(err)
	}
	swap := []TradeAsset{{From: "a", To: "b", Player: balCard}, {From: "b", To: "a", Player: pitCard}}
	forPlayer := []TradeAsset{{From: "a", To: "b", Player: balCard}, {From: "b", To: "a", Player: "2000"}}
	alone := []TradeAsset{{From: "a", To: "b", Player: balCard}}
	players := []TradeAsset{{From: "a", To: "b", Player: "1000"}, {From: "b", To: "a", Player: "2000"}}
	for _, gate := range []bool{true, false} {
		for name, c := range map[string]struct {
			assets []TradeAsset
			legal  bool
		}{"card for card": {swap, true}, "card for player": {forPlayer, false}, "card alone": {alone, false},
			"players only": {players, true}} {
			x := tradeExecution{seed: seed, state: state, rules: rules, facts: facts, zone: time.UTC}
			err := x.legal(TradeState{Parties: []string{"a", "b"}, Assets: c.assets, Policy: TradePolicy{RosterGate: gate}})
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
		after *RawNativeSnapshot
		legal bool
	}{
		"as seeded":    {seed, true},
		"cut the card": {withPlayers(seed, func(p []NativePlayer) []NativePlayer { return p[1:] }), false},
		"cut a player": {withPlayers(seed, func(p []NativePlayer) []NativePlayer { return append(p[:1], p[2:]...) }), true},
		"card to IR":   {withPlayers(seed, func(p []NativePlayer) []NativePlayer { p[0].Status = domain.RosterIR; return p }), false},
		"card to taxi": {withPlayers(seed, func(p []NativePlayer) []NativePlayer { p[0].Status = domain.RosterTaxi; return p }), false},
		"claim a second card": {withPlayers(seed, func(p []NativePlayer) []NativePlayer {
			return append(p, NativePlayer{ID: ariCard, Franchise: "a", Status: domain.RosterActive})
		}), false},
		"claim a player": {withPlayers(seed, func(p []NativePlayer) []NativePlayer {
			return append(p, NativePlayer{ID: "3000", Franchise: "a", Status: domain.RosterActive})
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
	w := &waiverWindow{ID: "w", Season: 2026, Player: NativePlayer{ID: ariCard, Franchise: "c", ExpirationYear: 2026}}
	if err := waiverClaimAllows(seed, rules, w, "a", facts, time.UTC); err == nil || !strings.Contains(err.Error(), "O-Line card") {
		t.Fatalf("second card claimed: %v", err)
	}
}

// Legacy has no OL bound, so cards are off there: the rule never refuses a Legacy roster.
func TestLineCardRuleIsOffInLegacy(t *testing.T) {
	rules := leaguerules.LegacyNFL()
	if err := nativecheck.LineCardsHeld(rules.Roster, nil); err != nil {
		t.Fatal(err)
	}
}

// realLineGames are two real 2024 games (2024_01_PIT_ATL, 2024_02_BUF_MIA) built from held nflverse
// play-by-play and PFR rushing, the fixtures the study components were checked against.
func realLineGames(t *testing.T) *source.HeldLineGames {
	t.Helper()
	open := func(name string) *os.File {
		f, err := os.Open("../fantasyscore/source/testdata/" + name)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = f.Close() })
		return f
	}
	plays, err := source.ReadLinePlays(open("line_plays.csv"))
	if err != nil {
		t.Fatal(err)
	}
	rushing, err := source.ReadLineRushing(open("line_rushing.csv"))
	if err != nil {
		t.Fatal(err)
	}
	rbs := map[string]bool{}
	for _, row := range rushing {
		rbs[row.Fields["pfr_player_id"]] = row.Fields["league_rb"] == "1"
	}
	games, err := source.LineGames(plays, rushing, func(id string) bool { return rbs[id] })
	if err != nil {
		t.Fatal(err)
	}
	return &source.HeldLineGames{SHA256: "held", Games: games}
}

// A started card scores its team's real game under Set B (the study's values); a team with no game in
// a held week scores 0 (bye); an unheld week and a Legacy league score nothing.
func TestLineCardScoresARealGame(t *testing.T) {
	evidence := source.NativeEvidence{OLine: realLineGames(t)}
	rules := leaguerules.Default32Scoring()
	for _, c := range []struct {
		team   string
		week   int
		status string
		points leaguerules.Decimal
	}{
		{"PIT", 1, "Complete", 13300}, {"ATL", 1, "Complete", 8700},
		{"BUF", 2, "Complete", 57800}, {"MIA", 2, "Complete", 25900},
		{"BAL", 1, "Complete", 0}, {"PIT", 2, "Complete", 0},
		{"PIT", 3, "Unavailable", -1},
	} {
		out := NativePlayerScore{Season: 2024, Week: c.week, Status: "Unavailable", Items: []NativeScoreItem{}}
		got, err := out.countLineCard(rules, c.team, evidence, nil, 0)
		if err != nil || got.Status != c.status {
			t.Fatalf("%s week %d: %+v %v", c.team, c.week, got, err)
		}
		if c.points >= 0 && (got.Points == nil || *got.Points != c.points) {
			t.Fatalf("%s week %d: %+v, study %d", c.team, c.week, got.Points, c.points)
		}
		if c.points < 0 && got.Points != nil {
			t.Fatalf("%s week %d scored an unheld week", c.team, c.week)
		}
	}
	legacy := leaguerules.LegacyNFL().Scoring
	if got, _ := (NativePlayerScore{Week: 1}).countLineCard(legacy, "PIT", evidence, nil, 0); got.Points != nil {
		t.Fatalf("Legacy scored a card: %+v", got)
	}
}
