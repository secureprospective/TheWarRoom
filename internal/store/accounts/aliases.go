package accounts

import (
	"context"
	"database/sql"
	"fmt"

	identity "github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/governance"
)

// Linked desktop identities are append-only: an identity belongs to one account for good.
func (s *Store) initializeAliases(ctx context.Context) error {
	_, err := s.pools.ExecWrite(ctx, func(tx *sql.Tx) (sql.Result, error) {
		return tx.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS account_aliases (
 alias TEXT PRIMARY KEY NOT NULL CHECK(length(alias) BETWEEN 1 AND 128 AND substr(alias, 1, 5) <> 'acct.'),
 account_id TEXT NOT NULL REFERENCES accounts(id),
 linked_at INTEGER NOT NULL
) STRICT;
CREATE TRIGGER IF NOT EXISTS account_aliases_immutable
 BEFORE UPDATE ON account_aliases
 BEGIN SELECT RAISE(ABORT, 'linked identities are immutable'); END;
CREATE TRIGGER IF NOT EXISTS account_aliases_no_delete
 BEFORE DELETE ON account_aliases
 BEGIN SELECT RAISE(ABORT, 'linked identities cannot be deleted'); END;`)
	})
	if err != nil {
		return fmt.Errorf("accounts: initialize linked identities: %w", err)
	}
	return nil
}

// BindAlias links alias to accountID once; linking it again to the same account is a no-op.
func (s *Store) BindAlias(ctx context.Context, accountID, alias string) error {
	if _, err := governance.AccountPrincipal(accountID); err != nil {
		return fmt.Errorf("accounts: link identity: %w", err)
	}
	tx, err := s.pools.BeginWrite(ctx)
	if err != nil {
		return fmt.Errorf("accounts: link identity: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var owner string
	err = tx.QueryRowContext(ctx, `SELECT COALESCE((SELECT account_id FROM account_aliases WHERE alias = ?), '')`, alias).Scan(&owner)
	if err != nil {
		return fmt.Errorf("accounts: read linked identity: %w", err)
	}
	switch owner {
	case accountID:
		return nil
	case "":
	default:
		return identity.ErrAliasTaken
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO account_aliases(alias, account_id, linked_at) VALUES (?, ?, ?)`,
		alias, accountID, s.now().UTC().UnixNano()); err != nil {
		return fmt.Errorf("accounts: link identity: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("accounts: link identity: %w", err)
	}
	return nil
}

func (s *Store) Aliases(ctx context.Context) ([]identity.Alias, error) {
	rows, err := s.pools.Read().QueryContext(ctx, `SELECT alias, account_id FROM account_aliases ORDER BY alias`)
	if err != nil {
		return nil, fmt.Errorf("accounts: read linked identities: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []identity.Alias
	for rows.Next() {
		var alias, account string
		if err := rows.Scan(&alias, &account); err != nil {
			return nil, fmt.Errorf("accounts: read linked identity: %w", err)
		}
		principal, err := governance.AccountPrincipal(account)
		if err != nil {
			return nil, fmt.Errorf("accounts: stored linked identity: %w", err)
		}
		out = append(out, identity.Alias{Alias: governance.Principal(alias), Account: principal})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("accounts: read linked identities: %w", err)
	}
	return out, nil
}

func (s *Store) SeatStatus(ctx context.Context, alias string) (identity.SeatStatus, error) {
	var status identity.SeatStatus
	err := s.pools.Read().QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM accounts),
 COALESCE((SELECT a.username FROM account_aliases l JOIN accounts a ON a.id = l.account_id WHERE l.alias = ?), '')`,
		alias).Scan(&status.HasAccount, &status.Username)
	if err != nil {
		return identity.SeatStatus{}, fmt.Errorf("accounts: read sign-in: %w", err)
	}
	return status, nil
}
