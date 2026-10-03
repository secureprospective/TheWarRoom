package state

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// player_status_events is the append-only availability log; a player's current status is his
// latest row. It exists because every removal path (cut, buyout, retirement, death, contract
// expiry) leaves the same empty footprint: without it a retired player looks like a signable free
// agent. The pool is players whose latest status is FREE_AGENT and who are on no roster. Triggers
// abort update and delete.
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

// initPlayerStatusSchema creates the table and its immutability triggers.
func (s *Store) initPlayerStatusSchema(ctx context.Context) error {
	if _, err := s.pools.Write().ExecContext(ctx, playerStatusDDL); err != nil {
		return fmt.Errorf("state: init player-status schema: %w", err)
	}
	return nil
}

// RecordStatus appends one availability event. ReleasePlayer calls it, so no release can skip
// marking where the player went.
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

// CurrentStatus returns the latest committed status, which is what SIGN must check: the status
// before the op. found=false means never released. An unknown stored status is drift.
func (w *txWriter) CurrentStatus(ctx context.Context, mflID string) (domain.PlayerStatus, bool, error) {
	return w.s.CurrentStatus(ctx, mflID)
}

// CurrentStatus reads the committed status. It is not on Reader; the App calls the concrete store.
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

// FreeAgents returns the pool by mfl id: latest status FREE_AGENT and on no roster. (A signed
// player keeps his old FREE_AGENT row until his next release; the roster check removes him.)
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
