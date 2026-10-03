package state

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// season_phases is the append-only phase-transition log; the current phase is the latest row's
// to_phase. Rows are only appended, and triggers abort any update or delete, because this log
// decides which transactions are legal. seq orders rows, so "latest" never relies on timestamps.
// meta is a JSON slot for directives that don't change the phase (signing window, trade
// deadline).
const seasonPhaseDDL = `
CREATE TABLE IF NOT EXISTS season_phases (
	seq         INTEGER PRIMARY KEY,
	league_id   TEXT NOT NULL,
	season      INTEGER NOT NULL,
	from_phase  TEXT NOT NULL,
	to_phase    TEXT NOT NULL,
	note        TEXT NOT NULL,
	meta        TEXT NOT NULL DEFAULT '',
	at          TEXT NOT NULL
);
CREATE TRIGGER IF NOT EXISTS season_phases_no_update
BEFORE UPDATE ON season_phases
BEGIN SELECT RAISE(ABORT, 'season_phases is append-only'); END;
CREATE TRIGGER IF NOT EXISTS season_phases_no_delete
BEFORE DELETE ON season_phases
BEGIN SELECT RAISE(ABORT, 'season_phases is append-only'); END;`

// initSeasonPhaseSchema creates the table and its immutability triggers.
func (s *Store) initSeasonPhaseSchema(ctx context.Context) error {
	if _, err := s.pools.Write().ExecContext(ctx, seasonPhaseDDL); err != nil {
		return fmt.Errorf("state: init season-phase schema: %w", err)
	}
	return nil
}

// seedInitialPhase writes the genesis row (to OFFSEASON) when the log is empty. It is the only
// source of the initial phase: CurrentPhase has no fallback, so a missing seed fails loudly.
// Idempotent; an existing history is never touched.
func (s *Store) seedInitialPhase(ctx context.Context) error {
	var seq int64
	row := s.pools.Read().QueryRowContext(ctx,
		`SELECT seq FROM season_phases WHERE league_id = ? ORDER BY seq DESC LIMIT 1`,
		s.leagueID)
	switch err := row.Scan(&seq); {
	case errors.Is(err, sql.ErrNoRows):
	case err != nil:
		return fmt.Errorf("state: seed initial phase: probe: %w", err)
	default:
		return nil // a phase already exists: never reseed
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := s.pools.Write().ExecContext(ctx, `
INSERT INTO season_phases (league_id, season, from_phase, to_phase, note, meta, at)
VALUES (?, ?, '', ?, 'seed', '', ?)`,
		s.leagueID, s.season, string(domain.PhaseOffseason), now); err != nil {
		return fmt.Errorf("state: seed initial phase: insert: %w", err)
	}
	return nil
}

// deriveCurrentSeason returns the latest phase row's season: after a rollover, the log, not the
// config, says what season it is. found=false only for an empty log (a brand-new DB).
func (s *Store) deriveCurrentSeason(ctx context.Context) (season int, found bool, err error) {
	row := s.pools.Read().QueryRowContext(ctx,
		`SELECT season FROM season_phases WHERE league_id = ? ORDER BY seq DESC LIMIT 1`,
		s.leagueID)
	switch scanErr := row.Scan(&season); {
	case errors.Is(scanErr, sql.ErrNoRows):
		return 0, false, nil
	case scanErr != nil:
		return 0, false, fmt.Errorf("state: derive current season: %w", scanErr)
	}
	return season, true, nil
}

// refreshSeason re-derives s.season from the log; called under wmu. After a rollover commits,
// every season-scoped read must follow the new year, or cap math runs against the wrong season.
func (s *Store) refreshSeason(ctx context.Context) error {
	season, found, err := s.deriveCurrentSeason(ctx)
	if err != nil {
		return err
	}
	if found {
		s.season = season
	}
	return nil
}

// AppendPhaseTransition appends one transition from the committed current phase to `to`. A no-op
// is rejected; any real target is allowed (commissioner correction). Moving the phase back does
// not reverse op counters, which the per-season limits bound.
func (w *txWriter) AppendPhaseTransition(ctx context.Context, to domain.Phase, note string) error {
	if !to.Valid() {
		return fmt.Errorf("state: AppendPhaseTransition: %q is not a known phase", to)
	}
	from, err := w.CurrentPhase(ctx)
	if err != nil {
		return fmt.Errorf("state: AppendPhaseTransition: read current phase: %w", err)
	}
	if to == from {
		return fmt.Errorf("state: AppendPhaseTransition: already in phase %q (no-op rejected)", to)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := w.tx.ExecContext(ctx, `
INSERT INTO season_phases (league_id, season, from_phase, to_phase, note, meta, at)
VALUES (?, ?, ?, ?, ?, '', ?)`,
		w.s.leagueID, w.s.season, string(from), string(to), note, now); err != nil {
		return fmt.Errorf("state: AppendPhaseTransition %s→%s: insert: %w", from, to, err)
	}
	return nil
}

// RolloverSeason moves PLAYOFFS(N) to OFFSEASON(N+1): it appends the transition carrying N+1 and
// advances the roster and contract snapshot. Ledgers and op counters are untouched (N+1 reads
// them as zero); is_restructured is a lifetime guard and stays. Legal only from PLAYOFFS, checked
// here as well as at the gate: a rollover from any other phase would strand the season's ledgers.
func (w *txWriter) RolloverSeason(ctx context.Context, note string) error {
	from, err := w.CurrentPhase(ctx)
	if err != nil {
		return fmt.Errorf("state: RolloverSeason: read current phase: %w", err)
	}
	if from != domain.PhasePlayoffs {
		return fmt.Errorf("state: RolloverSeason: only legal from PLAYOFFS (current phase is %q)", from)
	}
	// The in-memory season must match the log before rolling, so a drifted season row fails loudly
	// instead of rolling off the wrong year.
	logSeason, found, err := w.s.deriveCurrentSeason(ctx)
	if err != nil {
		return fmt.Errorf("state: RolloverSeason: derive current season: %w", err)
	}
	if !found || logSeason != w.s.season {
		return fmt.Errorf("state: RolloverSeason: in-memory season %d disagrees with phase log %d (found=%v) — refusing to roll a drifted season", w.s.season, logSeason, found)
	}
	// A rollover must be the only op in its WriteTx: any other op would still read season N and
	// charge the wrong year. Today the coordinator runs one request per WriteTx.
	cur := w.s.season
	next := cur + 1
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := w.tx.ExecContext(ctx, `
INSERT INTO season_phases (league_id, season, from_phase, to_phase, note, meta, at)
VALUES (?, ?, ?, ?, ?, '', ?)`,
		w.s.leagueID, next, string(domain.PhasePlayoffs), string(domain.PhaseOffseason), note, now); err != nil {
		return fmt.Errorf("state: RolloverSeason: append transition %d→%d: %w", cur, next, err)
	}
	// Release expired contracts before advancing the snapshot: the roster rows must still carry
	// season N for ReleasePlayer's delete, while "expired" is judged against N+1.
	if err := w.promoteExpiredContracts(ctx, next); err != nil {
		return fmt.Errorf("state: RolloverSeason: promote expired: %w", err)
	}
	if _, err := w.tx.ExecContext(ctx,
		`UPDATE rosters SET season = ? WHERE league_id = ? AND season = ?`, next, w.s.leagueID, cur); err != nil {
		return fmt.Errorf("state: RolloverSeason: advance roster snapshot: %w", err)
	}
	if _, err := w.tx.ExecContext(ctx,
		`UPDATE contracts SET season = ? WHERE league_id = ? AND season = ?`, next, w.s.leagueID, cur); err != nil {
		return fmt.Errorf("state: RolloverSeason: advance contract snapshot: %w", err)
	}
	return nil
}

const ufaExpiryReason = "ufa-expiry §14"

// promoteExpiredContracts releases every rostered player with no PAID cell for `next` to free
// agency, with no dead cap: an expiry is not a cut. The cursor is drained before any write,
// because the transaction shares one connection.
func (w *txWriter) promoteExpiredContracts(ctx context.Context, next int) error {
	expired, err := w.readExpiredRosterIDs(ctx, next)
	if err != nil {
		return err
	}
	for _, id := range expired {
		if rerr := w.ReleasePlayer(ctx, id, domain.PlayerFreeAgent, ufaExpiryReason); rerr != nil {
			return fmt.Errorf("state: promoteExpiredContracts: release %q: %w", id, rerr)
		}
	}
	return nil
}

// readExpiredRosterIDs returns the expired players and closes the cursor before returning.
func (w *txWriter) readExpiredRosterIDs(ctx context.Context, next int) ([]string, error) {
	rows, err := w.tx.QueryContext(ctx, `
SELECT r.mfl_id FROM rosters r
WHERE r.league_id = ? AND r.season = ?
  AND NOT EXISTS (
	SELECT 1 FROM contract_years cy
	WHERE cy.league_id = r.league_id AND cy.mfl_id = r.mfl_id
	  AND cy.year_status = ? AND cy.league_year >= ?
  )
ORDER BY r.mfl_id`, w.s.leagueID, w.s.season, yearStatusPaid, next)
	if err != nil {
		return nil, fmt.Errorf("state: promoteExpiredContracts: read: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var expired []string
	for rows.Next() {
		var id string
		if serr := rows.Scan(&id); serr != nil {
			return nil, fmt.Errorf("state: promoteExpiredContracts: scan: %w", serr)
		}
		expired = append(expired, id)
	}
	if ierr := rows.Err(); ierr != nil {
		return nil, fmt.Errorf("state: promoteExpiredContracts: iterate: %w", ierr)
	}
	return expired, nil
}

// CurrentPhase reads committed state, which is what the eligibility gate needs: the phase before
// the op. A missing seed or an unknown stored phase is an error.
func (w *txWriter) CurrentPhase(ctx context.Context) (domain.Phase, error) {
	return w.s.CurrentPhase(ctx)
}

// CurrentPhase reads the committed phase. It is not on Reader; the App calls the concrete store.
func (s *Store) CurrentPhase(ctx context.Context) (domain.Phase, error) {
	var p string
	row := s.pools.Read().QueryRowContext(ctx, `
SELECT to_phase FROM season_phases
WHERE league_id = ?
ORDER BY seq DESC LIMIT 1`, s.leagueID)
	switch err := row.Scan(&p); {
	case errors.Is(err, sql.ErrNoRows):
		return "", fmt.Errorf("state: CurrentPhase: no phase row for league %q (seed missing)", s.leagueID)
	case err != nil:
		return "", fmt.Errorf("state: CurrentPhase: %w", err)
	}
	ph := domain.Phase(p)
	if !ph.Valid() {
		return "", fmt.Errorf("state: CurrentPhase: stored phase %q is not a known phase (drift)", p)
	}
	return ph, nil
}

// Signing-window directives. A row carrying one keeps the phase (from == to) and only moves the
// window; the window stays as set until the next toggle.
const (
	signingWindowMetaOpen   = `{"ufa_window":"open"}`
	signingWindowMetaClosed = `{"ufa_window":"closed"}`
)

// phaseMeta decodes the season_phases.meta JSON slot.
type phaseMeta struct {
	UFAWindow string `json:"ufa_window"`
}

// SigningWindowClosed lets the SIGN gate read the window inside the transaction.
func (w *txWriter) SigningWindowClosed(ctx context.Context) (bool, error) {
	return w.s.signingWindowClosed(ctx)
}

// signingWindowClosed reads the latest window directive; none means open. An unknown stored value
// is drift and fails loudly. The LIKE scan is unindexed, which is fine at a handful of rows per
// season; move the window to its own table if the log grows.
func (s *Store) signingWindowClosed(ctx context.Context) (bool, error) {
	var meta string
	row := s.pools.Read().QueryRowContext(ctx, `
SELECT meta FROM season_phases
WHERE league_id = ? AND meta LIKE '%ufa_window%'
ORDER BY seq DESC LIMIT 1`, s.leagueID)
	switch err := row.Scan(&meta); {
	case errors.Is(err, sql.ErrNoRows):
		return false, nil // no directive: open
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

// AppendSigningWindow appends a window directive that keeps the phase; a redundant toggle is
// rejected.
func (w *txWriter) AppendSigningWindow(ctx context.Context, open bool, note string) error {
	closed, err := w.s.signingWindowClosed(ctx)
	if err != nil {
		return fmt.Errorf("state: AppendSigningWindow: read current window: %w", err)
	}
	currentlyOpen := !closed
	if open == currentlyOpen {
		return fmt.Errorf("state: AppendSigningWindow: signing window already %s (no-op rejected)", windowWord(currentlyOpen))
	}
	// from == to: a directive, not a phase change.
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

func windowWord(open bool) string {
	if open {
		return "open"
	}
	return "closed"
}

// tradeDeadlineMeta decodes a trade-deadline directive; an empty Deadline means none.
type tradeDeadlineMeta struct {
	TradeDeadline string `json:"trade_deadline"`
}

// TradeDeadlinePassed lets the TRADE gate read the deadline inside the transaction.
func (w *txWriter) TradeDeadlinePassed(ctx context.Context) (bool, error) {
	return w.s.tradeDeadlinePassed(ctx)
}

// tradeDeadlinePassed reports whether the latest deadline directive has passed. None, or a
// cleared one, means no block; an unparseable stamp fails loudly.
func (s *Store) tradeDeadlinePassed(ctx context.Context) (bool, error) {
	stamp, err := s.currentTradeDeadlineStamp(ctx)
	if err != nil {
		return false, err
	}
	if stamp == "" {
		return false, nil // none, or cleared
	}
	deadline, err := time.Parse(time.RFC3339, stamp)
	if err != nil {
		return false, fmt.Errorf("state: trade deadline: stored value %q is not RFC3339 (drift): %w", stamp, err)
	}
	return !time.Now().Before(deadline), nil
}

// currentTradeDeadlineStamp returns the latest deadline stamp, or "" when none or cleared.
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

// AppendTradeDeadline appends a deadline directive that keeps the phase; a zero deadline clears
// it. Re-sending the exact same stamp is rejected, while a different past deadline is accepted.
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
