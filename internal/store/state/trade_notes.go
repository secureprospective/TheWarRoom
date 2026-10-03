package state

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// initTradeNotesSchema creates trade_notes, the audit row for each executed trade, written in the
// same transaction as its player moves. picks_note is free text: there is no pick-ownership
// ledger yet, so "2027 1st to Franchise X" is a note, not data. rationale is required.
// involved_franchises is comma-joined for cheap "trades involving X" queries.
func (s *Store) initTradeNotesSchema(ctx context.Context) error {
	const ddl = `
CREATE TABLE IF NOT EXISTS trade_notes (
	id                  TEXT PRIMARY KEY,
	league_id           TEXT NOT NULL,
	season              INTEGER NOT NULL,
	created_at          TEXT NOT NULL,
	picks_note          TEXT NOT NULL DEFAULT '',
	rationale           TEXT NOT NULL,
	involved_franchises TEXT NOT NULL
);
`
	if _, err := s.pools.Write().ExecContext(ctx, ddl); err != nil {
		return fmt.Errorf("state: init trade notes schema: %w", err)
	}
	return nil
}

// LogTradeNote appends the trade's audit row inside the trade's transaction.
func (w *txWriter) LogTradeNote(ctx context.Context, picksNote, rationale string, involvedFranchises []string) error {
	id := fmt.Sprintf("tn:%s:%d", w.s.leagueID, time.Now().UnixNano())
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := w.tx.ExecContext(ctx, `
INSERT INTO trade_notes (id, league_id, season, created_at, picks_note, rationale, involved_franchises)
VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, w.s.leagueID, w.s.season, now, picksNote, rationale, strings.Join(involvedFranchises, ",")); err != nil {
		return fmt.Errorf("state: log trade note: %w", err)
	}
	return nil
}
