package state

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// calendar_events is the append-only COMMISSIONER CALENDAR log — the schedule of planned league
// ops (phase advances, signing-window toggles, rollover, and scheduled §13 commissioner acts) that
// the Google-Calendar-style board renders as draggable "blobs". It is DELIBERATELY append-only,
// heavy-on-the-write: "people's lives are not mutable" (Christopher, 2026-07-13). A blob is never
// UPDATEd — scheduling, rescheduling (a drag to a new time), cancelling, and firing all APPEND a
// new row that carries the SAME logical event_id. The CURRENT position/status of a logical event
// is the latest row (MAX(seq)) for that event_id, exactly the season_phases / cap_relief_ledger
// latest-row-wins idiom. Full drag history is preserved for the audit trail.
//
// Like season_phases / dead_cap_ledger / cap_relief_ledger it is DOUBLE-immutable — no update/
// delete Go API AND BEFORE UPDATE/DELETE RAISE(ABORT) triggers — so a raw mutation that bypasses
// the Go layer still aborts (the audit-log idiom).
//
// seq is a monotonic INTEGER PRIMARY KEY (rowid alias): inserts serialize under wmu and rows are
// never deleted, so seq strictly increases in insertion order — "latest row per event_id" is an
// unambiguous MAX(seq), never a wall-clock tie in scheduled_at/created_at.
//
// Each row is SELF-CONTAINED: a reschedule re-carries kind + payload with the new scheduled_at, so
// the head row alone fully describes the event (no join back to the original scheduling row). kind
// is the EVENTUAL op (ADVANCE_PHASE / ROLLOVER_SEASON / SET_SIGNING_WINDOW / RETIREMENT / DEATH /
// CAP_RELIEF); payload is the opaque JSON of that op's TransactionRequest fields, stored VERBATIM
// and NOT executed at schedule time — the calendar only records intent. scheduled_at is ISO-8601
// (RFC3339). status is one of PLANNED / FIRED / CANCELLED.
const calendarEventsDDL = `
CREATE TABLE IF NOT EXISTS calendar_events (
	seq          INTEGER PRIMARY KEY,
	league_id    TEXT NOT NULL,
	event_id     TEXT NOT NULL,
	kind         TEXT NOT NULL,
	scheduled_at TEXT NOT NULL,
	payload      TEXT NOT NULL,
	status       TEXT NOT NULL,
	note         TEXT NOT NULL DEFAULT '',
	created_at   TEXT NOT NULL
);
CREATE TRIGGER IF NOT EXISTS calendar_events_no_update
BEFORE UPDATE ON calendar_events
BEGIN SELECT RAISE(ABORT, 'calendar_events is append-only'); END;
CREATE TRIGGER IF NOT EXISTS calendar_events_no_delete
BEFORE DELETE ON calendar_events
BEGIN SELECT RAISE(ABORT, 'calendar_events is append-only'); END;`

// Calendar event status values. A logical event's current status is its latest row's status:
// PLANNED (scheduled, not yet fired), FIRED (its op has been executed), or CANCELLED (withdrawn).
const (
	CalStatusPlanned   = "PLANNED"
	CalStatusFired     = "FIRED"
	CalStatusCancelled = "CANCELLED"
)

// validCalStatus reports whether s is one of the three known calendar statuses — the whitelist the
// append primitive enforces so a drift value never reaches the log.
func validCalStatus(s string) bool {
	switch s {
	case CalStatusPlanned, CalStatusFired, CalStatusCancelled:
		return true
	default:
		return false
	}
}

// CalendarEvent is one append-only calendar row as it crosses the store boundary. On a WRITE the
// caller supplies EventID/Kind/ScheduledAt/Payload/Status/Note (CreatedAt is stamped by the store);
// on a READ the head-view queries return every field, CreatedAt included. ScheduledAt is ISO-8601
// (RFC3339); Payload is opaque JSON the store never interprets — the eventual op's fields, resolved
// authoritatively only when the event is fired.
type CalendarEvent struct {
	EventID     string
	Kind        string
	ScheduledAt string
	Payload     string
	Status      string
	Note        string
	CreatedAt   string
}

// initCalendarSchema creates the calendar_events table and its immutability triggers. Called from
// initSchema, mirroring initCapReliefSchema / initSeasonPhaseSchema (own file, store-no-siblings +
// the 400-line cap).
func (s *Store) initCalendarSchema(ctx context.Context) error {
	if _, err := s.pools.Write().ExecContext(ctx, calendarEventsDDL); err != nil {
		return fmt.Errorf("state: init calendar schema: %w", err)
	}
	return nil
}

// AppendCalendarEvent appends one row to the calendar log in the shared tx — the write primitive
// behind SCHEDULE_EVENT / RESCHEDULE_EVENT / CANCEL_EVENT (and, once the §12 gate opens, the FIRED
// mark). It is APPEND-ONLY (no update/delete here, and the DB triggers reject a raw mutation): a
// reschedule or cancel is a NEW row carrying the same event_id, never an edit of the old one. The
// store records the intent VERBATIM — whether firing the payload is legal is judged only when the
// op actually runs, not by this pure data layer. Fails loud on a missing field, an unparseable
// scheduled_at, or an unknown status.
func (w *txWriter) AppendCalendarEvent(ctx context.Context, e CalendarEvent) error {
	if strings.TrimSpace(e.EventID) == "" {
		return fmt.Errorf("state: AppendCalendarEvent requires an event id")
	}
	if strings.TrimSpace(e.Kind) == "" {
		return fmt.Errorf("state: AppendCalendarEvent requires a kind")
	}
	if strings.TrimSpace(e.Payload) == "" {
		return fmt.Errorf("state: AppendCalendarEvent requires a payload")
	}
	if !validCalStatus(e.Status) {
		return fmt.Errorf("state: AppendCalendarEvent: unknown status %q", e.Status)
	}
	if _, err := time.Parse(time.RFC3339, e.ScheduledAt); err != nil {
		return fmt.Errorf("state: AppendCalendarEvent: scheduled_at %q is not RFC3339: %w", e.ScheduledAt, err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := w.tx.ExecContext(ctx, `
INSERT INTO calendar_events (league_id, event_id, kind, scheduled_at, payload, status, note, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		w.s.leagueID, e.EventID, e.Kind, e.ScheduledAt, e.Payload, e.Status, e.Note, now); err != nil {
		return fmt.Errorf("state: append calendar event %q: %w", e.EventID, err)
	}
	return nil
}

// calendarHeadSQL selects the HEAD row (latest seq) of every logical event for this league — the
// current position/status of each blob. Reschedule/cancel/fire append rows sharing an event_id;
// this join keeps only the newest per event_id. Ordered by scheduled_at so the board renders in
// time order. The WHERE-clause tail is appended by the callers (all-events vs due-planned).
const calendarHeadSQL = `
SELECT c.event_id, c.kind, c.scheduled_at, c.payload, c.status, c.note, c.created_at
FROM calendar_events c
JOIN (
	SELECT event_id, MAX(seq) AS mseq
	FROM calendar_events
	WHERE league_id = ?
	GROUP BY event_id
) h ON c.seq = h.mseq`

// CalendarEvents returns the current head view — the latest row per logical event_id for this
// league, in scheduled order. It reads the read pool (committed state) and backs GetCalendarEvents
// (the board render). CANCELLED and FIRED heads are included so the UI can show history/state; the
// frontend decides how to style them. Never returns a nil slice on success (empty is a valid board).
func (s *Store) CalendarEvents(ctx context.Context) ([]CalendarEvent, error) {
	rows, err := s.pools.Read().QueryContext(ctx, calendarHeadSQL+`
ORDER BY c.scheduled_at`, s.leagueID)
	if err != nil {
		return nil, fmt.Errorf("state: calendar events: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make([]CalendarEvent, 0)
	for rows.Next() {
		var e CalendarEvent
		if serr := rows.Scan(&e.EventID, &e.Kind, &e.ScheduledAt, &e.Payload, &e.Status, &e.Note, &e.CreatedAt); serr != nil {
			return nil, fmt.Errorf("state: calendar events scan: %w", serr)
		}
		out = append(out, e)
	}
	if ierr := rows.Err(); ierr != nil {
		return nil, fmt.Errorf("state: calendar events iterate: %w", ierr)
	}
	return out, nil
}

// DuePlannedEvents returns the head events that are DUE and still PLANNED — status PLANNED, whose
// latest scheduled_at is at or before `now`, restricted to the `kinds` allow-list. It is the
// auto-fire scheduler's idempotency-safe query (Phase 6, behind the §12 gate): the kinds filter is
// how the scheduler is SCOPED to the season-clock ops and can never pick up a destructive
// commissioner blob (RETIREMENT/DEATH/CAP_RELIEF stay click-to-fire). A CANCELLED or already-FIRED
// head is excluded by the status check, so a committed FIRED row is a durable at-most-once marker
// across restarts. `now` is an RFC3339 string compared lexicographically (RFC3339 UTC sorts
// chronologically). An empty kinds list matches nothing (fail-closed — the scheduler must name its
// scope), never everything.
func (s *Store) DuePlannedEvents(ctx context.Context, now string, kinds []string) ([]CalendarEvent, error) {
	if len(kinds) == 0 {
		return make([]CalendarEvent, 0), nil // fail-closed: no scope named → nothing is due
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(kinds)), ",")
	args := make([]any, 0, len(kinds)+3)
	args = append(args, s.leagueID) // calendarHeadSQL's league filter
	args = append(args, s.leagueID, CalStatusPlanned, now)
	for _, k := range kinds {
		args = append(args, k)
	}
	rows, err := s.pools.Read().QueryContext(ctx, calendarHeadSQL+`
WHERE c.league_id = ? AND c.status = ? AND c.scheduled_at <= ?
  AND c.kind IN (`+placeholders+`)
ORDER BY c.scheduled_at`, args...)
	if err != nil {
		return nil, fmt.Errorf("state: due planned events: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make([]CalendarEvent, 0)
	for rows.Next() {
		var e CalendarEvent
		if serr := rows.Scan(&e.EventID, &e.Kind, &e.ScheduledAt, &e.Payload, &e.Status, &e.Note, &e.CreatedAt); serr != nil {
			return nil, fmt.Errorf("state: due planned events scan: %w", serr)
		}
		out = append(out, e)
	}
	if ierr := rows.Err(); ierr != nil {
		return nil, fmt.Errorf("state: due planned events iterate: %w", ierr)
	}
	return out, nil
}
