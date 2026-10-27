package draft

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
)

func TestRefreshFeedsOrderLazyPauseAndCancellation(t *testing.T) {
	var order []string
	RefreshFeeds(t.Context(), func() func(context.Context, time.Duration) error {
		order = append(order, "select pause")
		return func(_ context.Context, d time.Duration) error {
			if d != time.Second {
				t.Fatal(d)
			}
			order = append(order, "pause")
			return nil
		}
	}, func(context.Context) { order = append(order, "results") }, func(context.Context) { order = append(order, "future") })
	if !reflect.DeepEqual(order, []string{"results", "select pause", "pause", "future"}) {
		t.Fatal(order)
	}
	RefreshFeeds(t.Context(), func() func(context.Context, time.Duration) error {
		return func(context.Context, time.Duration) error { return errors.New("canceled") }
	}, func(context.Context) {}, func(context.Context) { t.Fatal("future after canceled pause") })
}

func TestDraftRefreshAdmissionAndDeadline(t *testing.T) {
	at := time.Now()
	fresh := func() (report.Feed, report.Feed) { return report.Feed{FetchedAt: at}, report.Feed{FetchedAt: at} }
	parent := func() context.Context { t.Fatal("parent captured for fresh or refused request"); return nil }
	if err := RefreshReports(false, func() error { return nil }, fresh, parent, func(context.Context) { t.Fatal("fresh run") }); err != nil {
		t.Fatal(err)
	}
	if err := RefreshReports(true, func() error { return errors.New("failed") }, func() (report.Feed, report.Feed) {
		t.Fatal("capture before ready")
		return report.Feed{}, report.Feed{}
	}, parent, nil); err == nil || err.Error() != "draft refresh: startup: failed" {
		t.Fatal(err)
	}
	var observed context.Context
	if err := RefreshReports(true, func() error { return nil }, fresh, func() context.Context { return t.Context() }, func(ctx context.Context) {
		observed = ctx
		if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > 2*time.Minute {
			t.Fatal("missing bounded context")
		}
	}); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(observed.Err(), context.Canceled) {
		t.Fatal("context not released")
	}
}

func TestDraftOpenAndDetailedPlayerAdmission(t *testing.T) {
	opened := false
	if err := OpenRoom(func() error { return errors.New("startup") }, func() string { t.Fatal("destination before ready"); return "" }, func(string) { opened = true }); err == nil {
		t.Fatal("startup admitted")
	}
	if err := OpenRoom(func() error { return nil }, func() string { return "" }, func(string) { opened = true }); err == nil || opened {
		t.Fatal(err, opened)
	}
	target := Destination("discovered.example", 2026)
	if target != "https://discovered.example/2026/options?L=14432&O=17" || Destination("", 2026) != "" {
		t.Fatal(target)
	}
	if err := OpenRoom(func() error { return nil }, func() string { return target }, func(got string) {
		if got != target {
			t.Fatal(got)
		}
		opened = true
	}); err != nil || !opened {
		t.Fatal(err)
	}
	if _, err := ParsePlayers(t.Context(), []byte(`{"players":{"player":[{"id":"1","name":"Held","position":"QB"}]}}`)); err == nil || !strings.Contains(err.Error(), "detailed selection fields not held") {
		t.Fatal(err)
	}
	if _, err := ParsePlayers(t.Context(), []byte(`broken`)); err == nil || !strings.Contains(err.Error(), "NFL draft: players:") {
		t.Fatal(err)
	}
	if rows, err := ParsePlayers(t.Context(), []byte(`{"players":{"player":[{"id":"1","name":"Held","position":"QB","draft_round":"1","draft_pick":"2"}]}}`)); err != nil || len(rows) != 1 {
		t.Fatal(rows, err)
	}
}
