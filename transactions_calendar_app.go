package main

import (
	"github.com/secureprospective/TheWarRoom/internal/transactions"
)

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
