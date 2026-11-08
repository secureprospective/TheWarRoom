package accounts

import (
	"context"
	"fmt"

	identity "github.com/secureprospective/TheWarRoom/internal/accounts"
)

// ChangePassword replaces one account's hash and moves its credential generation. Sessions,
// helper tokens and recent-auth proofs issued before the move no longer resolve.
func (s *Store) ChangePassword(ctx context.Context, accountID, passwordHash string) error {
	if _, _, err := identity.DecodePasswordHash(passwordHash); err != nil {
		return fmt.Errorf("accounts: change password: %w", err)
	}
	tx, err := s.pools.BeginWrite(ctx)
	if err != nil {
		return fmt.Errorf("accounts: begin password change: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	result, err := tx.ExecContext(ctx, `UPDATE accounts SET password_hash = ? WHERE id = ?`, passwordHash, accountID)
	if err != nil {
		return fmt.Errorf("accounts: change password: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("accounts: change password result: %w", err)
	}
	if count != 1 {
		return identity.ErrNotFound
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO account_auth_generation(account_id, generation) VALUES (?, 1)
 ON CONFLICT(account_id) DO UPDATE SET generation = generation + 1`, accountID); err != nil {
		return fmt.Errorf("accounts: revoke credentials: %w", err)
	}
	if err := recoveryAudit(ctx, tx, accountID, "password_changed", "", s.now().UTC().UnixNano()); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("accounts: commit password change: %w", err)
	}
	return nil
}
