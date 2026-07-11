package state

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// player_status_events is the append-only PLAYER-AVAILABILITY log — the free-agency pool's
// membership record (Free_Agency_Design). Each row is one transition of a player's off-roster
// availability (FREE_AGENT / RETIRED / DECEASED); the CURRENT status is the latest row for a
// player. There is no CRUD: status changes only by appending a row, never by updating one.
//
// It exists because ReleasePlayer — the SOLE roster-removal primitive — is shared by four
// terminal paths (waiver-cut §8, buyout §12, retirement §13, death §13) plus the §14 rollover
// UFA-expiry, all of which leave IDENTICAL empty footprints (no rosters/contracts rows). Without
// this marker a retired or deceased player is indistinguishable from a signable free agent. The
// pool = players whose latest status is FREE_AGENT; a SIGN appends nothing here (it clears the
// player by rostering him — the pool query excludes rostered players anyway), and the NEXT
// release appends his new status. A player may cycle FREE_AGENT → rostered → FREE_AGENT across
// seasons; the append-only log holds every transition with its reason.
//
// Like dead_cap_ledger / cap_relief_ledger / season_phases it is DOUBLE-immutable — no
// update/delete Go API AND BEFORE UPDATE/DELETE RAISE(ABORT) triggers, so a raw mutation that
// bypasses the Go layer still aborts (the audit-log idiom). seq is a monotonic INTEGER PRIMARY
// KEY (rowid alias): inserts are serialized under wmu and rows are never deleted, so "latest row"
// is an unambiguous ORDER BY seq DESC LIMIT 1 with no reliance on wall-clock ties in `at`.
const playerStatusDDL = `
CREATE TABLE IF NOT EXISTS player_status_events (
	seq        INTEGER PRIMARY KEY,
	league_id  TEXT NOT NULL,
	mfl_id     TEXT NOT NULL,
	status     TEXT NOT NULL,
	reason     TEXT NOT NULL,
	at         TEXT NOT NULL
);
CREATE TRIGGER IF NOT EXISTS player_status_events_no_update
BEFORE UPDATE ON player_status_events
BEGIN SELECT RAISE(ABORT, 'player_status_events is append-only'); END;
CREATE TRIGGER IF NOT EXISTS player_status_events_no_delete
BEFORE DELETE ON player_status_events
BEGIN SELECT RAISE(ABORT, 'player_status_events is append-only'); END;`

// initPlayerStatusSchema creates the player_status_events table and its immutability triggers.
// Called from initSchema, mirroring initCapReliefSchema / initSeasonPhaseSchema (own file,
// store-no-siblings + the 400-line cap).
func (s *Store) initPlayerStatusSchema(ctx context.Context) error {
	if _, err := s.pools.Write().ExecContext(ctx, playerStatusDDL); err != nil {
		return fmt.Errorf("state: init player-status schema: %w", err)
	}
	return nil
}

// RecordStatus appends one player-availability event in the shared tx — the pool's write
// primitive. It is the ENFORCED chokepoint: ReleasePlayer takes a status+reason and calls this,
// so no release path can silently forget to mark where a removed player went (the expert-panel's
// unanimous top risk — a player who is neither rostered nor findable). Append-only: no update/
// delete here, and the DB triggers reject a raw mutation. Fails loud on an unknown status or an
// empty reason.
func (w *txWriter) RecordStatus(ctx context.Context, mflID string, status domain.PlayerStatus, reason string) error {
	if !status.Valid() {
		return fmt.Errorf("state: RecordStatus %q: %q is not a known player status", mflID, status)
	}
	if reason == "" {
		return fmt.Errorf("state: RecordStatus %q: reason is required", mflID)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := w.tx.ExecContext(ctx, `
INSERT INTO player_status_events (league_id, mfl_id, status, reason, at)
VALUES (?, ?, ?, ?, ?)`,
		w.s.leagueID, mflID, string(status), reason, now); err != nil {
		return fmt.Errorf("state: RecordStatus %q → %s: %w", mflID, status, err)
	}
	return nil
}

// CurrentStatus returns a player's latest availability status and whether any status event
// exists for him. It reads the read pool (COMMITTED state), NOT this tx's own uncommitted
// writes, which is exactly right for the SIGN eligibility check: SIGN gates on the status as it
// stood BEFORE the op (set by a prior committed release), and the single-writer law serializes
// transactions so nothing can slip in between. found=false means the player has never been
// released (he is rostered, or unknown) — SIGN treats that as not-a-free-agent. Fails loud on a
// stored value that is not a known status (drift).
func (w *txWriter) CurrentStatus(ctx context.Context, mflID string) (domain.PlayerStatus, bool, error) {
	return w.s.CurrentStatus(ctx, mflID)
}

// CurrentStatus reads a player's latest availability status off the read pool (committed state).
// It backs both the txWriter surface (the SIGN eligibility gate) and read-only callers (the dev
// IPC pool listing). Deliberately NOT on the Reader interface — the App holds the concrete *Store
// and calls it directly, so the read-only boundary contract stays at its member count.
func (s *Store) CurrentStatus(ctx context.Context, mflID string) (domain.PlayerStatus, bool, error) {
	var st string
	row := s.pools.Read().QueryRowContext(ctx, `
SELECT status FROM player_status_events
WHERE league_id = ? AND mfl_id = ?
ORDER BY seq DESC LIMIT 1`, s.leagueID, mflID)
	switch err := row.Scan(&st); {
	case errors.Is(err, sql.ErrNoRows):
		return "", false, nil
	case err != nil:
		return "", false, fmt.Errorf("state: CurrentStatus %q: %w", mflID, err)
	}
	ps := domain.PlayerStatus(st)
	if !ps.Valid() {
		return "", false, fmt.Errorf("state: CurrentStatus %q: stored status %q is not known (drift)", mflID, st)
	}
	return ps, true, nil
}

// FreeAgents returns the mfl ids of every player currently in the pool — those whose LATEST
// status event is FREE_AGENT AND who are not on any roster (a signed player keeps his old
// FREE_AGENT event until his next release, so the roster exclusion is what removes him from the
// live pool). Ordered by mfl id for a deterministic listing. This is the read the dev IPC pool
// panel consumes.
func (s *Store) FreeAgents(ctx context.Context) ([]string, error) {
	rows, err := s.pools.Read().QueryContext(ctx, `
SELECT e.mfl_id FROM player_status_events e
JOIN (
	SELECT mfl_id, MAX(seq) AS seq FROM player_status_events
	WHERE league_id = ? GROUP BY mfl_id
) latest ON latest.mfl_id = e.mfl_id AND latest.seq = e.seq
WHERE e.league_id = ? AND e.status = ?
  AND NOT EXISTS (
	SELECT 1 FROM rosters r
	WHERE r.league_id = e.league_id AND r.mfl_id = e.mfl_id AND r.season = ?
  )
ORDER BY e.mfl_id`, s.leagueID, s.leagueID, string(domain.PlayerFreeAgent), s.season)
	if err != nil {
		return nil, fmt.Errorf("state: free agents: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var id string
		if serr := rows.Scan(&id); serr != nil {
			return nil, fmt.Errorf("state: free agents scan: %w", serr)
		}
		out = append(out, id)
	}
	if ierr := rows.Err(); ierr != nil {
		return nil, fmt.Errorf("state: free agents iterate: %w", ierr)
	}
	return out, nil
}
