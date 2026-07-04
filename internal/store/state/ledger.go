package state

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// Year-status values for a contract_years cell. PAID counts against the cap; UFA is the
// free-agency placeholder slot the offseason after the last paid year ($0); VOID is a
// waived/relieved cell kept for history (never deleted — the ledger is king, history is
// preserved). Ship 1 seeds only PAID + UFA; VOID lands when the waiver op is refit.
// VOID (a waived/relieved cell kept for history) lands with the Ship-2 waiver refit; only
// PAID + UFA are written at seed.
const (
	yearStatusPaid = "PAID"
	yearStatusUFA  = "UFA"
)

// seedLedgerPlayer flat-fills one player's per-year salary cells from the single seed
// figures (annual salary + final contract year), inside the SAME transaction as the
// contracts seed so the ledger and the legacy row are always born together. It is the
// fencepost LOCKED with Christopher: the stated contract year is the LAST PAID year, and
// the offseason after it is a UFA slot — e.g. "$2M, expires 2028" seeded in season 2026
// yields PAID cells for 2026/2027/2028 (each $2M) and a UFA cell for 2029.
//
// Flat math (rulebook §6): every PAID cell gets the SAME salary — bids/seeds never
// back/frontload; only a later restructure/extension varies cells. The salary is snapped
// to $10k (§1, universal rule) so a cell is never born off-grid. A record with no/expired
// contract year still gets at least the current season as one PAID cell (max(year, season))
// so the ledger always covers the live year — never fabricating extra years.
//
// contract_id ties every cell of this initial term together (deterministic from the seed
// key — no external uuid dependency), so once live "6 total years" = COUNT(PAID) for a
// contract_id and "already extended" is answerable from the cells. Every seeded cell logs
// an immutable INIT row in contract_year_changes (the dated-reason audit spine).
func seedLedgerPlayer(ctx context.Context, tx *sql.Tx, leagueID string, season int, now string, p domain.PlayerRecord) error {
	mflID := p.MFLID.String()
	contractID := "ct:" + leagueID + ":" + fmt.Sprint(season) + ":" + mflID
	salary := domain.RoundToNearest10k(p.Salary)

	lastPaid := p.ContractYear
	if lastPaid < season {
		lastPaid = season // absent/expired contract year → at least the live season is PAID
	}
	for year := season; year <= lastPaid; year++ {
		if err := insertCell(ctx, tx, leagueID, mflID, contractID, year, salary, yearStatusPaid, now); err != nil {
			return err
		}
	}
	// The UFA placeholder slot the offseason after the last paid year ($0, not cap-counting).
	return insertCell(ctx, tx, leagueID, mflID, contractID, lastPaid+1, 0, yearStatusUFA, now)
}

// insertCell writes one contract_years cell and its INIT change-log row atomically within
// the seed tx. The change row records the cell's birth (old 0 → new salary) with a dated
// reason, the first entry in the human-verifiable audit trail for that cell.
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
