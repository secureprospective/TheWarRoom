// Package accounts persists credentials in the server-owned SQLite pool, never the league ledger.
package accounts

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	identity "github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/db"
)

type Store struct{ pools *db.Pools }

func New(pools *db.Pools) *Store { return &Store{pools: pools} }

// Initialize adds account-owned tables without changing the shared database's migration version.
func (s *Store) Initialize(ctx context.Context) error {
	_, err := s.pools.Write().ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS accounts (
 id TEXT PRIMARY KEY NOT NULL,
 username TEXT NOT NULL UNIQUE,
 password_hash TEXT NOT NULL
) STRICT;
CREATE TRIGGER IF NOT EXISTS accounts_identity_immutable
 BEFORE UPDATE OF id, username ON accounts
 BEGIN SELECT RAISE(ABORT, 'account identity is immutable'); END;
CREATE TRIGGER IF NOT EXISTS accounts_no_delete
 BEFORE DELETE ON accounts
 BEGIN SELECT RAISE(ABORT, 'accounts cannot be deleted'); END;`)
	if err != nil {
		return fmt.Errorf("accounts: initialize schema: %w", err)
	}
	return nil
}

// Insert serializes uniqueness and first-account bootstrap at the single writer.
func (s *Store) Insert(ctx context.Context, credential identity.Credential, onlyIfEmpty bool) error {
	if err := credential.Validate(); err != nil {
		return fmt.Errorf("accounts: insert: %w", err)
	}
	result, err := s.pools.Write().ExecContext(ctx, `
INSERT INTO accounts(id, username, password_hash)
 SELECT ?, ?, ? WHERE NOT ? OR NOT EXISTS (SELECT 1 FROM accounts)
 ON CONFLICT(username) DO NOTHING`, credential.ID, credential.Username, credential.PasswordHash, onlyIfEmpty)
	if err != nil {
		return fmt.Errorf("accounts: insert: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("accounts: insert result: %w", err)
	}
	if count == 0 {
		if onlyIfEmpty {
			return identity.ErrBootstrapClosed
		}
		return identity.ErrUsernameTaken
	}
	return nil
}

func (s *Store) Credential(ctx context.Context, username string) (identity.Credential, error) {
	var credential identity.Credential
	err := s.pools.Read().QueryRowContext(ctx,
		`SELECT id, username, password_hash FROM accounts WHERE username = ?`, username).
		Scan(&credential.ID, &credential.Username, &credential.PasswordHash)
	if errors.Is(err, sql.ErrNoRows) {
		return identity.Credential{}, identity.ErrNotFound
	}
	if err != nil {
		return identity.Credential{}, fmt.Errorf("accounts: read credential: %w", err)
	}
	if err := credential.Validate(); err != nil {
		return identity.Credential{}, fmt.Errorf("accounts: stored credential: %w", err)
	}
	return credential, nil
}
