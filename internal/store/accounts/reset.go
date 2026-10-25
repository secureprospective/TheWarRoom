package accounts

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"unicode"
)

// ResetSignIns is the operator-only incident port. A new auth epoch ends every device
// session, helper token and recent second-factor proof issued before it; passwords and
// authenticators are kept. Never expose it as a league command or an HTTP route.
func (s *Store) ResetSignIns(ctx context.Context, reason string) (int, error) {
	return s.resetAuth(ctx, reason, "sign_ins_reset", false)
}

// ResetAfterRestore runs on a restored copy before any listener opens. Besides the new
// epoch it removes what the copy may have revived: a printed recovery code or invitation
// used after the snapshot reads as unused in it, so every printed code and unredeemed
// invitation goes. Owners print new codes after signing in with password and authenticator.
func (s *Store) ResetAfterRestore(ctx context.Context, reason string) (int, error) {
	return s.resetAuth(ctx, reason, "restored_sign_ins_reset", true)
}

func (s *Store) resetAuth(ctx context.Context, reason, operation string, restored bool) (int, error) {
	if reason == "" || len(reason) > 256 || strings.IndexFunc(reason, func(r rune) bool { return !unicode.IsPrint(r) }) >= 0 {
		return 0, fmt.Errorf("accounts: a printable reason of up to 256 characters is required")
	}
	epoch, err := newAuthEpoch()
	if err != nil {
		return 0, err
	}
	tx, err := s.pools.BeginWrite(ctx)
	if err != nil {
		return 0, fmt.Errorf("accounts: begin sign-in reset: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	statements := []string{`UPDATE account_auth_epoch SET epoch = ? WHERE singleton = 1`,
		`UPDATE account_totp SET proof_hash = '', authenticated_at = 0`}
	if restored {
		statements = append(statements, `DELETE FROM account_recovery_codes`,
			`DELETE FROM account_enrollments WHERE redeemed_account IS NULL`)
	}
	for i, statement := range statements {
		args := []any{}
		if i == 0 {
			args = append(args, epoch)
		}
		if _, err := tx.ExecContext(ctx, statement, args...); err != nil {
			return 0, fmt.Errorf("accounts: reset sign-ins: %w", err)
		}
	}
	ids, err := accountIDs(ctx, tx)
	if err != nil {
		return 0, err
	}
	now := s.now().UTC().UnixNano()
	for _, id := range ids {
		if err := recoveryAudit(ctx, tx, id, operation, reason, now); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("accounts: commit sign-in reset: %w", err)
	}
	return len(ids), nil
}

func accountIDs(ctx context.Context, tx *sql.Tx) ([]string, error) {
	rows, err := tx.QueryContext(ctx, `SELECT id FROM accounts ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("accounts: list accounts: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("accounts: list accounts: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("accounts: list accounts: %w", err)
	}
	return ids, nil
}
