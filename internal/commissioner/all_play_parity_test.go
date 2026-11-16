package commissioner

import (
	"maps"
	"reflect"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/engine/competition"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func TestAllPlayMatchesPairwiseReference(t *testing.T) {
	a, b, c := leaguerules.Decimal(100), leaguerules.Decimal(200), leaguerules.Decimal(200)
	cases := [][]FranchiseWeekScore{
		nil,
		{{Franchise: "a", Points: &a}},
		{{Franchise: "a", Points: &a}, {Franchise: "b", Points: &b}, {Franchise: "c", Points: &c}},
		{{Franchise: "a", Points: &a}, {Franchise: "a", Points: &b}, {Franchise: "b", Points: &c}},
		{{Franchise: "a", Points: &a}, {Franchise: "b"}},
	}
	for i, scores := range cases {
		initial := map[string]competition.Tally{"a": {W: 3, L: 2, T: 1}}
		want := maps.Clone(initial)
		complete := true
		for _, score := range scores {
			if score.Points == nil {
				complete = false
			}
		}
		if complete {
			for _, mine := range scores {
				for _, theirs := range scores {
					if mine.Franchise != theirs.Franchise {
						want[mine.Franchise] = competition.ResultOf(*mine.Points, *theirs.Points).Add(want[mine.Franchise])
					}
				}
			}
		}
		got := seasonGames{allPlay: maps.Clone(initial)}
		got.addAllPlay(scores)
		if !reflect.DeepEqual(got.allPlay, want) {
			t.Errorf("case %d: got %v want %v", i, got.allPlay, want)
		}
	}
}
