package state

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// LogTradeNote is the TRADE audit-record write primitive — see the TxWriter interface doc. It
// appends one trade_notes row inside the shared tx, alongside the player moves the same Trade
// executes, so the event itself (picksNote + rationale) is never lost even though it touches no
// roster/contract row on its own.
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
