package accounts

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
)

// Moving a one-file hosted server's sign-ins into their own file: copy every sign-in row and the
// security chain's rows as stored, prove the copy (per set: row count and ordered-row digest), then
// remove the sign-in tables from the league file in one transaction that also records the move.

func signInRows() []db.MovedRows {
	var out []db.MovedRows
	for _, table := range SignInTables() {
		out = append(out, db.MovedRows{Table: table})
	}
	where := "source='" + SecurityAuditSource + "'"
	return append(out, db.MovedRows{Table: "audit_chain_head", Where: where}, db.MovedRows{Table: "audit_chain", Where: where})
}

// CopySignIns copies every moved row from the league file into the new sign-in file, in one
// transaction. The new file's own first auth epoch is replaced by the league file's, so existing
// sign-ins keep working.
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
		if err := db.CopyRows(ctx, source, tx, rows); err != nil {
			return fmt.Errorf("accounts: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("accounts: commit the sign-in copy: %w", err)
	}
	return nil
}

// ProveSignIns compares every moved set, source against target: same count, same ordered-row digest.
func ProveSignIns(ctx context.Context, source, target *db.Pools) error {
	if err := db.ProveMoved(ctx, source, target, signInRows()); err != nil {
		return fmt.Errorf("accounts: %w", err)
	}
	return nil
}

// MoveProof is one moved set's row count and ordered-row digest.
type MoveProof = db.RowProof

// MoveProofs reads the proof of every moved set in pools, keyed by table and filter.
func MoveProofs(ctx context.Context, pools *db.Pools) (map[string]MoveProof, error) {
	out := map[string]MoveProof{}
	for _, m := range signInRows() {
		p, err := db.ProveRows(ctx, pools.Read(), m)
		if err != nil {
			return nil, fmt.Errorf("accounts: %w", err)
		}
		out[m.Key()] = p
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
	return db.HasTable(ctx, pools, name) //nolint:wrapcheck // the caller names the file
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
