package accounts

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/db"
)

const securityAuditSource = "account_security_audit"
const securityAuditColumns = `SELECT CAST(sequence AS TEXT),json_array(sequence,account_id,operation,reason,occurred_at) FROM account_security_audit`
const securityAuditQuery = securityAuditColumns + ` ORDER BY sequence`

func (s *Store) initializeAudit(ctx context.Context) error {
	tx, err := s.pools.BeginWrite(ctx)
	if err != nil {
		return fmt.Errorf("accounts: begin audit setup: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := db.InitializeAudit(ctx, tx, securityAuditSource, securityAuditQuery); err != nil {
		return fmt.Errorf("accounts: initialize audit: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `CREATE TRIGGER IF NOT EXISTS account_security_audit_no_update BEFORE UPDATE ON account_security_audit
 BEGIN SELECT RAISE(ABORT,'security audit is append-only'); END;
 CREATE TRIGGER IF NOT EXISTS account_security_audit_no_delete BEFORE DELETE ON account_security_audit
 BEGIN SELECT RAISE(ABORT,'security audit is append-only'); END;`); err != nil {
		return fmt.Errorf("accounts: protect audit: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("accounts: commit audit setup: %w", err)
	}
	return nil
}

// VerifyAudit returns a source-scoped checkpoint only after checking the original audit rows.
func (s *Store) VerifyAudit(ctx context.Context) (db.AuditHead, error) {
	tx, err := s.pools.Read().BeginTx(ctx, nil)
	if err != nil {
		return db.AuditHead{}, fmt.Errorf("accounts audit: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	head, err := db.VerifyAudit(ctx, tx, securityAuditSource, securityAuditQuery)
	if err != nil {
		return db.AuditHead{}, fmt.Errorf("accounts: verify audit: %w", err)
	}
	return head, nil
}

func appendSecurityAudit(ctx context.Context, tx *sql.Tx, sequence int64) error {
	var key, payload string
	if err := tx.QueryRowContext(ctx, securityAuditColumns+` WHERE sequence=?`, sequence).Scan(&key, &payload); err != nil {
		return fmt.Errorf("accounts: audit row: %w", err)
	}
	if err := db.AppendAudit(ctx, tx, securityAuditSource, key, payload); err != nil {
		return fmt.Errorf("accounts: append audit: %w", err)
	}
	return nil
}

// VerifyAuditAnchor checks the full local chain and its independently held prefix in one snapshot.
func (s *Store) VerifyAuditAnchor(ctx context.Context, anchor db.AuditHead) error {
	tx, err := s.pools.Read().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin anchored audit verification: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	current, err := db.VerifyAudit(ctx, tx, securityAuditSource, securityAuditQuery)
	if err != nil {
		return fmt.Errorf("verify anchored audit: %w", err)
	}
	if err := db.CheckAuditAnchor(ctx, tx, current, anchor); err != nil {
		return fmt.Errorf("verify anchored audit prefix: %w", err)
	}
	return nil
}
