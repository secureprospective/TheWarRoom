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

// SetCell sets one PAID cell to an ABSOLUTE value and logs the old→new change with a dated
// reason, in the shared tx. Unlike MoveCellMoney it does NOT conserve the contract total —
// it is the §9 franchise-tag primitive: a tag replaces the season salary with a fresh
// one-year figure (the resolved tag price), it does not move money between years. Fails loud
// if the cell is missing (the seed guarantees a season PAID cell) or the value is negative.
func (w *txWriter) SetCell(ctx context.Context, mflID string, year int, value domain.Money, reason string) error {
	if value < 0 {
		return fmt.Errorf("state: SetCell %q/%d: value %s must not be negative", mflID, year, value)
	}
	oldCents, err := w.readCell(ctx, mflID, year)
	if err != nil {
		return err
	}
	return w.writeCell(ctx, mflID, year, oldCents, int64(value), reason)
}

// adjustCell applies a signed delta to one PAID cell and logs the old→new change with a
// dated reason, in the shared tx. It reads the cell's current value THROUGH the tx (so a
// prior write in this same tx is visible), fails loud if the cell is missing or would go
// negative, and appends an immutable change row.
func (w *txWriter) adjustCell(ctx context.Context, mflID string, year int, delta domain.Money, reason string) error {
	oldCents, err := w.readCell(ctx, mflID, year)
	if err != nil {
		return err
	}
	newCents := oldCents + int64(delta)
	if newCents < 0 {
		return fmt.Errorf("state: adjustCell %q/%d: result %d cents is negative", mflID, year, newCents)
	}
	return w.writeCell(ctx, mflID, year, oldCents, newCents, reason)
}

// readCell reads one cell's current salary THROUGH the shared tx (so a prior write in the
// same tx is visible), failing loud if the cell does not exist.
func (w *txWriter) readCell(ctx context.Context, mflID string, year int) (int64, error) {
	var oldCents int64
	row := w.tx.QueryRowContext(ctx, `
SELECT salary_cents FROM contract_years
WHERE league_id = ? AND mfl_id = ? AND league_year = ?`, w.s.leagueID, mflID, year)
	switch err := row.Scan(&oldCents); {
	case errors.Is(err, sql.ErrNoRows):
		return 0, fmt.Errorf("state: cell %q/%d: no cell to write", mflID, year)
	case err != nil:
		return 0, fmt.Errorf("state: cell %q/%d read: %w", mflID, year, err)
	}
	return oldCents, nil
}

// writeCell updates one cell to newCents and appends an immutable old→new change row, in the
// shared tx. The caller has already read oldCents and validated newCents (>= 0); this is the
// shared write+log tail of both the absolute-set (SetCell) and delta (adjustCell) paths.
func (w *txWriter) writeCell(ctx context.Context, mflID string, year int, oldCents, newCents int64, reason string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := w.tx.ExecContext(ctx, `
UPDATE contract_years SET salary_cents = ?, last_updated = ?
WHERE league_id = ? AND mfl_id = ? AND league_year = ?`,
		newCents, now, w.s.leagueID, mflID, year)
	if err != nil {
		return fmt.Errorf("state: writeCell %q/%d write: %w", mflID, year, err)
	}
	if rerr := requireOneRow(res, mflID); rerr != nil {
		return rerr
	}
	id := fmt.Sprintf("cyc:%s:%s:%d:%d", w.s.leagueID, mflID, year, time.Now().UnixNano())
	if _, err := w.tx.ExecContext(ctx, `
INSERT INTO contract_year_changes (id, league_id, mfl_id, league_year, old_cents, new_cents, reason, source, changed_at)
VALUES (?, ?, ?, ?, ?, ?, ?, 'op', ?)`,
		id, w.s.leagueID, mflID, year, oldCents, newCents, reason, now); err != nil {
		return fmt.Errorf("state: writeCell %q/%d log: %w", mflID, year, err)
	}
	return nil
}
