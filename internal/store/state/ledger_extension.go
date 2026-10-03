package state

import (
	"context"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// SourceExtension tags every cell a §10 extension writes, so a later extension can see from the
// cells alone that the contract was already extended.
const SourceExtension = "extension"

// PaidCells reads a player's PAID cells through the transaction, by year.
func (w *txWriter) PaidCells(ctx context.Context, mflID string) ([]LedgerCell, error) {
	rows, err := w.tx.QueryContext(ctx, `
SELECT league_year, salary_cents, source FROM contract_years
WHERE league_id = ? AND mfl_id = ? AND year_status = ?
ORDER BY league_year`, w.s.leagueID, mflID, yearStatusPaid)
	if err != nil {
		return nil, fmt.Errorf("state: PaidCells %q: read: %w", mflID, err)
	}
	defer func() { _ = rows.Close() }()
	var cells []LedgerCell
	for rows.Next() {
		var c LedgerCell
		var cents int64
		if serr := rows.Scan(&c.Year, &cents, &c.Source); serr != nil {
			return nil, fmt.Errorf("state: PaidCells %q: scan: %w", mflID, serr)
		}
		c.Salary = domain.Money(cents)
		cells = append(cells, c)
	}
	if ierr := rows.Err(); ierr != nil {
		return nil, fmt.Errorf("state: PaidCells %q: iterate: %w", mflID, ierr)
	}
	return cells, nil
}

// AppendExtensionYears adds `addedYears` PAID cells at pricePerYear (§10): the UFA slot becomes the
// first new year, further years are inserted, and a new UFA slot follows. All are tagged and
// logged. No PAID cell, or a missing or misplaced UFA slot, is an error.
func (w *txWriter) AppendExtensionYears(ctx context.Context, mflID string, addedYears int, pricePerYear domain.Money, reason string) error {
	if addedYears <= 0 {
		return fmt.Errorf("state: AppendExtensionYears %q: addedYears must be positive, got %d", mflID, addedYears)
	}
	if pricePerYear <= 0 {
		return fmt.Errorf("state: AppendExtensionYears %q: price must be positive, got %s", mflID, pricePerYear)
	}
	tail, err := w.readContractTail(ctx, mflID)
	if err != nil {
		return err
	}
	firstNew := tail.maxPaidYear + 1
	if tail.ufaYear != firstNew {
		return fmt.Errorf("state: extension %q: UFA slot at %d is not the offseason after the last paid year %d (drift)",
			mflID, tail.ufaYear, tail.maxPaidYear)
	}
	lastNew := tail.maxPaidYear + addedYears
	if err := w.promoteUFAToPaid(ctx, mflID, firstNew, pricePerYear, reason); err != nil {
		return err
	}
	for year := firstNew + 1; year <= lastNew; year++ {
		if err := w.insertExtensionCell(ctx, mflID, tail.contractID, year, pricePerYear, yearStatusPaid, reason); err != nil {
			return err
		}
	}
	return w.insertExtensionCell(ctx, mflID, tail.contractID, lastNew+1, 0, yearStatusUFA, reason)
}

// contractTail is what an extension needs, read once before any write: the contract_id, the last
// PAID year and the UFA slot's year.
type contractTail struct {
	contractID  string
	maxPaidYear int
	ufaYear     int
}

// readContractTail locates the contract's end through the transaction. No PAID cell, or no UFA
// slot, is an error.
func (w *txWriter) readContractTail(ctx context.Context, mflID string) (contractTail, error) {
	rows, err := w.tx.QueryContext(ctx, `
SELECT league_year, year_status, contract_id FROM contract_years
WHERE league_id = ? AND mfl_id = ? ORDER BY league_year`, w.s.leagueID, mflID)
	if err != nil {
		return contractTail{}, fmt.Errorf("state: extension tail %q: read: %w", mflID, err)
	}
	defer func() { _ = rows.Close() }()
	var (
		t        contractTail
		havePaid bool
		ufaCount int
	)
	for rows.Next() {
		var year int
		var status, cid string
		if serr := rows.Scan(&year, &status, &cid); serr != nil {
			return contractTail{}, fmt.Errorf("state: extension tail %q: scan: %w", mflID, serr)
		}
		switch status {
		case yearStatusPaid:
			if !havePaid || year > t.maxPaidYear {
				t.maxPaidYear = year
			}
			t.contractID = cid
			havePaid = true
		case yearStatusUFA:
			t.ufaYear = year
			ufaCount++
		}
	}
	if ierr := rows.Err(); ierr != nil {
		return contractTail{}, fmt.Errorf("state: extension tail %q: iterate: %w", mflID, ierr)
	}
	if !havePaid {
		return contractTail{}, fmt.Errorf("state: extension %q: no PAID cell — nothing to extend", mflID)
	}
	// Exactly one UFA slot: a stray second one would pass the placement check below unseen.
	if ufaCount != 1 {
		return contractTail{}, fmt.Errorf("state: extension %q: expected exactly one UFA slot, found %d (drift)", mflID, ufaCount)
	}
	return t, nil
}

// promoteUFAToPaid turns the UFA slot at `year` into a PAID extension cell. Matching only UFA
// means a slot that isn't one fails requireOneRow instead of overwriting a paid cell.
func (w *txWriter) promoteUFAToPaid(ctx context.Context, mflID string, year int, price domain.Money, reason string) error {
	t := time.Now().UTC()
	now := t.Format(time.RFC3339)
	res, err := w.tx.ExecContext(ctx, `
UPDATE contract_years SET salary_cents = ?, year_status = ?, source = ?, last_updated = ?
WHERE league_id = ? AND mfl_id = ? AND league_year = ? AND year_status = ?`,
		price.Cents(), yearStatusPaid, SourceExtension, now,
		w.s.leagueID, mflID, year, yearStatusUFA)
	if err != nil {
		return fmt.Errorf("state: promote UFA %q/%d: %w", mflID, year, err)
	}
	if rerr := requireOneRow(res, mflID); rerr != nil {
		return rerr
	}
	return w.logCellChange(ctx, mflID, year, 0, price.Cents(), reason, t)
}

// insertExtensionCell inserts one new extension or UFA cell; an existing cell at that year fails
// the insert rather than being overwritten.
func (w *txWriter) insertExtensionCell(ctx context.Context, mflID, contractID string, year int, salary domain.Money, status, reason string) error {
	t := time.Now().UTC()
	now := t.Format(time.RFC3339)
	res, err := w.tx.ExecContext(ctx, `
INSERT INTO contract_years (league_id, mfl_id, league_year, salary_cents, year_status, contract_id, source, last_updated)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		w.s.leagueID, mflID, year, salary.Cents(), status, contractID, SourceExtension, now)
	if err != nil {
		return fmt.Errorf("state: insert extension cell %q/%d: %w", mflID, year, err)
	}
	if rerr := requireOneRow(res, mflID); rerr != nil {
		return rerr
	}
	return w.logCellChange(ctx, mflID, year, 0, salary.Cents(), reason, t)
}

// logCellChange appends one change row. The id carries the clock reading and the year, so two
// cells written in the same nanosecond still get distinct ids.
func (w *txWriter) logCellChange(ctx context.Context, mflID string, year int, oldCents, newCents int64, reason string, t time.Time) error {
	now := t.Format(time.RFC3339)
	id := fmt.Sprintf("cyc:%s:%s:%d:%d", w.s.leagueID, mflID, year, t.UnixNano())
	if _, err := w.tx.ExecContext(ctx, `
INSERT INTO contract_year_changes (id, league_id, mfl_id, league_year, old_cents, new_cents, reason, source, changed_at)
VALUES (?, ?, ?, ?, ?, ?, ?, 'op', ?)`,
		id, w.s.leagueID, mflID, year, oldCents, newCents, reason, now); err != nil {
		return fmt.Errorf("state: log cell change %q/%d: %w", mflID, year, err)
	}
	return nil
}
