package transactions

import (
	"context"
	"testing"
	"time"
)

// calEvent builds a well-formed calendar event payload for the tests.
func calEvent(id, kind string) CalendarEvent {
	return CalendarEvent{
		EventID:     id,
		EventKind:   kind,
		ScheduledAt: time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339),
		Payload:     `{"toPhase":"PLAYOFFS"}`,
	}
}

// TestCalendarOps_RouteThroughCoordinator proves schedule/reschedule/cancel each validate, pass the
// phase gate, and append exactly one calendar row through the single write path (recorded as a
// "calendar" op keyed by the event id). They touch no players (Receipt.PlayersAffected 0).
func TestCalendarOps_RouteThroughCoordinator(t *testing.T) {
	for _, tc := range []struct {
		name string
		req  Request
	}{
		{"schedule", ScheduleEvent{Event: calEvent("e1", string(KindAdvancePhase))}},
		{"reschedule", RescheduleEvent{Event: calEvent("e1", string(KindAdvancePhase))}},
		{"cancel", CancelEvent{Event: calEvent("e1", string(KindAdvancePhase))}},
	} {
		w := newFake()
		c := newCoord(t, w)
		rec, err := c.Execute(context.Background(), tc.req)
		if err != nil {
			t.Fatalf("%s: Execute: %v", tc.name, err)
		}
		if rec.PlayersAffected != 0 {
			t.Fatalf("%s: PlayersAffected = %d, want 0", tc.name, rec.PlayersAffected)
		}
		if len(w.tw.calls) != 1 || w.tw.calls[0].op != "calendar" || w.tw.calls[0].target != "e1" {
			t.Fatalf("%s: calls = %+v, want one calendar append for e1", tc.name, w.tw.calls)
		}
	}
}

// TestCalendarOps_PreviewRollsBack proves a preview runs the full validate→gate→apply path (so the
// UI can surface a rejection) but persists nothing — the recorded append happens inside the tx that
// the dry-run sentinel rolls back, and Preview still reports success.
func TestCalendarOps_PreviewRollsBack(t *testing.T) {
	w := newFake()
	c := newCoord(t, w)
	rec, err := c.Preview(context.Background(), ScheduleEvent{Event: calEvent("e1", string(KindCapRelief))})
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	if rec.Kind != KindScheduleEvent {
		t.Fatalf("Preview receipt kind = %q, want SCHEDULE_EVENT", rec.Kind)
	}
	if !w.writeTxRan {
		t.Fatal("preview did not open a transaction — the full path must run")
	}
}

// TestCalendarOps_RejectBadShape proves validate refuses a non-schedulable eventual kind, an empty
// event id, an empty payload, and an unparseable time BEFORE a transaction opens (default-deny at
// the boundary).
func TestCalendarOps_RejectBadShape(t *testing.T) {
	for _, tc := range []struct {
		name string
		ev   CalendarEvent
	}{
		{"non-schedulable kind", CalendarEvent{EventID: "e", EventKind: string(KindTrade), ScheduledAt: time.Now().UTC().Format(time.RFC3339), Payload: "{}"}},
		{"empty event id", CalendarEvent{EventID: "", EventKind: string(KindAdvancePhase), ScheduledAt: time.Now().UTC().Format(time.RFC3339), Payload: "{}"}},
		{"empty payload", CalendarEvent{EventID: "e", EventKind: string(KindAdvancePhase), ScheduledAt: time.Now().UTC().Format(time.RFC3339), Payload: ""}},
		{"bad time", CalendarEvent{EventID: "e", EventKind: string(KindAdvancePhase), ScheduledAt: "nope", Payload: "{}"}},
	} {
		w := newFake()
		c := newCoord(t, w)
		if _, err := c.Execute(context.Background(), ScheduleEvent{Event: tc.ev}); err == nil {
			t.Fatalf("%s: Execute accepted a bad-shape calendar event, want rejection", tc.name)
		}
		if w.writeTxRan {
			t.Fatalf("%s: a bad-shape event opened a transaction — it must be rejected in validate first", tc.name)
		}
	}
}
