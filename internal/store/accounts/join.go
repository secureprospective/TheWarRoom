package accounts

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	identity "github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/governance"
)

// initializeJoin holds at most one live join link per league, as a digest only. Rotating replaces the
// row; revoking deletes it, so a revoked or replaced link stops matching at once.
func (s *Store) initializeJoin(ctx context.Context) error {
	_, err := s.pools.ExecWrite(ctx, func(tx *sql.Tx) (sql.Result, error) {
		return tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS league_join_links (
 league TEXT PRIMARY KEY CHECK(length(league) BETWEEN 1 AND 128),
 code_hash TEXT NOT NULL CHECK(length(code_hash) = 64),
 issuer TEXT NOT NULL CHECK(length(issuer) > 5),
 issued_at INTEGER NOT NULL
) STRICT`)
	})
	if err != nil {
		return fmt.Errorf("accounts: join link schema: %w", err)
	}
	return nil
}

// SetJoinLink stores league's join link digest, replacing any earlier link.
func (s *Store) SetJoinLink(ctx context.Context, league, digest string, issuer governance.Principal) error {
	canonical, err := governance.AccountPrincipal(strings.TrimPrefix(string(issuer), "acct."))
	if !validSecretDigest(digest) || league == "" || len(league) > 128 || err != nil || canonical != issuer {
		return identity.ErrJoinRefused
	}
	_, err = s.pools.ExecWrite(ctx, func(tx *sql.Tx) (sql.Result, error) {
		return tx.ExecContext(ctx, `INSERT INTO league_join_links(league, code_hash, issuer, issued_at)
 SELECT ?, ?, ?, ? WHERE EXISTS (SELECT 1 FROM accounts WHERE 'acct.' || id = ?)
 ON CONFLICT(league) DO UPDATE SET code_hash = excluded.code_hash, issuer = excluded.issuer, issued_at = excluded.issued_at`,
			league, digest, string(issuer), s.now().UTC().UnixNano(), string(issuer))
	})
	if err != nil {
		return fmt.Errorf("accounts: set join link: %w", err)
	}
	return nil
}

// RevokeJoinLink deletes league's join link; revoking an absent link is not an error.
func (s *Store) RevokeJoinLink(ctx context.Context, league string) error {
	_, err := s.pools.ExecWrite(ctx, func(tx *sql.Tx) (sql.Result, error) {
		return tx.ExecContext(ctx, `DELETE FROM league_join_links WHERE league = ?`, league)
	})
	if err != nil {
		return fmt.Errorf("accounts: revoke join link: %w", err)
	}
	return nil
}

// JoinLink returns league's live link digest and the commissioner who issued it.
func (s *Store) JoinLink(ctx context.Context, league string) (string, governance.Principal, error) {
	var digest, issuer string
	err := s.pools.Read().QueryRowContext(ctx, `SELECT code_hash, issuer FROM league_join_links WHERE league = ?`, league).Scan(&digest, &issuer)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", identity.ErrJoinRefused
	}
	if err != nil {
		return "", "", fmt.Errorf("accounts: read join link: %w", err)
	}
	return digest, governance.Principal(issuer), nil
}

// SignInName resolves an account principal's sign-in name for the join listing. A seat held by a
// non-account identity, or an unknown account, has no sign-in name to show.
func (s *Store) SignInName(ctx context.Context, accountID string) (string, error) {
	if _, err := governance.AccountPrincipal(accountID); err != nil {
		return "", identity.ErrNotFound
	}
	var username string
	err := s.pools.Read().QueryRowContext(ctx, `SELECT username FROM accounts WHERE id = ?`, accountID).Scan(&username)
	if errors.Is(err, sql.ErrNoRows) {
		return "", identity.ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("accounts: read sign-in name: %w", err)
	}
	return username, nil
}
