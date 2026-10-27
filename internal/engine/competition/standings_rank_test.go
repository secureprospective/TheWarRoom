package competition

import (
	"reflect"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func rankFixture() RankInput {
	return RankInput{Teams: []RankTeam{{ID: "a", Name: "Alpha", Division: "d", Conference: "c"},
		{ID: "b", Name: "Beta", Division: "d", Conference: "c"}, {ID: "x", Name: "Other", Division: "e", Conference: "e"}},
		Games: []RankGame{{Home: "a", Away: "b", HomePoints: 1, AwayPoints: 0},
			{Home: "a", Away: "x", HomePoints: 0, AwayPoints: 100}, {Home: "b", Away: "x", HomePoints: 10, AwayPoints: 0}},
		Points:  map[string]leaguerules.Decimal{"a": 1, "b": 10, "x": 100},
		AllPlay: map[string]Tally{"a": {W: 1, L: 3}, "b": {W: 3, L: 1}, "x": {W: 2, L: 2}}}
}

func TestNativeRankingHonorsCriterionOrderAndExplains(t *testing.T) {
	in := rankFixture()
	in.Criteria = []leaguerules.Criterion{leaguerules.WinPercentage, leaguerules.HeadToHeadRecord, leaguerules.Points}
	// Restrict the tied group to the two teams whose head-to-head conflicts with their points.
	in.Teams = in.Teams[:2]
	got := RankStandings(in)
	if !reflect.DeepEqual(got.Order, []string{"a", "b"}) || !strings.Contains(strings.Join(got.Why["a"], " "), "Head-to-head record") {
		t.Fatalf("head-to-head skipped: %+v", got)
	}
	in.Criteria = []leaguerules.Criterion{leaguerules.WinPercentage, leaguerules.Points, leaguerules.HeadToHeadRecord}
	got = RankStandings(in)
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
		got := RankStandings(in)
		want := "a"
		if c == leaguerules.AllPlayPercentage || c == leaguerules.Points {
			want = "b"
		}
		if got.Order[0] != want || !strings.Contains(strings.ToLower(strings.Join(got.Why[want], " ")), CriterionLabel(c)) {
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
	got := RankStandings(in)
	if len(got.Skipped) != 7 || !reflect.DeepEqual(got.Order, []string{"a", "b", "x"}) {
		t.Fatalf("missing data invented a competitive value: %+v", got)
	}
	in = rankFixture()
	in.Teams = in.Teams[:2]
	in.AllPlay = map[string]Tally{"b": {W: 3, L: 1}}
	in.Criteria = []leaguerules.Criterion{leaguerules.AllPlayPercentage, leaguerules.HeadToHeadRecord}
	got = RankStandings(in)
	if got.Order[0] != "a" || !strings.Contains(strings.Join(got.Skipped, " "), "Alpha has no games yet") {
		t.Fatalf("missing data treated as zero: %+v", got)
	}
}
