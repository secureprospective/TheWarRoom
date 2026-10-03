package main

import (
	"context"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/transactions"
)

// CalendarEventDTO is one calendar event's current state (its latest row). Payload is the
// opaque JSON of the operation it will run; the frontend sends it back unchanged on a
// reschedule or cancel. Status is PLANNED, FIRED or CANCELLED.
type CalendarEventDTO struct {
	EventID     string `json:"eventID"`
	Kind        string `json:"kind"`
	ScheduledAt string `json:"scheduledAt"`
	Payload     string `json:"payload"`
	Status      string `json:"status"`
	Note        string `json:"note"`
	CreatedAt   string `json:"createdAt"`
}

// CalendarEventsResult is the calendar in scheduled order. Events is never nil on success.
type CalendarEventsResult struct {
	OK     bool               `json:"ok"`
	Events []CalendarEventDTO `json:"events"`
	Detail string             `json:"detail"`
}

// GetCalendarEvents reads every event's current state. After a failed reload it reports stale
// rather than showing an out-of-date schedule as current.
func (a *App) GetCalendarEvents() CalendarEventsResult {
	if a.startupErr != nil {
		return CalendarEventsResult{Detail: a.startupErr.Error()}
	}
	if a.whatif == nil {
		return CalendarEventsResult{Detail: "state store not initialized"}
	}
	if err := a.whatif.Err(); err != nil {
		return CalendarEventsResult{Detail: "state is stale after a failed reload: " + err.Error()}
	}
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()
	evs, err := a.whatif.CalendarEvents(ctx)
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

// calendarEvent maps the DTO's calendar fields. The payload stays opaque here; the operation's
// own builder parses it when the event fires.
func calendarEvent(req TransactionRequest) transactions.CalendarEvent {
	return transactions.CalendarEvent{
		EventID:     req.EventID,
		EventKind:   req.EventKind,
		ScheduledAt: req.ScheduledAt,
		Payload:     req.Payload,
	}
}
