package transactions

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// The calendar ops record intent only. A stored payload is not executed when scheduled: it runs
// later, from a commissioner's "fire now" or the scoped auto-fire scheduler.

// schedulableKind is the whitelist of ops a calendar event may carry: the season-clock ops and the
// §13 commissioner acts. Per-player transactions, calendar ops and corrections are not schedulable.
func schedulableKind(k string) bool {
	switch Kind(k) {
	case KindAdvancePhase, KindRolloverSeason, KindSetSigningWindow, KindSetTradeDeadline, KindRetirement, KindDeath, KindCapRelief:
		return true
	case KindTrade, KindRosterStatus, KindWaiver, KindRestructure, KindTag, KindExtension, KindBuyout, KindSign,
		KindScheduleEvent, KindRescheduleEvent, KindCancelEvent, KindCorrect:
		return false
	default:
		return false
	}
}

// CalendarEvent is the shared shape of the three calendar ops. Each op sets the status itself:
// PLANNED for schedule and reschedule, CANCELLED for cancel.
type CalendarEvent struct {
	EventID     string
	EventKind   string // the eventual op
	ScheduledAt string // RFC3339
	Payload     string // opaque JSON, stored as-is
}

// validateShape is the cheap check all three share; the store re-checks on append.
func (e CalendarEvent) validateShape() error {
	if strings.TrimSpace(e.EventID) == "" {
		return fmt.Errorf("transactions: calendar event has an empty event id")
	}
	if !schedulableKind(e.EventKind) {
		return fmt.Errorf("transactions: calendar event kind %q is not schedulable", e.EventKind)
	}
	if strings.TrimSpace(e.Payload) == "" {
		return fmt.Errorf("transactions: calendar event %q has an empty payload", e.EventID)
	}
	if _, err := time.Parse(time.RFC3339, e.ScheduledAt); err != nil {
		return fmt.Errorf("transactions: calendar event %q scheduled_at %q is not RFC3339: %w", e.EventID, e.ScheduledAt, err)
	}
	return nil
}

// appendWith appends one calendar row with the op's status. Calendar ops move no players and carry
// no cap impact.
func (e CalendarEvent) appendWith(ctx context.Context, w state.TxWriter, status string) (applyResult, error) {
	row := state.CalendarEvent{
		EventID:     e.EventID,
		Kind:        e.EventKind,
		ScheduledAt: e.ScheduledAt,
		Payload:     e.Payload,
		Status:      status,
	}
	if err := w.AppendCalendarEvent(ctx, row); err != nil {
		return applyResult{}, fmt.Errorf("calendar append: %w", err)
	}
	return applyResult{}, nil
}

// ScheduleEvent adds a PLANNED event; the frontend mints the id.
type ScheduleEvent struct{ Event CalendarEvent }

func (ScheduleEvent) Kind() Kind { return KindScheduleEvent }
func (ScheduleEvent) sealed()    {}

func (s ScheduleEvent) validate() error { return s.Event.validateShape() }

func (s ScheduleEvent) apply(ctx context.Context, w state.TxWriter) (applyResult, error) {
	return s.Event.appendWith(ctx, w, state.CalStatusPlanned)
}

// RescheduleEvent moves an event by appending a new PLANNED row with the same event_id; the old
// position stays as history.
type RescheduleEvent struct{ Event CalendarEvent }

func (RescheduleEvent) Kind() Kind { return KindRescheduleEvent }
func (RescheduleEvent) sealed()    {}

func (r RescheduleEvent) validate() error { return r.Event.validateShape() }

func (r RescheduleEvent) apply(ctx context.Context, w state.TxWriter) (applyResult, error) {
	return r.Event.appendWith(ctx, w, state.CalStatusPlanned)
}

// CancelEvent appends a CANCELLED row; the scheduler skips cancelled events.
type CancelEvent struct{ Event CalendarEvent }

func (CancelEvent) Kind() Kind { return KindCancelEvent }
func (CancelEvent) sealed()    {}

func (c CancelEvent) validate() error { return c.Event.validateShape() }

func (c CancelEvent) apply(ctx context.Context, w state.TxWriter) (applyResult, error) {
	return c.Event.appendWith(ctx, w, state.CalStatusCancelled)
}
