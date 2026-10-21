package core_test

import (
	"context"
	"errors"
	"reflect"
	"sync/atomic"
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

func TestRaceRefreshSerializationAndCaptureOrder(t *testing.T) {
	s := coretest.New(t).Build()
	var calls atomic.Int32
	entered, release := make(chan struct{}, 2), make(chan struct{})
	s.RefreshRaceFeeds = func(ctx context.Context) {
		if _, ok := ctx.Deadline(); !ok {
			t.Error("refresh has no deadline")
		}
		if s.RefreshMu.TryLock() {
			s.RefreshMu.Unlock()
			t.Error("refresh not serialized")
		}
		calls.Add(1)
		entered <- struct{}{}
		<-release
	}
	captured := make(chan struct{}, 2)
	snapshot := func() season.Held {
		if s.SeasonMu.TryLock() {
			s.SeasonMu.Unlock()
			t.Error("freshness not under SeasonMu")
		}
		captured <- struct{}{}
		return season.Held{}
	}
	s.RefreshMu.Lock()
	done := make(chan error, 2)
	go func() { done <- s.RefreshRace(false, snapshot) }()
	waitRace(t, captured)
	if calls.Load() != 0 {
		t.Fatal("refresh ran before serializer released")
	}
	s.RefreshMu.Unlock()
	waitRace(t, entered)
	go func() { done <- s.RefreshRace(true, snapshot) }()
	waitRace(t, captured)
	if calls.Load() != 1 {
		t.Fatal("two refreshes entered together")
	}
	release <- struct{}{}
	waitRace(t, entered)
	release <- struct{}{}
	for range 2 {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
	if !s.RefreshMu.TryLock() {
		t.Fatal("refresh retained serializer")
	}
	s.RefreshMu.Unlock()
}

func TestRaceRefreshFreshAndStartupDenial(t *testing.T) {
	s := coretest.New(t).Build()
	s.RefreshRaceFeeds = func(context.Context) { t.Fatal("fresh feed refreshed") }
	fresh := func() season.Held {
		at := time.Now()
		return season.Held{Weeks: season.Feed[[]leaguefeed.WeekResult]{FetchedAt: at}, Brackets: season.Feed[[]leaguefeed.Bracket]{FetchedAt: at}, Standings: season.Held{}.Standings}
	}
	snapshot := func() season.Held { h := fresh(); h.Standings.FetchedAt = h.Weeks.FetchedAt; return h }
	if err := s.RefreshRace(false, snapshot); err != nil {
		t.Fatal(err)
	}
	s.StartupErr = errors.New("fixture startup failed")
	err := s.RefreshRace(true, func() season.Held { t.Fatal("captured before ready"); return season.Held{} })
	if err == nil || err.Error() != "race refresh: startup: fixture startup failed" {
		t.Fatal(err)
	}
}

func waitRace(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatal("race host operation blocked")
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
