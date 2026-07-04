package state

import (
	"context"
	"fmt"
)

// initLedgerSchema creates the per-year salary LEDGER and its append-only change log —
// the pivot from one annual salary per player (contracts.annual_salary_cents) to one cell
// per (player, league_year). It is ADDITIVE: Ship 1 creates and seeds these tables while
// nothing reads them yet, so the live single-salary cap path is untouched and main stays
// green (the cutover to reading cells is a later ship). The ledger is KING once live —
// every money figure (cap usage, dead cap, years remaining, highest remaining) is COMPUTED
// from these cells, never stored as competing truth.
//
// contract_years: one row per player-year. year_status is PAID (counts against the cap),
// UFA (the free-agency slot after the last paid year — a placeholder, $0), or VOID (a
// waived/relieved cell kept for history, not deleted). salary_cents is exact cents with a
// CHECK(>= 0); every figure written here is already snapped to $10k by the writing op
// (domain.RoundToNearest10k). contract_id ties every cell of one contract term together so
// "6 total years" = COUNT(PAID) and "no 2nd extension" is answerable from the cells without
// a separate is_extended flag. UNIQUE(league_id, mfl_id, league_year): one cell per
// player-year.
//
// contract_year_changes: the immutable audit trail — every deviation of a cell from its
// original seeded value logs a dated reason in the SAME tx that moves the cell. It is
// DOUBLE-immutable, the dead_cap_ledger idiom: no update/delete Go API AND BEFORE
// UPDATE/DELETE RAISE(ABORT) triggers, so a raw UPDATE/DELETE that bypasses the Go layer
// still aborts at the DB. This is the spine of the commissioner-override audit path
// ("if there is a reason the number differs, it gets a dated tag, human-verifiable").
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
