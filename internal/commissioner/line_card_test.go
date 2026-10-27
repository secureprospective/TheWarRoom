package commissioner

import (
	"os"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

const (
	balCard = "990003"
	pitCard = "990027"
	ariCard = "990001"
)

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
