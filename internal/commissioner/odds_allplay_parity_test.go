package commissioner

import (
	"reflect"

	"testing"

	"github.com/secureprospective/TheWarRoom/internal/engine/competition"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func TestOddsAllPlayRecordedSeed(t *testing.T) {
	s := NativeStandings{
		teams:    []competition.RankTeam{{ID: "a"}, {ID: "b"}, {ID: "c"}, {ID: "d"}},
		criteria: []leaguerules.Criterion{leaguerules.AllPlayPercentage},
		counted: seasonGames{allPlay: map[string]competition.Tally{}, points: map[string]leaguerules.Decimal{}, open: []OpenGame{
			{Week: 1, Home: "a", Away: "b"}, {Week: 1, Home: "c", Away: "d"},
			{Week: 2, Home: "a", Away: "c"}, {Week: 2, Home: "b", Away: "d"},
		}},
	}
	field := competition.FieldInput{Conferences: []string{"league"}, Rules: competition.PlayoffRules{PerConference: 2}, Teams: map[string]competition.FieldTeam{
		"a": {Conference: "league"}, "b": {Conference: "league"}, "c": {Conference: "league"}, "d": {Conference: "league"},
	}}
	got, words := playOut(s, map[string]float64{"a": 110, "b": 100, "c": 90, "d": 80}, field, 250, 17)
	want := map[string]int{"a": 176, "b": 148, "c": 109, "d": 67}
	if words != "" || !reflect.DeepEqual(got, want) {
		t.Fatalf("counts %v words %q", got, words)
	}
}
