package state

import (
	"context"
	"database/sql"
	"fmt"
)

// initSchema creates the rosters, contracts, and dead_cap_ledger tables if absent
// (Backend_Architecture §8). B3c OWNS rosters + contracts; B7b added dead_cap_ledger
// (the §8 waiver-cut charge surface). waiver_order / league_state stay out of scope.
//
// dead_cap_ledger is APPEND-ONLY, enforced BOTH in Go (no update/delete API — WriteTx
// only exposes AddDeadCap) AND at the DB via BEFORE UPDATE/DELETE RAISE(ABORT) triggers
// (the B6 double-immutability idiom): a raw UPDATE/DELETE that bypasses the Go API still
// aborts. Rows are keyed to an ABSOLUTE league_year (never a relative slot) with a UNIQUE
// (league_id, franchise_id, league_year, mfl_id) — one charge per released player per year.
// dead_cap_cents is exact cents (OQ-014) with a CHECK(>= 0): rollover is off, so cap debt
// never goes negative (§1).
func (s *Store) initSchema(ctx context.Context) error {
	const ddl = `
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
	annual_salary         REAL NOT NULL DEFAULT 0,
	adjusted_salary       REAL NOT NULL DEFAULT 0,
	annual_salary_cents   INTEGER NOT NULL DEFAULT 0 CHECK (annual_salary_cents >= 0),
	adjusted_salary_cents INTEGER NOT NULL DEFAULT 0 CHECK (adjusted_salary_cents >= 0),
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
BEGIN SELECT RAISE(ABORT, 'dead_cap_ledger is append-only'); END;`
	if _, err := s.pools.Write().ExecContext(ctx, ddl); err != nil {
		return fmt.Errorf("state: init schema: %w", err)
	}
	return s.migrateMoneyCents(ctx)
}

// migrateMoneyCents is the additive REAL→int64-cents migration (OQ-014). On a fresh DB
// the cents columns already exist from the DDL and this is a no-op. On an existing DB
// with only the legacy REAL columns, it adds the cents columns, backfills them exactly
// from the REAL values ($millions × 1e8), and verifies zero mismatches — failing loud
// rather than silently serving under-converted money. The legacy REAL columns are kept
// (frozen, unread) one release for rollback safety, then dropped in B7c.
func (s *Store) migrateMoneyCents(ctx context.Context) error {
	have, err := s.columnExists(ctx, "contracts", "annual_salary_cents")
	if err != nil {
		return err
	}
	if have {
		return nil // fresh DB (DDL created them) or an already-migrated DB
	}
	// Existing DB: ADD COLUMN cannot carry a CHECK here portably, so non-negativity is
	// enforced by the Go write layer (ApplyContract) on migrated DBs; the fresh DDL keeps
	// the CHECK. Backfill rounds to the nearest cent (REAL millions × 1e8).
	const migrate = `
ALTER TABLE contracts ADD COLUMN annual_salary_cents INTEGER NOT NULL DEFAULT 0;
ALTER TABLE contracts ADD COLUMN adjusted_salary_cents INTEGER NOT NULL DEFAULT 0;
UPDATE contracts SET
	annual_salary_cents   = CAST(ROUND(annual_salary   * 100000000) AS INTEGER),
	adjusted_salary_cents = CAST(ROUND(adjusted_salary * 100000000) AS INTEGER);`
	if _, err := s.pools.Write().ExecContext(ctx, migrate); err != nil {
		return fmt.Errorf("state: money-cents migration: %w", err)
	}
	// Verify: every backfilled cent must round-trip to within half a cent of the REAL
	// value it came from (1 cent = 1e-8 millions). Any mismatch aborts startup.
	var bad int
	row := s.pools.Read().QueryRowContext(ctx, `
SELECT COUNT(1) FROM contracts
WHERE ABS(annual_salary_cents   / 100000000.0 - annual_salary)   > 0.000000005
   OR ABS(adjusted_salary_cents / 100000000.0 - adjusted_salary) > 0.000000005`)
	if err := row.Scan(&bad); err != nil {
		return fmt.Errorf("state: money-cents migration verify: %w", err)
	}
	if bad != 0 {
		return fmt.Errorf("state: money-cents migration left %d row(s) mismatched — aborting", bad)
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
