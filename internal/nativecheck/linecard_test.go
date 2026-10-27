package nativecheck

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func TestLineCardIDs(t *testing.T) {
	if len(LineCardTeams()) != 32 {
		t.Fatalf("%d teams", len(LineCardTeams()))
	}
	for _, team := range LineCardTeams() {
		id, ok := LineCardID(team)
		back, isCard := LineCardTeam(id)
		if !ok || !isCard || back != team {
			t.Fatalf("%s -> %s -> %s", team, id, back)
		}
	}
	if id, _ := LineCardID("BAL"); id != "990003" {
		t.Fatalf("BAL card %s", id)
	}
	for _, id := range []string{"990000", "990033", "0990003", "15248", "0001", "BAL", ""} {
		if _, ok := LineCardTeam(id); ok {
			t.Fatalf("%q read as a card", id)
		}
	}
	if f, _ := LineCardFacts("990003"); f.Name != "BAL O-Line" || f.Position != leaguerules.OL || f.Team != "BAL" {
		t.Fatalf("facts %+v", f)
	}
}

// The roster check places a card from its id; no player feed holds it.
func TestRosterCountsTheCardWithoutFacts(t *testing.T) {
	rules := leaguerules.Default32().Roster
	rules.Limits = leaguerules.Known(leaguerules.Bounds{Min: leaguerules.Known(0), Max: leaguerules.Known(44)})
	active := func(ids ...string) []RosterPlayer {
		out := []RosterPlayer{}
		for _, id := range ids {
			out = append(out, RosterPlayer{ID: id, Status: domain.RosterActive})
		}
		return out
	}
	for name, c := range map[string]struct {
		players []RosterPlayer
		fits    bool
	}{"one": {active("990003"), true}, "none": {active(), false}, "two": {active("990003", "990027"), false}} {
		if v := CheckRoster(rules, "a", c.players, Facts{}); v.Fits != c.fits {
			t.Errorf("%s: fits %v, want %v (%v %v)", name, v.Fits, c.fits, v.Problems, v.Notes)
		}
		if err := LineCardsHeld(rules, c.players); (err == nil) != c.fits {
			t.Errorf("%s: held rule %v", name, err)
		}
	}
}
