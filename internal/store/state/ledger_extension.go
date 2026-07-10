package state

import (
	"context"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// SourceExtension tags every contract_years cell a §10 extension writes or promotes, so a
// later extension can detect "already extended off a prior extension" straight from the cells
// (the ledger_schema design intent — no separate is_extended flag). Seeded cells stay "seed";
// a restructure/tag leaves a cell's source column unchanged (they only touch salary/status),
// so ONLY genuine extension years ever carry this tag. Exported so the §10 handler (which
// reads a player's PaidCells) can test LedgerCell.Source against it without duplicating the
// string.
const SourceExtension = "extension"

// PaidCells reads a player's PAID contract-year cells (year, salary, source) THROUGH the
// shared tx, ordered by year. VOID/UFA cells are omitted (they carry no cap). See the
// LedgerWriter interface doc for how the §10 handler derives its rule facts from these.
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

// AppendExtensionYears appends `addedYears` new PAID contract-year cells at pricePerYear — the
// §10 extension write primitive. It promotes the existing UFA slot (the offseason after the
// last paid year) to the first PAID extension year, inserts any further added years as new
// PAID cells, and creates a fresh UFA slot the offseason after the new last paid year, so the
// ledger invariant (contiguous PAID cells followed by exactly one UFA slot) still holds. Every
// new/promoted cell is tagged source "extension" and logged to the immutable change log. Fails
// loud on a non-positive addedYears/price, a player with no PAID cell, or a missing/misplaced
// UFA slot (drift). The §10 rule math is the handler's — this is pure mechanics.
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
	// 1. Promote the existing UFA slot into the first PAID extension year (no new row — the
	//    slot was already reserved as the offseason placeholder; it now becomes a paid year).
	if err := w.promoteUFAToPaid(ctx, mflID, firstNew, pricePerYear, reason); err != nil {
		return err
	}
	// 2. Insert the remaining extension years as new PAID cells.
	for year := firstNew + 1; year <= lastNew; year++ {
		if err := w.insertExtensionCell(ctx, mflID, tail.contractID, year, pricePerYear, yearStatusPaid, reason); err != nil {
			return err
		}
	}
	// 3. Create a fresh UFA slot the offseason after the new last paid year.
	return w.insertExtensionCell(ctx, mflID, tail.contractID, lastNew+1, 0, yearStatusUFA, reason)
}

// contractTail is the end-of-contract shape AppendExtensionYears needs: the contract_id every
// new cell inherits, the last PAID year (extension years start after it), and the UFA slot's
// year (promoted to the first extension year). Read once, before any write.
type contractTail struct {
	contractID  string
	maxPaidYear int
	ufaYear     int
}

// readContractTail reads a player's cell layout THROUGH the tx and locates the contract tail.
// Fails loud if the player has no PAID cell (nothing to extend) or no UFA slot (drift — the
// seed and every op keep exactly one UFA slot after the last paid year).
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
	// Exactly one UFA slot must exist (GLM m1): the "contiguous PAID + one trailing UFA"
	// invariant. A stray second UFA (e.g. at a year ≤ maxPaidYear) would otherwise pass the
	// placement check below unseen, since ufaYear records only the last one scanned.
	if ufaCount != 1 {
		return contractTail{}, fmt.Errorf("state: extension %q: expected exactly one UFA slot, found %d (drift)", mflID, ufaCount)
	}
	return t, nil
}

// promoteUFAToPaid flips the reserved UFA slot at `year` to a PAID extension cell at `price`,
// tags it source "extension", and logs the 0→price change — all in the shared tx. The WHERE
// guards year_status = UFA so a slot that is not actually the UFA placeholder (drift) matches
// 0 rows and requireOneRow fails loud rather than silently overwriting a paid cell.
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

// insertExtensionCell inserts one brand-new contract-year cell (a further PAID extension year
// or the fresh trailing UFA slot) tagged source "extension" and logs its 0→salary birth, in
// the shared tx. A pre-existing cell at the year makes the INSERT hit the (league, player,
// year) PK and error — fail loud, never a silent overwrite.
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

// logCellChange appends one immutable old→new change-log row for a cell mutation, in the
// shared tx. The change-log ID embeds the same clock reading as changed_at (one time.Now) plus
// the year, so IDs never collide even when two cells are written within the same nanosecond
// (the year differs). source is "op" — the runtime-op tag, matching writeCell/voidCell.
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
