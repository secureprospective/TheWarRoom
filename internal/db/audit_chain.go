package db

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

// AuditHead is a versioned, source-scoped checkpoint suitable for off-host anchoring.
// Baseline counts rows trusted at first adoption, not independently authenticated history.
type AuditHead struct {
	Version  int    `json:"version"`
	Source   string `json:"source"`
	Size     int64  `json:"size"`
	Hash     string `json:"hash"`
	Baseline int64  `json:"baseline"`
}

const auditDDL = `
CREATE TABLE IF NOT EXISTS audit_chain_head (
 source TEXT PRIMARY KEY, size INTEGER NOT NULL, hash TEXT NOT NULL, baseline INTEGER NOT NULL
) STRICT;
CREATE TABLE IF NOT EXISTS audit_chain (
 source TEXT NOT NULL, sequence INTEGER NOT NULL, entry_key TEXT NOT NULL,
 digest TEXT NOT NULL, previous TEXT NOT NULL, hash TEXT NOT NULL,
 PRIMARY KEY(source,sequence), UNIQUE(source,entry_key)
) STRICT;
CREATE TRIGGER IF NOT EXISTS audit_chain_no_update BEFORE UPDATE ON audit_chain
 BEGIN SELECT RAISE(ABORT,'audit chain is append-only'); END;
CREATE TRIGGER IF NOT EXISTS audit_chain_no_delete BEFORE DELETE ON audit_chain
 BEGIN SELECT RAISE(ABORT,'audit chain is append-only'); END;`

// InitializeAudit adopts existing rows once, in the caller's startup transaction.
// query must be a fixed store-owned SELECT of (stable key, canonical text), in append order.
func InitializeAudit(ctx context.Context, tx *sql.Tx, source, query string) error {
	if _, err := tx.ExecContext(ctx, auditDDL); err != nil {
		return fmt.Errorf("audit schema: %w", err)
	}
	_, err := ReadAuditHead(ctx, tx, source)
	if err == nil {
		_, err = VerifyAudit(ctx, tx, source, query)
		return err
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	var held int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM audit_chain WHERE source=?`, source).Scan(&held); err != nil {
		return fmt.Errorf("audit adoption: %w", err)
	}
	if held != 0 {
		return fmt.Errorf("audit head is missing")
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO audit_chain_head VALUES (?,0,'',0)`, source)
	if err := auditWriteResult(result, err); err != nil {
		return fmt.Errorf("audit adoption: %w", err)
	}
	if err := baselineAudit(ctx, tx, source, query); err != nil {
		return err
	}
	result, err = tx.ExecContext(ctx, `UPDATE audit_chain_head SET baseline=size WHERE source=?`, source)
	if err := auditWriteResult(result, err); err != nil {
		return fmt.Errorf("audit baseline: %w", err)
	}
	return nil
}

func baselineAudit(ctx context.Context, tx *sql.Tx, source, query string) error {
	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("audit baseline rows: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var key, payload string
		if err := rows.Scan(&key, &payload); err != nil {
			return fmt.Errorf("audit baseline row: %w", err)
		}
		if err := AppendAudit(ctx, tx, source, key, payload); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("audit baseline rows: %w", err)
	}
	return nil
}

// ReadAuditHead must be used on the same transaction as the data it checkpoints.
func ReadAuditHead(ctx context.Context, tx *sql.Tx, source string) (AuditHead, error) {
	head := AuditHead{Version: 1, Source: source}
	err := tx.QueryRowContext(ctx, `SELECT size,hash,baseline FROM audit_chain_head WHERE source=?`, source).Scan(&head.Size, &head.Hash, &head.Baseline)
	if err != nil {
		return AuditHead{}, fmt.Errorf("audit head: %w", err)
	}
	return head, nil
}

func auditDigest(payload string) string {
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])
}

func auditHash(source string, size int64, key, digest, previous string) string {
	// A string array cannot fail JSON encoding and unambiguously frames embedded separators.
	body, _ := json.Marshal([]string{"twr-audit-v1", source, strconv.FormatInt(size, 10), key, digest, previous})
	return auditDigest(string(body))
}

// AppendAudit records a source row in its owner's transaction; any error must abort that transaction.
func AppendAudit(ctx context.Context, tx *sql.Tx, source, key, payload string) error {
	head, err := ReadAuditHead(ctx, tx, source)
	if err != nil {
		return err
	}
	if err := checkAuditTip(ctx, tx, head); err != nil {
		return err
	}
	digest := auditDigest(payload)
	hash := auditHash(source, head.Size+1, key, digest, head.Hash)
	result, err := tx.ExecContext(ctx, `INSERT INTO audit_chain VALUES (?,?,?,?,?,?)`, source, head.Size+1, key, digest, head.Hash, hash)
	if err := auditWriteResult(result, err); err != nil {
		return fmt.Errorf("append audit: %w", err)
	}
	result, err = tx.ExecContext(ctx, `UPDATE audit_chain_head SET size=?,hash=? WHERE source=?`, head.Size+1, hash, source)
	if err := auditWriteResult(result, err); err != nil {
		return fmt.Errorf("advance audit head: %w", err)
	}
	return nil
}

func auditWriteResult(result sql.Result, err error) error {
	if err != nil {
		return fmt.Errorf("write audit: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("audit write count: %w", err)
	}
	if count != 1 {
		return fmt.Errorf("audit write did not persist")
	}
	return nil
}

func checkAuditTip(ctx context.Context, tx *sql.Tx, head AuditHead) error {
	var size int64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence),0) FROM audit_chain WHERE source=?`, head.Source).Scan(&size); err != nil {
		return fmt.Errorf("audit tip: %w", err)
	}
	if size != head.Size || size < 0 || (size == 0 && head.Hash != "") {
		return fmt.Errorf("audit tip does not match head")
	}
	if size == 0 {
		return nil
	}
	var hash string
	if err := tx.QueryRowContext(ctx, `SELECT hash FROM audit_chain WHERE source=? AND sequence=?`, head.Source, size).Scan(&hash); err != nil {
		return fmt.Errorf("audit tip hash: %w", err)
	}
	if hash != head.Hash {
		return fmt.Errorf("audit tip does not match head")
	}
	return nil
}

// VerifyAudit compares every original row (before decoding/upcasting) and chain link in one snapshot.
func VerifyAudit(ctx context.Context, tx *sql.Tx, source, query string) (AuditHead, error) {
	head, err := ReadAuditHead(ctx, tx, source)
	if err != nil {
		return AuditHead{}, err
	}
	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return AuditHead{}, fmt.Errorf("verify audit source: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var size int64
	previous := ""
	for rows.Next() {
		var key, payload string
		if err := rows.Scan(&key, &payload); err != nil {
			return AuditHead{}, fmt.Errorf("verify audit row: %w", err)
		}
		size++
		hash := auditHash(source, size, key, auditDigest(payload), previous)
		var matches bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM audit_chain WHERE source=? AND sequence=? AND entry_key=? AND digest=? AND previous=? AND hash=?)`, source, size, key, auditDigest(payload), previous, hash).Scan(&matches); err != nil {
			return AuditHead{}, fmt.Errorf("verify audit link: %w", err)
		}
		if !matches {
			return AuditHead{}, fmt.Errorf("audit integrity check failed at entry %d", size)
		}
		previous = hash
	}
	if err := rows.Err(); err != nil {
		return AuditHead{}, fmt.Errorf("verify audit rows: %w", err)
	}
	var count int64
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM audit_chain WHERE source=?`, source).Scan(&count); err != nil {
		return AuditHead{}, fmt.Errorf("verify audit size: %w", err)
	}
	if count != size || head.Size != size || head.Hash != previous || head.Baseline < 0 || head.Baseline > size {
		return AuditHead{}, fmt.Errorf("audit integrity check failed: head or tail mismatch")
	}
	return head, nil
}
