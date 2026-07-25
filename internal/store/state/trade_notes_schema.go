package state

import (
	"context"
	"fmt"
)

// initTradeNotesSchema creates the trade_notes table — the audit record for a TRADE event
// itself. Before this table existed, acquisitions.Trade() only moved players (rosters/contracts
// rows updated), so a trade's rationale and any draft picks discussed alongside it were never
// persisted anywhere — only the player-move side effects were. trade_notes is APPEND-ONLY (one
// row per executed Trade, written inside the same tx as its player moves) and ADDITIVE: a plain
// CREATE TABLE IF NOT EXISTS, no migration-registry entry, matching every sibling init*Schema.
//
// picks_note is Alpha-scope free text (unvalidated by design — no pick-ownership ledger yet,
// deliberately deferred post-Alpha per the locked panel decision) so it can carry "2027 1st to
// Franchise X" as a human-readable note without the app understanding pick ownership at all.
// rationale is required (Trade.validate() enforces non-empty before a tx ever opens).
// involved_franchises is a comma-joined list of every franchise a move in the trade touched —
// denormalized for cheap "trades involving franchise X" queries without joining back through
// rosters history.
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
