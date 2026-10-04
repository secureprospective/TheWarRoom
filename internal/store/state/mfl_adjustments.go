package state

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// AdjustmentSource is a Source that also supplies MFL's salary adjustments. A fresh what-if league
// seeds them with the rosters, so its cap starts equal to the mirror's; the Mirror is one.
type AdjustmentSource interface {
	Source
	Adjustments(ctx context.Context) ([]Adjustment, error)
}

// mfl_salary_adjustments holds MFL's salary adjustments as seeded: dead-cap charges, and credits
// when negative. They are MFL's figures, kept apart from the app's own dead_cap_ledger, which
// must stay non-negative.
const mflAdjustmentsDDL = `
CREATE TABLE IF NOT EXISTS mfl_salary_adjustments (
	league_id    TEXT NOT NULL,
	season       INTEGER NOT NULL,
	id           TEXT NOT NULL,
	franchise_id TEXT NOT NULL,
	amount_cents INTEGER NOT NULL,
	description  TEXT NOT NULL,
	PRIMARY KEY (league_id, season, id)
);`

func (s *Store) initMFLAdjustmentsSchema(ctx context.Context) error {
	if _, err := s.pools.Write().ExecContext(ctx, mflAdjustmentsDDL); err != nil {
		return fmt.Errorf("state: init mfl adjustments schema: %w", err)
	}
	return nil
}

// Adjustments returns the mirror's salary adjustments, so it can seed a what-if Store.
func (m *Mirror) Adjustments(_ context.Context) ([]Adjustment, error) {
	m.wmu.Lock()
	defer m.wmu.Unlock()
	return append([]Adjustment(nil), m.snap.Adjustments...), nil
}

// seedAdjustments writes src's salary adjustments in the seed transaction, when src has them.
func (s *Store) seedAdjustments(ctx context.Context, tx *sql.Tx, src Source) error {
	as, ok := src.(AdjustmentSource)
	if !ok {
		return nil
	}
	adjs, err := as.Adjustments(ctx)
	if err != nil {
		return fmt.Errorf("state: seed adjustments: %w", err)
	}
	for _, a := range adjs {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO mfl_salary_adjustments (league_id, season, id, franchise_id, amount_cents, description)
VALUES (?, ?, ?, ?, ?, ?)`, s.leagueID, s.season, a.ID, a.FranchiseID, a.Amount.Cents(), a.Description); err != nil {
			return fmt.Errorf("state: seed adjustment %s: %w", a.ID, err)
		}
	}
	return nil
}

// applyMFLAdjustments adds this season's MFL salary adjustments to each franchise's cap.
func (s *Store) applyMFLAdjustments(ctx context.Context, fr map[string]*FranchiseState) error {
	rows, err := s.pools.Read().QueryContext(ctx, `
SELECT franchise_id, SUM(amount_cents) FROM mfl_salary_adjustments
WHERE league_id = ? AND season = ? GROUP BY franchise_id`, s.leagueID, s.season)
	if err != nil {
		return fmt.Errorf("state: load mfl adjustments: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var fid string
		var cents int64
		if err := rows.Scan(&fid, &cents); err != nil {
			return fmt.Errorf("state: mfl adjustments scan: %w", err)
		}
		if fr[fid] == nil {
			fr[fid] = &FranchiseState{FranchiseID: fid}
		}
		fr[fid].CapUsed += domain.Money(cents)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("state: mfl adjustments iterate: %w", err)
	}
	return nil
}
