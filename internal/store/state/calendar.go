package state

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// calendar_events is the append-only commissioner calendar: planned league ops shown as blobs on
// the board. Rows are never updated ("people's lives are not mutable", Christopher): scheduling,
// rescheduling, cancelling and firing each append a row with the same event_id, and the latest row
// (by seq) is the event's current state. Each row is self-contained: kind is the eventual op,
// payload is its fields as opaque JSON, stored and not executed. Triggers abort update and delete.
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

// Calendar statuses; an event's current status is its latest row's.
const (
	CalStatusPlanned   = "PLANNED"
	CalStatusFired     = "FIRED"
	CalStatusCancelled = "CANCELLED"
)

// validCalStatus guards the append, so an unknown status never reaches the log.
func validCalStatus(s string) bool {
	switch s {
	case CalStatusPlanned, CalStatusFired, CalStatusCancelled:
		return true
	default:
		return false
	}
}

// CalendarEvent is one calendar row. On write the store stamps CreatedAt. Payload is opaque JSON,
// resolved only when the event fires.
type CalendarEvent struct {
	EventID     string
	Kind        string
	ScheduledAt string
	Payload     string
	Status      string
	Note        string
	CreatedAt   string
}

// initCalendarSchema creates the table and its immutability triggers.
func (s *Store) initCalendarSchema(ctx context.Context) error {
	if _, err := s.pools.Write().ExecContext(ctx, calendarEventsDDL); err != nil {
		return fmt.Errorf("state: init calendar schema: %w", err)
	}
	return nil
}

// AppendCalendarEvent appends one calendar row; a reschedule or cancel is a new row with the same
// event_id. Whether firing the payload is legal is judged when it runs, not here.
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

// calendarHeadSQL selects each event's latest row, in scheduled order. Callers append the WHERE
// tail.
const calendarHeadSQL = `
SELECT c.event_id, c.kind, c.scheduled_at, c.payload, c.status, c.note, c.created_at
FROM calendar_events c
JOIN (
	SELECT event_id, MAX(seq) AS mseq
	FROM calendar_events
	WHERE league_id = ?
	GROUP BY event_id
) h ON c.seq = h.mseq`

// CalendarEvents returns each event's current row in scheduled order, CANCELLED and FIRED
// included. Never nil on success.
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

// DuePlannedEvents returns events still PLANNED and due by `now`, limited to `kinds`. The kinds
// list keeps the auto-fire scheduler to the season-clock ops: destructive commissioner events
// (retirement, death, cap relief) stay click-to-fire. A FIRED row is a durable at-most-once
// marker across restarts. An empty kinds list matches nothing.
func (s *Store) DuePlannedEvents(ctx context.Context, now string, kinds []string) ([]CalendarEvent, error) {
	if len(kinds) == 0 {
		return make([]CalendarEvent, 0), nil // no scope named: nothing is due
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(kinds)), ",")
	args := make([]any, 0, len(kinds)+3)
	args = append(args, s.leagueID)
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
