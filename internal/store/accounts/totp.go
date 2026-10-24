package accounts

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	identity "github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/governance"
)

func validateTOTPAccount(id string) error {
	if _, err := governance.AccountPrincipal(id); err != nil {
		return identity.ErrTOTPRequired
	}
	return nil
}

// BeginTOTP cannot replace a confirmed factor; a fresh password is checked by the service.
func (s *Store) BeginTOTP(ctx context.Context, accountID string, sealedSeed []byte) error {
	if err := validateTOTPAccount(accountID); err != nil {
		return err
	}
	if len(sealedSeed) != 48 {
		return identity.ErrTOTPRequired
	}
	tx, err := s.pools.Write().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("accounts: begin authenticator setup: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	now := s.now().UTC()
	result, err := tx.ExecContext(ctx, `INSERT INTO account_totp(account_id, sealed_seed, pending_until)
 VALUES (?, ?, ?) ON CONFLICT(account_id) DO UPDATE
 SET sealed_seed = excluded.sealed_seed, pending_until = excluded.pending_until
 WHERE account_totp.confirmed = 0`, accountID, sealedSeed, now.Add(identity.TOTPEnrollmentLifetime).UnixNano())
	if err != nil {
		return fmt.Errorf("accounts: save authenticator setup: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("accounts: authenticator setup result: %w", err)
	}
	if count != 1 {
		return identity.ErrTOTPRequired
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("accounts: commit authenticator setup: %w", err)
	}
	return nil
}

func (s *Store) TOTP(ctx context.Context, accountID string) (identity.TOTPState, error) {
	if err := validateTOTPAccount(accountID); err != nil {
		return identity.TOTPState{}, err
	}
	var state identity.TOTPState
	err := s.pools.Read().QueryRowContext(ctx, `SELECT sealed_seed, confirmed FROM account_totp WHERE account_id = ?`, accountID).
		Scan(&state.SealedSeed, &state.Confirmed)
	if errors.Is(err, sql.ErrNoRows) {
		return identity.TOTPState{}, identity.ErrTOTPRequired
	}
	if err != nil {
		return identity.TOTPState{}, fmt.Errorf("accounts: read authenticator: %w", err)
	}
	state.Now = s.now().UTC()
	return state, nil
}

// ConsumeTOTP atomically advances the replay counter and (only on step-up) replaces
// the recent-auth proof. Seed comparison fences concurrent pending setup replacement.
func (s *Store) ConsumeTOTP(ctx context.Context, accountID string, sealedSeed []byte, counter int64, confirming bool, proofDigest string) error {
	if err := validateTOTPAccount(accountID); err != nil {
		return err
	}
	if len(sealedSeed) != 48 || (confirming && proofDigest != "") {
		return identity.ErrTOTPRequired
	}
	if !confirming && !validSecretDigest(proofDigest) {
		return identity.ErrTOTPRequired
	}
	tx, err := s.pools.Write().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("accounts: begin authenticator verification: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	now := s.now().UTC()
	if !identity.ValidTOTPCounter(counter, now) {
		return identity.ErrTOTPRequired
	}
	result, err := tx.ExecContext(ctx, `UPDATE account_totp SET confirmed = 1, last_counter = ?,
 proof_hash = ?, authenticated_at = ? WHERE account_id = ? AND sealed_seed = ? AND last_counter < ?
 AND ((? AND confirmed = 0 AND pending_until > ?) OR (NOT ? AND confirmed = 1))`,
		counter, proofDigest, now.UnixNano(), accountID, sealedSeed, counter, confirming, now.UnixNano(), confirming)
	if err != nil {
		return fmt.Errorf("accounts: consume authenticator code: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("accounts: authenticator verification result: %w", err)
	}
	if count != 1 {
		return identity.ErrTOTPRequired
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("accounts: commit authenticator verification: %w", err)
	}
	return nil
}

func (s *Store) CheckRecentAuth(ctx context.Context, accountID, proofDigest string) error {
	_, err := s.RecentAuthRemaining(ctx, accountID, proofDigest)
	return err
}

// RecentAuthRemaining is how long a valid recent-authentication proof has left on the server clock.
func (s *Store) RecentAuthRemaining(ctx context.Context, accountID, proofDigest string) (time.Duration, error) {
	if err := validateTOTPAccount(accountID); err != nil {
		return 0, err
	}
	if !validSecretDigest(proofDigest) {
		return 0, identity.ErrTOTPRequired
	}
	now := s.now().UTC().UnixNano()
	var authenticated int64
	err := s.pools.Read().QueryRowContext(ctx, `SELECT COALESCE((SELECT authenticated_at FROM account_totp
 WHERE account_id = ? AND confirmed = 1 AND proof_hash = ?
 AND authenticated_at <= ? AND authenticated_at > ?), 0)`, accountID, proofDigest, now, now-int64(identity.RecentAuthLifetime)).Scan(&authenticated)
	if err != nil {
		return 0, fmt.Errorf("accounts: check recent authentication: %w", err)
	}
	if authenticated == 0 {
		return 0, identity.ErrTOTPRequired
	}
	return time.Duration(authenticated + int64(identity.RecentAuthLifetime) - now), nil
}
