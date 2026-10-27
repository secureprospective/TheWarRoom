package state

import (
	"context"
	"database/sql"
	"fmt"
)

// tradeScratchGlob matches the generations older builds seeded into whatif.db for every trade
// projection. Projections now run on a throwaway scratch pool, so nothing references these rows.
const tradeScratchGlob = `(league_id GLOB 'trade-counterparty:*' OR league_id GLOB 'trade-preview:*')`

// dropTradeScratch (v3) deletes the trade-projection generations once. Real what-if generations
// (whatif:*), the league's own rows, saved sessions and trade plans are untouched. Append-only
// tables refuse deletes, so their delete guards are lifted inside this transaction and restored
// verbatim from sqlite_master; a failure rolls the guards back with the rows.
func dropTradeScratch(ctx context.Context, tx *sql.Tx) error {
	tables, err := queryStrings(ctx, tx, `SELECT m.name FROM sqlite_master m, pragma_table_info(m.name) c
WHERE m.type = 'table' AND c.name = 'league_id' ORDER BY m.name`)
	if err != nil {
		return fmt.Errorf("state: trade scratch tables: %w", err)
	}
	// Each lifted guard is restored from its own stored text; the guards of other tables stay put.
	guards, err := queryStrings(ctx, tx, `SELECT t.sql FROM sqlite_master t WHERE t.type = 'trigger'
AND t.name = t.tbl_name || '_no_delete'
AND EXISTS (SELECT 1 FROM pragma_table_info(t.tbl_name) c WHERE c.name = 'league_id') ORDER BY t.name`)
	if err != nil {
		return fmt.Errorf("state: trade scratch guards: %w", err)
	}
	for _, table := range tables {
		if _, err := tx.ExecContext(ctx, `DROP TRIGGER IF EXISTS "`+table+`_no_delete"`); err != nil {
			return fmt.Errorf("state: lift %s delete guard: %w", table, err)
		}
	}
	for _, table := range tables {
		if _, err := tx.ExecContext(ctx, `DELETE FROM "`+table+`" WHERE `+tradeScratchGlob); err != nil { //nolint:gosec // names come from sqlite_master
			return fmt.Errorf("state: drop trade scratch from %s: %w", table, err)
		}
	}
	for _, guard := range guards {
		if _, err := tx.ExecContext(ctx, guard); err != nil {
			return fmt.Errorf("state: restore delete guard: %w", err)
		}
	}
	return nil
}

func queryStrings(ctx context.Context, tx *sql.Tx, query string) ([]string, error) {
	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return nil, err //nolint:wrapcheck // callers name the query
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, err //nolint:wrapcheck // callers name the query
		}
		out = append(out, value)
	}
	return out, rows.Err() //nolint:wrapcheck // callers name the query
}
