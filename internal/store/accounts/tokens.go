package accounts

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	identity "github.com/secureprospective/TheWarRoom/internal/accounts"
)

func (s *Store) initializeTokens(ctx context.Context) error {
	_, err := s.pools.ExecWrite(ctx, func(tx *sql.Tx) (sql.Result, error) {
		return tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS account_agent_tokens (
 token_hash TEXT PRIMARY KEY CHECK(length(token_hash) = 64),
 id TEXT UNIQUE NOT NULL, account_id TEXT NOT NULL REFERENCES accounts(id),
 scope TEXT NOT NULL, generation INTEGER NOT NULL, epoch TEXT NOT NULL,
 issued_at INTEGER NOT NULL, expires_at INTEGER NOT NULL,
 revoked INTEGER NOT NULL DEFAULT 0 CHECK(revoked IN (0,1)), CHECK(expires_at > issued_at)
 ) STRICT;
 CREATE TRIGGER IF NOT EXISTS agent_token_ceiling_immutable
 BEFORE UPDATE OF token_hash,id,account_id,scope,generation,epoch,issued_at,expires_at ON account_agent_tokens
 BEGIN SELECT RAISE(ABORT, 'agent issuance ceiling is immutable'); END;`)
	})
	if err != nil {
		return fmt.Errorf("accounts: token schema: %w", err)
	}
	return nil
}

func (s *Store) IssueToken(ctx context.Context, sessionDigest, digest, id string, scope identity.RawTokenScope, authorize func(identity.Account) error) (identity.AgentToken, error) {
	if !validSecretDigest(sessionDigest) || !validSecretDigest(digest) || (identity.RawSession{Bearer: id}).Validate() != nil || scope.Validate() != nil || authorize == nil {
		return identity.AgentToken{}, identity.ErrTokenRefused
	}
	tx, err := s.pools.BeginWrite(ctx)
	if err != nil {
		return identity.AgentToken{}, fmt.Errorf("accounts: begin token: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	account, err := s.readSession(ctx, tx, sessionDigest)
	if err != nil {
		return identity.AgentToken{}, err
	}
	if err := authorize(account); err != nil {
		return identity.AgentToken{}, fmt.Errorf("accounts: token issuer rights: %w", err)
	}
	body, err := json.Marshal(scope)
	if err != nil {
		return identity.AgentToken{}, fmt.Errorf("accounts: encode scope: %w", err)
	}
	now := s.now().UTC()
	expires := now.Add(scope.Lifetime())
	_, err = tx.ExecContext(ctx, `INSERT INTO account_agent_tokens(token_hash,id,account_id,scope,generation,epoch,issued_at,expires_at)
 SELECT ?,?,?,?,COALESCE((SELECT generation FROM account_auth_generation WHERE account_id = ?),0),
 (SELECT epoch FROM account_auth_epoch WHERE singleton = 1),?,?`, digest, id, account.ID, string(body), account.ID, now.UnixNano(), expires.UnixNano())
	if err != nil {
		return identity.AgentToken{}, fmt.Errorf("accounts: save token: %w", err)
	}
	if err := recoveryAudit(ctx, tx, account.ID, "agent_token_issued", id, now.UnixNano()); err != nil {
		return identity.AgentToken{}, err
	}
	if err := tx.Commit(); err != nil {
		return identity.AgentToken{}, fmt.Errorf("accounts: commit token: %w", err)
	}
	return identity.AgentToken{ID: id, Account: account, Scope: scope, ExpiresAt: expires}, nil
}

// The independent reader is safe under history's shared single writer. Revocation,
// recovery and epoch changes cannot commit between this recheck and ledger append.
func (s *Store) Token(ctx context.Context, digest string) (identity.AgentToken, error) {
	if !validSecretDigest(digest) {
		return identity.AgentToken{}, identity.ErrTokenRefused
	}
	var token identity.AgentToken
	var body string
	var issued, expires int64
	err := s.pools.Read().QueryRowContext(ctx, `SELECT t.id,a.id,a.username,t.scope,t.issued_at,t.expires_at
 FROM account_agent_tokens t JOIN accounts a ON a.id = t.account_id
 JOIN account_auth_epoch e ON e.singleton = 1 AND e.epoch = t.epoch
 WHERE t.token_hash = ? AND t.revoked = 0 AND t.generation = COALESCE((SELECT generation FROM account_auth_generation WHERE account_id = a.id),0)`, digest).Scan(&token.ID, &token.Account.ID, &token.Account.Username, &body, &issued, &expires)
	if errors.Is(err, sql.ErrNoRows) {
		return identity.AgentToken{}, identity.ErrTokenRefused
	}
	if err != nil {
		return identity.AgentToken{}, fmt.Errorf("accounts: read token: %w", err)
	}
	if err := json.Unmarshal([]byte(body), &token.Scope); err != nil {
		return identity.AgentToken{}, fmt.Errorf("accounts: decode token scope: %w", err)
	}
	token.ExpiresAt = time.Unix(0, expires).UTC()
	now := s.now().UTC()
	if token.Scope.Validate() != nil || now.Before(time.Unix(0, issued)) || !now.Before(token.ExpiresAt) || token.ExpiresAt.Sub(time.Unix(0, issued)) != token.Scope.Lifetime() {
		return identity.AgentToken{}, identity.ErrTokenRefused
	}
	return token, nil
}

func (s *Store) RevokeToken(ctx context.Context, sessionDigest, id string) error {
	tx, err := s.pools.BeginWrite(ctx)
	if err != nil {
		return fmt.Errorf("accounts: begin token revoke: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	account, err := s.readSession(ctx, tx, sessionDigest)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `UPDATE account_agent_tokens SET revoked = 1 WHERE id = ? AND account_id = ?`, id, account.ID)
	if err != nil {
		return fmt.Errorf("accounts: revoke token: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("accounts: revoke token result: %w", err)
	}
	if count != 1 {
		return identity.ErrTokenRefused
	}
	if err := recoveryAudit(ctx, tx, account.ID, "agent_token_revoked", id, s.now().UTC().UnixNano()); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("accounts: commit token revoke: %w", err)
	}
	return nil
}
