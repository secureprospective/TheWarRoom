package state

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// SignContract lays a brand-new flat contract for a free agent and rosters him — the free-agency
// SIGN write primitive (Free_Agency_Design). In the shared tx it: (1) CLEARS the player's prior
// contract_years cells (a signed free agent's previous contract is closed — clearing forward AND
// historical live cells guarantees the "contiguous PAID + exactly one trailing UFA" invariant and
// removes the boundary-year collision the panel flagged, since ReleasePlayer leaves old cells
// behind); each cleared cell is logged old→0 to the append-only contract_year_changes audit spine,
// so history survives even though the live cell is removed; (2) inserts the rosters + contracts
// parent rows (the player was off every roster); (3) lays `years` contiguous PAID cells at the
// current season..season+years-1, each at the flat `salary`, plus one trailing UFA slot the
// offseason after — the same fencepost the seed and the extension layer hold. Every new cell is
// tagged `source` ("signing"). The store owns the mechanics; the rule math (min-salary floor,
// 1..4 years, flat salary, eligibility, lockout) is the handler's, already resolved into the args.
// Fails loud on a non-positive years, a negative salary, or a player already on a roster.
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
	if err := w.clearPriorCells(ctx, mflID, reason); err != nil {
		return err
	}
	season := w.s.season
	if err := w.insertSignedRosterRows(ctx, mflID, franchiseID, salary, years, season); err != nil {
		return err
	}
	contractID := fmt.Sprintf("ct:%s:%d:%s", w.s.leagueID, season, mflID)
	for year := season; year <= season+years-1; year++ {
		if err := w.layCell(ctx, mflID, contractID, year, salary, yearStatusPaid, source, reason); err != nil {
			return err
		}
	}
	// The UFA placeholder slot the offseason after the last paid year ($0, not cap-counting).
	return w.layCell(ctx, mflID, contractID, season+years, 0, yearStatusUFA, source, reason)
}

// clearPriorCells removes every existing contract_years cell for a player (any status) inside the
// shared tx, logging each removed cell old→0 to the immutable change log first so the audit spine
// stays complete. It reads the cells into a slice and closes the cursor BEFORE any write (one tx =
// one SQLite connection). A player with no prior cells (never contracted) clears nothing.
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

// priorCell is one existing contract_years cell (its year + current cents), drained before any
// write so the read cursor is closed first (one tx = one SQLite connection).
type priorCell struct {
	year  int
	cents int64
}

// readAllCells drains every contract_years cell (any status) for a player and closes the cursor
// before returning, so the caller's clear/lay writes never overlap an open read.
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

// insertSignedRosterRows creates the rosters + contracts parent rows for a freshly signed player,
// mirroring the seed insert. roster_status is ROSTER (active); contract_status is UFA (the benign
// default — the contract resolves to a UFA at its trailing slot). expiration_year is the last paid
// year (season+years-1), the figure §8/§12 read for "remaining years". Money lives in the ledger
// cells; the base cents column carries the flat salary for display parity with the seed.
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
		   contract_years, expiration_year, contract_status,
		   is_restructured, is_tagged, season, last_updated)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0, 0, ?, ?)`,
		"c:"+key, w.s.leagueID, mflID, franchiseID, salary.Cents(), years, season+years-1,
		string(domain.CStatusUFA), season, now); err != nil {
		return fmt.Errorf("state: SignContract %q: insert contract: %w", mflID, err)
	}
	return nil
}

// layCell inserts one fresh contract_years cell and logs its birth (0→salary) to the immutable
// change log, in the shared tx — the write half of a signing. Unlike the seed's insertCell it uses
// a nanosecond-suffixed change id (so a re-sign in a year the player previously held does not
// collide on the change-log primary key) and records the caller's source tag.
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

// ActiveBuyoutLockout reports whether the player has an ACTIVE buyout lockout — a dead_cap_ledger
// row with the given reason whose league_year is >= season (§12 "cannot bid on a bought-out player
// until the following offseason"). It is DERIVED with zero new writes: a buyout already records its
// dead-cap row keyed to the buyout season, and the append-only ledger never purges it, so the row
// gates signing for its own season then ages out as the season advances. The reason is passed IN
// (the store stays agnostic to the deadcap package's audit string). Multiple buyouts → multiple
// rows; EXISTS with >= season catches the most recent live one. The lockout is GLOBAL (any
// franchise), matching the rulebook.
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
