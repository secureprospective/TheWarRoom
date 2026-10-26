package research

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/researchfeeds"
)

func TestFinishedScoreRefreshPinsAndCancels(t *testing.T) {
	at := time.Now()
	values := Values{}
	first := PinFinished(&values, 2, at)
	if got := PinFinished(&values, 2, at.Add(time.Hour)); !got.Equal(first) {
		t.Fatal("finish moved", got)
	}
	for _, canceled := range []bool{false, true} {
		calls := []string{}
		s := FinishedScoreRefresh{
			Week:     func() int { return 3 },
			Schedule: func(context.Context, int) { calls = append(calls, "schedule") },
			Held:     func(int) time.Time { return time.Time{} },
			Archive:  func(context.Context, int) time.Time { calls = append(calls, "archive"); return at.Add(-time.Hour) },
			Cutoff:   func(int) (time.Time, bool) { return at, true },
			Pin:      func(w int, end time.Time) time.Time { return PinFinished(&values, w, end) },
			Pause: func() func(context.Context, time.Duration) error {
				return func(context.Context, time.Duration) error {
					if canceled {
						return context.Canceled
					}
					return nil
				}
			},
			Refresh: func(_ context.Context, period string) { calls = append(calls, period) },
		}
		s.RefreshDue(context.Background())
		want := []string{"schedule", "archive", "2", "YTD"}
		if canceled {
			want = want[:2]
		}
		if !reflect.DeepEqual(calls, want) {
			t.Fatal(calls, want)
		}
		calls = nil
		s.Week = func() int { return 1 }
		s.RefreshDue(context.Background())
		if len(calls) != 0 {
			t.Fatal("read before first finished week", calls)
		}
	}
}

func TestResearchFeedAndDepthRefreshRetention(t *testing.T) {
	ctx := context.Background()
	at := time.Now()
	failure := errors.New("offline")
	old := Feed[[]researchfeeds.Feed]{FetchedAt: at, Value: []researchfeeds.Feed{{}}}
	reads, publishes := 0, 0
	read := func(context.Context) ([]researchfeeds.Feed, domain.Freshness, error) {
		reads++
		return nil, domain.Freshness{}, failure
	}
	publish := func(next Feed[[]researchfeeds.Feed]) {
		publishes++
		if !next.FetchedAt.Equal(at) || len(next.Value) != 1 || next.RefreshError == "" {
			t.Fatal(next)
		}
	}
	RefreshFeed(ctx, old, false, read, publish)
	if reads != 0 || publishes != 0 {
		t.Fatal("fresh read performed IO")
	}
	RefreshFeed(ctx, old, true, read, publish)
	if reads != 1 || publishes != 1 {
		t.Fatal(reads, publishes)
	}
	values := Values{Status: "latest", Links: map[string]string{"old": "id"}}
	PublishFeed(&values, "playerScores", "2", old)
	if values.Status != "latest" || len(values.Feeds) != 1 {
		t.Fatal(values)
	}
	oldDepth := Feed[[]researchfeeds.RawDepth]{FetchedAt: at, Value: []researchfeeds.RawDepth{{}}}
	RefreshDepth(ctx, oldDepth, true,
		func(context.Context) ([]researchfeeds.RawDepth, domain.Freshness, error) {
			return nil, domain.Freshness{}, failure
		},
		func(context.Context) (map[string]string, error) { return nil, failure },
		func(held Feed[[]researchfeeds.RawDepth], links map[string]string, err error) {
			PublishDepth(&values, held, links, err)
		})
	if values.Links["old"] != "id" || values.LinkError != "offline" || len(values.Depth.Value) != 1 || values.Status != "latest" {
		t.Fatal(values)
	}
	PublishDepth(&values, oldDepth, map[string]string{"new": "id"}, nil)
	if values.LinkError != "" || values.Links["new"] != "id" {
		t.Fatal(values)
	}
	PrepareDepth(false, func() context.Context { t.Fatal("parent captured without transport"); return ctx }, time.Second, nil)
	PrepareDepth(true, func() context.Context { return ctx }, time.Second, func(ctx context.Context, force bool) {
		if _, ok := ctx.Deadline(); !ok || force {
			t.Fatal("unbounded or forced preparation")
		}
	})
}

func TestResearchRefreshStepsOrderAndCancellation(t *testing.T) {
	calls := []string{}
	s := RefreshSteps{
		Week: func() int { return 2 }, Status: func(string) {}, Pause: func(context.Context, time.Duration) error { return nil },
		Feed: func(_ context.Context, kind, week string, force bool) {
			if !force {
				t.Fatal("lost force")
			}
			calls = append(calls, Key(kind, week))
		},
		Injuries: func(context.Context) { calls = append(calls, "injuries") },
		Schedule: func(_ context.Context, week int, force bool) {
			if week != 2 || !force {
				t.Fatal(week, force)
			}
			calls = append(calls, "schedule")
		},
		Depth: func(context.Context, bool) { calls = append(calls, "depth") },
	}
	if err := s.Run(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	want := []string{}
	for _, kind := range Kinds() {
		week := ""
		if kind == "playerScores" {
			week = "YTD"
		}
		want = append(want, Key(kind, week))
	}
	want = append(want, "playerScores:1", "playerScores:2", "injuries", "schedule", "depth")
	if !reflect.DeepEqual(calls, want) {
		t.Fatal(calls, want)
	}
	calls = nil
	s.Pause = func(context.Context, time.Duration) error { return context.Canceled }
	if err := s.Run(context.Background(), true); !errors.Is(err, context.Canceled) || len(calls) != 0 {
		t.Fatal(err, calls)
	}
}

func TestNewsRefreshAdmissionAndFailurePublication(t *testing.T) {
	fail := errors.New("read lost")
	published, unlocked := false, false
	s := NewsRefresh{Ready: func() error { return nil }, Names: func() map[string]string { return map[string]string{"0001": "One"} },
		Parent: context.Background, TryLock: func() bool { return true }, Unlock: func() { unlocked = true },
		Players: func(string) []string { return []string{"0010"} }, Held: func() map[string]Feed[[]researchfeeds.News] { return nil },
		Read: func(ctx context.Context, held map[string]Feed[[]researchfeeds.News], ids []string) error {
			if _, ok := ctx.Deadline(); !ok || len(ids) != 1 {
				t.Fatal("unbounded or lost scope")
			}
			held[ids[0]] = Feed[[]researchfeeds.News]{RefreshError: "read lost"}
			return fail
		},
		Publish: func(held map[string]Feed[[]researchfeeds.News]) { published = held["0010"].RefreshError == "read lost" },
	}
	if err := s.Run("0001"); !errors.Is(err, fail) || !published || !unlocked {
		t.Fatal(err, published, unlocked)
	}
	published, unlocked = false, false
	if err := s.Run("0002"); err == nil || published || unlocked {
		t.Fatal("unknown franchise admitted")
	}
	s.TryLock = func() bool { return false }
	if err := s.Run(""); err == nil || published || unlocked {
		t.Fatal("busy refresh admitted")
	}
	s.Ready = func() error { return fail }
	s.Names = func() map[string]string { t.Fatal("read names before ready"); return nil }
	if err := s.Run("0001"); !errors.Is(err, fail) {
		t.Fatal(err)
	}
}

func TestReadDepthLiveAvoidsArchive(t *testing.T) {
	rows, fresh, err := ReadDepth(context.Background(), context.Background(), 2026, nil,
		func(context.Context) ([]researchfeeds.RawDepth, error) { return []researchfeeds.RawDepth{{}}, nil },
		func(string) { t.Fatal("logged live success") })
	if err != nil || len(rows) != 1 || fresh.State != domain.FreshLive {
		t.Fatal(rows, fresh, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := report.RefreshPause(nil)(ctx, time.Hour); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
