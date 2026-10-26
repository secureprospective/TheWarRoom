package research

import (
	"reflect"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/researchfeeds"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

func TestFinishedScoresScopeAndCounts(t *testing.T) {
	t.Parallel()
	id, err := playerid.New("0042")
	if err != nil {
		t.Fatal(err)
	}
	lookup, err := normalize.NewLookup([]players.RawPlayer{{ID: "0042", Position: "QB", Name: "Quarterback"}})
	if err != nil {
		t.Fatal(err)
	}
	early := time.Unix(1800000000, 0)
	late := early.Add(time.Hour)
	feed := func(points float64, at time.Time) Feed[[]researchfeeds.Feed] {
		return Feed[[]researchfeeds.Feed]{Value: []researchfeeds.Feed{{Players: []researchfeeds.Entry{{ID: id, Value: points}}}}, FetchedAt: at}
	}
	in := Snapshot{Week: 5, Lookup: lookup, Held: Values{Feeds: map[string]Feed[[]researchfeeds.Feed]{
		Key("playerScores", "1"): feed(12, early), Key("playerScores", "2"): feed(-2, late),
		Key("playerScores", "3"): {Value: []researchfeeds.Feed{{}}, FetchedAt: early},
		Key("playerScores", "4"): {FetchedAt: late.Add(time.Hour)},
		Key("playerScores", "5"): feed(100, late.Add(2*time.Hour)), Key("playerScores", "YTD"): feed(200, late.Add(3*time.Hour)),
	}}}
	before := in.Held.Feeds[Key("playerScores", "1")]
	got := FinishedScores(in)
	if got.Held != 3 || got.Weeks != 4 || got.Latest != late || !reflect.DeepEqual(got.Rows, map[string]*Score{"0042": {Position: "QB", Sum: 10, Games: 2}}) {
		t.Fatalf("finished: %+v rows %+v", got, got.Rows)
	}
	got.Rows["0042"].Sum = 999
	if !reflect.DeepEqual(in.Held.Feeds[Key("playerScores", "1")], before) || FinishedScores(in).Rows["0042"].Sum != 10 {
		t.Fatal("output aliases published inputs")
	}
	for _, week := range []int{-1, 0, 1} {
		in.Week = week
		empty := FinishedScores(in)
		if empty.Weeks != 0 || empty.Held != 0 || len(empty.Rows) != 0 || !empty.Latest.IsZero() || empty.Rows == nil {
			t.Fatalf("week %d: %+v", week, empty)
		}
	}
}

func TestFinishedScoresUsesOnlyFirstFeed(t *testing.T) {
	t.Parallel()
	id, err := playerid.New("0042")
	if err != nil {
		t.Fatal(err)
	}
	in := Snapshot{Week: 2, Held: Values{Feeds: map[string]Feed[[]researchfeeds.Feed]{Key("playerScores", "1"): {Value: []researchfeeds.Feed{{Players: []researchfeeds.Entry{{ID: id, Value: 0}, {ID: id, Value: 3}}}, {Players: []researchfeeds.Entry{{ID: id, Value: 99}}}}}}}}
	got := FinishedScores(in)
	if got.Rows["0042"].Sum != 3 || got.Rows["0042"].Games != 2 || got.Rows["0042"].Position != "Unknown" {
		t.Fatalf("first feed: %+v", got.Rows["0042"])
	}
}
