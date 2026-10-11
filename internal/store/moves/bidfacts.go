package moves

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine/bid"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// bidFactsDDL is the one side table for bid plans, keyed by the envelope's correlation id. It
// holds only the owner's own figures: no franchise column, so no other team can be recorded.
// The rulebook limits are check-time code, not constraints here. Facts change only while the
// plan is unposted; once handed off they are the record of what was posted.
const bidFactsDDL = `
CREATE TABLE IF NOT EXISTS bid_facts (
 correlation_id TEXT PRIMARY KEY REFERENCES move_envelopes(correlation_id),
 player TEXT NOT NULL,
 salary_cents INTEGER NOT NULL,
 years INTEGER NOT NULL,
 points_milli INTEGER NOT NULL,
 snipe INTEGER NOT NULL CHECK (snipe IN (0, 1)),
 beat_milli INTEGER,
 evidence TEXT NOT NULL,
 closes TEXT NOT NULL
);
CREATE TRIGGER IF NOT EXISTS bid_facts_no_delete
BEFORE DELETE ON bid_facts
BEGIN SELECT RAISE(ABORT, 'bid_facts is append-only'); END;
CREATE TRIGGER IF NOT EXISTS bid_facts_frozen
BEFORE UPDATE ON bid_facts
WHEN (
 SELECT to_state FROM move_audit WHERE correlation_id = OLD.correlation_id
 ORDER BY sequence DESC LIMIT 1
) NOT IN ('draft', 'blocked', 'ready', 'stale')
BEGIN SELECT RAISE(ABORT, 'bid_facts is frozen once the bid is handed off'); END;`

// SaveBid saves a bid plan's envelope and its facts in one transaction, so a plan never exists
// without its facts. The facts of a plan already handed off cannot change.
func (s *Store) SaveBid(ctx context.Context, e envelope.Envelope, f envelope.BidFacts) error {
	spec := e.Receipt().Spec
	if spec.Intent != envelope.BidIntent || spec.Expected.Player != f.Player {
		return fmt.Errorf("moves: save bid %q: facts do not belong to this plan", e.ID())
	}
	return s.save(ctx, e, func(tx *sql.Tx) error { return putBidFacts(ctx, tx, e.ID(), f) })
}

func putBidFacts(ctx context.Context, tx *sql.Tx, id string, f envelope.BidFacts) error {
	var beat sql.NullInt64
	if f.Beat != nil {
		beat = sql.NullInt64{Int64: int64(*f.Beat), Valid: true}
	}
	_, err := tx.ExecContext(ctx, `
INSERT INTO bid_facts (correlation_id, player, salary_cents, years, points_milli, snipe, beat_milli, evidence, closes)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (correlation_id) DO UPDATE SET player = excluded.player, salary_cents = excluded.salary_cents,
 years = excluded.years, points_milli = excluded.points_milli, snipe = excluded.snipe,
 beat_milli = excluded.beat_milli, evidence = excluded.evidence, closes = excluded.closes`,
		id, f.Player.String(), f.Salary.Cents(), f.Years, int64(f.Points), f.Snipe, beat, f.Evidence, f.Closes)
	if err != nil {
		return fmt.Errorf("moves: save %q bid facts: %w", id, err)
	}
	return nil
}

// BidFacts reads one plan's facts; a plan without any is ErrNotFound.
func (s *Store) BidFacts(ctx context.Context, id string) (envelope.BidFacts, error) {
	rows, err := s.queryBidFacts(ctx, id)
	if err != nil {
		return envelope.BidFacts{}, err
	}
	f, ok := rows[id]
	if !ok {
		return envelope.BidFacts{}, fmt.Errorf("moves: bid facts %q: %w", id, ErrNotFound)
	}
	return f, nil
}

// AllBidFacts reads every plan's facts by correlation id.
func (s *Store) AllBidFacts(ctx context.Context) (map[string]envelope.BidFacts, error) {
	return s.queryBidFacts(ctx, "")
}

func (s *Store) queryBidFacts(ctx context.Context, id string) (map[string]envelope.BidFacts, error) {
	rows, err := s.pools.Read().QueryContext(ctx, `
SELECT correlation_id, player, salary_cents, years, points_milli, snipe, beat_milli, evidence, closes
FROM bid_facts WHERE ? = '' OR correlation_id = ?`, id, id)
	if err != nil {
		return nil, fmt.Errorf("moves: bid facts: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := map[string]envelope.BidFacts{}
	for rows.Next() {
		var key, player string
		var salary, points int64
		var beat sql.NullInt64
		var f envelope.BidFacts
		if err := rows.Scan(&key, &player, &salary, &f.Years, &points, &f.Snipe, &beat, &f.Evidence, &f.Closes); err != nil {
			return nil, fmt.Errorf("moves: bid facts scan: %w", err)
		}
		pid, err := playerid.New(player)
		if err != nil {
			return nil, fmt.Errorf("moves: bid facts %q player: %w", key, err)
		}
		f.Player, f.Salary, f.Points = pid, domain.Money(salary), bid.Points(points)
		if beat.Valid {
			b := bid.Points(beat.Int64)
			f.Beat = &b
		}
		out[key] = f
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("moves: bid facts rows: %w", err)
	}
	return out, nil
}
