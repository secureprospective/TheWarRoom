package race

import (
	"reflect"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/app/season"
)

func TestFailedPowerReadKeepsLastGoodValues(t *testing.T) {
	held := &Held{}
	in := heldFixture()
	good := in.Power["season"]
	good.OK = true
	in.Power["season"] = good
	publish(held, in)
	held.PublishStandings(in.Standings, map[string]season.PowerRankingsResult{"season": {Error: "history: scan run: context deadline exceeded"}}, true)
	board := held.Snapshot().Power["season"]
	if !board.OK || !reflect.DeepEqual(board.Rows, good.Rows) || board.Error != "Power rankings could not be updated. Try refreshing again." {
		t.Fatalf("lost last good board: %+v", board)
	}
	held.PublishStandings(in.Standings, in.Power, true)
	if !reflect.DeepEqual(held.Snapshot().Power["season"], good) {
		t.Fatal("successful retry changed values or kept failure")
	}
}
