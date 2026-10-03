package state

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// SignContract gives a free agent a new flat contract and rosters him. It clears his prior cells
// (each logged), so the contract is PAID years followed by exactly one UFA slot with no collision
// from old cells; inserts his roster and contract rows; and lays `years` PAID cells from the
// current season at `salary` plus the trailing UFA slot, all tagged `source`. The rule math is
// already resolved by the handler.
func (w *txWriter) SignContract(ctx context.Context, mflID, franchiseID string, salary domain.Money, years int, source, reason string) error {
	if years < 1 {
		return fmt.Errorf("state: SignContract %q: years must be >= 1, got %d", mflID, years)
	}
	if salary < 0 {
		return fmt.Errorf("state: SignContract %q: salary %s must not be negative", mflID, salary)
	}
	if w.s.exists(mflID) {
		return fmt.Errorf("state: SignContract %q: already on a roster", mflID)
	}
	season := w.s.season
	// The franchise must field a roster this season: franchises exist only through their players, so
	// a mistyped id would strand the signed player where no screen shows him. A franchise with no
	// players at all can't sign until it holds one, which is rare and fixed by a trade.
	if err := w.requireKnownFranchise(ctx, franchiseID, season); err != nil {
		return err
	}
	if err := w.clearPriorCells(ctx, mflID, reason); err != nil {
		return err
	}
	if err := w.insertSignedRosterRows(ctx, mflID, franchiseID, salary, years, season); err != nil {
		return err
	}
	contractID := fmt.Sprintf("ct:%s:%d:%s", w.s.leagueID, season, mflID)
	for year := season; year <= season+years-1; year++ {
		if err := w.layCell(ctx, mflID, contractID, year, salary, yearStatusPaid, source, reason); err != nil {
			return err
		}
	}
	return w.layCell(ctx, mflID, contractID, season+years, 0, yearStatusUFA, source, reason)
}

// requireKnownFranchise fails unless franchiseID fields a roster this season.
func (w *txWriter) requireKnownFranchise(ctx context.Context, franchiseID string, season int) error {
	var one int
	row := w.tx.QueryRowContext(ctx,
		`SELECT 1 FROM rosters WHERE league_id = ? AND franchise_id = ? AND season = ? LIMIT 1`,
		w.s.leagueID, franchiseID, season)
	switch err := row.Scan(&one); {
	case errors.Is(err, sql.ErrNoRows):
		return fmt.Errorf("state: SignContract: unknown franchise %q (no roster this season)", franchiseID)
	case err != nil:
		return fmt.Errorf("state: SignContract: check franchise %q: %w", franchiseID, err)
	}
	return nil
}

// clearPriorCells removes every existing cell for a player, logging each first. Cells are read and
// the cursor closed before any write.
func (w *txWriter) clearPriorCells(ctx context.Context, mflID, reason string) error {
	prior, err := w.readAllCells(ctx, mflID)
	if err != nil {
		return err
	}
	for _, c := range prior {
		clearReason := fmt.Sprintf("%s: prior contract closed on signing", reason)
		if lerr := w.logCellChange(ctx, mflID, c.year, c.cents, 0, clearReason, time.Now().UTC()); lerr != nil {
			return lerr
		}
	}
	if _, derr := w.tx.ExecContext(ctx,
		`DELETE FROM contract_years WHERE league_id = ? AND mfl_id = ?`, w.s.leagueID, mflID); derr != nil {
		return fmt.Errorf("state: SignContract %q: clear prior cells: %w", mflID, derr)
	}
	return nil
}

// priorCell is one existing cell, read before any write.
type priorCell struct {
	year  int
	cents int64
}

// readAllCells reads every cell for a player and closes the cursor before returning.
func (w *txWriter) readAllCells(ctx context.Context, mflID string) ([]priorCell, error) {
	rows, err := w.tx.QueryContext(ctx, `
SELECT league_year, salary_cents FROM contract_years
WHERE league_id = ? AND mfl_id = ?
ORDER BY league_year`, w.s.leagueID, mflID)
	if err != nil {
		return nil, fmt.Errorf("state: SignContract %q: read prior cells: %w", mflID, err)
	}
	defer func() { _ = rows.Close() }()
	var prior []priorCell
	for rows.Next() {
		var c priorCell
		if serr := rows.Scan(&c.year, &c.cents); serr != nil {
			return nil, fmt.Errorf("state: SignContract %q: scan prior cell: %w", mflID, serr)
		}
		prior = append(prior, c)
	}
	if ierr := rows.Err(); ierr != nil {
		return nil, fmt.Errorf("state: SignContract %q: iterate prior cells: %w", mflID, ierr)
	}
	return prior, nil
}

// insertSignedRosterRows creates the roster and contract rows for a signed player.
// expiration_year is the last paid year, which §8 and §12 read for remaining years; the base
// salary column is set only for display parity with the seed.
func (w *txWriter) insertSignedRosterRows(ctx context.Context, mflID, franchiseID string, salary domain.Money, years, season int) error {
	key := fmt.Sprintf("%s:%d:%s", w.s.leagueID, season, mflID)
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := w.tx.ExecContext(ctx,
		`INSERT INTO rosters (id, league_id, mfl_id, franchise_id, roster_status, season, as_of)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"r:"+key, w.s.leagueID, mflID, franchiseID, string(domain.RosterActive), season, now); err != nil {
		return fmt.Errorf("state: SignContract %q: insert roster: %w", mflID, err)
	}
	if _, err := w.tx.ExecContext(ctx,
		`INSERT INTO contracts (id, league_id, mfl_id, franchise_id, annual_salary_cents,
		   expiration_year, contract_status, is_restructured, is_tagged, season, last_updated)
		 VALUES (?, ?, ?, ?, ?, ?, ?, 0, 0, ?, ?)`,
		"c:"+key, w.s.leagueID, mflID, franchiseID, salary.Cents(), season+years-1,
		string(domain.CStatusUFA), season, now); err != nil {
		return fmt.Errorf("state: SignContract %q: insert contract: %w", mflID, err)
	}
	return nil
}

// layCell inserts one cell and logs its birth. The change id carries nanoseconds, so re-signing a
// player in a year he held before doesn't collide.
func (w *txWriter) layCell(ctx context.Context, mflID, contractID string, year int, salary domain.Money, status, source, reason string) error {
	t := time.Now().UTC()
	if _, err := w.tx.ExecContext(ctx,
		`INSERT INTO contract_years (league_id, mfl_id, league_year, salary_cents, year_status, contract_id, source, last_updated)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		w.s.leagueID, mflID, year, salary.Cents(), status, contractID, source, t.Format(time.RFC3339)); err != nil {
		return fmt.Errorf("state: SignContract %q/%d: insert cell: %w", mflID, year, err)
	}
	return w.logCellChange(ctx, mflID, year, 0, salary.Cents(), reason, t)
}

// ActiveBuyoutLockout reports a dead_cap_ledger row with `reason` for this season or later (§12:
// no bidding on a bought-out player until the next offseason). Nothing new is written: the
// buyout's own dead-cap row is the lockout, and it ages out as seasons pass. League-wide, per the
// rulebook.
func (w *txWriter) ActiveBuyoutLockout(ctx context.Context, mflID, reason string, season int) (bool, error) {
	var one int
	row := w.s.pools.Read().QueryRowContext(ctx, `
SELECT 1 FROM dead_cap_ledger
WHERE league_id = ? AND mfl_id = ? AND reason = ? AND league_year >= ?
LIMIT 1`, w.s.leagueID, mflID, reason, season)
	switch err := row.Scan(&one); {
	case err == sql.ErrNoRows:
		return false, nil
	case err != nil:
		return false, fmt.Errorf("state: ActiveBuyoutLockout %q: %w", mflID, err)
	}
	return true, nil
}
