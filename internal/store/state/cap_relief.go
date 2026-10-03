package state

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// cap_relief_ledger is the append-only log of commissioner cap-relief credits (§13), each against
// an absolute league year. It is separate from dead cap, which must stay non-negative. CapUsed is
// cells + dead cap − relief, floored at 0. A franchise may get many reliefs a season, so there is
// no unique key. Triggers abort update and delete.
const capReliefDDL = `
CREATE TABLE IF NOT EXISTS cap_relief_ledger (
	seq          INTEGER PRIMARY KEY,
	league_id    TEXT NOT NULL,
	franchise_id TEXT NOT NULL,
	league_year  INTEGER NOT NULL,
	relief_cents INTEGER NOT NULL CHECK (relief_cents > 0),
	reason       TEXT NOT NULL,
	created_at   TEXT NOT NULL
);
CREATE TRIGGER IF NOT EXISTS cap_relief_ledger_no_update
BEFORE UPDATE ON cap_relief_ledger
BEGIN SELECT RAISE(ABORT, 'cap_relief_ledger is append-only'); END;
CREATE TRIGGER IF NOT EXISTS cap_relief_ledger_no_delete
BEFORE DELETE ON cap_relief_ledger
BEGIN SELECT RAISE(ABORT, 'cap_relief_ledger is append-only'); END;`

// initCapReliefSchema creates the table and its immutability triggers.
func (s *Store) initCapReliefSchema(ctx context.Context) error {
	if _, err := s.pools.Write().ExecContext(ctx, capReliefDDL); err != nil {
		return fmt.Errorf("state: init cap-relief schema: %w", err)
	}
	return nil
}

// AddCapRelief appends one cap-relief credit (§13). Whether the appeal is justified is the
// commissioner's call.
func (w *txWriter) AddCapRelief(ctx context.Context, e CapReliefEntry) error {
	if e.Amount <= 0 {
		return fmt.Errorf("state: AddCapRelief: relief must be positive, got %d", e.Amount)
	}
	if strings.TrimSpace(e.FranchiseID) == "" {
		return fmt.Errorf("state: AddCapRelief requires a franchise")
	}
	if strings.TrimSpace(e.Reason) == "" {
		return fmt.Errorf("state: AddCapRelief requires a reason")
	}
	if e.LeagueYear <= 0 {
		return fmt.Errorf("state: AddCapRelief requires an absolute league year, got %d", e.LeagueYear)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := w.tx.ExecContext(ctx, `
INSERT INTO cap_relief_ledger (league_id, franchise_id, league_year, relief_cents, reason, created_at)
VALUES (?, ?, ?, ?, ?, ?)`,
		w.s.leagueID, e.FranchiseID, e.LeagueYear, e.Amount.Cents(), e.Reason, now); err != nil {
		return fmt.Errorf("state: add cap relief: %w", err)
	}
	return nil
}

// applyCapRelief subtracts this season's relief from CapUsed, floored at 0. Relief for a franchise
// with no cap hit is ignored rather than creating a phantom franchise.
func (s *Store) applyCapRelief(ctx context.Context, fr map[string]*FranchiseState) error {
	cr, err := s.loadCapRelief(ctx)
	if err != nil {
		return err
	}
	for fid, amt := range cr {
		f, ok := fr[fid]
		if !ok {
			continue
		}
		if amt >= f.CapUsed {
			f.CapUsed = 0
		} else {
			f.CapUsed -= amt
		}
	}
	return nil
}

// loadCapRelief sums this season's relief credits per franchise.
func (s *Store) loadCapRelief(ctx context.Context) (map[string]domain.Money, error) {
	rows, err := s.pools.Read().QueryContext(ctx, `
SELECT franchise_id, COALESCE(SUM(relief_cents), 0)
FROM cap_relief_ledger
WHERE league_id = ? AND league_year = ?
GROUP BY franchise_id`, s.leagueID, s.season)
	if err != nil {
		return nil, fmt.Errorf("state: load cap relief: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := map[string]domain.Money{}
	for rows.Next() {
		var fid string
		var cents int64
		if err := rows.Scan(&fid, &cents); err != nil {
			return nil, fmt.Errorf("state: cap relief scan: %w", err)
		}
		out[fid] = domain.Money(cents)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("state: cap relief iterate: %w", err)
	}
	return out, nil
}
