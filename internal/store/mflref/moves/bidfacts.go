package moves

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine/bid"
	"github.com/secureprospective/TheWarRoom/internal/mflref/envelope"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// SaveBid saves a bid plan's envelope and its facts in one transaction, so a plan never exists
// without its facts. The facts of a plan already handed off cannot change.
func (s *Store) SaveBid(ctx context.Context, e envelope.Envelope, f envelope.BidFacts) error {
	spec := e.Receipt().Spec
	if spec.Intent != envelope.BidIntent || spec.Expected.Player != f.Player {
		return fmt.Errorf("moves: save bid %q: facts do not belong to this plan", e.ID())
	}
	return s.save(ctx, e, func(tx *sql.Tx) error {
		if err := putBidFacts(ctx, tx, e.ID(), f); err != nil {
			return err
		}
		return putBidCondition(ctx, tx, e.ID(), len(e.Receipt().Audit)-1, f)
	})
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

func putBidCondition(ctx context.Context, tx *sql.Tx, id string, sequence int, f envelope.BidFacts) error {
	var player, name string
	err := tx.QueryRowContext(ctx, `SELECT player, name FROM bid_conditions
 WHERE correlation_id = ? AND sequence = ?`, id, sequence).Scan(&player, &name)
	if err == nil {
		if player != f.ConditionalDrop.String() || name != f.ConditionalName {
			return fmt.Errorf("moves: conditional drop changed without a new plan version")
		}
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("moves: read bid condition: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO bid_conditions
 (correlation_id, sequence, player, name) VALUES (?, ?, ?, ?)`,
		id, sequence, f.ConditionalDrop.String(), f.ConditionalName); err != nil {
		return fmt.Errorf("moves: save bid condition: %w", err)
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
SELECT f.correlation_id, f.player, salary_cents, years, points_milli, snipe, beat_milli, evidence, closes,
 COALESCE(c.player, ''), COALESCE(c.name, '')
FROM bid_facts f LEFT JOIN bid_conditions c ON c.correlation_id = f.correlation_id
 AND c.sequence = (SELECT MAX(sequence) FROM bid_conditions WHERE correlation_id = f.correlation_id)
WHERE ? = '' OR f.correlation_id = ?`, id, id)
	if err != nil {
		return nil, fmt.Errorf("moves: bid facts: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := map[string]envelope.BidFacts{}
	for rows.Next() {
		var key, player, conditional string
		var salary, points int64
		var beat sql.NullInt64
		var f envelope.BidFacts
		if err := rows.Scan(&key, &player, &salary, &f.Years, &points, &f.Snipe, &beat, &f.Evidence, &f.Closes, &conditional, &f.ConditionalName); err != nil {
			return nil, fmt.Errorf("moves: bid facts scan: %w", err)
		}
		pid, err := playerid.New(player)
		if err != nil {
			return nil, fmt.Errorf("moves: bid facts %q player: %w", key, err)
		}
		if conditional != "" {
			f.ConditionalDrop, err = playerid.New(conditional)
			if err != nil {
				return nil, fmt.Errorf("moves: bid condition player: %w", err)
			}
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
