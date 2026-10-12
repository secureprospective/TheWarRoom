package main

import (
	"context"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
	"github.com/secureprospective/TheWarRoom/internal/transactions"
)

type RawWindowDate struct{ Kind, At string }

func (r RawWindowDate) Validate() error {
	for _, rule := range leagueclock.Rules(0) {
		if string(rule.Kind) != r.Kind {
			continue
		}
		if r.At == "" {
			return nil
		}
		if _, err := time.Parse(time.RFC3339, r.At); err != nil {
			return fmt.Errorf("set a date and time with a timezone: %w", err)
		}
		return nil
	}
	return fmt.Errorf("unknown league window %q", r.Kind)
}

func (a *App) TargetSetWindow(kind, at string) error {
	if err := (RawWindowDate{Kind: kind, At: at}).Validate(); err != nil {
		return err
	}
	if err := a.ready(); err != nil {
		return fmt.Errorf("set league date: startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, m2Timeout)
	defer cancel()
	if err := a.whatif.Err(); err != nil {
		return fmt.Errorf("set league date: state: %w", err)
	}
	var err error
	if kind == string(leagueclock.TradeDeadline) {
		err = a.setTradeWindow(ctx, at)
	} else {
		err = a.setCalendarWindow(ctx, kind, at)
	}
	if err != nil {
		return err
	}
	a.clockChanged(ctx)
	return nil
}

func (a *App) setTradeWindow(ctx context.Context, at string) error {
	deadline := time.Time{}
	if at == "" {
		current, err := a.whatif.TradeDeadline(ctx)
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
	if _, err := a.coordinator.Execute(ctx, transactions.SetTradeDeadline{
		Deadline: deadline, Note: "commissioner set trade_deadline",
	}); err != nil {
		return fmt.Errorf("set trade date: %w", err)
	}
	return nil
}

func (a *App) setCalendarWindow(ctx context.Context, kind, at string) error {
	heads, err := a.whatif.CalendarEvents(ctx)
	if err != nil {
		return fmt.Errorf("read league dates: %w", err)
	}
	id := fmt.Sprintf("window:%d:%s", a.league.Season(), kind)
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
	if _, err := a.coordinator.Execute(ctx, request); err != nil {
		return fmt.Errorf("set league date: %w", err)
	}
	return nil
}
