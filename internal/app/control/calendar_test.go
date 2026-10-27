package control

import (
	"context"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
	"github.com/secureprospective/TheWarRoom/internal/transactions"
)

func TestMergeMFLCalendarHeldOnlyOrderingAndLabels(t *testing.T) {
	t.Parallel()
	now := time.Unix(1800000000, 0)
	later := now.Add(time.Hour)
	held := report.HeldFeed[[]ingestion.RawCalendarEvent]{FetchedAt: now, Value: []ingestion.RawCalendarEvent{
		{ID: "past", Title: "Past", Start: strconv.FormatInt(now.Add(-time.Second).Unix(), 10)},
		{ID: "next", Title: "Dated title", Start: strconv.FormatInt(later.Unix(), 10)},
		{ID: "undated", Type: "ROOKIE_DRAFT"},
		{ID: "now", Type: "LINEUP_LOCK", Start: strconv.FormatInt(now.Unix(), 10)},
	}}
	before := append([]ingestion.RawCalendarEvent(nil), held.Value...)
	clock := snapshot.Sourced[leagueclock.Reading]{Value: leagueclock.Reading{Season: 2026, Deadlines: []leagueclock.Deadline{{ID: "app", At: &later, Urgency: leagueclock.U3}}}, Provenance: snapshot.Provenance{Source: "app", Freshness: domain.Freshness{Note: "base"}}}
	MergeMFLCalendar(now, &clock, held)
	ids := []string{}
	labels := map[string]string{}
	for _, d := range clock.Value.Deadlines {
		ids = append(ids, d.ID)
		labels[d.ID] = d.Label
	}
	if !reflect.DeepEqual(ids, []string{"mfl-calendar:now", "app", "mfl-calendar:next", "mfl-calendar:undated"}) {
		t.Fatalf("order: %v", ids)
	}
	if labels["mfl-calendar:next"] != "MFL · Dated title" || labels["mfl-calendar:undated"] != "MFL · rookie draft" {
		t.Fatalf("labels: %v", labels)
	}
	if clock.Value.Season != 2026 || clock.Provenance.Source != "app+held-mfl-calendar" || !strings.HasPrefix(clock.Provenance.Freshness.Note, "base; MFL calendar ") || !reflect.DeepEqual(held.Value, before) {
		t.Fatalf("preserved fields/inputs: %+v", clock)
	}
}

func TestMergeMFLCalendarUnavailable(t *testing.T) {
	t.Parallel()
	clock := snapshot.Sourced[leagueclock.Reading]{Provenance: snapshot.Provenance{Source: "app"}}
	MergeMFLCalendar(time.Unix(1800000000, 0), &clock, report.HeldFeed[[]ingestion.RawCalendarEvent]{})
	if clock.Provenance.Source != "app+held-mfl-calendar" || clock.Provenance.Freshness.Note != "; MFL calendar unavailable; showing app clocks only" || len(clock.Value.Deadlines) != 0 {
		t.Fatalf("unavailable: %+v", clock)
	}
}

func TestSeasonGuess(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 12, 31, 23, 59, 0, 0, time.FixedZone("owner", -6*60*60))
	if SeasonGuess(2025, now) != 2025 || SeasonGuess(0, now) != 2026 {
		t.Fatal("held season must win, fallback uses caller calendar year")
	}
}

func TestWindowSetterRefusalsPreserveStateAndNotifyAfterExecute(t *testing.T) {
	changed, executions := 0, 0
	heads := []state.CalendarEvent{}
	service := WindowSetter{StateError: func() error { return nil }, TradeDeadline: func(context.Context) (time.Time, error) { return time.Time{}, nil },
		CalendarEvents: func(context.Context) ([]state.CalendarEvent, error) { return heads, nil }, Season: func() int { return 2026 },
		Execute: func(context.Context, transactions.Request) (transactions.Receipt, error) {
			executions++
			return transactions.Receipt{}, nil
		}, Changed: func(context.Context) {
			if executions == 0 {
				t.Fatal("event before execute")
			}
			changed++
		},
	}
	if err := service.Set(context.Background(), string(leagueclock.TradeDeadline), ""); err == nil || executions != 0 || changed != 0 {
		t.Fatal(err, executions, changed)
	}
	if err := service.Set(context.Background(), string(leagueclock.TradeDeadline), "bad date"); err == nil || executions != 0 || changed != 0 {
		t.Fatal(err, executions, changed)
	}
	if err := service.Set(context.Background(), "auction_start", ""); err == nil || executions != 0 || changed != 0 {
		t.Fatal(err, executions, changed)
	}
	heads = []state.CalendarEvent{{EventID: "window:2026:auction_start", Status: state.CalStatusPlanned, ScheduledAt: "2026-10-01T00:00:00Z"}}
	service.Execute = func(_ context.Context, request transactions.Request) (transactions.Receipt, error) {
		cancel, ok := request.(transactions.CancelEvent)
		if !ok || cancel.Event.ScheduledAt != heads[0].ScheduledAt {
			t.Fatal(request)
		}
		executions++
		return transactions.Receipt{}, nil
	}
	if err := service.Set(context.Background(), "auction_start", ""); err != nil || changed != 1 || executions != 1 {
		t.Fatal(err, executions, changed)
	}
	service.Execute = func(context.Context, transactions.Request) (transactions.Receipt, error) {
		return transactions.Receipt{}, context.Canceled
	}
	if err := service.Set(context.Background(), "auction_start", "2026-10-02T00:00:00Z"); !errors.Is(err, context.Canceled) || changed != 1 {
		t.Fatal(err, changed)
	}
}
