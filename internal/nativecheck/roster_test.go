package nativecheck

import (
	"fmt"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func players(status domain.RosterStatus, n int, prefix string) []RosterPlayer {
	out := make([]RosterPlayer, n)
	for i := range out {
		out[i] = RosterPlayer{ID: fmt.Sprintf("%s%03d", prefix, i), Status: status}
	}
	return out
}

func roster(active, ir, taxi int) []RosterPlayer {
	all := players(domain.RosterActive, active, "a")
	all = append(all, players(domain.RosterIR, ir, "i")...)
	return append(all, players(domain.RosterTaxi, taxi, "t")...)
}

func TestRosterSizeIsJudgedOnTheActiveRosterOnly(t *testing.T) {
	rules := leaguerules.LegacyNFL().Roster
	for _, c := range []struct {
		name            string
		active, ir, tax int
		fits            bool
		words           string
	}{
		{"too many", 82, 0, 0, false, "Active roster 82 of 80: 2 over"},
		{"too few", 15, 0, 0, false, "Active roster 15 of 18: 3 under"},
		{"lowest", 18, 0, 0, true, "Within 18 to 80"},
		{"highest", 80, 0, 0, true, "Within 18 to 80"},
		{"reserves do not count", 80, 12, 8, true, "Within 18 to 80"},
		{"reserves do not rescue a short roster", 17, 12, 8, false, "Active roster 17 of 18: 1 under"},
	} {
		v := CheckRoster(rules, "franchise.01", roster(c.active, c.ir, c.tax), nil)
		if v.Fits != c.fits || v.Words != c.words || v.Counted != c.active {
			t.Errorf("%s: %+v", c.name, v)
		}
	}
}

func TestLeagueThatCountsReservesCountsThem(t *testing.T) {
	rules := leaguerules.LegacyNFL().Roster
	rules.ReserveExcluded = leaguerules.Known(false)
	v := CheckRoster(rules, "franchise.01", roster(70, 8, 6), nil)
	if v.Fits || v.Counted != 84 || v.Words != "Whole roster 84 of 80: 4 over" {
		t.Fatalf("%+v", v)
	}
}

func TestRosterSizeWithOneEndUnset(t *testing.T) {
	rules := leaguerules.LegacyNFL().Roster
	rules.Limits = leaguerules.Known(leaguerules.Bounds{Max: leaguerules.Known(30)})
	if v := CheckRoster(rules, "f", roster(30, 0, 0), nil); !v.Fits || v.Words != "Within the maximum of 30" {
		t.Fatalf("%+v", v)
	}
	if v := CheckRoster(rules, "f", roster(0, 0, 0), nil); !v.Fits {
		t.Fatalf("an unset minimum was applied: %+v", v)
	}
	rules.Limits = leaguerules.Needs[leaguerules.Bounds]("")
	if v := CheckRoster(rules, "f", roster(30, 0, 0), nil); v.Fits || !strings.HasPrefix(v.Words, "Needs a ruling") {
		t.Fatalf("an unset size was judged: %+v", v)
	}
	rules = leaguerules.LegacyNFL().Roster
	rules.ReserveExcluded = leaguerules.Needs[bool]("")
	if v := CheckRoster(rules, "f", roster(30, 0, 0), nil); v.Fits || !strings.HasPrefix(v.Words, "Needs a ruling") {
		t.Fatalf("an unset reserve question was guessed: %+v", v)
	}
}

func TestRosterPositionLimits(t *testing.T) {
	rules := leaguerules.LegacyNFL().Roster
	rules.PositionBounds = leaguerules.Known(map[leaguerules.Position]leaguerules.Bounds{
		leaguerules.QB: {Min: leaguerules.Known(2), Max: leaguerules.Known(4)},
		leaguerules.PK: {Max: leaguerules.Known(1)},
	})
	facts := Facts{}
	roster := players(domain.RosterActive, 30, "a")
	for i, p := range roster {
		position := leaguerules.WR
		switch i {
		case 0:
			position = leaguerules.QB
		case 1, 2:
			position = leaguerules.PK
		}
		facts[p.ID] = PlayerFacts{Position: position}
	}
	v := CheckRoster(rules, "f", roster, facts)
	if v.Fits || len(v.Problems) != 2 || v.Problems[0] != "QB 1 of 2: 1 under" || v.Problems[1] != "PK 2 of 1: 1 over" {
		t.Fatalf("%+v", v)
	}
	if v.Words != "Within 18 to 80" {
		t.Fatalf("a position problem changed the size headline: %q", v.Words)
	}
	facts["a001"], facts["a002"] = PlayerFacts{Position: leaguerules.QB}, PlayerFacts{Position: leaguerules.QB}
	if v := CheckRoster(rules, "f", roster, facts); !v.Fits || len(v.Problems) != 0 {
		t.Fatalf("a roster inside its position limits did not fit: %+v", v)
	}
}

func TestRosterPositionsNotHeldAreSaidNotGuessed(t *testing.T) {
	rules := leaguerules.LegacyNFL().Roster
	rules.PositionBounds = leaguerules.Known(map[leaguerules.Position]leaguerules.Bounds{
		leaguerules.QB: {Min: leaguerules.Known(1), Max: leaguerules.Known(4)}})
	v := CheckRoster(rules, "f", roster(30, 0, 0), nil)
	if v.Fits || len(v.Problems) != 0 || len(v.Notes) != 1 || !strings.Contains(v.Notes[0], "30 players") {
		t.Fatalf("%+v", v)
	}
	if v := CheckRoster(leaguerules.LegacyNFL().Roster, "f", roster(30, 0, 0), nil); !v.Fits || len(v.Notes) != 0 {
		t.Fatalf("a league with no position limits needed positions: %+v", v)
	}
}
