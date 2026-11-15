package accounts

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	identity "github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/governance"
)

// addSeatClaims holds the one-time claims a moved league brings for each person it seats (identity
// SeatClaim), as digests only. Redeeming one deletes it.
func addSeatClaims(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, `CREATE TABLE league_seat_claims (
 league TEXT NOT NULL CHECK(length(league) BETWEEN 1 AND 128),
 principal TEXT NOT NULL CHECK(length(principal) > 5),
 franchise TEXT NOT NULL CHECK(length(franchise) BETWEEN 1 AND 64),
 name TEXT NOT NULL,
 code_hash TEXT NOT NULL UNIQUE CHECK(length(code_hash) = 64),
 issued_at INTEGER NOT NULL,
 PRIMARY KEY(league, principal)
) STRICT`); err != nil {
		return fmt.Errorf("accounts: seat claims: %w", err)
	}
	return nil
}

// SetSeatClaims replaces league's seat claims.
func (s *Store) SetSeatClaims(ctx context.Context, league string, claims []identity.StoredSeatClaim) error {
	tx, err := s.pools.BeginWrite(ctx)
	if err != nil {
		return fmt.Errorf("accounts: set seat claims: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM league_seat_claims WHERE league = ?`, league); err != nil {
		return fmt.Errorf("accounts: set seat claims: %w", err)
	}
	now := s.now().UTC().UnixNano()
	for _, claim := range claims {
		if !validSecretDigest(claim.Digest) {
			return identity.ErrJoinRefused
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO league_seat_claims(league, principal, franchise, name, code_hash, issued_at) VALUES(?,?,?,?,?,?)`,
			league, string(claim.Principal), claim.Franchise, claim.Name, claim.Digest, now); err != nil {
			return fmt.Errorf("accounts: set seat claims: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("accounts: set seat claims: %w", err)
	}
	return nil
}

// SeatClaim reads league's live claim whose code has digest; identity.ErrJoinRefused when none.
func (s *Store) SeatClaim(ctx context.Context, league, digest string) (identity.StoredSeatClaim, error) {
	var claim identity.StoredSeatClaim
	var principal string
	err := s.pools.Read().QueryRowContext(ctx, `SELECT principal, franchise, name, code_hash FROM league_seat_claims WHERE league = ? AND code_hash = ?`,
		league, digest).Scan(&principal, &claim.Franchise, &claim.Name, &claim.Digest)
	if errors.Is(err, sql.ErrNoRows) {
		return claim, identity.ErrJoinRefused
	}
	if err != nil {
		return claim, fmt.Errorf("accounts: read seat claim: %w", err)
	}
	claim.Principal = governance.Principal(principal)
	return claim, nil
}

// RedeemSeatClaim spends league's claim with digest and makes credential's account under the claimed
// identity, in one transaction: a taken username or a spent claim changes nothing.
func (s *Store) RedeemSeatClaim(ctx context.Context, league, digest string, credential identity.Credential) error {
	if err := credential.Validate(); err != nil {
		return fmt.Errorf("accounts: redeem seat claim: %w", err)
	}
	tx, err := s.pools.BeginWrite(ctx)
	if err != nil {
		return fmt.Errorf("accounts: redeem seat claim: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	spent, err := tx.ExecContext(ctx, `DELETE FROM league_seat_claims WHERE league = ? AND code_hash = ? AND principal = 'acct.' || ?`, league, digest, credential.ID)
	if err != nil {
		return fmt.Errorf("accounts: redeem seat claim: %w", err)
	}
	if n, err := spent.RowsAffected(); err != nil || n != 1 {
		return identity.ErrJoinRefused
	}
	var held bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM accounts WHERE id = ?)`, credential.ID).Scan(&held); err != nil {
		return fmt.Errorf("accounts: redeem seat claim: %w", err)
	}
	if held {
		return identity.ErrJoinRefused // this server already has that person's account
	}
	made, err := tx.ExecContext(ctx, `INSERT INTO accounts(id, username, password_hash) VALUES(?,?,?) ON CONFLICT(username) DO NOTHING`,
		credential.ID, credential.Username, credential.PasswordHash)
	if err != nil {
		return fmt.Errorf("accounts: redeem seat claim: %w", err)
	}
	if n, err := made.RowsAffected(); err != nil || n != 1 {
		return identity.ErrUsernameTaken
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("accounts: redeem seat claim: %w", err)
	}
	return nil
}

// SeatClaimsLeft counts league's claims not yet redeemed.
func (s *Store) SeatClaimsLeft(ctx context.Context, league string) (int, error) {
	var n int
	if err := s.pools.Read().QueryRowContext(ctx, `SELECT count(*) FROM league_seat_claims WHERE league = ?`, league).Scan(&n); err != nil {
		return 0, fmt.Errorf("accounts: count seat claims: %w", err)
	}
	return n, nil
}
