package race

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/season"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leagueschedule"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leaguestandings"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
)

func TestRefreshKeepsPublicationOrderAndOneSchedule(t *testing.T) {
	t.Parallel()
	for _, failed := range []bool{false, true} {
		var calls []string
		var network context.Context
		schedules := 0
		source := RefreshSource{Pause: func(context.Context, time.Duration) error { calls = append(calls, "pause"); return nil },
			Weeks: func(context.Context) ([]leaguefeed.WeekResult, domain.Freshness, error) {
				calls = append(calls, "weeks")
				return nil, domain.Freshness{}, nil
			}, PublishWeeks: func([]leaguefeed.WeekResult, domain.Freshness, error) { calls = append(calls, "publish weeks") },
			Brackets: func(context.Context) ([]leaguefeed.Bracket, domain.Freshness, error) {
				calls = append(calls, "brackets")
				return nil, domain.Freshness{}, nil
			}, PublishBrackets: func([]leaguefeed.Bracket, domain.Freshness, error) { calls = append(calls, "publish brackets") },
			Standings: func(ctx context.Context) ([]leaguestandings.RawStanding, domain.Freshness, error) {
				network = ctx
				calls = append(calls, "standings")
				if failed {
					return nil, domain.Freshness{}, errors.New("offline")
				}
				return nil, domain.Freshness{}, nil
			},
			Schedule: func(context.Context) []leagueschedule.RawScheduleWeek { schedules++; return nil }, Power: func(ctx context.Context, _ []leaguestandings.RawStanding, _ domain.Freshness, schedule func(context.Context) []leagueschedule.RawScheduleWeek, mode, view string) season.PowerRankingsResult {
				if network.Err() == nil || ctx.Err() != nil || network == ctx {
					t.Fatal("history shares the spent network budget")
				}
				calls = append(calls, view+":"+mode)
				schedule(ctx)
				schedule(ctx)
				return season.PowerRankingsResult{View: view, AggMode: mode}
			},
			PublishStandings: func(_ []leaguestandings.RawStanding, _ domain.Freshness, _ error, power map[string]season.PowerRankingsResult, replace bool) {
				calls = append(calls, "publish standings")
				if replace == failed || (failed && len(power) != 0) || (!failed && len(power) != 6) {
					t.Fatal("power replacement policy changed")
				}
			},
		}
		Refresh(t.Context(), source)
		want := []string{"weeks", "publish weeks", "pause", "brackets", "publish brackets", "pause", "standings"}
		if !failed {
			for _, view := range []string{season.ViewSeason, season.ViewFranchise} {
				for _, mode := range []string{"lineup", "sum", "topn"} {
					want = append(want, view+":"+mode)
				}
			}
		}
		want = append(want, "publish standings")
		if !reflect.DeepEqual(calls, want) || (!failed && schedules != 1) || (failed && schedules != 0) {
			t.Fatalf("calls %v schedules %d", calls, schedules)
		}
	}
}

func TestBracketsAggregateFreshnessAndRefusePartialSet(t *testing.T) {
	t.Parallel()
	pause := func(context.Context, time.Duration) error { return nil }
	list := func(context.Context) ([]leaguefeed.Bracket, domain.Freshness, error) {
		return []leaguefeed.Bracket{{ID: 1, Name: "One"}, {ID: 2, Name: "Two"}}, domain.Freshness{State: domain.FreshLive}, nil
	}
	_, fresh, err := ReadBrackets(t.Context(), pause, list, func(_ context.Context, b leaguefeed.Bracket) ([]leaguefeed.BracketGame, domain.Freshness, error) {
		return nil, domain.Freshness{State: domain.FreshStale, Note: b.Name}, nil
	})
	if err != nil || fresh.Note != "Two" {
		t.Fatalf("last nonlive part precedence changed: %+v %v", fresh, err)
	}
	failure := errors.New("missing")
	result, _, err := ReadBrackets(t.Context(), pause, list, func(context.Context, leaguefeed.Bracket) ([]leaguefeed.BracketGame, domain.Freshness, error) {
		return nil, domain.Freshness{}, failure
	})
	if result != nil || !errors.Is(err, failure) {
		t.Fatal("partial bracket set escaped")
	}
	result, _, err = ReadBrackets(t.Context(), func(context.Context, time.Duration) error { return context.Canceled }, list, nil)
	if result != nil || !errors.Is(err, context.Canceled) {
		t.Fatal("canceled bracket pause continued")
	}
}

func TestRaceRefreshReadingAdmissionAndDeadline(t *testing.T) {
	at := time.Now()
	held := season.Held{}
	held.Weeks.FetchedAt, held.Brackets.FetchedAt, held.Standings.FetchedAt = at, at, at
	fresh := func() season.Held { return held }
	if err := RefreshReading(false, func() error { return nil }, fresh, func() context.Context { t.Fatal("fresh context capture"); return nil }, nil); err != nil {
		t.Fatal(err)
	}
	var observed context.Context
	if err := RefreshReading(true, func() error { return nil }, fresh, func() context.Context { return t.Context() }, func(ctx context.Context) {
		observed = ctx
		if ctx != t.Context() {
			t.Fatal("read budget started before the refresh lock")
		}
	}); err != nil {
		t.Fatal(err)
	}
	if observed.Err() != nil {
		t.Fatal("refresh canceled its caller's lifetime")
	}
	if err := RefreshReading(true, func() error { return errors.New("failed") }, func() season.Held { t.Fatal("capture before ready"); return held }, nil, nil); err == nil || err.Error() != "race refresh: startup: failed" {
		t.Fatal(err)
	}
}
