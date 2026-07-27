package transactions

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// This file groups the COMMISSIONER-CALENDAR ops (schedule / reschedule / cancel). They share the
// append-only calendar_events log as their write surface and are the CRUD-by-append verbs behind the
// Google-Calendar-style board: a blob is scheduled, dragged (rescheduled), or cancelled — each an
// APPEND of a new row sharing the logical event_id, never an edit (Christopher: "people's lives are
// not mutable, heavy on the write"). Split from request.go to keep both within the 400-line cap.
//
// CRITICAL: these ops record INTENT only. The stored payload (the eventual op's TransactionRequest
// fields) is NOT executed at schedule time — the calendar just remembers what should happen when.
// The payload is executed later, either by a commissioner "fire now" click (routed through the
// eventual op's own request) or, once the §12 gate opens, by the scoped auto-fire scheduler.

// schedulableKinds is the whitelist of op kinds a calendar blob may carry — the season-clock ops
// (advance/rollover/signing-window) plus the §13 commissioner acts (retirement/death/cap-relief).
// The three per-player transaction ops (trade/waiver/etc.) are NOT schedulable in v1: the calendar
// is the league-global commissioner schedule, not a per-roster to-do list. A blob whose eventual
// kind is outside this set is rejected at validate, before a transaction opens (default-deny).
func schedulableKind(k string) bool {
	switch Kind(k) {
	case KindAdvancePhase, KindRolloverSeason, KindSetSigningWindow, KindSetTradeDeadline, KindRetirement, KindDeath, KindCapRelief:
		// The season-clock ops + the §13 commissioner acts — the league-global schedule.
		return true
	case KindTrade, KindRosterStatus, KindWaiver, KindRestructure, KindTag, KindExtension, KindBuyout, KindSign,
		KindScheduleEvent, KindRescheduleEvent, KindCancelEvent, KindCorrect:
		// The per-player transaction ops are NOT schedulable in v1 (the calendar is the commissioner
		// schedule, not a per-roster to-do list); a calendar op cannot schedule another calendar op.
		// A correction is likewise not schedulable — it corrects a past entry, it is never a future intent.
		return false
	default:
		// A non-enum string (never a valid Kind) is not schedulable — default-deny.
		return false
	}
}

// CalendarEvent is the shape shared by the three calendar ops: the logical event id, the eventual op
// kind, its ISO-8601 scheduled time, and the opaque JSON payload of the eventual op's fields. It is
// distinct from state.CalendarEvent (which also carries the status + created_at the store stamps) —
// each op sets the status itself (schedule/reschedule → PLANNED, cancel → CANCELLED).
type CalendarEvent struct {
	EventID     string
	EventKind   string // the EVENTUAL op (ADVANCE_PHASE / … / CAP_RELIEF), not a calendar Kind
	ScheduledAt string // ISO-8601 (RFC3339)
	Payload     string // opaque JSON of the eventual op's fields, stored verbatim
}

// validateShape is the cheap pre-transaction gate shared by all three calendar ops: a non-empty
// event id, a schedulable eventual kind, a parseable RFC3339 time, and a non-empty payload. The
// store re-checks these atomically on append; this rejects a malformed request before a tx opens.
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

// appendWith is the shared apply body: it appends one calendar row carrying the op's status.
// It writes the row and returns an empty applyResult: a calendar op moves no players and
// carries no cap-impact line, so PlayersAffected stays 0 and Deltas nil — the quote for a schedule/
// reschedule/cancel is a pure will-succeed/rejection with no dollar figure.
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

// ScheduleEvent puts a new PLANNED blob on the commissioner calendar — the SCHEDULE_EVENT op. It
// appends a PLANNED row for a fresh event_id (the frontend mints the id). It is legal in every phase
// (the commissioner plans whenever) and records intent only — the eventual op is not run here.
type ScheduleEvent struct{ Event CalendarEvent }

func (ScheduleEvent) Kind() Kind { return KindScheduleEvent }
func (ScheduleEvent) sealed()    {}

func (s ScheduleEvent) validate() error { return s.Event.validateShape() }

func (s ScheduleEvent) apply(ctx context.Context, w state.TxWriter) (applyResult, error) {
	return s.Event.appendWith(ctx, w, state.CalStatusPlanned)
}

// RescheduleEvent moves an existing blob to a new time — the RESCHEDULE_EVENT op, the drag on the
// board. It appends a NEW PLANNED row sharing the same event_id with the new scheduled_at (never an
// UPDATE); the head view then reflects the new time and the prior position is preserved as history.
// The frontend re-sends the blob's kind + payload (it holds the head row), keeping each row
// self-contained.
type RescheduleEvent struct{ Event CalendarEvent }

func (RescheduleEvent) Kind() Kind { return KindRescheduleEvent }
func (RescheduleEvent) sealed()    {}

func (r RescheduleEvent) validate() error { return r.Event.validateShape() }

func (r RescheduleEvent) apply(ctx context.Context, w state.TxWriter) (applyResult, error) {
	return r.Event.appendWith(ctx, w, state.CalStatusPlanned)
}

// CancelEvent withdraws a blob — the CANCEL_EVENT op. It appends a CANCELLED row sharing the
// event_id, so the head view flips to CANCELLED while the full schedule/reschedule history is
// preserved. A cancelled blob is inert: the scheduler's DuePlannedEvents excludes it (status guard).
type CancelEvent struct{ Event CalendarEvent }

func (CancelEvent) Kind() Kind { return KindCancelEvent }
func (CancelEvent) sealed()    {}

func (c CancelEvent) validate() error { return c.Event.validateShape() }

func (c CancelEvent) apply(ctx context.Context, w state.TxWriter) (applyResult, error) {
	return c.Event.appendWith(ctx, w, state.CalStatusCancelled)
}
