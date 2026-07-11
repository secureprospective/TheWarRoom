package state

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// This file carries the COMMISSIONER UFA-CALENDAR surface (Free_Agency_Design Q4) — the §6
// free-agency signing window's open/closed state. It rides the append-only season_phases log via
// the meta slot rather than a new table: a window toggle is a SUB-PHASE directive (the phase does
// not change), and meta is exactly the nullable slot built for that granularity. Split from
// season_phase.go so both files stay within the 400-line store cap (store-no-siblings).

// The ufa_window meta directives. A season_phases row whose meta carries one of these is a
// commissioner window toggle, not a phase transition — the phase (from == to) is unchanged; only
// the signing window moves. Ordinary transitions write meta=” and so leave the window untouched
// (the directive persists until the next toggle: "stays closed until the commissioner reopens it").
const (
	signingWindowMetaOpen   = `{"ufa_window":"open"}`
	signingWindowMetaClosed = `{"ufa_window":"closed"}`
)

// phaseMeta decodes the season_phases.meta JSON slot. Only the ufa_window key exists in v1; the
// slot stays freeform so finer phases can add keys without a schema migration.
type phaseMeta struct {
	UFAWindow string `json:"ufa_window"`
}

// SigningWindowClosed is the txWriter surface for the UFA-calendar read — see the SeasonScope
// interface doc. It delegates to the Store read (committed state) so the SIGN phase gate can
// consult the window inside the transaction.
func (w *txWriter) SigningWindowClosed(ctx context.Context) (bool, error) {
	return w.s.signingWindowClosed(ctx)
}

// signingWindowClosed reads the latest season_phases row that carries a ufa_window directive and
// reports whether it CLOSED the window. No directive → false (open by default). The LIKE filter
// narrows to directive-bearing rows (ordinary transitions write meta=”), then the JSON is parsed
// for the authoritative value — a stored value that is neither "open" nor "closed" fails loud
// (drift), mirroring CurrentPhase's unknown-phase guard.
//
// PERF (v1-acceptable): the LIKE '%ufa_window%' is an unindexable scan run on every SIGN gate and
// window read. At v1 scale (~a handful of season_phases rows per league-year) it is negligible, but
// it grows with league age — if seasons accumulate, promote the window to a derived
// signing_window_state table (or a covering index) so the SIGN hot path stays O(1) (GLM advisory).
func (s *Store) signingWindowClosed(ctx context.Context) (bool, error) {
	var meta string
	row := s.pools.Read().QueryRowContext(ctx, `
SELECT meta FROM season_phases
WHERE league_id = ? AND meta LIKE '%ufa_window%'
ORDER BY seq DESC LIMIT 1`, s.leagueID)
	switch err := row.Scan(&meta); {
	case errors.Is(err, sql.ErrNoRows):
		return false, nil // no commissioner directive ever set — window open (v1 default)
	case err != nil:
		return false, fmt.Errorf("state: signing window: %w", err)
	}
	var m phaseMeta
	if err := json.Unmarshal([]byte(meta), &m); err != nil {
		return false, fmt.Errorf("state: signing window: decode meta %q: %w", meta, err)
	}
	switch m.UFAWindow {
	case "closed":
		return true, nil
	case "open":
		return false, nil
	default:
		return false, fmt.Errorf("state: signing window: stored ufa_window %q is neither open nor closed (drift)", m.UFAWindow)
	}
}

// AppendSigningWindow is the UFA-calendar write primitive — see the SeasonScope interface doc. It
// appends a season_phases row that keeps the current phase (from == to) and stamps the ufa_window
// directive in meta, inside the shared tx, and rejects a redundant toggle.
func (w *txWriter) AppendSigningWindow(ctx context.Context, open bool, note string) error {
	closed, err := w.s.signingWindowClosed(ctx)
	if err != nil {
		return fmt.Errorf("state: AppendSigningWindow: read current window: %w", err)
	}
	currentlyOpen := !closed
	if open == currentlyOpen {
		return fmt.Errorf("state: AppendSigningWindow: signing window already %s (no-op rejected)", windowWord(currentlyOpen))
	}
	// from == to: this is a sub-phase directive, not a phase change. CurrentPhase supplies the
	// unchanged phase so the log row is well-formed (from/to are NOT NULL) and downstream phase
	// derivation is untouched (to_phase equals the standing phase).
	phase, err := w.CurrentPhase(ctx)
	if err != nil {
		return fmt.Errorf("state: AppendSigningWindow: read current phase: %w", err)
	}
	meta := signingWindowMetaOpen
	if !open {
		meta = signingWindowMetaClosed
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := w.tx.ExecContext(ctx, `
INSERT INTO season_phases (league_id, season, from_phase, to_phase, note, meta, at)
VALUES (?, ?, ?, ?, ?, ?, ?)`,
		w.s.leagueID, w.s.season, string(phase), string(phase), note, meta, now); err != nil {
		return fmt.Errorf("state: AppendSigningWindow: insert %s directive: %w", windowWord(open), err)
	}
	return nil
}

// windowWord renders the window state for error/audit text.
func windowWord(open bool) string {
	if open {
		return "open"
	}
	return "closed"
}
