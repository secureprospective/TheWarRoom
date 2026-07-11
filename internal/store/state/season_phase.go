package state

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// season_phases is the append-only SEASON-PHASE transition log (Vision-2026 D3). Each row
// is one transition (from_phase → to_phase) for a (league, season); the CURRENT phase is
// the latest row's to_phase. There is no CRUD: the phase advances only by appending a row
// (the ADVANCE_PHASE op), never by updating one. Like dead_cap_ledger / contract_year_changes
// it is DOUBLE-immutable — no update/delete Go API AND BEFORE UPDATE/DELETE RAISE(ABORT)
// triggers, so a raw mutation that bypasses the Go layer still aborts (GLM panel A4: parity
// with the other audit logs; the phase log dictates the legal validity of every transaction,
// so it earns the same defense-in-depth).
//
// seq is a monotonic INTEGER PRIMARY KEY (rowid alias): inserts are serialized under wmu and
// rows are never deleted, so seq strictly increases in insertion order — "latest row" is an
// unambiguous ORDER BY seq DESC LIMIT 1, with no reliance on wall-clock ties in `at`.
//
// meta is a nullable freeform JSON slot (empty by default) so later finer phases can carry
// granularity (e.g. {"week":9}) without a schema migration (DeepSeek panel A5). v1 writes "".
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

// initSeasonPhaseSchema creates the season_phases table and its immutability triggers.
// Called from initSchema (own file, store-no-siblings + the 400-line cap), mirroring
// initLedgerSchema.
func (s *Store) initSeasonPhaseSchema(ctx context.Context) error {
	if _, err := s.pools.Write().ExecContext(ctx, seasonPhaseDDL); err != nil {
		return fmt.Errorf("state: init season-phase schema: %w", err)
	}
	return nil
}

// seedInitialPhase inserts the genesis transition row (→ OFFSEASON) for this league+season
// if the log is empty. It is the SOLE source of the initial phase — CurrentPhase reads the
// table with NO hardcoded fallback, so a missing seed fails loud rather than silently
// defaulting (GLM panel A2: one source of truth for the initial phase). Idempotent: an
// existing log (fresh-seeded earlier, or a pre-phase DB advanced since) is left untouched, so
// it never clobbers a real transition history. The genesis row's from_phase is "" (no prior
// phase). Runs independently of the roster seed — a DB that pre-dates this feature gets its
// seed row on the first startup that carries it. Season = OFFSEASON at the loaded season int
// realizes the "offseason = start of season N" invariant (domain.Phase).
func (s *Store) seedInitialPhase(ctx context.Context) error {
	var seq int64
	row := s.pools.Read().QueryRowContext(ctx,
		`SELECT seq FROM season_phases WHERE league_id = ? ORDER BY seq DESC LIMIT 1`,
		s.leagueID)
	switch err := row.Scan(&seq); {
	case errors.Is(err, sql.ErrNoRows):
		// empty log — insert the genesis row below
	case err != nil:
		return fmt.Errorf("state: seed initial phase: probe: %w", err)
	default:
		return nil // a phase already exists for this league — never reseed (league-global, D1/D7)
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

// deriveCurrentSeason reads the league-global latest phase row's season — the runtime source of
// truth for "current season" once a ROLLOVER_SEASON op has moved it past the config seed year
// (Season_Rollover_Design D1/D7). The season is DERIVED from the append-only phase log, never
// stored as a competing mutable cursor. Returns found=false only when the phase log is empty (a
// brand-new DB, before seedInitialPhase), in which case the caller keeps the config season.
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

// refreshSeason re-derives s.season from the phase log and updates the frozen field. Called under
// wmu (from Initialize and load) — s.season is written only here and read only under wmu (SQL
// scoping in load/rosterCount/writes) so no separate lock is needed. This is the make-or-break
// rollover seam (Season_Rollover_Design D6): after a rollover commits, every scoped read must track
// the new year, or the cap engine silently computes against the wrong season.
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

// AppendPhaseTransition advances the season phase by APPENDING one transition row (from the
// current phase → to) inside the shared tx — the ADVANCE_PHASE write primitive. It reads the
// committed current phase for the `from` side, rejects a no-op (to == current — the no-silent-
// no-op house rule), and writes the row through w.tx so the transition commits or rolls back
// with the op. Any target phase is permitted (v1 supports commissioner correction/rollback);
// the append-only log audits it, and rollback does NOT auto-reverse transaction_counts — a
// documented v1 posture (expert-panel A6), bounded by the per-season op ceilings. `note` is the
// commissioner's freeform reason (may be empty). Fails loud on an unknown target phase, a
// missing seed (via CurrentPhase), or a no-op.
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

// RolloverSeason advances the league from PLAYOFFS(N) to OFFSEASON(N+1) — the ROLLOVER_SEASON write
// primitive (Season_Rollover_Design D5/D7). In the shared tx it (1) appends the PLAYOFFS→OFFSEASON
// transition row carrying season N+1 (the season bump lives in this row, so D1's "latest row's
// season" derivation makes N+1 the current season after commit), and (2) advances the mutable
// roster/contract snapshot from N to N+1 so every season-scoped read follows the new year. It does
// NOT touch dead_cap_ledger / cap_relief_ledger (per-year audit rows stay put; the new season sums
// them to 0 by absence), transaction_counts (N+1 reads 0 by absence — the per-season limits reset),
// or is_restructured (the §11 lifetime guard is not season-scoped, D4). The season is MONOTONIC:
// this is the only primitive that moves it, forward by exactly one. Legal ONLY from PLAYOFFS — the
// op-eligibility gate enforces it, and this asserts it again as defense in depth (a rollover from
// any other phase would strand the current season's ledgers). load() re-derives s.season post-commit.
func (w *txWriter) RolloverSeason(ctx context.Context, note string) error {
	from, err := w.CurrentPhase(ctx)
	if err != nil {
		return fmt.Errorf("state: RolloverSeason: read current phase: %w", err)
	}
	if from != domain.PhasePlayoffs {
		return fmt.Errorf("state: RolloverSeason: only legal from PLAYOFFS (current phase is %q)", from)
	}
	// Monotonic guard (D5): re-derive the committed latest season from the log and assert the
	// in-memory season agrees before rolling. Today they always agree (refreshSeason set s.season
	// from this same row at load), so this is defense in depth — but it makes a drifted or
	// out-of-band season row fail LOUD here rather than silently rolling off the wrong year, the
	// posture the rest of the store holds. It also pins the +1 to the committed log, not just memory.
	logSeason, found, err := w.s.deriveCurrentSeason(ctx)
	if err != nil {
		return fmt.Errorf("state: RolloverSeason: derive current season: %w", err)
	}
	if !found || logSeason != w.s.season {
		return fmt.Errorf("state: RolloverSeason: in-memory season %d disagrees with phase log %d (found=%v) — refusing to roll a drifted season", w.s.season, logSeason, found)
	}
	// SOLE-OCCUPANCY INVARIANT (D6): a rollover MUST be the only op in its WriteTx. It moves the
	// season to N+1 while every co-resident contract op would still read w.s.season == N (load has
	// not run yet) and charge dead cap to the wrong year — silently. Enforced structurally today:
	// Coordinator.Execute runs exactly one leaf Request per WriteTx and RolloverSeason is a leaf, so
	// nothing can co-reside. Any future composite/batch op must keep a rollover in its own tx.
	cur := w.s.season
	next := cur + 1
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := w.tx.ExecContext(ctx, `
INSERT INTO season_phases (league_id, season, from_phase, to_phase, note, meta, at)
VALUES (?, ?, ?, ?, ?, '', ?)`,
		w.s.leagueID, next, string(domain.PhasePlayoffs), string(domain.PhaseOffseason), note, now); err != nil {
		return fmt.Errorf("state: RolloverSeason: append transition %d→%d: %w", cur, next, err)
	}
	// UFA promotion (§14, Free_Agency_Design): release every player whose contract has no PAID
	// cell for the upcoming season `next` into the free-agent pool, at $0 dead cap (expiry ≠ cut).
	// It runs BEFORE the snapshot advance below — while roster rows still carry season `cur`, so
	// ReleasePlayer's season-scoped delete (bound to w.s.season == cur, unchanged until load()
	// re-derives post-commit) matches — but "expired" is judged against `next`. The survivors are
	// what the advance then carries forward. Ordering is the panel's Q2 must-get-right.
	if err := w.promoteExpiredContracts(ctx, next); err != nil {
		return fmt.Errorf("state: RolloverSeason: promote expired: %w", err)
	}
	// Advance the mutable snapshot to N+1 (rosters + contracts are runtime state, not append-only).
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

// ufaExpiryReason is the audit string on the §14 rollover UFA-expiry status event.
const ufaExpiryReason = "ufa-expiry §14"

// promoteExpiredContracts releases every rostered player whose contract has no PAID cell for the
// UPCOMING season `next` into the free-agent pool (status FREE_AGENT) at $0 dead cap — an expired
// contract is an expiry, not a cut, so no dead-cap charge is written. Called ONLY from
// RolloverSeason, BEFORE the roster/contract snapshot advance: roster rows still carry the current
// season (matching ReleasePlayer's season-scoped delete), while "expired" is evaluated against
// `next` (the PAID-cell existence check). The roster cursor is fully drained and closed BEFORE any
// ReleasePlayer write — one tx shares one SQLite connection, so a read cursor must not overlap the
// writes. A player with a PAID cell in year >= next survives and is advanced by the caller.
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

// readExpiredRosterIDs drains the mfl ids of every current-season roster player with no PAID cell
// for `next` (the upcoming season) — the expired contracts — and closes the cursor before
// returning, so the caller's ReleasePlayer writes never overlap an open read on the shared
// connection.
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

// CurrentPhase returns the league-year's current season phase — the latest transition's
// to_phase. It reads the read pool (committed state), NOT this tx's own uncommitted writes,
// which is exactly right for the op-eligibility gate: the gate checks the phase as it stood
// BEFORE the op, and the single-writer law serializes transactions so nothing can slip in
// between. Fails loud on a missing seed row (no fallback — the seed is the one source of
// truth) or on a stored value that is not a known phase (drift).
func (w *txWriter) CurrentPhase(ctx context.Context) (domain.Phase, error) {
	return w.s.CurrentPhase(ctx)
}

// CurrentPhase reads the current season phase off the read pool (committed state). It backs
// both the txWriter surface (the op-eligibility gate) and a read-only caller (the dev IPC that
// shows the phase). It is deliberately NOT on the Reader interface — the App holds the concrete
// *Store and calls it directly, so the read-only boundary contract stays at five methods.
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
