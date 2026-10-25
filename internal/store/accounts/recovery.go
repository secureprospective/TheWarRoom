package accounts

import (
	"context"
	"database/sql"
	"fmt"

	identity "github.com/secureprospective/TheWarRoom/internal/accounts"
)

func (s *Store) ReplaceRecoveryCodes(ctx context.Context, accountID, proofDigest string, digests [identity.RecoveryCodeCount]string) error {
	if err := validateTOTPAccount(accountID); err != nil {
		return err
	}
	if !validSecretDigest(proofDigest) {
		return identity.ErrRecoveryRefused
	}
	for _, digest := range digests {
		if !validSecretDigest(digest) {
			return identity.ErrRecoveryRefused
		}
	}
	tx, err := s.pools.BeginWrite(ctx)
	if err != nil {
		return fmt.Errorf("accounts: begin recovery codes: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	now := s.now().UTC().UnixNano()
	var allowed bool
	err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM account_totp WHERE account_id = ? AND confirmed = 1 AND proof_hash = ? AND authenticated_at <= ? AND authenticated_at > ?)`, accountID, proofDigest, now, now-int64(identity.RecentAuthLifetime)).Scan(&allowed)
	if err != nil {
		return fmt.Errorf("accounts: recovery authentication: %w", err)
	}
	if !allowed {
		return identity.ErrRecoveryRefused
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM account_recovery_codes WHERE account_id = ?`, accountID); err != nil {
		return fmt.Errorf("accounts: replace recovery codes: %w", err)
	}
	for _, digest := range digests {
		if _, err := tx.ExecContext(ctx, `INSERT INTO account_recovery_codes(account_id, code_hash) VALUES (?, ?)`, accountID, digest); err != nil {
			return fmt.Errorf("accounts: save recovery code: %w", err)
		}
	}
	if err := recoveryAudit(ctx, tx, accountID, "recovery_codes_printed", "", now); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("accounts: commit recovery codes: %w", err)
	}
	return nil
}

// ResetAccount is a trusted service port. An empty digest is allowed only after
// offline-key verification by the local console service, never by league policy.
func (s *Store) ResetAccount(ctx context.Context, username, passwordHash, codeDigest, reason string) error {
	if _, _, err := identity.DecodePasswordHash(passwordHash); err != nil {
		return fmt.Errorf("accounts: reset password: %w", err)
	}
	if (codeDigest == "" && (reason == "" || len(reason) > 256)) || (codeDigest != "" && (!validSecretDigest(codeDigest) || reason != "")) {
		return identity.ErrRecoveryRefused
	}
	tx, err := s.pools.BeginWrite(ctx)
	if err != nil {
		return fmt.Errorf("accounts: begin recovery: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var accountID string
	err = tx.QueryRowContext(ctx, `SELECT id FROM accounts WHERE username = ? AND (? = '' OR EXISTS(SELECT 1 FROM account_recovery_codes WHERE account_id = accounts.id AND code_hash = ?))`, username, codeDigest, codeDigest).Scan(&accountID)
	if err == sql.ErrNoRows {
		return identity.ErrRecoveryRefused
	}
	if err != nil {
		return fmt.Errorf("accounts: recovery identity: %w", err)
	}
	if err := resetCredentials(ctx, tx, accountID, passwordHash); err != nil {
		return err
	}
	operation := "recovery_code_reset"
	if codeDigest == "" {
		operation = "offline_key_reset"
	}
	if err := recoveryAudit(ctx, tx, accountID, operation, reason, s.now().UTC().UnixNano()); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("accounts: commit recovery: %w", err)
	}
	return nil
}

func resetCredentials(ctx context.Context, tx *sql.Tx, accountID, passwordHash string) error {
	if _, err := tx.ExecContext(ctx, `UPDATE accounts SET password_hash = ? WHERE id = ?`, passwordHash, accountID); err != nil {
		return fmt.Errorf("accounts: reset password: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM account_totp WHERE account_id = ?`, accountID); err != nil {
		return fmt.Errorf("accounts: reset authenticator: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM account_recovery_codes WHERE account_id = ?`, accountID); err != nil {
		return fmt.Errorf("accounts: revoke recovery codes: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO account_auth_generation(account_id, generation) VALUES (?, 1) ON CONFLICT(account_id) DO UPDATE SET generation = generation + 1`, accountID); err != nil {
		return fmt.Errorf("accounts: revoke credentials: %w", err)
	}
	return nil
}

func recoveryAudit(ctx context.Context, tx *sql.Tx, accountID, operation, reason string, now int64) error {
	result, err := tx.ExecContext(ctx, `INSERT INTO account_security_audit(account_id, operation, reason, occurred_at) VALUES (?, ?, ?, ?)`, accountID, operation, reason, now)
	if err != nil {
		return fmt.Errorf("accounts: audit recovery: %w", err)
	}
	sequence, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("accounts: audit sequence: %w", err)
	}
	return appendSecurityAudit(ctx, tx, sequence)
}

// AuthGeneration fences every credential/delegation issued before a reset. Future
// admission must compare the issuance generation again inside the write transaction.
func (s *Store) AuthGeneration(ctx context.Context, accountID string) (int64, error) {
	var generation int64
	err := s.pools.Read().QueryRowContext(ctx, `SELECT COALESCE((SELECT generation FROM account_auth_generation WHERE account_id = accounts.id), 0) FROM accounts WHERE id = ?`, accountID).Scan(&generation)
	if err != nil {
		return 0, fmt.Errorf("accounts: credential generation: %w", err)
	}
	return generation, nil
}
