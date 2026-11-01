package now

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/marketfeeds"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
)

func TestHQProbabilityUsesCompleteHeldProjections(t *testing.T) {
	t.Parallel()
	in := realNow(t, 5)
	v, err := newNowView(in)
	if err != nil {
		t.Fatal(err)
	}
	me, them, _, ok := v.myGame()
	if !ok {
		t.Fatal("missing matchup")
	}
	players := []marketfeeds.Player{}
	for _, side := range []leaguefeed.Side{me, them} {
		for _, p := range v.starters(side) {
			players = append(players, marketfeeds.Player{ID: p.ID, Value: 10})
		}
	}
	in.Projections.Value = []marketfeeds.Feed{{Week: 5, Players: players}}
	probability := func() string {
		r := nowReport(t, nowBoard(t, in), "M-018")
		for _, row := range r.Summary {
			if row.Cells[0] == "My win probability" {
				return row.Cells[1]
			}
		}
		return ""
	}
	if got := probability(); got != "50.0%" {
		t.Fatal(got)
	}
	for i := range len(v.starters(me)) {
		in.Projections.Value[0].Players[i].Value = 20
	}
	if got := probability(); got == "50.0%" || got == "" {
		t.Fatal(got)
	}
	in.Projections.Value[0].Players = in.Projections.Value[0].Players[1:]
	if got := probability(); got != "" {
		t.Fatalf("partial projection became probability: %s", got)
	}
}
