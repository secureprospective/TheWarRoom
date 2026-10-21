package race

import (
	"reflect"
	"sync"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/app/season"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leaguestandings"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
)

func heldFixture() season.Held {
	return season.Held{
		Weeks:     season.Feed[[]leaguefeed.WeekResult]{Value: []leaguefeed.WeekResult{{Week: 1, Games: []leaguefeed.Game{{Home: leaguefeed.Side{Players: []leaguefeed.WeekPlayer{{Points: 1}}}, Away: leaguefeed.Side{Players: []leaguefeed.WeekPlayer{{Points: 2}}}}}, Others: []leaguefeed.Side{{Players: []leaguefeed.WeekPlayer{{Points: 3}}}}}}},
		Brackets:  season.Feed[[]leaguefeed.Bracket]{Value: []leaguefeed.Bracket{{Games: []leaguefeed.BracketGame{{ID: 1}}}}},
		Standings: season.Feed[[]leaguestandings.RawStanding]{Value: []leaguestandings.RawStanding{{FranchiseID: "0001"}}},
		Power:     map[string]season.PowerRankingsResult{"season": {Rows: []season.PowerRow{{Name: "held"}}}},
	}
}

func publish(h *Held, in season.Held) {
	h.PublishWeeks(in.Weeks)
	h.PublishBrackets(in.Brackets)
	h.PublishStandings(in.Standings, in.Power, true)
}

func mutate(in season.Held) {
	in.Weeks.Value[0].Games[0].Home.Players[0].Points = 100
	in.Weeks.Value[0].Games[0].Away.Players[0].Points = 200
	in.Weeks.Value[0].Others[0].Players[0].Points = 300
	in.Weeks.Value[0].Games[0].Regular = true
	in.Weeks.Value[0].Week = 2
	in.Brackets.Value[0].Games[0].ID = 2
	in.Brackets.Value[0].Name = "changed"
	in.Standings.Value[0].FranchiseID = "0002"
	in.Power["season"].Rows[0].Name = "changed"
	delete(in.Power, "season")
}

func TestHeldDetachesIngressAndEgress(t *testing.T) {
	h := &Held{}
	input, want := heldFixture(), heldFixture()
	publish(h, input)
	mutate(input)
	if got := h.Snapshot(); !reflect.DeepEqual(got, want) {
		t.Fatal("publication retained caller storage", got)
	}
	mutate(h.Snapshot())
	if got := h.Snapshot(); !reflect.DeepEqual(got, want) {
		t.Fatal("snapshot exposed held storage", got)
	}
	h.PublishStandings(season.Feed[[]leaguestandings.RawStanding]{RefreshError: "failed"}, nil, false)
	if got := h.Snapshot(); !reflect.DeepEqual(got.Power, want.Power) {
		t.Fatal("failed refresh cleared last good power")
	}
}

func TestHeldPreservesNilAndEmpty(t *testing.T) {
	for _, input := range []season.Held{
		{},
		{Weeks: season.Feed[[]leaguefeed.WeekResult]{Value: []leaguefeed.WeekResult{}}, Brackets: season.Feed[[]leaguefeed.Bracket]{Value: []leaguefeed.Bracket{}}, Standings: season.Feed[[]leaguestandings.RawStanding]{Value: []leaguestandings.RawStanding{}}, Power: map[string]season.PowerRankingsResult{}},
		{Weeks: season.Feed[[]leaguefeed.WeekResult]{Value: []leaguefeed.WeekResult{{Games: []leaguefeed.Game{}, Others: []leaguefeed.Side{}}, {Games: []leaguefeed.Game{{Home: leaguefeed.Side{Players: []leaguefeed.WeekPlayer{}}}}, Others: []leaguefeed.Side{{Players: []leaguefeed.WeekPlayer{}}}}}}, Brackets: season.Feed[[]leaguefeed.Bracket]{Value: []leaguefeed.Bracket{{Games: []leaguefeed.BracketGame{}}}}, Power: map[string]season.PowerRankingsResult{"nil": {}, "empty": {Rows: []season.PowerRow{}}}},
	} {
		h := &Held{}
		publish(h, input)
		if got := h.Snapshot(); !reflect.DeepEqual(got, input) {
			t.Fatal("nil/empty changed", got, input)
		}
	}
}

func TestHeldPublishReadRace(t *testing.T) {
	h := &Held{}
	input := heldFixture()
	publish(h, input)
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Go(func() {
			for j := 0; j < 300; j++ {
				publish(h, input)
				mutate(h.Snapshot())
			}
		})
	}
	wg.Wait()
	if !reflect.DeepEqual(h.Snapshot(), input) {
		t.Fatal("concurrent reads mutated publication")
	}
}
