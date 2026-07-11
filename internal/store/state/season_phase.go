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
		`SELECT seq FROM season_phases WHERE league_id = ? AND season = ? ORDER BY seq DESC LIMIT 1`,
		s.leagueID, s.season)
	switch err := row.Scan(&seq); {
	case errors.Is(err, sql.ErrNoRows):
		// empty log — insert the genesis row below
	case err != nil:
		return fmt.Errorf("state: seed initial phase: probe: %w", err)
	default:
		return nil // a phase already exists for this league-year — never reseed
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
WHERE league_id = ? AND season = ?
ORDER BY seq DESC LIMIT 1`, s.leagueID, s.season)
	switch err := row.Scan(&p); {
	case errors.Is(err, sql.ErrNoRows):
		return "", fmt.Errorf("state: CurrentPhase: no phase row for league %q season %d (seed missing)", s.leagueID, s.season)
	case err != nil:
		return "", fmt.Errorf("state: CurrentPhase: %w", err)
	}
	ph := domain.Phase(p)
	if !ph.Valid() {
		return "", fmt.Errorf("state: CurrentPhase: stored phase %q is not a known phase (drift)", p)
	}
	return ph, nil
}
