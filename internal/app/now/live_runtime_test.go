package now

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/season"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
)

func TestWeekFeedRefreshFreshnessAndPacing(t *testing.T) {
	t.Parallel()
	at := time.Now()
	for _, tc := range []struct {
		name                string
		force               bool
		scheduleAge, byeAge time.Duration
		failure             string
		cancel              bool
		want                []string
	}{
		{name: "fresh", scheduleAge: time.Minute, byeAge: time.Hour},
		{name: "schedule cutoff", scheduleAge: 5 * time.Minute, byeAge: time.Hour, want: []string{"schedule", "pause"}},
		{name: "bye cutoff", scheduleAge: time.Minute, byeAge: 24 * time.Hour, want: []string{"byes"}},
		{name: "failed", scheduleAge: time.Minute, byeAge: time.Hour, failure: "failed", want: []string{"schedule", "pause", "byes"}},
		{name: "forced", force: true, want: []string{"schedule", "pause", "byes"}},
		{name: "canceled pause", force: true, cancel: true, want: []string{"schedule", "pause"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []string
			RefreshWeekFeeds(t.Context(), tc.force, at.Add(-tc.scheduleAge), tc.failure, at.Add(-tc.byeAge), tc.failure, WeekFeedSource{
				Since:    at.Sub,
				Schedule: func(context.Context) { calls = append(calls, "schedule") },
				Pause: func(context.Context, time.Duration) error {
					calls = append(calls, "pause")
					if tc.cancel {
						return context.Canceled
					}
					return nil
				},
				Byes: func(context.Context) { calls = append(calls, "byes") },
			})
			if !reflect.DeepEqual(calls, tc.want) {
				t.Fatalf("calls %v want %v", calls, tc.want)
			}
		})
	}
}

func TestPollLiveMFLBackoffAndReset(t *testing.T) {
	t.Parallel()
	at := time.Now()
	live := LiveGame{NextMFL: at}
	calls := 0
	refresh := func(context.Context) bool { calls++; return true }
	PollLiveMFL(t.Context(), &live, at, false, refresh)
	PollLiveMFL(t.Context(), &live, at.Add(-time.Nanosecond), true, refresh)
	if calls != 0 {
		t.Fatal("inactive or early poll ran")
	}
	for _, delay := range []time.Duration{60, 120, 240, 480, 480} {
		now := live.NextMFL
		PollLiveMFL(t.Context(), &live, now, true, refresh)
		if live.NextMFL.Sub(now) != delay*time.Second {
			t.Fatalf("backoff: %+v", live)
		}
	}
	now := live.NextMFL
	PollLiveMFL(t.Context(), &live, now, true, func(context.Context) bool { return false })
	if live.MFLFailures != 0 || live.NextMFL.Sub(now) != 30*time.Second {
		t.Fatalf("success did not reset: %+v", live)
	}
}

func TestLiveInitializationAndWeekReset(t *testing.T) {
	t.Parallel()
	failure := errors.New("unavailable")
	adapter := livegame.Adapter{}
	if err := StartLiveGame(&adapter, func() (livegame.Fetch, livegame.Finals, error) { return nil, nil, failure }); err != failure || adapter.Fetch != nil {
		t.Fatal("failed initialization published")
	}
	fetch := func(context.Context, string, string) ([]byte, error) { t.Fatal("unexpected network"); return nil, nil }
	adapter.Fetch = fetch
	if err := StartLiveGame(&adapter, func() (livegame.Fetch, livegame.Finals, error) { t.Fatal("reinitialized"); return nil, nil, nil }); err != nil {
		t.Fatal(err)
	}
	live := LiveGame{Week: 1, Adapter: adapter, MFLFailures: 3, NextMFL: time.Now()}
	source := LiveSource{Start: func(*livegame.Adapter) error { return nil }, Now: time.Now, Windows: func() []livegame.Window { return nil }, Week: func() int { return 2 }, Seed: func(time.Time) livegame.Held { return livegame.Held{Games: []livegame.Game{{ID: "held"}}} }}
	if _, err := RefreshLiveGame(t.Context(), &live, false, "missing", "", source); err == nil {
		t.Fatal("unknown follow accepted")
	}
	if live.Week != 2 || live.MFLFailures != 0 || !live.NextMFL.IsZero() || live.Adapter.Fetch == nil || len(live.Adapter.Board.Games) != 1 {
		t.Fatalf("week reset lost transport or retained backoff: %+v", live)
	}
	source.Poll = func(context.Context, *LiveGame, time.Time, bool) {}
	source.Spines = func(context.Context, *LiveGameView, livegame.Adapter, string) error { return failure }
	view, err := RefreshLiveGame(t.Context(), &live, false, "", "", source)
	if err != nil || len(view.Games) != 1 || view.DetailFailure == "" {
		t.Fatalf("detail failure lost held board: %+v %v", view, err)
	}
}

func TestSpinesSkipUnneededReadsAndStopOnFailure(t *testing.T) {
	t.Parallel()
	view := LiveGameView{}
	if err := FillLiveSpines(t.Context(), &view, livegame.Adapter{}, "", SpineSource{}); err != nil {
		t.Fatal(err)
	}
	view.Followed = &livegame.Game{ID: "follow"}
	if err := FillLiveSpines(t.Context(), &view, livegame.Adapter{}, "", SpineSource{}); err == nil {
		t.Fatal("missing directory accepted")
	}
	failure := errors.New("crosswalk failed")
	if err := FillLiveSpines(t.Context(), &view, livegame.Adapter{}, "", SpineSource{Available: true, Known: func(context.Context) (map[string]bool, error) { return nil, failure }}); !errors.Is(err, failure) {
		t.Fatal(err)
	}
}

func TestSelectedWeekAndRaceOverlay(t *testing.T) {
	t.Parallel()
	calls := 0
	if SelectedRefreshWeek(3, func() int { t.Fatal("explicit week read current"); return 0 }) != 3 {
		t.Fatal("explicit week changed")
	}
	if SelectedRefreshWeek(0, func() int { return 4 }) != 4 {
		t.Fatal("default week lost")
	}
	refresh := func(context.Context, int, bool) { calls++ }
	weeks := []leaguefeed.WeekResult{{Week: 2}}
	for _, week := range []int{2, 3} {
		if err := RefreshSelectedWeek(t.Context(), week, weeks, 3, true, refresh); err != nil {
			t.Fatal(err)
		}
	}
	if err := RefreshSelectedWeek(t.Context(), 1, weeks, 3, true, refresh); err == nil || calls != 2 {
		t.Fatal("unlisted noncurrent week refreshed")
	}
	if err := RefreshSelectedWeek(t.Context(), 0, nil, 0, true, refresh); err != nil || calls != 2 {
		t.Fatal("zero week refreshed")
	}
	failure := errors.New("race failed")
	if err := RefreshNow(1, false, func(bool) error { return failure }, func(int, bool) error { t.Fatal("continued after race failure"); return nil }); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	if _, err := CheckScheduleWeek(nflschedule.RawWeek{Week: "2"}, 1); err == nil {
		t.Fatal("wrong week accepted")
	}
	held := season.Held{Weeks: season.Feed[[]leaguefeed.WeekResult]{Value: []leaguefeed.WeekResult{{Week: 1}, {Week: 2}}}}
	snap := RaceSnapshot(held, Feed[leaguefeed.Lineups]{Value: leaguefeed.Lineups{Week: 2}}, true)
	if len(snap.LiveWeeks) != 1 || snap.LiveWeeks[2].Week != 2 || !snap.LiveUnplayed {
		t.Fatalf("overlay scope: %+v", snap)
	}
}

func TestRefreshFirstsKeepsHeldOnArchiveFailure(t *testing.T) {
	t.Parallel()
	logged := ""
	RefreshFirsts(t.Context(), FirstScores{}, 2026, func(context.Context, string, func(string, []byte, time.Time) bool) (bool, error) {
		return false, errors.New("read failed")
	}, func(FirstScores) { t.Fatal("failed archive published") }, func(s string) { logged = s })
	if logged == "" {
		t.Fatal("archive failure lost")
	}
}
