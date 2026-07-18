package main

import (
	"context"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/transactions"
)

// This file groups the COMMISSIONER-CALENDAR IPC surface — the board read (GetCalendarEvents) and
// the DTO→request projection its schedule/reschedule/cancel ops share. Split from transactions_app.go
// to keep both within the 400-line file cap (AD-14/AD-17). The calendar write ops route through the
// shared buildRequest dispatch in transactions_app.go; only the read + the calendar-field projection
// live here.

// CalendarEventDTO is one commissioner-calendar blob's current (head) state as it crosses the IPC
// boundary — the latest row for its logical event id. EventID is the logical blob id; Kind is the
// EVENTUAL op it will run; ScheduledAt is its ISO-8601 time; Payload is the opaque JSON of that op's
// fields (the frontend re-sends it verbatim on a reschedule/cancel and decodes it for a "fire now");
// Status is PLANNED / FIRED / CANCELLED; CreatedAt is when this head row was appended.
type CalendarEventDTO struct {
	EventID     string `json:"eventID"`
	Kind        string `json:"kind"`
	ScheduledAt string `json:"scheduledAt"`
	Payload     string `json:"payload"`
	Status      string `json:"status"`
	Note        string `json:"note"`
	CreatedAt   string `json:"createdAt"`
}

// CalendarEventsResult is a read of the commissioner calendar — the head view (latest row per event
// id), in scheduled order. Events is never null on success (Wails marshals a nil Go slice to JSON
// null, which the frontend guards with `?? []`; here it is always a non-nil slice, empty for a
// league with no blobs yet).
type CalendarEventsResult struct {
	OK     bool               `json:"ok"`
	Events []CalendarEventDTO `json:"events"`
	Detail string             `json:"detail"`
}

// GetCalendarEvents reads the commissioner calendar's head view off the concrete store (a read-only
// query, never the writer), so the board can render every blob's current position and status. It
// surfaces a stale-after-failed-reload state instead of confidently showing an out-of-date schedule.
func (a *App) GetCalendarEvents() CalendarEventsResult {
	if a.startupErr != nil {
		return CalendarEventsResult{Detail: a.startupErr.Error()}
	}
	if a.state == nil {
		return CalendarEventsResult{Detail: "state store not initialized"}
	}
	if err := a.state.Err(); err != nil {
		return CalendarEventsResult{Detail: "state is stale after a failed reload: " + err.Error()}
	}
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()
	evs, err := a.state.CalendarEvents(ctx)
	if err != nil {
		return CalendarEventsResult{Detail: err.Error()}
	}
	out := make([]CalendarEventDTO, len(evs))
	for i, e := range evs {
		out[i] = CalendarEventDTO{
			EventID:     e.EventID,
			Kind:        e.Kind,
			ScheduledAt: e.ScheduledAt,
			Payload:     e.Payload,
			Status:      e.Status,
			Note:        e.Note,
			CreatedAt:   e.CreatedAt,
		}
	}
	return CalendarEventsResult{OK: true, Events: out}
}

// calendarEvent projects the DTO's calendar fields onto the transactions.CalendarEvent the three
// CRUD-by-append ops share. No money parsing happens here — a blob's payload is opaque JSON stored
// verbatim; the eventual op's own builder parses it (millions→cents server-side) only when it fires.
func calendarEvent(req TransactionRequest) transactions.CalendarEvent {
	return transactions.CalendarEvent{
		EventID:     req.EventID,
		EventKind:   req.EventKind,
		ScheduledAt: req.ScheduledAt,
		Payload:     req.Payload,
	}
}
