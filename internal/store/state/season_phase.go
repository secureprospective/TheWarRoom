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

// Directive keys in season_phases.meta. A directive row keeps the phase (from == to) and sets
// one key; the newest row carrying a key is its current value.
const (
	directiveSigningWindow = "ufa_window"     // "open" | "closed"; none means open
	directiveTradeDeadline = "trade_deadline" // RFC3339, or "" for cleared
)

// latestDirective returns the newest value stored under key, and whether one exists. It reads
// committed state, not this transaction's writes.
func (s *Store) latestDirective(ctx context.Context, key string) (string, bool, error) {
	path := "$." + key
	var v string
	row := s.pools.Read().QueryRowContext(ctx, `
SELECT json_extract(meta, ?) FROM season_phases
WHERE league_id = ? AND CASE WHEN json_valid(meta) THEN json_type(meta, ?) END IS NOT NULL
ORDER BY seq DESC LIMIT 1`, path, s.leagueID, path)
	switch err := row.Scan(&v); {
	case errors.Is(err, sql.ErrNoRows):
		return "", false, nil
	case err != nil:
		return "", false, fmt.Errorf("state: read %s directive: %w", key, err)
	}
	return v, true, nil
}

// appendDirective appends a row that keeps the current phase and sets key to value.
func (w *txWriter) appendDirective(ctx context.Context, key, value, note string) error {
	phase, err := w.CurrentPhase(ctx)
	if err != nil {
		return fmt.Errorf("state: %s directive: read current phase: %w", key, err)
	}
	meta, err := json.Marshal(map[string]string{key: value})
	if err != nil {
		return fmt.Errorf("state: %s directive: encode: %w", key, err)
	}
	if _, err := w.tx.ExecContext(ctx, `
INSERT INTO season_phases (league_id, season, from_phase, to_phase, note, meta, at)
VALUES (?, ?, ?, ?, ?, ?, ?)`,
		w.s.leagueID, w.s.season, string(phase), string(phase), note, string(meta),
		time.Now().UTC().Format(time.RFC3339)); err != nil {
		return fmt.Errorf("state: %s directive: insert: %w", key, err)
	}
	return nil
}

// SigningWindowClosed lets the SIGN gate read the window inside the transaction.
func (w *txWriter) SigningWindowClosed(ctx context.Context) (bool, error) {
	return w.s.signingWindowClosed(ctx)
}

// signingWindowClosed reads the window; no directive means open, and an unknown value is drift.
func (s *Store) signingWindowClosed(ctx context.Context) (bool, error) {
	v, _, err := s.latestDirective(ctx, directiveSigningWindow)
	if err != nil {
		return false, err
	}
	switch v {
	case "closed":
		return true, nil
	case "open", "":
		return false, nil
	default:
		return false, fmt.Errorf("state: signing window: stored value %q is neither open nor closed (drift)", v)
	}
}

// AppendSigningWindow opens or closes the window; a toggle to the current state is rejected.
func (w *txWriter) AppendSigningWindow(ctx context.Context, open bool, note string) error {
	closed, err := w.s.signingWindowClosed(ctx)
	if err != nil {
		return fmt.Errorf("state: AppendSigningWindow: %w", err)
	}
	word := "closed"
	if open {
		word = "open"
	}
	if open == !closed {
		return fmt.Errorf("state: AppendSigningWindow: signing window already %s (no-op rejected)", word)
	}
	return w.appendDirective(ctx, directiveSigningWindow, word, note)
}

// TradeDeadlinePassed lets the TRADE gate read the deadline inside the transaction.
func (w *txWriter) TradeDeadlinePassed(ctx context.Context) (bool, error) {
	return w.s.tradeDeadlinePassed(ctx)
}

// tradeDeadlinePassed reports whether the current deadline has passed. None or cleared means no
// block; an unparseable stamp is drift.
func (s *Store) tradeDeadlinePassed(ctx context.Context) (bool, error) {
	stamp, _, err := s.latestDirective(ctx, directiveTradeDeadline)
	if err != nil || stamp == "" {
		return false, err
	}
	deadline, err := time.Parse(time.RFC3339, stamp)
	if err != nil {
		return false, fmt.Errorf("state: trade deadline: stored value %q is not RFC3339 (drift): %w", stamp, err)
	}
	return !time.Now().Before(deadline), nil
}

// AppendTradeDeadline sets the deadline; a zero deadline clears it. Re-sending the current stamp
// is rejected, while a different past deadline is accepted.
func (w *txWriter) AppendTradeDeadline(ctx context.Context, deadline time.Time, note string) error {
	stamp := ""
	if !deadline.IsZero() {
		stamp = deadline.UTC().Format(time.RFC3339)
	}
	current, _, err := w.s.latestDirective(ctx, directiveTradeDeadline)
	if err != nil {
		return fmt.Errorf("state: AppendTradeDeadline: %w", err)
	}
	if stamp == current {
		word := "cleared"
		if stamp != "" {
			word = "set to " + stamp
		}
		return fmt.Errorf("state: AppendTradeDeadline: already %s (no-op rejected)", word)
	}
	return w.appendDirective(ctx, directiveTradeDeadline, stamp, note)
}
