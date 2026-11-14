package accounts

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
)

// Moving a one-file hosted server's sign-ins into their own file: copy every sign-in row and the
// security chain's rows as stored, prove the copy (per set: row count and ordered-row digest), then
// remove the sign-in tables from the league file in one transaction that also records the move.

// movedRows is one set of rows the conversion moves: a whole sign-in table, or the security chain's
// rows inside the league file's shared audit tables.
type movedRows struct{ table, where string }

func signInRows() []movedRows {
	var out []movedRows
	for _, table := range SignInTables() {
		out = append(out, movedRows{table: table})
	}
	where := "source='" + SecurityAuditSource + "'"
	return append(out, movedRows{"audit_chain_head", where}, movedRows{"audit_chain", where})
}

func (m movedRows) filter() string {
	if m.where == "" {
		return ""
	}
	return " WHERE " + m.where
}

// copySignIns copies every moved row, as stored, in one transaction on the new file. The new file's
// own first auth epoch is replaced by the league file's, so existing sign-ins keep working.
// CopySignIns copies every moved row from the league file into the new sign-in file.
func CopySignIns(ctx context.Context, from, target *db.Pools) error {
	source := from.Read()
	tx, err := target.Write().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("accounts: begin the sign-in copy: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM account_auth_epoch`); err != nil {
		return fmt.Errorf("accounts: clear the new auth epoch: %w", err)
	}
	for _, rows := range signInRows() {
		if err := copyRows(ctx, source, tx, rows); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("accounts: commit the sign-in copy: %w", err)
	}
	return nil
}

func copyRows(ctx context.Context, source *sql.DB, tx *sql.Tx, m movedRows) error {
	columns, err := tableColumns(ctx, source, m.table)
	if err != nil {
		return err
	}
	list := strings.Join(columns, ",")
	rows, err := source.QueryContext(ctx, `SELECT `+list+` FROM `+m.table+m.filter()) //nolint:gosec // fixed table names; columns come from the file's own schema
	if err != nil {
		return fmt.Errorf("accounts: read %s: %w", m.table, err)
	}
	defer func() { _ = rows.Close() }()
	insert := `INSERT INTO ` + m.table + `(` + list + `) VALUES (?` + strings.Repeat(",?", len(columns)-1) + `)` //nolint:gosec // as the select above
	values := make([]any, len(columns))
	pointers := make([]any, len(columns))
	for i := range values {
		pointers[i] = &values[i]
	}
	for rows.Next() {
		if err := rows.Scan(pointers...); err != nil {
			return fmt.Errorf("accounts: read %s: %w", m.table, err)
		}
		if _, err := tx.ExecContext(ctx, insert, values...); err != nil {
			return fmt.Errorf("accounts: copy %s: %w", m.table, err)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("accounts: read %s: %w", m.table, err)
	}
	return nil
}

func tableColumns(ctx context.Context, q *sql.DB, table string) ([]string, error) {
	rows, err := q.QueryContext(ctx, `SELECT name FROM pragma_table_info(?) ORDER BY cid`, table)
	if err != nil {
		return nil, fmt.Errorf("accounts: read %s columns: %w", table, err)
	}
	defer func() { _ = rows.Close() }()
	var columns []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("accounts: read %s columns: %w", table, err)
		}
		columns = append(columns, `"`+name+`"`)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("accounts: read %s columns: %w", table, err)
	}
	if len(columns) == 0 {
		return nil, fmt.Errorf("accounts: the league file has no %s table", table)
	}
	return columns, nil
}

// rowProof is one moved set's row count and the digest of its rows in a fixed order, values quoted
// by SQLite itself so a changed type shows as a changed digest.
type rowProof struct {
	rows   int64
	digest string
}

func proveRows(ctx context.Context, q *sql.DB, m movedRows) (rowProof, error) {
	columns, err := tableColumns(ctx, q, m.table)
	if err != nil {
		return rowProof{}, err
	}
	quoted := make([]string, len(columns))
	for i, c := range columns {
		quoted[i] = "quote(" + c + ")"
	}
	// Ordered by every column, so the digest does not depend on rowid or insertion order.
	rows, err := q.QueryContext(ctx, `SELECT `+strings.Join(quoted, "||','||")+` FROM `+m.table+m.filter()+` ORDER BY `+strings.Join(columns, ",")) //nolint:gosec // as copyRows
	if err != nil {
		return rowProof{}, fmt.Errorf("accounts: digest %s: %w", m.table, err)
	}
	defer func() { _ = rows.Close() }()
	h := sha256.New()
	var proof rowProof
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			return rowProof{}, fmt.Errorf("accounts: digest %s: %w", m.table, err)
		}
		h.Write([]byte(line))
		h.Write([]byte{'\n'})
		proof.rows++
	}
	if err := rows.Err(); err != nil {
		return rowProof{}, fmt.Errorf("accounts: digest %s: %w", m.table, err)
	}
	proof.digest = hex.EncodeToString(h.Sum(nil))
	return proof, nil
}

// ProveSignIns compares every moved set, source against target: same count, same ordered-row digest.
func ProveSignIns(ctx context.Context, source, target *db.Pools) error {
	for _, m := range signInRows() {
		want, err := proveRows(ctx, source.Read(), m)
		if err != nil {
			return err
		}
		got, err := proveRows(ctx, target.Read(), m)
		if err != nil {
			return err
		}
		if want != got {
			return fmt.Errorf("accounts: %s did not copy exactly (%d rows before, %d after)", m.table, want.rows, got.rows)
		}
	}
	return nil
}

// MoveProof is one moved set's row count and ordered-row digest.
type MoveProof struct {
	Rows   int64
	Digest string
}

// MoveProofs reads the proof of every moved set in pools, keyed by table and filter.
func MoveProofs(ctx context.Context, pools *db.Pools) (map[string]MoveProof, error) {
	out := map[string]MoveProof{}
	for _, m := range signInRows() {
		p, err := proveRows(ctx, pools.Read(), m)
		if err != nil {
			return nil, err
		}
		out[m.table+m.filter()] = MoveProof{Rows: p.rows, Digest: p.digest}
	}
	return out, nil
}

// MoveRecordTable is the league file's record that its sign-ins live in the sign-in file.
const MoveRecordTable = "signin_move"

// MoveRecord is what the league file records about the move.
type MoveRecord struct {
	File     string
	Accounts int64
	Security db.AuditHead
}

// HasTable reports whether pools' file holds the named table.
func HasTable(ctx context.Context, pools *db.Pools, name string) (bool, error) {
	var held bool
	if err := pools.Read().QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name=?)`, name).Scan(&held); err != nil {
		return false, fmt.Errorf("accounts: inspect file: %w", err)
	}
	return held, nil
}

// SecurityHead reads the security chain's stored head; found is false when the file never adopted it.
func SecurityHead(ctx context.Context, pools *db.Pools) (head db.AuditHead, found bool, err error) {
	err = pools.Read().QueryRowContext(ctx, `SELECT size, hash FROM audit_chain_head WHERE source=?`, SecurityAuditSource).Scan(&head.Size, &head.Hash)
	if errors.Is(err, sql.ErrNoRows) {
		return head, false, nil
	}
	if err != nil {
		return head, false, fmt.Errorf("accounts: read the security history head: %w", err)
	}
	return head, true, nil
}

// CountAccounts counts the accounts in pools' file.
func CountAccounts(ctx context.Context, pools *db.Pools) (int64, error) {
	var n int64
	if err := pools.Read().QueryRowContext(ctx, `SELECT count(*) FROM accounts`).Scan(&n); err != nil {
		return 0, fmt.Errorf("accounts: count accounts: %w", err)
	}
	return n, nil
}

// CommitMove removes the sign-in tables from the league file (when held) and records the move, in
// one transaction. secure_delete overwrites the freed pages, so password and token material does not
// linger in the file. The security chain's rows stay: the audit tables are append-only, and they hold
// hashes, not secrets.
func CommitMove(ctx context.Context, league *db.Pools, held bool, record MoveRecord) error {
	conn, err := league.Write().Conn(ctx)
	if err != nil {
		return fmt.Errorf("accounts: reserve the league writer: %w", err)
	}
	defer func() { _ = conn.Close() }()
	if _, err := conn.ExecContext(ctx, `PRAGMA secure_delete=ON`); err != nil {
		return fmt.Errorf("accounts: prepare secure delete: %w", err)
	}
	defer func() { _, _ = conn.ExecContext(context.WithoutCancel(ctx), `PRAGMA secure_delete=OFF`) }()
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("accounts: begin the move: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if held {
		tables := SignInTables()
		for i := len(tables) - 1; i >= 0; i-- {
			if _, err := tx.ExecContext(ctx, `DROP TABLE `+tables[i]); err != nil {
				return fmt.Errorf("accounts: remove %s from the league file: %w", tables[i], err)
			}
		}
	}
	if _, err := tx.ExecContext(ctx, `CREATE TABLE `+MoveRecordTable+` (
 id INTEGER PRIMARY KEY CHECK(id = 1), moved_at TEXT NOT NULL, file TEXT NOT NULL,
 accounts INTEGER NOT NULL, security_size INTEGER NOT NULL, security_hash TEXT NOT NULL) STRICT`); err != nil {
		return fmt.Errorf("accounts: create the move record: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO `+MoveRecordTable+` VALUES (1, ?, ?, ?, ?, ?)`,
		time.Now().UTC().Format(time.RFC3339Nano), record.File, record.Accounts, record.Security.Size, record.Security.Hash); err != nil {
		return fmt.Errorf("accounts: write the move record: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("accounts: commit the move: %w", err)
	}
	return nil
}

// MissingAccounts returns the named principals that have no account in pools' file.
func MissingAccounts(ctx context.Context, pools *db.Pools, named []string) ([]string, error) {
	rows, err := pools.Read().QueryContext(ctx, `SELECT 'acct.' || id FROM accounts`)
	if err != nil {
		return nil, fmt.Errorf("accounts: read accounts: %w", err)
	}
	defer func() { _ = rows.Close() }()
	held := map[string]bool{}
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, fmt.Errorf("accounts: read accounts: %w", err)
		}
		held[p] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("accounts: read accounts: %w", err)
	}
	var missing []string
	for _, p := range named {
		if !held[p] {
			missing = append(missing, p)
		}
	}
	return missing, nil
}
