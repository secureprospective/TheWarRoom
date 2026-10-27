package market

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/app/season"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/marketfeeds"
)

func TestMarketRefreshSelectionAndCancellation(t *testing.T) {
	t.Parallel()
	at := time.Now()
	held := map[string]HeldFeed{}
	for _, kind := range Kinds() {
		held[kind] = HeldFeed{Feed: report.Feed{FetchedAt: at}, Value: []marketfeeds.Feed{{Week: 2}}}
	}
	var calls []string
	RefreshFeeds(t.Context(), false, 3, held, func(time.Time) time.Duration { return time.Hour }, func(context.Context, time.Duration) error { calls = append(calls, "pause"); return nil }, func(_ context.Context, kind string, week int) {
		if week != 3 {
			t.Fatal(week)
		}
		calls = append(calls, kind)
	})
	if !reflect.DeepEqual(calls, []string{"pause", marketfeeds.Projections}) {
		t.Fatal(calls)
	}
	calls = nil
	RefreshFeeds(t.Context(), true, 3, held, func(time.Time) time.Duration { return 0 }, func(context.Context, time.Duration) error { return context.Canceled }, func(_ context.Context, kind string, _ int) { calls = append(calls, kind) })
	if len(calls) != 0 {
		t.Fatal("canceled pause continued")
	}
}

func TestMarketFailureNeverPublishesWrongWeek(t *testing.T) {
	t.Parallel()
	at := time.Now().Truncate(time.Second)
	held := season.Feed[[]marketfeeds.Feed]{Value: []marketfeeds.Feed{{Week: 2}}, FetchedAt: at}
	failure := errors.New("offline")
	changed := RefreshedFeed(marketfeeds.Projections, 3, held, nil, report.LiveFreshness(at), failure)
	if changed.Value != nil || !changed.FetchedAt.IsZero() || changed.RefreshError == "" {
		t.Fatal("wrong-week projection survived failure")
	}
	unchanged := RefreshedFeed(marketfeeds.Projections, 2, held, nil, report.LiveFreshness(at), failure)
	if len(unchanged.Value) != 1 || unchanged.Value[0].Week != 2 || !unchanged.FetchedAt.Equal(at) {
		t.Fatal("same-week last-good lost")
	}
	if _, err := ParseArchiveFeed(marketfeeds.Projections, 2, []byte(`{`)); err == nil {
		t.Fatal("bad archive accepted")
	}
}

func TestMarketRefreshReportsReadyBeforeContextAndBoundedRun(t *testing.T) {
	err := RefreshReports(true, func() error { return errors.New("failed") }, func() context.Context { t.Fatal("context before ready"); return nil }, func(context.Context, bool) { t.Fatal("run before ready") })
	if err == nil || err.Error() != "market refresh: startup: failed" {
		t.Fatal(err)
	}
	var observed context.Context
	if err := RefreshReports(true, func() error { return nil }, func() context.Context { return t.Context() }, func(ctx context.Context, force bool) {
		observed = ctx
		if !force {
			t.Fatal("lost force")
		}
		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("missing bound")
		}
	}); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(observed.Err(), context.Canceled) {
		t.Fatal("context retained")
	}
}

func TestReadFeedScopesFallbackBeforeParsing(t *testing.T) {
	t.Parallel()
	body, err := os.ReadFile("../../ingestion/marketfeeds/testdata/projectedScores.json")
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	failure := errors.New("no network")
	bodies := func(_ context.Context, part string, accept func(string, []byte, time.Time) bool) (bool, error) {
		if part != "TYPE=projectedScores" {
			t.Fatal(part)
		}
		for _, query := range []string{"L=14432&W=5", "L=other&W=6", "L=14432&W=6"} {
			if accept("https://fixture.invalid/2026/export?TYPE=projectedScores&"+query, body, at) {
				if query != "L=14432&W=6" {
					t.Fatal("wrong archive admitted", query)
				}
				return true, nil
			}
		}
		return false, nil
	}
	rows, fresh, err := ReadFeed(t.Context(), t.Context(), 2026, marketfeeds.Projections, 6,
		func(context.Context) (marketfeeds.Feed, error) { return marketfeeds.Feed{}, failure }, bodies, func(string) {})
	if err != nil || len(rows) != 1 || rows[0].Week != 6 || fresh.FetchedAt != at.Format(time.RFC3339) || !strings.Contains(fresh.Note, "market refresh: no network") {
		t.Fatal(rows, fresh, err)
	}
}

func TestReadFeedLiveSuccessNeverReadsArchive(t *testing.T) {
	t.Parallel()
	rows, _, err := ReadFeed(t.Context(), t.Context(), 2026, marketfeeds.Ranks, 6,
		func(context.Context) (marketfeeds.Feed, error) { return marketfeeds.Feed{Week: 6}, nil },
		func(context.Context, string, func(string, []byte, time.Time) bool) (bool, error) {
			t.Fatal("archive after success")
			return false, nil
		}, func(string) {})
	if err != nil || len(rows) != 1 || rows[0].Week != 6 {
		t.Fatal(rows, err)
	}
}
