package commissioner

import (
	"context"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// perStarter scores each starter from its own prepared reading, so a week can mix a final starter
// with one whose game is still being played.
type perStarter struct {
	lineup WeekLineup
	scores map[string]NativePlayerScore
}

func (p perStarter) Lineup(context.Context, []Record, string, int) (WeekLineup, error) {
	return p.lineup, nil
}
func (p perStarter) PlayerScore(_ context.Context, _ []Record, player string, _ int) (NativePlayerScore, error) {
	return p.scores[player], nil
}

func points(v leaguerules.Decimal) *leaguerules.Decimal { return &v }

// A week in play has a running score from the starters who have points so far, and still has no
// result: provisional points are never a result.
func TestWeekInPlayHasAProvisionalScoreAndNoResult(t *testing.T) {
	t.Parallel()
	seat := NativeDirectorySeat{PeopleSeat: PeopleSeat{Franchise: "a", Name: "Alpha"}}
	src := perStarter{lineup: WeekLineup{Set: true,
		Starters: []Starter{{Player: "0099", Name: "Done"}, {Player: "0100", Name: "Playing"}}},
		scores: map[string]NativePlayerScore{
			"0099": {Status: "Complete", Points: points(120000), Words: "12 points"},
			"0100": {Status: "Incomplete", PartialPoints: points(85000),
				Words: "8.5 points counted so far; incomplete: the game is being played"},
		}}
	got, err := franchiseWeekScore(t.Context(), nil, src, leaguerules.LineupRules{}, seat, 2026, 8)
	if err != nil {
		t.Fatal(err)
	}
	if got.Points != nil {
		t.Fatalf("a week in play was given a result: %+v", got.Points)
	}
	if got.Provisional == nil || *got.Provisional != 205000 {
		t.Fatalf("provisional %v, want 20.5 points", got.Provisional)
	}
	if got.Starters[1].Provisional == nil || *got.Starters[1].Provisional != 85000 {
		t.Fatalf("starter provisional %+v", got.Starters[1])
	}
	if got.Starters[0].Points == nil || got.Starters[0].Provisional != nil {
		t.Fatalf("a final starter carries points, not provisional points: %+v", got.Starters[0])
	}
	if !strings.Contains(got.Words, "so far, from evidence that is not final") ||
		!strings.Contains(got.Words, "incomplete") {
		t.Fatalf("words %q", got.Words)
	}
}

// A starter with no evidence at all contributes nothing: a missing stat is pending, never zero.
func TestAStarterWithNoEvidenceAddsNoProvisionalPoints(t *testing.T) {
	t.Parallel()
	seat := NativeDirectorySeat{PeopleSeat: PeopleSeat{Franchise: "a", Name: "Alpha"}}
	src := perStarter{lineup: WeekLineup{Set: true, Starters: []Starter{{Player: "0099", Name: "Nothing"}}},
		scores: map[string]NativePlayerScore{
			"0099": {Status: "Incomplete", Words: "No NFL statistics are held for this season and week"},
		}}
	got, err := franchiseWeekScore(t.Context(), nil, src, leaguerules.LineupRules{}, seat, 2026, 8)
	if err != nil {
		t.Fatal(err)
	}
	if got.Points != nil || got.Provisional != nil {
		t.Fatalf("points were invented from no evidence: %+v %+v", got.Points, got.Provisional)
	}
	if !strings.Contains(got.Words, "incomplete") {
		t.Fatalf("words %q", got.Words)
	}
}

// The game carries both sides' running scores while it has no result, and neither as a result.
func TestGameWithoutAResultCarriesBothRunningScores(t *testing.T) {
	t.Parallel()
	home := NativeDirectorySeat{PeopleSeat: PeopleSeat{Franchise: "a", Name: "Alpha"}}
	away := NativeDirectorySeat{PeopleSeat: PeopleSeat{Franchise: "b", Name: "Beta"}}
	got := gameResult(8, home, away,
		FranchiseWeekScore{Provisional: points(205000), Words: "Alpha is still playing"},
		FranchiseWeekScore{Points: points(310000)})
	if got.Status != GameNoResult || got.HomePoints != nil || got.AwayPoints != nil {
		t.Fatalf("a game in play was decided: %+v", got)
	}
	if got.HomeProvisional == nil || *got.HomeProvisional != 205000 {
		t.Fatalf("home provisional %v", got.HomeProvisional)
	}
	if got.AwayProvisional == nil || *got.AwayProvisional != 310000 {
		t.Fatalf("a side already final still shows its running score: %v", got.AwayProvisional)
	}
	decided := gameResult(8, home, away, FranchiseWeekScore{Points: points(310000)},
		FranchiseWeekScore{Points: points(205000)})
	if decided.Winner != "a" || decided.HomeProvisional != nil || decided.AwayProvisional != nil {
		t.Fatalf("a decided game kept provisional points: %+v", decided)
	}
}
