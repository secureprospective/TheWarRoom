package accounts

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	identity "github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/governance"
)

func validSecretDigest(digest string) bool {
	decoded, err := hex.DecodeString(digest)
	return err == nil && len(decoded) == 32 && strings.ToLower(digest) == digest
}

func validateEnrollment(digest, username string) error {
	if !validSecretDigest(digest) {
		return identity.ErrEnrollmentRefused
	}
	canonical, err := identity.CanonicalUsername(username)
	if err != nil || username != canonical {
		return identity.ErrEnrollmentRefused
	}
	return nil
}

// IssueEnrollment persists only the digest; the caller authorizes the local issuer.
func (s *Store) IssueEnrollment(ctx context.Context, digest, username, league string, issuer governance.Principal) (time.Time, error) {
	if err := validateEnrollment(digest, username); err != nil {
		return time.Time{}, err
	}
	if league == "" || len(league) > 128 {
		return time.Time{}, identity.ErrEnrollmentRefused
	}
	canonicalIssuer, err := governance.AccountPrincipal(strings.TrimPrefix(string(issuer), "acct."))
	if err != nil || canonicalIssuer != issuer {
		return time.Time{}, identity.ErrEnrollmentRefused
	}
	tx, err := s.pools.Write().BeginTx(ctx, nil)
	if err != nil {
		return time.Time{}, fmt.Errorf("accounts: begin enrollment issue: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	now := s.now().UTC()
	expires := now.Add(identity.EnrollmentLifetime)
	result, err := tx.ExecContext(ctx, `INSERT INTO account_enrollments
 (code_hash, username, league, issuer, issued_at, expires_at)
 SELECT ?, ?, ?, ?, ?, ? WHERE EXISTS (SELECT 1 FROM accounts WHERE 'acct.' || id = ?)
 AND NOT EXISTS (SELECT 1 FROM accounts WHERE username = ?)`,
		digest, username, league, issuer, now.UnixNano(), expires.UnixNano(), issuer, username)
	if err != nil {
		return time.Time{}, fmt.Errorf("accounts: issue enrollment: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return time.Time{}, fmt.Errorf("accounts: enrollment issue result: %w", err)
	}
	if count != 1 {
		return time.Time{}, identity.ErrEnrollmentRefused
	}
	if err := tx.Commit(); err != nil {
		return time.Time{}, fmt.Errorf("accounts: commit enrollment issue: %w", err)
	}
	return expires, nil
}

// RedeemEnrollment admits after acquiring the single writer. Consumption and the
// new account commit together; any refusal or persistence error rolls back both.
func (s *Store) RedeemEnrollment(ctx context.Context, digest string, credential identity.Credential) error {
	if err := credential.Validate(); err != nil {
		return fmt.Errorf("accounts: enrollment credential: %w", err)
	}
	if err := validateEnrollment(digest, credential.Username); err != nil {
		return err
	}
	tx, err := s.pools.Write().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("accounts: begin enrollment redemption: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	now := s.now().UTC().UnixNano()
	result, err := tx.ExecContext(ctx, `INSERT INTO accounts(id, username, password_hash)
 SELECT ?, ?, ? WHERE EXISTS (SELECT 1 FROM account_enrollments
 WHERE code_hash = ? AND username = ? AND redeemed_account IS NULL AND issued_at <= ? AND expires_at > ?)
 ON CONFLICT(username) DO NOTHING`, credential.ID, credential.Username, credential.PasswordHash,
		digest, credential.Username, now, now)
	if err != nil {
		return fmt.Errorf("accounts: enroll account: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("accounts: enrollment result: %w", err)
	}
	if count != 1 {
		return identity.ErrEnrollmentRefused
	}
	if _, err := tx.ExecContext(ctx, `UPDATE account_enrollments SET redeemed_account = ? WHERE code_hash = ?`, credential.ID, digest); err != nil {
		return fmt.Errorf("accounts: consume enrollment: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("accounts: commit enrollment redemption: %w", err)
	}
	return nil
}
