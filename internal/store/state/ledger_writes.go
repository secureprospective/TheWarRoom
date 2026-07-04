package state

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// MoveCellMoney moves `amount` from fromYear to toYear for one player, conserving the
// contract total. It is the §11 restructure primitive: two equal-and-opposite cell deltas,
// each logged to the immutable change log, in the shared tx. The store guarantees
// conservation and non-negativity; the handler owns the rule math (how much, which years).
func (w *txWriter) MoveCellMoney(ctx context.Context, mflID string, fromYear, toYear int, amount domain.Money, reason string) error {
	if amount <= 0 {
		return fmt.Errorf("state: MoveCellMoney %q: amount must be positive, got %s", mflID, amount)
	}
	if fromYear == toYear {
		return fmt.Errorf("state: MoveCellMoney %q: from and to year are both %d", mflID, fromYear)
	}
	if err := w.adjustCell(ctx, mflID, fromYear, -amount, reason); err != nil {
		return err
	}
	return w.adjustCell(ctx, mflID, toYear, amount, reason)
}

// adjustCell applies a signed delta to one PAID cell and logs the old→new change with a
// dated reason, in the shared tx. It reads the cell's current value THROUGH the tx (so a
// prior write in this same tx is visible), fails loud if the cell is missing or would go
// negative, and appends an immutable change row.
func (w *txWriter) adjustCell(ctx context.Context, mflID string, year int, delta domain.Money, reason string) error {
	var oldCents int64
	row := w.tx.QueryRowContext(ctx, `
SELECT salary_cents FROM contract_years
WHERE league_id = ? AND mfl_id = ? AND league_year = ?`, w.s.leagueID, mflID, year)
	switch err := row.Scan(&oldCents); {
	case errors.Is(err, sql.ErrNoRows):
		return fmt.Errorf("state: adjustCell %q/%d: no cell to adjust", mflID, year)
	case err != nil:
		return fmt.Errorf("state: adjustCell %q/%d read: %w", mflID, year, err)
	}
	newCents := oldCents + int64(delta)
	if newCents < 0 {
		return fmt.Errorf("state: adjustCell %q/%d: result %d cents is negative", mflID, year, newCents)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := w.tx.ExecContext(ctx, `
UPDATE contract_years SET salary_cents = ?, last_updated = ?
WHERE league_id = ? AND mfl_id = ? AND league_year = ?`,
		newCents, now, w.s.leagueID, mflID, year)
	if err != nil {
		return fmt.Errorf("state: adjustCell %q/%d write: %w", mflID, year, err)
	}
	if rerr := requireOneRow(res, mflID); rerr != nil {
		return rerr
	}
	id := fmt.Sprintf("cyc:%s:%s:%d:%d", w.s.leagueID, mflID, year, time.Now().UnixNano())
	if _, err := w.tx.ExecContext(ctx, `
INSERT INTO contract_year_changes (id, league_id, mfl_id, league_year, old_cents, new_cents, reason, source, changed_at)
VALUES (?, ?, ?, ?, ?, ?, ?, 'op', ?)`,
		id, w.s.leagueID, mflID, year, oldCents, newCents, reason, now); err != nil {
		return fmt.Errorf("state: adjustCell %q/%d log: %w", mflID, year, err)
	}
	return nil
}
