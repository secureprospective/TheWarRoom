package db

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
)

// Moving part of a one-file hosted server into its own file copies rows as stored and proves the copy
// set by set: the row count and a digest of every row in a fixed order. The sign-in move and the
// reference move both use these; a set is a whole table or a filtered part of one.

// MovedRows is one set of rows a move copies: a table, optionally narrowed by a fixed SQL filter.
// Order, when set, fixes the order CopyRows inserts in (proofs never depend on it).
type MovedRows struct{ Table, Where, Order string }

func (m MovedRows) filter() string {
	if m.Where == "" {
		return ""
	}
	return " WHERE " + m.Where
}

// Key names the set in a proof map: the table and its filter.
func (m MovedRows) Key() string { return m.Table + m.filter() }

// RowProof is one set's row count and the digest of its rows ordered by every column, each value quoted
// by SQLite itself, so a changed type shows as a changed digest.
type RowProof struct {
	Rows   int64
	Digest string
}

// HasTable reports whether pools' own file (schema main) holds the named table.
func HasTable(ctx context.Context, pools *Pools, name string) (bool, error) {
	var held bool
	if err := pools.Read().QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM main.sqlite_master WHERE type='table' AND name=?)`, name).Scan(&held); err != nil {
		return false, fmt.Errorf("db: inspect file: %w", err)
	}
	return held, nil
}

// CopyRows copies one set from source into tx's file, as stored.
func CopyRows(ctx context.Context, source *sql.DB, tx *sql.Tx, m MovedRows) error {
	columns, err := tableColumns(ctx, source, m.Table)
	if err != nil {
		return err
	}
	list := strings.Join(columns, ",")
	order := ""
	if m.Order != "" {
		order = " ORDER BY " + m.Order
	}
	rows, err := source.QueryContext(ctx, `SELECT `+list+` FROM main.`+m.Table+m.filter()+order) //nolint:gosec // fixed table names; columns come from the file's own schema
	if err != nil {
		return fmt.Errorf("db: read %s: %w", m.Table, err)
	}
	defer func() { _ = rows.Close() }()
	insert := `INSERT INTO main.` + m.Table + `(` + list + `) VALUES (?` + strings.Repeat(",?", len(columns)-1) + `)` //nolint:gosec // as the select above
	values := make([]any, len(columns))
	pointers := make([]any, len(columns))
	for i := range values {
		pointers[i] = &values[i]
	}
	for rows.Next() {
		if err := rows.Scan(pointers...); err != nil {
			return fmt.Errorf("db: read %s: %w", m.Table, err)
		}
		if _, err := tx.ExecContext(ctx, insert, values...); err != nil {
			return fmt.Errorf("db: copy %s: %w", m.Table, err)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("db: read %s: %w", m.Table, err)
	}
	return nil
}

func tableColumns(ctx context.Context, q *sql.DB, table string) ([]string, error) {
	rows, err := q.QueryContext(ctx, `SELECT name FROM pragma_table_info(?, 'main') ORDER BY cid`, table)
	if err != nil {
		return nil, fmt.Errorf("db: read %s columns: %w", table, err)
	}
	defer func() { _ = rows.Close() }()
	var columns []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("db: read %s columns: %w", table, err)
		}
		columns = append(columns, `"`+name+`"`)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("db: read %s columns: %w", table, err)
	}
	if len(columns) == 0 {
		return nil, fmt.Errorf("db: the file has no %s table", table)
	}
	return columns, nil
}

// ProveRows reads one set's count and ordered-row digest from q's own file.
func ProveRows(ctx context.Context, q *sql.DB, m MovedRows) (RowProof, error) {
	columns, err := tableColumns(ctx, q, m.Table)
	if err != nil {
		return RowProof{}, err
	}
	quoted := make([]string, len(columns))
	for i, c := range columns {
		quoted[i] = "quote(" + c + ")"
	}
	// Ordered by every column, so the digest does not depend on rowid or insertion order.
	rows, err := q.QueryContext(ctx, `SELECT `+strings.Join(quoted, "||','||")+` FROM main.`+m.Table+m.filter()+` ORDER BY `+strings.Join(columns, ",")) //nolint:gosec // as CopyRows
	if err != nil {
		return RowProof{}, fmt.Errorf("db: digest %s: %w", m.Table, err)
	}
	defer func() { _ = rows.Close() }()
	h := sha256.New()
	var proof RowProof
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			return RowProof{}, fmt.Errorf("db: digest %s: %w", m.Table, err)
		}
		h.Write([]byte(line))
		h.Write([]byte{'\n'})
		proof.Rows++
	}
	if err := rows.Err(); err != nil {
		return RowProof{}, fmt.Errorf("db: digest %s: %w", m.Table, err)
	}
	proof.Digest = hex.EncodeToString(h.Sum(nil))
	return proof, nil
}

// ProveMoved compares every set, source against target: same count, same ordered-row digest.
func ProveMoved(ctx context.Context, source, target *Pools, sets []MovedRows) error {
	for _, m := range sets {
		want, err := ProveRows(ctx, source.Read(), m)
		if err != nil {
			return err
		}
		got, err := ProveRows(ctx, target.Read(), m)
		if err != nil {
			return err
		}
		if want != got {
			return fmt.Errorf("db: %s did not copy exactly (%d rows before, %d after)", m.Table, want.Rows, got.Rows)
		}
	}
	return nil
}
