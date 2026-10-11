package moves

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/envelope"
)

const releaseFactsDDL = `
CREATE TABLE IF NOT EXISTS release_facts (
 correlation_id TEXT PRIMARY KEY REFERENCES move_envelopes(correlation_id),
 facts TEXT NOT NULL CHECK (json_valid(facts))
);
CREATE TRIGGER IF NOT EXISTS release_facts_no_delete
BEFORE DELETE ON release_facts
BEGIN SELECT RAISE(ABORT, 'release_facts cannot be deleted'); END;`

func (s *Store) SaveRelease(ctx context.Context, e envelope.Envelope, f envelope.ReleaseFacts) error {
	spec := e.Receipt().Spec
	if !envelope.IsRelease(spec.Intent) || spec.Expected.Player != f.Player || f.Season < 2000 || f.Charge < 0 {
		return fmt.Errorf("moves: release facts do not belong to plan")
	}
	return s.save(ctx, e, func(tx *sql.Tx) error {
		if err := uniqueRelease(ctx, tx, e); err != nil {
			return err
		}
		body, err := json.Marshal(f)
		if err != nil {
			return fmt.Errorf("moves: encode release facts: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO release_facts (correlation_id,facts) VALUES (?,?) ON CONFLICT(correlation_id) DO UPDATE SET facts=excluded.facts`, e.ID(), string(body)); err != nil {
			return fmt.Errorf("moves: save release facts: %w", err)
		}
		return nil
	})
}
func uniqueRelease(ctx context.Context, tx *sql.Tx, e envelope.Envelope) error {
	if !envelope.ReleaseOpen(e.State()) {
		return nil
	}
	spec := e.Receipt().Spec
	var count int
	err := tx.QueryRowContext(ctx, `
 SELECT COUNT(*) FROM move_envelopes e WHERE correlation_id != ?
 AND json_extract(spec,'$.leagueId')=?
 AND json_extract(spec,'$.expected.player')=?
 AND json_extract(spec,'$.intent') IN (?,?)
 AND COALESCE((SELECT to_state FROM move_audit a WHERE a.correlation_id=e.correlation_id ORDER BY sequence DESC LIMIT 1),'draft') NOT IN ('landed','failed','withdrawn','awarded_off_app')`,
		e.ID(), spec.LeagueID, spec.Expected.Player.String(), envelope.CutIntent, envelope.BuyoutIntent).Scan(&count)
	if err != nil {
		return fmt.Errorf("moves: open release lookup: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("moves: player already has an open cut/buyout plan")
	}
	return nil
}
func (s *Store) ReleaseFacts(ctx context.Context, id string) (envelope.ReleaseFacts, error) {
	var body string
	if err := s.pools.Read().QueryRowContext(ctx, `SELECT facts FROM release_facts WHERE correlation_id=?`, id).Scan(&body); err != nil {
		return envelope.ReleaseFacts{}, fmt.Errorf("moves: read release facts: %w", err)
	}
	var f envelope.ReleaseFacts
	if err := json.Unmarshal([]byte(body), &f); err != nil {
		return f, fmt.Errorf("moves: decode release facts: %w", err)
	}
	if f.Player.String() == "" || f.Season < 2000 || f.Charge < 0 {
		return f, fmt.Errorf("moves: invalid release facts")
	}
	return f, nil
}
