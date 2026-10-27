package core_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/core"
	"github.com/secureprospective/TheWarRoom/internal/app/core/coretest"
	"github.com/secureprospective/TheWarRoom/internal/app/season"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

func TestRaceHostReadOrderAndDetachment(t *testing.T) {
	s := coretest.New(t).Build()
	live := leaguefeed.Lineups{Week: 5, Matchups: []leaguefeed.Matchup{{Home: "0001"}}, Franchises: []leaguefeed.Lineup{{Players: []leaguefeed.PlayerScore{{Score: 1}}, Starters: []playerid.PlayerID{racePlayer(t, "0001")}, NonStarters: []playerid.PlayerID{racePlayer(t, "0002")}}}}
	s.SeasonLineups.Value = live
	s.CaptureRace = func(snapshot func() season.Held) season.Snapshot {
		s.SeasonMu.Lock()
		defer s.SeasonMu.Unlock()
		h := snapshot()
		detached := core.CloneLineups(s.SeasonLineups.Value)
		return season.Snapshot{Held: h, Live: season.Feed[leaguefeed.Lineups]{Value: detached}}
	}
	in := s.RaceInput(func() season.Held {
		if s.SeasonMu.TryLock() {
			s.SeasonMu.Unlock()
			t.Fatal("capture lost outer shared lock")
		}
		return season.Held{}
	})
	in.Live.Value.Matchups[0].Home = "changed"
	in.Live.Value.Franchises[0].Players[0].Score = 2
	in.Live.Value.Franchises[0].Starters[0] = racePlayer(t, "0003")
	in.Live.Value.Franchises[0].NonStarters[0] = racePlayer(t, "0004")
	if !reflect.DeepEqual(s.SeasonLineups.Value, live) || live.Matchups[0].Home != "0001" || live.Franchises[0].Players[0].Score != 1 || live.Franchises[0].Starters[0].String() != "0001" || live.Franchises[0].NonStarters[0].String() != "0002" {
		t.Fatal("live capture retained mutable storage")
	}
	if in.Zone != time.UTC {
		t.Fatal("host did not pin zone")
	}
}

func racePlayer(t *testing.T, raw string) playerid.PlayerID {
	t.Helper()
	id, err := playerid.New(raw)
	if err != nil {
		t.Fatal(err)
	}
	return id
}
