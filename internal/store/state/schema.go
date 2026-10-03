package state

import (
	"context"
	"database/sql"
	"fmt"
)

// initSchema creates the base tables, then each feature's, then runs the migrations.
// dead_cap_ledger is append-only (no Go update API, and triggers abort update and delete), keyed
// to an absolute league year, one charge per released player per year, and never negative (§1:
// no cap rollover).
func (s *Store) initSchema(ctx context.Context) error {
	if _, err := s.pools.Write().ExecContext(ctx, baseSchemaDDL); err != nil {
		return fmt.Errorf("state: init schema: %w", err)
	}
	if err := s.initLedgerSchema(ctx); err != nil {
		return err
	}
	if err := s.initSeasonPhaseSchema(ctx); err != nil {
		return err
	}
	if err := s.initCapReliefSchema(ctx); err != nil {
		return err
	}
	if err := s.initPlayerStatusSchema(ctx); err != nil {
		return err
	}
	if err := s.initCalendarSchema(ctx); err != nil {
		return err
	}
	if err := s.initStandingsCacheSchema(ctx); err != nil {
		return err
	}
	if err := s.initLeagueScheduleCacheSchema(ctx); err != nil {
		return err
	}
	if err := s.initTradeNotesSchema(ctx); err != nil {
		return err
	}
	if err := s.initCorrectionSchema(ctx); err != nil {
		return err
	}
	return s.runMigrations(ctx)
}

// baseSchemaDDL creates rosters, contracts, dead_cap_ledger and transaction_counts.
const baseSchemaDDL = `
CREATE TABLE IF NOT EXISTS rosters (
	id            TEXT PRIMARY KEY,
	league_id     TEXT NOT NULL,
	mfl_id        TEXT NOT NULL,
	franchise_id  TEXT NOT NULL,
	roster_status TEXT NOT NULL,
	season        INTEGER NOT NULL,
	as_of         TEXT NOT NULL,
	UNIQUE (league_id, season, mfl_id)
);
CREATE TABLE IF NOT EXISTS contracts (
	id                    TEXT PRIMARY KEY,
	league_id             TEXT NOT NULL,
	mfl_id                TEXT NOT NULL,
	franchise_id          TEXT NOT NULL,
	annual_salary_cents   INTEGER NOT NULL DEFAULT 0 CHECK (annual_salary_cents >= 0),
	expiration_year INTEGER NOT NULL DEFAULT 0,
	contract_status TEXT NOT NULL DEFAULT '',
	is_restructured INTEGER NOT NULL DEFAULT 0,
	is_tagged       INTEGER NOT NULL DEFAULT 0,
	season          INTEGER NOT NULL,
	last_updated    TEXT NOT NULL,
	UNIQUE (league_id, season, mfl_id)
);
CREATE TABLE IF NOT EXISTS dead_cap_ledger (
	id             TEXT PRIMARY KEY,
	league_id      TEXT NOT NULL,
	franchise_id   TEXT NOT NULL,
	league_year    INTEGER NOT NULL,
	mfl_id         TEXT NOT NULL,
	dead_cap_cents INTEGER NOT NULL CHECK (dead_cap_cents >= 0),
	reason         TEXT NOT NULL,
	created_at     TEXT NOT NULL,
	UNIQUE (league_id, franchise_id, league_year, mfl_id)
);
CREATE TRIGGER IF NOT EXISTS dead_cap_ledger_no_update
BEFORE UPDATE ON dead_cap_ledger
BEGIN SELECT RAISE(ABORT, 'dead_cap_ledger is append-only'); END;
CREATE TRIGGER IF NOT EXISTS dead_cap_ledger_no_delete
BEFORE DELETE ON dead_cap_ledger
BEGIN SELECT RAISE(ABORT, 'dead_cap_ledger is append-only'); END;
CREATE TABLE IF NOT EXISTS transaction_counts (
	league_id    TEXT NOT NULL,
	franchise_id TEXT NOT NULL,
	season       INTEGER NOT NULL,
	op_kind      TEXT NOT NULL,
	count        INTEGER NOT NULL DEFAULT 0 CHECK (count >= 0),
	PRIMARY KEY (league_id, franchise_id, season, op_kind)
);`

// migrateMoneyCents (v1) converts an old DB's REAL salaries to exact cents and verifies every
// value, failing loudly rather than serving under-converted money. Only the base salary is carried;
// the cap figure now comes from the ledger cells.
func (s *Store) migrateMoneyCents(ctx context.Context) error {
	have, err := s.columnExists(ctx, "contracts", "annual_salary_cents")
	if err != nil {
		return err
	}
	if have {
		return nil // fresh or already migrated
	}
	// One transaction for add, backfill and verify. v2 drops the REAL column that was the recovery
	// source, so a crash between the add and the backfill would otherwise zero every salary for good.
	// The verify reads this transaction, because the new column isn't visible elsewhere until commit.
	tx, err := s.pools.Write().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("state: money-cents migration begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `
ALTER TABLE contracts ADD COLUMN annual_salary_cents INTEGER NOT NULL DEFAULT 0;
UPDATE contracts SET annual_salary_cents = CAST(ROUND(annual_salary * 100000000) AS INTEGER);`); err != nil {
		return fmt.Errorf("state: money-cents migration: %w", err)
	}
	// Every cent must round-trip within half a cent of its REAL source. This reads annual_salary,
	// which v2 drops next: never run it after the drop.
	var bad int
	if err := tx.QueryRowContext(ctx, `
SELECT COUNT(1) FROM contracts
WHERE ABS(annual_salary_cents / 100000000.0 - annual_salary) > 0.000000005`).Scan(&bad); err != nil {
		return fmt.Errorf("state: money-cents migration verify: %w", err)
	}
	if bad != 0 {
		return fmt.Errorf("state: money-cents migration left %d row(s) mismatched — aborting", bad)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("state: money-cents migration commit: %w", err)
	}
	return nil
}

// dropLegacyMoneyColumns (v2) removes the dead money columns. annual_salary_cents stays: it is the
// base salary load reads. Runs after v1, and drops all three in one transaction, each only if
// present, so every DB vintage ends the same.
func (s *Store) dropLegacyMoneyColumns(ctx context.Context) error {
	present := make([]string, 0, 3)
	for _, col := range []string{"annual_salary", "adjusted_salary", "adjusted_salary_cents"} {
		have, err := s.columnExists(ctx, "contracts", col)
		if err != nil {
			return err
		}
		if have {
			present = append(present, col)
		}
	}
	if len(present) == 0 {
		return nil // already dropped
	}
	tx, err := s.pools.Write().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("state: drop legacy columns begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	for _, col := range present {
		// col comes from the fixed list above; ALTER TABLE cannot bind identifiers.
		if _, err := tx.ExecContext(ctx,
			fmt.Sprintf("ALTER TABLE contracts DROP COLUMN %s", col)); err != nil {
			return fmt.Errorf("state: drop legacy column %s: %w", col, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("state: drop legacy columns commit: %w", err)
	}
	return nil
}

// dropContractYearsColumn (v3) drops contracts.contract_years. Only SignContract ever wrote it and
// nothing read it: expiration_year and the contract_years ledger cells carry the term.
func (s *Store) dropContractYearsColumn(ctx context.Context) error {
	have, err := s.columnExists(ctx, "contracts", "contract_years")
	if err != nil || !have {
		return err
	}
	if _, err := s.pools.Write().ExecContext(ctx, `ALTER TABLE contracts DROP COLUMN contract_years`); err != nil {
		return fmt.Errorf("state: drop contracts.contract_years: %w", err)
	}
	return nil
}

// contractYearsColumnDropped reports whether v3 is already in place (always, on a fresh DB).
func (s *Store) contractYearsColumnDropped(ctx context.Context) (bool, error) {
	have, err := s.columnExists(ctx, "contracts", "contract_years")
	return !have, err
}

// columnExists reports whether a table has a column.
func (s *Store) columnExists(ctx context.Context, table, column string) (bool, error) {
	rows, err := s.pools.Read().QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false, fmt.Errorf("state: table_info(%s): %w", table, err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var cid, notnull, pk int
		var name, ctype string
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return false, fmt.Errorf("state: table_info scan: %w", err)
		}
		if name == column {
			return true, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("state: table_info(%s) rows: %w", table, err)
	}
	return false, nil
}
