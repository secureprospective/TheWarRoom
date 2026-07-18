package state

import (
	"context"
	"strings"
	"testing"
	"time"
)

// appendEvent runs AppendCalendarEvent through a WriteTx (the surface the calendar handlers use).
func appendEvent(t *testing.T, s *Store, e CalendarEvent) error {
	t.Helper()
	return s.WriteTx(context.Background(), func(w TxWriter) error {
		return w.AppendCalendarEvent(context.Background(), e)
	})
}

// findEvent returns the head-view row for a logical event id, or fails the test.
func findEvent(t *testing.T, evs []CalendarEvent, id string) CalendarEvent {
	t.Helper()
	for _, e := range evs {
		if e.EventID == id {
			return e
		}
	}
	t.Fatalf("event %q not in head view", id)
	return CalendarEvent{}
}

// TestAppendCalendarEvent pins the write path and its shape guards: a valid PLANNED blob lands and
// shows up in the head view; a missing field, an unparseable scheduled_at, and an unknown status
// are each rejected before any row is written.
func TestAppendCalendarEvent(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	ctx := context.Background()
	at := time.Now().UTC().Add(48 * time.Hour).Format(time.RFC3339)

	if err := appendEvent(t, s, CalendarEvent{EventID: "e1", Kind: "ADVANCE_PHASE", ScheduledAt: at, Payload: `{"toPhase":"PLAYOFFS"}`, Status: CalStatusPlanned, Note: "playoffs open"}); err != nil {
		t.Fatalf("valid event: %v", err)
	}
	evs, err := s.CalendarEvents(ctx)
	if err != nil {
		t.Fatalf("CalendarEvents: %v", err)
	}
	e := findEvent(t, evs, "e1")
	if e.Status != CalStatusPlanned || e.Kind != "ADVANCE_PHASE" || e.ScheduledAt != at {
		t.Fatalf("head row = %+v, want the PLANNED advance-phase blob", e)
	}
	if e.CreatedAt == "" {
		t.Fatal("created_at was not stamped by the store")
	}

	for _, tc := range []struct {
		name string
		e    CalendarEvent
	}{
		{"empty event id", CalendarEvent{EventID: "", Kind: "ADVANCE_PHASE", ScheduledAt: at, Payload: "{}", Status: CalStatusPlanned}},
		{"empty kind", CalendarEvent{EventID: "e", Kind: "", ScheduledAt: at, Payload: "{}", Status: CalStatusPlanned}},
		{"empty payload", CalendarEvent{EventID: "e", Kind: "ADVANCE_PHASE", ScheduledAt: at, Payload: "", Status: CalStatusPlanned}},
		{"bad status", CalendarEvent{EventID: "e", Kind: "ADVANCE_PHASE", ScheduledAt: at, Payload: "{}", Status: "BOGUS"}},
		{"unparseable time", CalendarEvent{EventID: "e", Kind: "ADVANCE_PHASE", ScheduledAt: "not-a-time", Payload: "{}", Status: CalStatusPlanned}},
	} {
		if err := appendEvent(t, s, tc.e); err == nil {
			t.Fatalf("%s: AppendCalendarEvent accepted a bad-shape entry, want rejection", tc.name)
		}
	}
}

// TestCalendarRescheduleCancelChainByEventID proves the append-only reschedule/cancel model: a drag
// and a cancel each APPEND a new row sharing the event_id, and the head view reflects only the
// latest row per event_id (never a mutation of the original). Full history is preserved underneath.
func TestCalendarRescheduleCancelChainByEventID(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	ctx := context.Background()
	t1 := time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339)
	t2 := time.Now().UTC().Add(72 * time.Hour).Format(time.RFC3339)

	// schedule
	if err := appendEvent(t, s, CalendarEvent{EventID: "ev", Kind: "SET_SIGNING_WINDOW", ScheduledAt: t1, Payload: `{"windowOpen":true}`, Status: CalStatusPlanned}); err != nil {
		t.Fatalf("schedule: %v", err)
	}
	// reschedule (drag) — same event_id, new time, still PLANNED
	if err := appendEvent(t, s, CalendarEvent{EventID: "ev", Kind: "SET_SIGNING_WINDOW", ScheduledAt: t2, Payload: `{"windowOpen":true}`, Status: CalStatusPlanned}); err != nil {
		t.Fatalf("reschedule: %v", err)
	}
	evs, _ := s.CalendarEvents(ctx)
	if got := findEvent(t, evs, "ev"); got.ScheduledAt != t2 {
		t.Fatalf("head after reschedule = %s, want the new time %s", got.ScheduledAt, t2)
	}
	if len(evs) != 1 {
		t.Fatalf("head view has %d events, want 1 (reschedule must not create a second logical event)", len(evs))
	}
	// underlying history preserved: two rows share the event id
	var rows int
	if err := s.pools.Read().QueryRowContext(ctx,
		`SELECT COUNT(1) FROM calendar_events WHERE league_id = ? AND event_id = 'ev'`, s.leagueID).Scan(&rows); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if rows != 2 {
		t.Fatalf("calendar_events holds %d rows for 'ev', want 2 (schedule + reschedule preserved)", rows)
	}
	// cancel — appended, head flips to CANCELLED
	if err := appendEvent(t, s, CalendarEvent{EventID: "ev", Kind: "SET_SIGNING_WINDOW", ScheduledAt: t2, Payload: `{"windowOpen":true}`, Status: CalStatusCancelled}); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	evs, _ = s.CalendarEvents(ctx)
	if got := findEvent(t, evs, "ev"); got.Status != CalStatusCancelled {
		t.Fatalf("head after cancel = %s, want CANCELLED", got.Status)
	}
}

// TestDuePlannedEvents proves the scheduler query is idempotency-safe and correctly scoped: it
// returns only PLANNED heads at/before now whose kind is in the allow-list — excluding future,
// cancelled, already-fired, and out-of-scope (destructive) blobs. An empty allow-list is fail-closed.
func TestDuePlannedEvents(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	ctx := context.Background()
	past := time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339)
	future := time.Now().UTC().Add(1 * time.Hour).Format(time.RFC3339)
	now := time.Now().UTC().Format(time.RFC3339)
	clock := []string{"ADVANCE_PHASE", "ROLLOVER_SEASON", "SET_SIGNING_WINDOW"}

	// due + in scope
	_ = appendEvent(t, s, CalendarEvent{EventID: "due", Kind: "ADVANCE_PHASE", ScheduledAt: past, Payload: "{}", Status: CalStatusPlanned})
	// future — not due
	_ = appendEvent(t, s, CalendarEvent{EventID: "later", Kind: "ADVANCE_PHASE", ScheduledAt: future, Payload: "{}", Status: CalStatusPlanned})
	// due but destructive — out of scope
	_ = appendEvent(t, s, CalendarEvent{EventID: "death", Kind: "DEATH", ScheduledAt: past, Payload: "{}", Status: CalStatusPlanned})
	// due but already fired — excluded by status
	_ = appendEvent(t, s, CalendarEvent{EventID: "fired", Kind: "ADVANCE_PHASE", ScheduledAt: past, Payload: "{}", Status: CalStatusPlanned})
	_ = appendEvent(t, s, CalendarEvent{EventID: "fired", Kind: "ADVANCE_PHASE", ScheduledAt: past, Payload: "{}", Status: CalStatusFired})
	// due but cancelled — excluded by status
	_ = appendEvent(t, s, CalendarEvent{EventID: "gone", Kind: "ADVANCE_PHASE", ScheduledAt: past, Payload: "{}", Status: CalStatusPlanned})
	_ = appendEvent(t, s, CalendarEvent{EventID: "gone", Kind: "ADVANCE_PHASE", ScheduledAt: past, Payload: "{}", Status: CalStatusCancelled})

	due, err := s.DuePlannedEvents(ctx, now, clock)
	if err != nil {
		t.Fatalf("DuePlannedEvents: %v", err)
	}
	if len(due) != 1 || due[0].EventID != "due" {
		ids := make([]string, len(due))
		for i, e := range due {
			ids[i] = e.EventID
		}
		t.Fatalf("due set = %v, want exactly [due]", ids)
	}

	// fail-closed: no scope named → nothing is due, even though 'due' matches on time+status.
	empty, err := s.DuePlannedEvents(ctx, now, nil)
	if err != nil {
		t.Fatalf("DuePlannedEvents(nil kinds): %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("empty-scope due set = %d, want 0 (fail-closed)", len(empty))
	}
}

// TestCalendarAppendOnlyTriggers proves the double-immutability: a raw UPDATE or DELETE that
// bypasses the (write-only) Go API still aborts at the DB.
func TestCalendarAppendOnlyTriggers(t *testing.T) {
	s := newStore(t, &fakeSource{rosters: baseRosters(t)})
	ctx := context.Background()
	at := time.Now().UTC().Format(time.RFC3339)
	if err := appendEvent(t, s, CalendarEvent{EventID: "seed", Kind: "ADVANCE_PHASE", ScheduledAt: at, Payload: "{}", Status: CalStatusPlanned}); err != nil {
		t.Fatalf("seed event: %v", err)
	}

	if _, err := s.pools.Write().ExecContext(ctx,
		`UPDATE calendar_events SET status = 'FIRED' WHERE league_id = ?`, s.leagueID); err == nil {
		t.Fatal("raw UPDATE on calendar_events succeeded — append-only trigger did not fire")
	} else if !strings.Contains(err.Error(), "append-only") {
		t.Fatalf("UPDATE error = %v, want an append-only abort", err)
	}

	if _, err := s.pools.Write().ExecContext(ctx,
		`DELETE FROM calendar_events WHERE league_id = ?`, s.leagueID); err == nil {
		t.Fatal("raw DELETE on calendar_events succeeded — append-only trigger did not fire")
	} else if !strings.Contains(err.Error(), "append-only") {
		t.Fatalf("DELETE error = %v, want an append-only abort", err)
	}
}
