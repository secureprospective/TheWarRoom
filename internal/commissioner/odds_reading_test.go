package commissioner

import (
	"math"
	"slices"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/engine/competition"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/powerrankings"
)

// smallLeague is four teams in one conference, one week played (a and b won), one week open (a–b, c–d) and a
// two-team field: the league's tie-breakers are the full line of win percentage, then points.
func smallLeague() (NativeStandings, competition.FieldInput) {
	teams := []competition.RankTeam{{ID: "a", Division: "east", Conference: "c"}, {ID: "b", Division: "west", Conference: "c"},
		{ID: "c", Division: "east", Conference: "c"}, {ID: "d", Division: "west", Conference: "c"}}
	pts := func(n int) leaguerules.Decimal { return leaguerules.Decimal(n * 10000) }
	s := NativeStandings{teams: teams, criteria: []leaguerules.Criterion{leaguerules.WinPercentage, leaguerules.Points},
		counted: seasonGames{
			games:   []competition.RankGame{{Home: "a", Away: "c", HomePoints: pts(100), AwayPoints: pts(90)}, {Home: "b", Away: "d", HomePoints: pts(100), AwayPoints: pts(90)}},
			points:  map[string]leaguerules.Decimal{"a": pts(100), "b": pts(100), "c": pts(90), "d": pts(90)},
			allPlay: map[string]competition.Tally{}, open: []OpenGame{{Week: 2, Home: "a", Away: "b"}, {Week: 2, Home: "c", Away: "d"}}}}
	field := competition.FieldInput{Teams: map[string]competition.FieldTeam{}, Conferences: []string{"c"},
		Rules: competition.PlayoffRules{PerConference: 2, WildCards: 2}}
	for _, t := range teams {
		field.Teams[t.ID] = competition.FieldTeam{Division: t.Division, Conference: t.Conference}
	}
	return s, field
}

func TestPlayoffOddsPlayTheOpenGamesUnderTheLeaguesRules(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		expected map[string]float64
		check    func(t *testing.T, odds map[string]float64)
	}{
		{"even teams: a and b are in unless c and d both win and pass them on points", map[string]float64{"a": 100, "b": 100, "c": 100, "d": 100},
			func(t *testing.T, odds map[string]float64) {
				if odds["a"] < 0.6 || odds["b"] < 0.6 || odds["c"] > 0.4 || odds["d"] > 0.4 {
					t.Fatalf("the leaders should be favoured: %v", odds)
				}
			}},
		{"a far stronger c–d winner always takes the second seed on points", map[string]float64{"a": 60, "b": 60, "c": 200, "d": 200},
			func(t *testing.T, odds map[string]float64) {
				// The a–b winner is 2-0; the c–d winner ties the a–b loser at 1-1 and passes it on points.
				if math.Abs(odds["a"]+odds["b"]-1) > 0.01 || math.Abs(odds["c"]+odds["d"]-1) > 0.01 {
					t.Fatalf("one of a–b and one of c–d should make it: %v", odds)
				}
			}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s, field := smallLeague()
			made, words := playOutChunks(s, tc.expected, field, 7)
			if words != "" {
				t.Fatal(words)
			}
			total, odds := 0, map[string]float64{}
			for id, n := range made {
				total += n
				odds[id] = float64(n) / OddsRuns
			}
			if total != 2*OddsRuns {
				t.Fatalf("every run seeds exactly two teams, got %d seeds over %d runs", total, OddsRuns)
			}
			tc.check(t, odds)
			again, _ := playOutChunks(s, tc.expected, field, 7)
			for id := range made {
				if again[id] != made[id] {
					t.Fatalf("the recorded seed must give the same odds: %v vs %v", made, again)
				}
			}
		})
	}
}

func TestPlayedOutScoresHaveTheSpreadWinProbabilityReads(t *testing.T) {
	t.Parallel()
	s, field := smallLeague()
	s.teams = s.teams[:2]
	s.counted.games, s.counted.points = nil, map[string]leaguerules.Decimal{}
	s.counted.open = s.counted.open[:1]
	delete(field.Teams, "c")
	delete(field.Teams, "d")
	field.Rules = competition.PlayoffRules{PerConference: 1, WildCards: 1}
	made, _ := playOutChunks(s, map[string]float64{"a": 110, "b": 100, "c": 0, "d": 0}, field, 3)
	want := powerrankings.WinProbability(110, 100)
	got := float64(made["a"]) / OddsRuns
	// A two-team league with one game and one seed: a's odds are its game chance.
	if math.Abs(got-want) > 0.035 {
		t.Fatalf("a made the field in %.3f of runs, WinProbability says %.3f", got, want)
	}
}

func TestCoinFlipsOnlyReorderTheLevelTeams(t *testing.T) {
	t.Parallel()
	r := competition.Ranking{Order: []string{"a", "b", "c", "d"}, Level: [][]string{{"b", "c"}}}
	seen := map[string]bool{}
	rng := newOddsRand(1)
	for range 50 {
		got := coinFlips(r, rng)
		if got[0] != "a" || got[3] != "d" || !slices.Contains(got[1:3], "b") || !slices.Contains(got[1:3], "c") {
			t.Fatalf("a coin flip moved a team the tie-breakers placed: %v", got)
		}
		seen[got[1]] = true
	}
	if !seen["b"] || !seen["c"] {
		t.Fatalf("both orders of a level pair must be possible: %v", seen)
	}
}
