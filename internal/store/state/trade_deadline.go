package state

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// This file carries the COMMISSIONER TRADE-DEADLINE surface (§14 Week-9 hard-block). Like the
// UFA signing window (signing_window.go), it rides the append-only season_phases log via the
// meta slot rather than a new phase-policy entry or a dedicated table: a deadline toggle is a
// SUB-PHASE directive (the phase does not change), and meta is exactly the nullable slot built
// for that granularity — neither per-week phases nor a separate deadline table exist in the
// state layer, and building either now would be premature (Alpha-scope panel lock). Split from
// signing_window.go so both stay within the 400-line store cap (store-no-siblings).

// tradeDeadlineMeta decodes the season_phases.meta JSON slot for a trade-deadline directive. An
// empty/absent Deadline means "no deadline set" (the v1 default — trades never blocked).
type tradeDeadlineMeta struct {
	TradeDeadline string `json:"trade_deadline"`
}

// TradeDeadlinePassed is the txWriter surface for the trade-deadline read — see the SeasonScope
// interface doc. It delegates to the Store read (committed state) so the TRADE phase gate can
// consult it inside the transaction.
func (w *txWriter) TradeDeadlinePassed(ctx context.Context) (bool, error) {
	return w.s.tradeDeadlinePassed(ctx)
}

// tradeDeadlinePassed reads the latest season_phases row that carries a trade_deadline
// directive and reports whether that deadline has passed. No directive, or a directive whose
// TradeDeadline is empty (a cleared deadline), → false (no block — the v1 default). The LIKE
// filter narrows to directive-bearing rows (ordinary transitions write meta=”), then the
// stored RFC3339 instant is parsed and compared against time.Now() — a stored value that fails
// to parse fails loud (drift), mirroring SigningWindowClosed's unknown-value guard.
func (s *Store) tradeDeadlinePassed(ctx context.Context) (bool, error) {
	stamp, err := s.currentTradeDeadlineStamp(ctx)
	if err != nil {
		return false, err
	}
	if stamp == "" {
		return false, nil // no directive ever set, or the most recent one CLEARED the deadline
	}
	deadline, err := time.Parse(time.RFC3339, stamp)
	if err != nil {
		return false, fmt.Errorf("state: trade deadline: stored value %q is not RFC3339 (drift): %w", stamp, err)
	}
	return !time.Now().Before(deadline), nil
}

// currentTradeDeadlineStamp reads the latest trade_deadline directive's raw stamp (empty string
// for "no directive yet" or "most recently CLEARED") — the shared read behind both
// tradeDeadlinePassed and AppendTradeDeadline's redundant-toggle guard.
func (s *Store) currentTradeDeadlineStamp(ctx context.Context) (string, error) {
	var meta string
	row := s.pools.Read().QueryRowContext(ctx, `
SELECT meta FROM season_phases
WHERE league_id = ? AND meta LIKE '%trade_deadline%'
ORDER BY seq DESC LIMIT 1`, s.leagueID)
	switch err := row.Scan(&meta); {
	case errors.Is(err, sql.ErrNoRows):
		return "", nil
	case err != nil:
		return "", fmt.Errorf("state: trade deadline: %w", err)
	}
	var m tradeDeadlineMeta
	if err := json.Unmarshal([]byte(meta), &m); err != nil {
		return "", fmt.Errorf("state: trade deadline: decode meta %q: %w", meta, err)
	}
	return m.TradeDeadline, nil
}

// AppendTradeDeadline is the trade-deadline write primitive — see the SeasonScope interface doc.
// It appends a season_phases row that keeps the current phase (from == to) and stamps the
// trade_deadline directive in meta, inside the shared tx. A zero Deadline CLEARS any standing
// deadline (writes an empty trade_deadline, so tradeDeadlinePassed reads false going forward) —
// the commissioner's "reopen the trade window" action, mirroring the signing window's reopen.
// Rejects a redundant toggle (the exact same stamp already standing — e.g. a double-click on the
// "set deadline" control) — the no-silent-no-op house rule, same guard AppendSigningWindow uses.
// Unlike that boolean toggle, this compares the STAMP, not a derived passed/not-passed state, so
// setting a genuinely different past deadline is never mistaken for a no-op.
func (w *txWriter) AppendTradeDeadline(ctx context.Context, deadline time.Time, note string) error {
	stamp := ""
	if !deadline.IsZero() {
		stamp = deadline.UTC().Format(time.RFC3339)
	}
	current, err := w.s.currentTradeDeadlineStamp(ctx)
	if err != nil {
		return fmt.Errorf("state: AppendTradeDeadline: read current directive: %w", err)
	}
	if stamp == current {
		word := "cleared"
		if stamp != "" {
			word = "set to " + stamp
		}
		return fmt.Errorf("state: AppendTradeDeadline: already %s (no-op rejected)", word)
	}
	phase, err := w.CurrentPhase(ctx)
	if err != nil {
		return fmt.Errorf("state: AppendTradeDeadline: read current phase: %w", err)
	}
	meta := fmt.Sprintf(`{"trade_deadline":%q}`, stamp)
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := w.tx.ExecContext(ctx, `
INSERT INTO season_phases (league_id, season, from_phase, to_phase, note, meta, at)
VALUES (?, ?, ?, ?, ?, ?, ?)`,
		w.s.leagueID, w.s.season, string(phase), string(phase), note, meta, now); err != nil {
		return fmt.Errorf("state: AppendTradeDeadline: insert directive: %w", err)
	}
	return nil
}
