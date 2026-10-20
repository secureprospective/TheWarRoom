package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// JSONColumn names a JSON storage boundary. Empty permits the legacy absent-metadata marker.
type JSONColumn struct {
	Table, Column    string
	Versioned, Empty bool
}

// GuardJSON installs write guards without rebuilding append-only tables or rewriting old rows.
// Identifiers are migration constants, never application input.
func GuardJSON(ctx context.Context, tx *sql.Tx, columns ...JSONColumn) error {
	for _, column := range columns {
		if err := guardJSONColumn(ctx, tx, column); err != nil {
			return err
		}
	}
	return nil
}

func guardJSONColumn(ctx context.Context, tx *sql.Tx, c JSONColumn) error {
	if !sqlIdentifier(c.Table) || !sqlIdentifier(c.Column) {
		return fmt.Errorf("db: invalid JSON guard identifier")
	}
	value := "NEW." + c.Column
	valid := "json_valid(" + value + ")"
	if c.Versioned {
		valid = "CASE WHEN " + valid + " THEN json_type(" + value + ", '$.version') = 'integer' AND json_extract(" + value + ", '$.version') > 0 ELSE 0 END"
	}
	if c.Empty {
		valid = value + " = '' OR (" + valid + ")"
	}
	for _, op := range []string{"INSERT", "UPDATE"} {
		// An older guard under this name may let a NULL verdict through: replace it, never keep it.
		//nolint:gosec // SQLite cannot bind DDL identifiers; names are checked ASCII identifiers, and expressions are fixed here.
		ddl := fmt.Sprintf(`DROP TRIGGER IF EXISTS %[1]s_%[2]s_json_%[3]s;
CREATE TRIGGER %[1]s_%[2]s_json_%[3]s BEFORE %[4]s ON %[1]s
WHEN NOT COALESCE((%[5]s), 0)
BEGIN SELECT RAISE(ABORT, 'invalid JSON payload'); END;`, c.Table, c.Column, strings.ToLower(op), op, valid)
		if _, err := tx.ExecContext(ctx, ddl); err != nil {
			return fmt.Errorf("db: guard %s.%s JSON: %w", c.Table, c.Column, err)
		}
	}
	return nil
}

func sqlIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r != '_' && (r < 'a' || r > 'z') {
			return false
		}
	}
	return true
}
