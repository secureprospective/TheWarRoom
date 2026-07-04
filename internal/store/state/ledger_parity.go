package state

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// rowQuerier is the read surface parity needs — satisfied by both *sql.Tx (the in-tx gate,
// which sees uncommitted writes) and *sql.DB (a committed-state diagnostic read).
type rowQuerier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// CheckLedgerParity runs the parity comparison against COMMITTED state (the read pool) — a
// diagnostic/verification entry point (tests, an admin audit) usable outside a tx. The
// in-transaction gate (Ship 2e) will call assertParity directly on the tx so a drift rolls
// back; both share the same comparison via the rowQuerier seam.
func (s *Store) CheckLedgerParity(ctx context.Context) error {
	return assertParity(ctx, s.pools.Read(), s.leagueID, s.season)
}

// assertParity compares each franchise's derived current-season cap from the LEDGER cells
// against the legacy contract columns. Both sides are built identically — each player's
// effective salary snapped to $10k (legacy) vs the sum of his PAID cells for the current
// season (ledger). Snapping is translation-invariant under the $10k-multiple moves the ops
// make, so seed parity and post-op parity both hold exactly. Compares DERIVED cap usage,
// never write targets (which legitimately diverge once restructure moves money into future
// cells the legacy column cannot hold). Any franchise mismatch fails loud.
func assertParity(ctx context.Context, q rowQuerier, leagueID string, season int) error {
	legacy, err := legacyCapByFranchise(ctx, q, leagueID, season)
	if err != nil {
		return err
	}
	cells, err := cellCapByFranchise(ctx, q, leagueID, season)
	if err != nil {
		return err
	}
	for fid, want := range legacy {
		if got := cells[fid]; got != want {
			return fmt.Errorf("state: ledger parity FAILED for franchise %q: legacy cap %s, ledger cap %s", fid, want, got)
		}
	}
	for fid, got := range cells {
		if _, ok := legacy[fid]; !ok && got != 0 {
			return fmt.Errorf("state: ledger parity FAILED for franchise %q: ledger cap %s with no legacy contracts", fid, got)
		}
	}
	return nil
}

// legacyCapByFranchise sums each franchise's rounded effective salary from the contracts
// columns. Effective = adjusted if set, else annual (EffectiveSalary), snapped per player so
// the sum matches the per-cell-snapped ledger.
func legacyCapByFranchise(ctx context.Context, q rowQuerier, leagueID string, season int) (map[string]domain.Money, error) {
	rows, err := q.QueryContext(ctx, `
SELECT c.franchise_id, c.annual_salary_cents, c.adjusted_salary_cents
FROM contracts c
JOIN rosters r ON r.league_id = c.league_id AND r.season = c.season AND r.mfl_id = c.mfl_id
WHERE c.league_id = ? AND c.season = ?`, leagueID, season)
	if err != nil {
		return nil, fmt.Errorf("state: parity legacy read: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := map[string]domain.Money{}
	for rows.Next() {
		var fid string
		var annual, adjusted int64
		if err := rows.Scan(&fid, &annual, &adjusted); err != nil {
			return nil, fmt.Errorf("state: parity legacy scan: %w", err)
		}
		eff := domain.Money(annual)
		if adjusted > 0 {
			eff = domain.Money(adjusted)
		}
		out[fid] += domain.RoundToNearest10k(eff)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("state: parity legacy iterate: %w", err)
	}
	return out, nil
}

// cellCapByFranchise sums each franchise's current-season PAID ledger cells, attributed to
// the franchise via the current-season roster join (cells are player-keyed).
func cellCapByFranchise(ctx context.Context, q rowQuerier, leagueID string, season int) (map[string]domain.Money, error) {
	rows, err := q.QueryContext(ctx, `
SELECT r.franchise_id, cy.salary_cents
FROM contract_years cy
JOIN rosters r ON r.league_id = cy.league_id AND r.mfl_id = cy.mfl_id AND r.season = ?
WHERE cy.league_id = ? AND cy.league_year = ? AND cy.year_status = ?`,
		season, leagueID, season, yearStatusPaid)
	if err != nil {
		return nil, fmt.Errorf("state: parity cell read: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := map[string]domain.Money{}
	for rows.Next() {
		var fid string
		var cents int64
		if err := rows.Scan(&fid, &cents); err != nil {
			return nil, fmt.Errorf("state: parity cell scan: %w", err)
		}
		out[fid] += domain.Money(cents)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("state: parity cell iterate: %w", err)
	}
	return out, nil
}
