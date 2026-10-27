package control

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
	"github.com/secureprospective/TheWarRoom/internal/transactions"
)

func MergeMFLCalendar(now time.Time, clock *snapshot.Sourced[leagueclock.Reading], held report.HeldFeed[[]ingestion.RawCalendarEvent]) {
	events := make([]leagueclock.Event, 0, len(held.Value))
	for _, r := range held.Value {
		var at time.Time
		if r.Start != "" {
			seconds, _ := strconv.ParseInt(r.Start, 10, 64)
			at = time.Unix(seconds, 0).UTC()
		}
		label := r.Title
		if label == "" {
			label = strings.ToLower(strings.ReplaceAll(r.Type, "_", " "))
		}
		events = append(events, leagueclock.Event{ID: "mfl-calendar:" + r.ID, Kind: "MFL · " + label, Status: "PLANNED", At: at})
	}
	added := leagueclock.Clock(now, leagueclock.Inputs{Events: events})
	// A calendar event that has passed is history, not a deadline: the clock lists what is coming.
	upcoming := slices.DeleteFunc(added.Deadlines, func(d leagueclock.Deadline) bool { return d.At != nil && d.At.Before(now) })
	clock.Value.Deadlines = append(clock.Value.Deadlines, upcoming...)
	slices.SortFunc(clock.Value.Deadlines, func(a, b leagueclock.Deadline) int {
		if a.Urgency != b.Urgency {
			if a.Urgency > b.Urgency {
				return -1
			}
			return 1
		}
		if a.At != nil && b.At != nil && !a.At.Equal(*b.At) {
			return a.At.Compare(*b.At)
		}
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})
	clock.Provenance.Source += "+held-mfl-calendar"
	if held.FetchedAt.IsZero() {
		clock.Provenance.Freshness.Note += "; MFL calendar unavailable; showing app clocks only"
	} else {
		clock.Provenance.Freshness.Note += "; MFL calendar " + report.FeedTime(held.FeedState(), time.Local)
	}
}

type WindowSetter struct {
	StateError     func() error
	TradeDeadline  func(context.Context) (time.Time, error)
	CalendarEvents func(context.Context) ([]state.CalendarEvent, error)
	Season         func() int
	Execute        func(context.Context, transactions.Request) (transactions.Receipt, error)
	Changed        func(context.Context)
}

// SetLeagueWindow writes one league date through the coordinator, then tells the views the clock
// changed. The caller has validated the request and waited for startup.
func (s WindowSetter) Set(ctx context.Context, kind, at string) error {
	if err := s.StateError(); err != nil {
		return fmt.Errorf("set league date: state: %w", err)
	}
	var err error
	if kind == string(leagueclock.TradeDeadline) {
		err = s.setTradeWindow(ctx, at)
	} else {
		err = s.setCalendarWindow(ctx, kind, at)
	}
	if err != nil {
		return err
	}
	s.Changed(ctx)
	return nil
}

func (s WindowSetter) setTradeWindow(ctx context.Context, at string) error {
	deadline := time.Time{}
	if at == "" {
		current, err := s.TradeDeadline(ctx)
		if err != nil {
			return fmt.Errorf("read trade date: %w", err)
		}
		if current.IsZero() {
			return fmt.Errorf("trade deadline already follows the rulebook; there is nothing to reset")
		}
	} else {
		var err error
		deadline, err = time.Parse(time.RFC3339, at)
		if err != nil {
			return fmt.Errorf("trade date: %w", err)
		}
	}
	if _, err := s.Execute(ctx, transactions.SetTradeDeadline{
		Deadline: deadline, Note: "commissioner set trade_deadline",
	}); err != nil {
		return fmt.Errorf("set trade date: %w", err)
	}
	return nil
}

func (s WindowSetter) setCalendarWindow(ctx context.Context, kind, at string) error {
	heads, err := s.CalendarEvents(ctx)
	if err != nil {
		return fmt.Errorf("read league dates: %w", err)
	}
	id := fmt.Sprintf("window:%d:%s", s.Season(), kind)
	var head *state.CalendarEvent
	for i := range heads {
		if heads[i].EventID == id {
			head = &heads[i]
			break
		}
	}
	event := transactions.CalendarEvent{EventID: id, EventKind: transactions.WindowDate,
		ScheduledAt: at, Payload: fmt.Sprintf(`{"window":%q}`, kind), Note: "commissioner set " + kind}
	var request transactions.Request = transactions.ScheduleEvent{Event: event}
	if at == "" {
		if head == nil || head.Status != state.CalStatusPlanned {
			return fmt.Errorf("this window already follows the rulebook; there is nothing to reset")
		}
		event.ScheduledAt = head.ScheduledAt
		request = transactions.CancelEvent{Event: event}
	} else if head != nil {
		request = transactions.RescheduleEvent{Event: event}
	}
	if _, err := s.Execute(ctx, request); err != nil {
		return fmt.Errorf("set league date: %w", err)
	}
	return nil
}
