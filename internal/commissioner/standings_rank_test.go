package commissioner

import (
	"context"
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/engine/competition"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func rankFixture() competition.RankInput {
	return competition.RankInput{Teams: []competition.RankTeam{{ID: "a", Name: "Alpha", Division: "d", Conference: "c"},
		{ID: "b", Name: "Beta", Division: "d", Conference: "c"}, {ID: "x", Name: "Other", Division: "e", Conference: "e"}},
		Games: []competition.RankGame{{Home: "a", Away: "b", HomePoints: 1, AwayPoints: 0},
			{Home: "a", Away: "x", HomePoints: 0, AwayPoints: 100}, {Home: "b", Away: "x", HomePoints: 10, AwayPoints: 0}},
		Points:  map[string]leaguerules.Decimal{"a": 1, "b": 10, "x": 100},
		AllPlay: map[string]competition.Tally{"a": {W: 1, L: 3}, "b": {W: 3, L: 1}, "x": {W: 2, L: 2}}}
}

func TestNativeRankingHonorsCriterionOrderAndExplains(t *testing.T) {
	in := rankFixture()
	in.Criteria = []leaguerules.Criterion{leaguerules.WinPercentage, leaguerules.HeadToHeadRecord, leaguerules.Points}
	// Restrict the tied group to the two teams whose head-to-head conflicts with their points.
	in.Teams = in.Teams[:2]
	got := competition.RankStandings(in)
	if !reflect.DeepEqual(got.Order, []string{"a", "b"}) || !strings.Contains(strings.Join(got.Why["a"], " "), "Head-to-head record") {
		t.Fatalf("head-to-head skipped: %+v", got)
	}
	in.Criteria = []leaguerules.Criterion{leaguerules.WinPercentage, leaguerules.Points, leaguerules.HeadToHeadRecord}
	got = competition.RankStandings(in)
	if !reflect.DeepEqual(got.Order, []string{"b", "a"}) || !strings.Contains(strings.Join(got.Why["b"], " "), "Points scored") {
		t.Fatalf("criteria applied out of order: %+v", got)
	}
}

func TestNativeRankingEachCriterionAndNoData(t *testing.T) {
	for _, c := range []leaguerules.Criterion{leaguerules.DivisionPercentage, leaguerules.ConferencePercentage,
		leaguerules.AllPlayPercentage, leaguerules.Points} {
		in := rankFixture()
		in.Teams = in.Teams[:2]
		in.Criteria = []leaguerules.Criterion{c}
		got := competition.RankStandings(in)
		want := "a"
		if c == leaguerules.AllPlayPercentage || c == leaguerules.Points {
			want = "b"
		}
		if got.Order[0] != want || !strings.Contains(strings.ToLower(strings.Join(got.Why[want], " ")), competition.CriterionLabel(c)) {
			t.Fatalf("%s: %+v", c, got)
		}
	}
	in := rankFixture()
	in.Games = nil
	in.AllPlay = nil
	in.Points = nil
	in.Criteria = []leaguerules.Criterion{leaguerules.WinPercentage, leaguerules.HeadToHeadRecord,
		leaguerules.DivisionPercentage, leaguerules.ConferencePercentage, leaguerules.AllPlayPercentage,
		leaguerules.Points, leaguerules.PowerRank}
	got := competition.RankStandings(in)
	if len(got.Skipped) != 7 || !reflect.DeepEqual(got.Order, []string{"a", "b", "x"}) {
		t.Fatalf("missing data invented a competitive value: %+v", got)
	}
	in = rankFixture()
	in.Teams = in.Teams[:2]
	in.AllPlay = map[string]competition.Tally{"b": {W: 3, L: 1}}
	in.Criteria = []leaguerules.Criterion{leaguerules.AllPlayPercentage, leaguerules.HeadToHeadRecord}
	got = competition.RankStandings(in)
	if got.Order[0] != "a" || !strings.Contains(strings.Join(got.Skipped, " "), "Alpha has no games yet") {
		t.Fatalf("missing data treated as zero: %+v", got)
	}
}

type scoreSource struct {
	lineup WeekLineup
	score  NativePlayerScore
	err    error
}

func (s scoreSource) Lineup(context.Context, []Record, string, int) (WeekLineup, error) {
	return s.lineup, s.err
}
func (s scoreSource) PlayerScore(context.Context, []Record, string, int) (NativePlayerScore, error) {
	return s.score, s.err
}

func TestNativeGameIncompleteTieAndExactPoints(t *testing.T) {
	a := NativeDirectorySeat{PeopleSeat: PeopleSeat{Franchise: "a", Name: "Alpha"}}
	b := NativeDirectorySeat{PeopleSeat: PeopleSeat{Franchise: "b", Name: "Beta"}}
	high, low := leaguerules.Decimal(10001), leaguerules.Decimal(10000)
	for _, why := range []string{"A starter is not reported", "Needs a ruling: tackle points"} {
		got := gameResult(1, a, b, FranchiseWeekScore{Words: why}, FranchiseWeekScore{Points: &low})
		if got.Status != GameNoResult || got.Winner != "" || got.HomePoints != nil || got.AwayPoints != nil || !strings.Contains(got.Words, why) {
			t.Fatalf("incomplete game given winner: %+v", got)
		}
	}
	got := gameResult(1, a, b, FranchiseWeekScore{Points: &high}, FranchiseWeekScore{Points: &low})
	if got.Winner != "a" || !strings.Contains(got.Words, "1.0001") {
		t.Fatal(got)
	}
	got = gameResult(1, a, b, FranchiseWeekScore{Points: &low}, FranchiseWeekScore{Points: &low})
	if got.Status != GameTie || got.Winner != "" {
		t.Fatal(got)
	}
}

func TestNativeWeekScoreNeverUsesPartialOrDuplicateStarter(t *testing.T) {
	total := leaguerules.Decimal(10000)
	seat := NativeDirectorySeat{PeopleSeat: PeopleSeat{Franchise: "a", Name: "Alpha"}}
	src := scoreSource{lineup: WeekLineup{Set: true, Starters: []Starter{{Player: "0099"}}},
		score: NativePlayerScore{Status: "Incomplete", PartialPoints: &total, Words: "Still waiting for a starter"}}
	got, err := franchiseWeekScore(t.Context(), nil, src, leaguerules.LineupRules{}, seat, 2026, 1)
	if err != nil || got.Points != nil || !strings.Contains(got.Words, "waiting") {
		t.Fatal(got, err)
	}
	src.score = NativePlayerScore{Status: "Complete", Points: &total}
	got, err = franchiseWeekScore(t.Context(), nil, src, leaguerules.LineupRules{}, seat, 2026, 1)
	if err != nil || got.Points == nil || *got.Points != total {
		t.Fatal(got, err)
	}
	src.lineup.Starters = append(src.lineup.Starters, src.lineup.Starters[0])
	if _, err = franchiseWeekScore(t.Context(), nil, src, leaguerules.LineupRules{}, seat, 2026, 1); err == nil {
		t.Fatal("duplicate starter counted")
	}
	src.err = errors.New("unreadable lineup")
	if _, err = franchiseWeekScore(t.Context(), nil, src, leaguerules.LineupRules{}, seat, 2026, 1); !errors.Is(err, src.err) {
		t.Fatal(err)
	}
	src.err = nil
	src.lineup.Starters[1].Player = "0100"
	total = leaguerules.Decimal(math.MaxInt64)
	if _, err = franchiseWeekScore(t.Context(), nil, src, leaguerules.LineupRules{}, seat, 2026, 1); err == nil {
		t.Fatal("overflow accepted")
	}
}

func TestNativeAllPlayIncludesByesAndRequiresCompleteWeek(t *testing.T) {
	one, two, three := leaguerules.Decimal(1), leaguerules.Decimal(2), leaguerules.Decimal(3)
	scores := []FranchiseWeekScore{{Franchise: "a", Points: &one}, {Franchise: "b", Points: &two}, {Franchise: "bye", Points: &three}}
	got := seasonGames{allPlay: map[string]competition.Tally{}}
	got.addAllPlay(scores)
	if got.allPlay["bye"] != (competition.Tally{W: 2}) || got.allPlay["a"] != (competition.Tally{L: 2}) {
		t.Fatal(got)
	}
	scores[2].Points = nil
	got.addAllPlay(scores)
	if got.allPlay["a"] != (competition.Tally{L: 2}) {
		t.Fatal("incomplete all-play counted")
	}
}

type unevenScores struct{}

func (unevenScores) Lineup(_ context.Context, _ []Record, id string, _ int) (WeekLineup, error) {
	return WeekLineup{Set: true, Starters: []Starter{{Player: id}}}, nil
}
func (unevenScores) PlayerScore(_ context.Context, _ []Record, id string, _ int) (NativePlayerScore, error) {
	if id == "b" {
		return NativePlayerScore{Status: "Incomplete", Words: "Not reported"}, nil
	}
	points := leaguerules.Decimal(10000)
	if id == "c" {
		points = 30000
	}
	return NativePlayerScore{Status: "Complete", Points: &points}, nil
}

func TestNativePointsCountCompleteSidesAndByeWithoutInventingResults(t *testing.T) {
	directory := NativeDirectory{}
	for _, id := range []string{"a", "b", "c"} {
		directory.Franchises = append(directory.Franchises, NativeDirectorySeat{PeopleSeat: PeopleSeat{Franchise: id, Name: id}})
	}
	weeks := []ScheduleWeek{{Week: 1, Games: []ScheduleGame{{Home: "a", Away: "b"}}, Bye: "c"}}
	got, err := countSeason(t.Context(), "native.test", nil, unevenScores{}, leaguerules.LineupRules{}, directory, weeks, 2026)
	if err != nil || got.points["a"] != 10000 || got.points["c"] != 30000 || got.decided != 0 || got.pending != 1 ||
		len(got.allPlay) != 0 {
		t.Fatalf("complete side/bye lost or result guessed: %+v %v", got, err)
	}
	if _, ok := got.points["b"]; ok {
		t.Fatal("incomplete points invented")
	}
	in := rankFixture()
	in.Points = got.points
	in.Criteria = []leaguerules.Criterion{leaguerules.Points, leaguerules.HeadToHeadRecord}
	in.Teams = in.Teams[:2]
	ranked := competition.RankStandings(in)
	if ranked.Order[0] != "a" || len(ranked.Skipped) != 1 || !strings.Contains(ranked.Skipped[0], "Beta has no complete weekly score") {
		t.Fatal("missing points treated as zero", ranked)
	}
}
