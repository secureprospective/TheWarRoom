package state

import (
	"context"
	"database/sql"
	"fmt"
)

// initSchema creates the rosters, contracts, and dead_cap_ledger tables if absent
// (Backend_Architecture §8). B3c OWNS rosters + contracts; B7b added dead_cap_ledger
// (the §8 waiver-cut charge surface); B7c added transaction_counts (the durable
// per-franchise-per-season op counter that enforces limits like §11 "one restructure per
// team per year"). waiver_order / league_state stay out of scope.
//
// dead_cap_ledger is APPEND-ONLY, enforced BOTH in Go (no update/delete API — WriteTx
// only exposes AddDeadCap) AND at the DB via BEFORE UPDATE/DELETE RAISE(ABORT) triggers
// (the B6 double-immutability idiom): a raw UPDATE/DELETE that bypasses the Go API still
// aborts. Rows are keyed to an ABSOLUTE league_year (never a relative slot) with a UNIQUE
// (league_id, franchise_id, league_year, mfl_id) — one charge per released player per year.
// dead_cap_cents is exact cents (OQ-014) with a CHECK(>= 0): rollover is off, so cap debt
// never goes negative (§1).
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
	// The forward-only migration runner owns the two money migrations below (v1/v2): it
	// tracks them in schema_migrations, reconciles pre-marker DBs on data predicates, and
	// takes a VACUUM INTO backup before any migration that does real work (D-V6, Tier 2).
	return s.runMigrations(ctx)
}

// baseSchemaDDL creates the rosters, contracts, dead_cap_ledger, and transaction_counts
// tables (B3c/B7b/B7c). The per-year ledger tables are created separately by
// initLedgerSchema (own file, store-no-siblings + the 400-line cap).
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
	contract_years  INTEGER NOT NULL DEFAULT 0,
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

// migrateMoneyCents is the additive REAL→int64-cents migration (OQ-014). On a fresh DB
// the cents column already exists from the DDL and this is a no-op. On an existing pre-cents
// DB (only the legacy REAL columns), it adds annual_salary_cents, backfills it exactly from
// annual_salary ($millions × 1e8), and verifies zero mismatches — failing loud rather than
// silently serving under-converted money. Only the BASE salary is carried: adjusted_salary
// is dead as of the Ship 3 read-flip (cap now derives from the ledger cells), so it is not
// converted here — dropLegacyMoneyColumns removes it next.
func (s *Store) migrateMoneyCents(ctx context.Context) error {
	have, err := s.columnExists(ctx, "contracts", "annual_salary_cents")
	if err != nil {
		return err
	}
	if have {
		return nil // fresh DB (DDL created it) or an already-migrated DB
	}
	// ATOMIC (z.ai Ship-4 M1): the ADD COLUMN + backfill + verify run in ONE transaction so a
	// crash between the add and the backfill rolls the whole thing back — leaving the DB
	// pre-cents, which re-runs cleanly next startup. Ship 4 drops the legacy REAL columns
	// (dropLegacyMoneyColumns) that used to be the recovery source, so a half-migrated DB (cents
	// column present but unbackfilled) would be UNRECOVERABLE: migrateMoneyCents would early-exit
	// on the present column and dropLegacyMoneyColumns would then destroy annual_salary, silently
	// zeroing every base salary. The transaction closes that window. The verify reads THIS tx (the
	// added column is not visible on the separate read pool until commit). ADD COLUMN cannot carry
	// a CHECK portably, so non-negativity is enforced by the Go write layer on migrated DBs; the
	// fresh DDL keeps the CHECK. Backfill rounds to the nearest cent (REAL millions × 1e8).
	tx, err := s.pools.Write().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("state: money-cents migration begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // no-op after a successful Commit
	if _, err := tx.ExecContext(ctx, `
ALTER TABLE contracts ADD COLUMN annual_salary_cents INTEGER NOT NULL DEFAULT 0;
UPDATE contracts SET annual_salary_cents = CAST(ROUND(annual_salary * 100000000) AS INTEGER);`); err != nil {
		return fmt.Errorf("state: money-cents migration: %w", err)
	}
	// Verify: every backfilled cent must round-trip to within half a cent of the REAL
	// value it came from (1 cent = 1e-8 millions). Any mismatch aborts startup (tx rolls back).
	// NOTE: this references annual_salary, which dropLegacyMoneyColumns removes right after — it
	// is safe only because the have-guard above runs verify solely on a pre-cents DB where the
	// column still exists. Do not reorder the drop before this, or call verify standalone.
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

// dropLegacyMoneyColumns removes the frozen dead money columns now that the Ship 3 read-flip
// made the ledger cells the sole cap source of truth: annual_salary / adjusted_salary (the
// pre-cents REAL columns) and adjusted_salary_cents (dead since the read-flip stopped writing
// it). annual_salary_cents STAYS — it is the base §9/§11 rule salary that load() reads. Runs
// AFTER migrateMoneyCents so a pre-cents DB is backfilled into annual_salary_cents before its
// REAL source column is dropped. Idempotent across every DB vintage: each column is dropped
// only if present, so a fresh Ship-4 DB (never had them) and an already-dropped DB both no-op.
// All three drops run in ONE transaction (GLM-5.2 Ship-4 review): the per-column present-guard
// already makes a partial drop recover cleanly next startup, but a single tx closes that window
// entirely at zero cost — either all dead columns are gone or none are, never a torn schema.
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
		return nil // fresh Ship-4 DB or already dropped — nothing to do
	}
	tx, err := s.pools.Write().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("state: drop legacy columns begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // no-op after a successful Commit
	for _, col := range present {
		// col is a fixed literal from the list above, never caller input — no injection surface
		// (same pattern as columnExists's pragma call). ALTER TABLE cannot bind identifiers.
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

// columnExists reports whether a table has a given column, via pragma table_info.
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
