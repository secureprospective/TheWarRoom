package history

import (
	"context"
	"math"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/powerrankings"
)

// weekOneLineups sets every lineup in week 1 only, so the later weeks stay open.
type weekOneLineups struct{}

func (weekOneLineups) WeekLineup(_ context.Context, _ []commissioner.Record, _, franchise string, _, week int) (commissioner.WeekLineup, error) {
	if week != 1 {
		return commissioner.WeekLineup{}, nil
	}
	return commissioner.WeekLineup{Set: true, Starters: []commissioner.Starter{{Player: "10" + strings.TrimPrefix(franchise, "franchise."),
		Name: franchise + " starter"}}}, nil
}

// oddsValues gives franchise n one available linebacker worth n points a game.
func oddsValues(n int) commissioner.OddsValues {
	var out commissioner.OddsValues
	for i := 1; i <= n; i++ {
		out.Players = append(out.Players, commissioner.OddsPlayer{Franchise: franchiseID(i), Position: "LB", NowPPG: float64(i), Available: true})
	}
	return out
}

// The 32-team default: two conferences of four divisions, a three-week season with week 1 played.
func TestNativeOddsOnTheDefaultLeague(t *testing.T) {
	t.Parallel()
	f := bracketTest(t)
	runScoreCommand(t, f.s, ruleCommand("schedule.regularWeeks", "3", "now", "Three weeks", f.rev(t)), "creator")
	f.s.SetNativeLineups(weekOneLineups{})
	cases := []struct {
		name   string
		values commissioner.OddsValues
		check  func(t *testing.T, out commissioner.NativeOdds)
	}{
		{"no values: standings stand, the three numbers say why they are missing", commissioner.OddsValues{Words: "Values are still being prepared"},
			func(t *testing.T, out commissioner.NativeOdds) {
				if len(out.Rows) != 32 || out.Rows[0].PlayoffOdds != nil || out.Rows[0].Power != 0 || len(out.Games) != 0 ||
					out.OddsWords != "Values are still being prepared" || out.PowerWords != out.OddsWords {
					t.Fatalf("missing values must leave every number empty with its words: %+v", out)
				}
			}},
		{"values: power, this week's chances and playoff odds", oddsValues(32), func(t *testing.T, out commissioner.NativeOdds) {
			bracket := f.bracket(t)
			field := bracket.Rules.PerConference * 2
			var sum float64
			power := map[string]int{}
			for _, row := range out.Rows {
				if row.PlayoffOdds == nil || *row.PlayoffOdds < 0 || *row.PlayoffOdds > 1 {
					t.Fatalf("%s has no playoff odds: %+v", row.Franchise, row)
				}
				sum += *row.PlayoffOdds
				power[row.Franchise] = row.Power
			}
			if field == 0 || math.Abs(sum-float64(field)) > 1e-9 {
				t.Fatalf("the odds must add to the field size %d, got %.4f (%s)", field, sum, out.OddsWords)
			}
			if power[franchiseID(32)] != 1 || power[franchiseID(1)] != 32 {
				t.Fatalf("the strongest lineup with the best result ranks first: %v", power)
			}
			if out.Week != 2 || len(out.Games) != 16 {
				t.Fatalf("week 2's sixteen games carry a chance: week %d, %d games", out.Week, len(out.Games))
			}
			expected := map[string]float64{}
			for _, row := range out.Rows {
				expected[row.Franchise] = row.Expected
			}
			for _, g := range out.Games {
				want := powerrankings.WinProbability(expected[g.Home], expected[g.Away])
				if math.Abs(g.HomeWin-want) > 1e-3 {
					t.Fatalf("%s v %s: %.4f, WinProbability %.4f", g.Home, g.Away, g.HomeWin, want)
				}
			}
			best, worst := odds(out, franchiseID(32)), odds(out, franchiseID(1))
			if best <= worst {
				t.Fatalf("1-0 and strongest (%.3f) must beat 0-1 and weakest (%.3f)", best, worst)
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := f.s.NativeOdds(t.Context(), bracketLeague, 2026, tc.values)
			if err != nil {
				t.Fatal(err)
			}
			if out.Status != "Ready" {
				t.Fatalf("standings are ranked: %s %s", out.Status, out.Words)
			}
			tc.check(t, out)
			again, err := f.s.NativeOdds(t.Context(), bracketLeague, 2026, tc.values)
			if err != nil || again.Seed != out.Seed {
				t.Fatalf("the same league reads the same seed: %v", err)
			}
			for _, row := range out.Rows {
				if odds(again, row.Franchise) != odds(out, row.Franchise) {
					t.Fatalf("the recorded seed must give the same odds for %s", row.Franchise)
				}
			}
		})
	}
}

// An unpublished stretch of the season holds the odds back and says so; power and chances still read.
func TestNativeOddsWaitForTheWholeSchedule(t *testing.T) {
	t.Parallel()
	f := bracketTest(t)
	runScoreCommand(t, f.s, ruleCommand("schedule.regularWeeks", "5", "now", "Five weeks", f.rev(t)), "creator")
	f.s.SetNativeLineups(weekOneLineups{})
	out, err := f.s.NativeOdds(t.Context(), bracketLeague, 2026, oddsValues(32))
	if err != nil {
		t.Fatal(err)
	}
	if out.Rows[0].PlayoffOdds != nil || !strings.Contains(out.OddsWords, "covers only 3 of 5 weeks") || out.Rows[0].Power == 0 ||
		len(out.Games) == 0 {
		t.Fatalf("odds wait for the schedule, the rest reads: %+v", out)
	}
}

func odds(out commissioner.NativeOdds, id string) float64 {
	for _, row := range out.Rows {
		if row.Franchise == id && row.PlayoffOdds != nil {
			return *row.PlayoffOdds
		}
	}
	return -1
}
