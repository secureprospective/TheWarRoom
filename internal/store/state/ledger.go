package state

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// Cell statuses. PAID counts against the cap; UFA is the $0 free-agency slot after the last
// paid year; VOID is a waived or relieved cell kept for history.
const (
	yearStatusPaid = "PAID"
	yearStatusUFA  = "UFA"
	yearStatusVoid = "VOID"
)

// seedLedgerPlayer lays one player's per-year cells from the seed figures, in the same transaction
// as his contract row. The stated contract year is the last PAID year and the offseason after it is
// the UFA slot: "$2M, expires 2028" seeded in 2026 gives PAID 2026-2028 at $2M each and UFA 2029.
// Cells are flat (§6) and snapped to $10k (§1). An expired or missing year still gets the current
// season as one PAID cell. contract_id groups the term's cells; each cell logs an INIT change row.
func seedLedgerPlayer(ctx context.Context, tx *sql.Tx, leagueID string, season int, now string, p domain.PlayerRecord) error {
	mflID := p.MFLID.String()
	contractID := "ct:" + leagueID + ":" + fmt.Sprint(season) + ":" + mflID
	salary := domain.RoundToNearest10k(p.Salary)

	lastPaid := p.ContractYear
	if lastPaid < season {
		lastPaid = season // expired or missing: at least the current season
	}
	for year := season; year <= lastPaid; year++ {
		if err := insertCell(ctx, tx, leagueID, mflID, contractID, year, salary, yearStatusPaid, now); err != nil {
			return err
		}
	}
	return insertCell(ctx, tx, leagueID, mflID, contractID, lastPaid+1, 0, yearStatusUFA, now)
}

// LedgerCells returns a player's committed PAID cells (year to salary). UFA and VOID cells carry
// no cap and are omitted.
func (s *Store) LedgerCells(ctx context.Context, mflID string) (map[int]domain.Money, error) {
	rows, err := s.pools.Read().QueryContext(ctx, `
SELECT league_year, salary_cents FROM contract_years
WHERE league_id = ? AND mfl_id = ? AND year_status = ?`, s.leagueID, mflID, yearStatusPaid)
	if err != nil {
		return nil, fmt.Errorf("state: ledger cells %q: %w", mflID, err)
	}
	defer func() { _ = rows.Close() }()
	out := map[int]domain.Money{}
	for rows.Next() {
		var year int
		var cents int64
		if err := rows.Scan(&year, &cents); err != nil {
			return nil, fmt.Errorf("state: ledger cells scan %q: %w", mflID, err)
		}
		out[year] = domain.Money(cents)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("state: ledger cells iterate %q: %w", mflID, err)
	}
	return out, nil
}

// insertCell writes one cell and its INIT change row in the seed transaction.
func insertCell(ctx context.Context, tx *sql.Tx, leagueID, mflID, contractID string, year int, salary domain.Money, status, now string) error {
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO contract_years (league_id, mfl_id, league_year, salary_cents, year_status, contract_id, source, last_updated)
		 VALUES (?, ?, ?, ?, ?, ?, 'seed', ?)`,
		leagueID, mflID, year, salary.Cents(), status, contractID, now); err != nil {
		return fmt.Errorf("state: seed ledger cell %q/%d: %w", mflID, year, err)
	}
	changeID := fmt.Sprintf("cyc:%s:%s:%d", leagueID, mflID, year)
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO contract_year_changes (id, league_id, mfl_id, league_year, old_cents, new_cents, reason, source, changed_at)
		 VALUES (?, ?, ?, ?, 0, ?, 'seed: flat-fill from MFL annual salary + expiration', 'seed', ?)`,
		changeID, leagueID, mflID, year, salary.Cents(), now); err != nil {
		return fmt.Errorf("state: seed ledger change %q/%d: %w", mflID, year, err)
	}
	return nil
}

// initLedgerSchema creates the per-year salary ledger and its change log. The cells are the source
// of truth: cap usage, dead cap, years remaining and the top-paid year are all computed from them.
//
// contract_years: one row per player-year (UNIQUE league, player, year). year_status is PAID, UFA
// or VOID; salary_cents is exact, non-negative, snapped to $10k by the writer. contract_id groups a
// term, so "6 total years" and "no second extension" are answerable from the cells.
//
// contract_year_changes: every change to a cell logs a dated reason in the same transaction.
// Triggers abort update and delete ("if there is a reason the number differs, it gets a dated tag,
// human-verifiable").
func (s *Store) initLedgerSchema(ctx context.Context) error {
	const ddl = `
CREATE TABLE IF NOT EXISTS contract_years (
	league_id    TEXT NOT NULL,
	mfl_id       TEXT NOT NULL,
	league_year  INTEGER NOT NULL,
	salary_cents INTEGER NOT NULL DEFAULT 0 CHECK (salary_cents >= 0),
	year_status  TEXT NOT NULL,
	contract_id  TEXT NOT NULL,
	source       TEXT NOT NULL,
	last_updated TEXT NOT NULL,
	PRIMARY KEY (league_id, mfl_id, league_year)
);
CREATE TABLE IF NOT EXISTS contract_year_changes (
	id           TEXT PRIMARY KEY,
	league_id    TEXT NOT NULL,
	mfl_id       TEXT NOT NULL,
	league_year  INTEGER NOT NULL,
	old_cents    INTEGER NOT NULL,
	new_cents    INTEGER NOT NULL CHECK (new_cents >= 0),
	reason       TEXT NOT NULL,
	source       TEXT NOT NULL,
	changed_at   TEXT NOT NULL
);
CREATE TRIGGER IF NOT EXISTS contract_year_changes_no_update
BEFORE UPDATE ON contract_year_changes
BEGIN SELECT RAISE(ABORT, 'contract_year_changes is append-only'); END;
CREATE TRIGGER IF NOT EXISTS contract_year_changes_no_delete
BEFORE DELETE ON contract_year_changes
BEGIN SELECT RAISE(ABORT, 'contract_year_changes is append-only'); END;`
	if _, err := s.pools.Write().ExecContext(ctx, ddl); err != nil {
		return fmt.Errorf("state: init ledger schema: %w", err)
	}
	return nil
}

// MoveCellMoney moves amount between two years for one player, conserving the total (§11
// restructure). Both deltas are logged.
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

// SetCell sets one PAID cell to an absolute value (§9 franchise tag) and logs the change. Fails if
// the cell is missing or the value negative.
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

// VoidCells flips every PAID cell to VOID (§8 waiver cut): the cap contribution goes to 0 and
// the history stays, each void logged. No PAID cell at all is drift. Cells are read and the cursor
// closed before any write, because the transaction shares one connection.
func (w *txWriter) VoidCells(ctx context.Context, mflID string, reason string) error {
	cells, err := w.readPaidCells(ctx, mflID)
	if err != nil {
		return err
	}
	if len(cells) == 0 {
		return fmt.Errorf("state: VoidCells %q: no PAID cell to void", mflID)
	}
	for _, c := range cells {
		if verr := w.voidCell(ctx, mflID, c.year, c.cents, reason); verr != nil {
			return verr
		}
	}
	return nil
}

// paidCell is one PAID cell, collected before the void writes.
type paidCell struct {
	year  int
	cents int64
}

// readPaidCells reads every PAID cell and closes the cursor before returning.
func (w *txWriter) readPaidCells(ctx context.Context, mflID string) ([]paidCell, error) {
	rows, err := w.tx.QueryContext(ctx, `
SELECT league_year, salary_cents FROM contract_years
WHERE league_id = ? AND mfl_id = ? AND year_status = ?
ORDER BY league_year`, w.s.leagueID, mflID, yearStatusPaid)
	if err != nil {
		return nil, fmt.Errorf("state: VoidCells %q: read paid cells: %w", mflID, err)
	}
	defer func() { _ = rows.Close() }()
	var cells []paidCell
	for rows.Next() {
		var c paidCell
		if serr := rows.Scan(&c.year, &c.cents); serr != nil {
			return nil, fmt.Errorf("state: VoidCells %q: scan: %w", mflID, serr)
		}
		cells = append(cells, c)
	}
	if ierr := rows.Err(); ierr != nil {
		return nil, fmt.Errorf("state: VoidCells %q: iterate: %w", mflID, ierr)
	}
	return cells, nil
}

// voidCell flips one cell to VOID at $0 and logs the change; oldCents is already read.
func (w *txWriter) voidCell(ctx context.Context, mflID string, year int, oldCents int64, reason string) error {
	t := time.Now().UTC()
	now := t.Format(time.RFC3339)
	// Match only PAID at write time: if the cell changed since the read, requireOneRow fails.
	res, err := w.tx.ExecContext(ctx, `
UPDATE contract_years SET salary_cents = 0, year_status = ?, last_updated = ?
WHERE league_id = ? AND mfl_id = ? AND league_year = ? AND year_status = ?`,
		yearStatusVoid, now, w.s.leagueID, mflID, year, yearStatusPaid)
	if err != nil {
		return fmt.Errorf("state: voidCell %q/%d write: %w", mflID, year, err)
	}
	if rerr := requireOneRow(res, mflID); rerr != nil {
		return rerr
	}
	// One clock reading for both the id and changed_at, so they never disagree.
	id := fmt.Sprintf("cyc:%s:%s:%d:%d", w.s.leagueID, mflID, year, t.UnixNano())
	if _, err := w.tx.ExecContext(ctx, `
INSERT INTO contract_year_changes (id, league_id, mfl_id, league_year, old_cents, new_cents, reason, source, changed_at)
VALUES (?, ?, ?, ?, ?, 0, ?, 'op', ?)`,
		id, w.s.leagueID, mflID, year, oldCents, reason, now); err != nil {
		return fmt.Errorf("state: voidCell %q/%d log: %w", mflID, year, err)
	}
	return nil
}

// adjustCell applies a signed delta to one PAID cell, reading through the transaction so earlier
// writes in it are visible. Fails if missing or it would go negative.
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

// readCell reads one cell through the transaction; a missing cell is an error.
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

// writeCell sets one cell and appends the change row: the shared tail of SetCell and adjustCell.
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
