package accounts

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	identity "github.com/secureprospective/TheWarRoom/internal/accounts"
)

func (s *Store) initializeSessions(ctx context.Context) error {
	epoch, err := newAuthEpoch()
	if err != nil {
		return err
	}
	_, err = s.pools.Write().ExecContext(ctx, `
 CREATE TABLE IF NOT EXISTS account_auth_epoch (
 singleton INTEGER PRIMARY KEY CHECK(singleton = 1), epoch TEXT NOT NULL CHECK(length(epoch) = 64)
 ) STRICT;
 CREATE TABLE IF NOT EXISTS account_sessions (
 token_hash TEXT PRIMARY KEY CHECK(length(token_hash) = 64),
 account_id TEXT NOT NULL REFERENCES accounts(id), device TEXT NOT NULL,
 generation INTEGER NOT NULL, epoch TEXT NOT NULL,
 issued_at INTEGER NOT NULL, last_used_at INTEGER NOT NULL,
 revoked INTEGER NOT NULL DEFAULT 0 CHECK(revoked IN (0,1))
 ) STRICT;`)
	if err != nil {
		return fmt.Errorf("accounts: session schema: %w", err)
	}
	if _, err := s.pools.Write().ExecContext(ctx, `INSERT INTO account_auth_epoch(singleton, epoch) VALUES (1, ?) ON CONFLICT(singleton) DO NOTHING`, epoch); err != nil {
		return fmt.Errorf("accounts: initialize auth epoch: %w", err)
	}
	return nil
}
func newAuthEpoch() (string, error) {
	var epoch [32]byte
	if _, err := rand.Read(epoch[:]); err != nil {
		return "", fmt.Errorf("accounts: generate auth epoch: %w", err)
	}
	return hex.EncodeToString(epoch[:]), nil
}

// RotateAuthEpoch is an operator-only restore/incident port. Never expose it as a
// league command. A restored file must rotate before accepting credentials.
func (s *Store) RotateAuthEpoch(ctx context.Context) error {
	epoch, err := newAuthEpoch()
	if err != nil {
		return err
	}
	if _, err := s.pools.Write().ExecContext(ctx, `UPDATE account_auth_epoch SET epoch = ? WHERE singleton = 1`, epoch); err != nil {
		return fmt.Errorf("accounts: rotate auth epoch: %w", err)
	}
	return nil
}

func (s *Store) CreateSession(ctx context.Context, credential identity.Credential, digest, device string) (time.Time, error) {
	if err := credential.Validate(); err != nil {
		return time.Time{}, fmt.Errorf("accounts: session credential: %w", err)
	}
	if !validSecretDigest(digest) || !identity.ValidDeviceName(device) {
		return time.Time{}, identity.ErrSessionRefused
	}
	tx, err := s.pools.Write().BeginTx(ctx, nil)
	if err != nil {
		return time.Time{}, fmt.Errorf("accounts: begin session: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	now := s.now().UTC()
	result, err := tx.ExecContext(ctx, `INSERT INTO account_sessions(token_hash,account_id,device,generation,epoch,issued_at,last_used_at)
 SELECT ?, id, ?, COALESCE((SELECT generation FROM account_auth_generation WHERE account_id = accounts.id),0),
 (SELECT epoch FROM account_auth_epoch WHERE singleton = 1), ?, ? FROM accounts WHERE id = ? AND password_hash = ?`,
		digest, device, now.UnixNano(), now.UnixNano(), credential.ID, credential.PasswordHash)
	if err != nil {
		return time.Time{}, fmt.Errorf("accounts: save session: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return time.Time{}, fmt.Errorf("accounts: session result: %w", err)
	}
	if count != 1 {
		return time.Time{}, identity.ErrSessionRefused
	}
	if err := tx.Commit(); err != nil {
		return time.Time{}, fmt.Errorf("accounts: commit session: %w", err)
	}
	return now.Add(identity.SessionAbsoluteLifetime), nil
}

type sessionRow struct {
	account      identity.Account
	issued, used int64
}

// Read admission uses the independent reader pool while the history writer is held.
// Revocations share that single writer, so none can commit between this check and append.
func (s *Store) SessionAccount(ctx context.Context, digest string, touch bool) (identity.Account, error) {
	if !validSecretDigest(digest) {
		return identity.Account{}, identity.ErrSessionRefused
	}
	if !touch {
		return s.readSession(ctx, nil, digest)
	}
	tx, err := s.pools.Write().BeginTx(ctx, nil)
	if err != nil {
		return identity.Account{}, fmt.Errorf("accounts: begin activity: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	account, err := s.readSession(ctx, tx, digest)
	if err != nil {
		return identity.Account{}, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE account_sessions SET last_used_at = ? WHERE token_hash = ?`, s.now().UTC().UnixNano(), digest); err != nil {
		return identity.Account{}, fmt.Errorf("accounts: record activity: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return identity.Account{}, fmt.Errorf("accounts: commit activity: %w", err)
	}
	return account, nil
}

func (s *Store) readSession(ctx context.Context, tx *sql.Tx, digest string) (identity.Account, error) {
	var row sessionRow
	const query = `SELECT a.id,a.username,s.issued_at,s.last_used_at FROM account_sessions s
 JOIN accounts a ON a.id = s.account_id JOIN account_auth_epoch e ON e.singleton = 1 AND e.epoch = s.epoch
 WHERE s.token_hash = ? AND s.revoked = 0 AND s.generation = COALESCE((SELECT generation FROM account_auth_generation WHERE account_id = a.id),0)`
	result := s.pools.Read().QueryRowContext
	if tx != nil {
		result = tx.QueryRowContext
	}
	err := result(ctx, query, digest).Scan(&row.account.ID, &row.account.Username, &row.issued, &row.used)
	if errors.Is(err, sql.ErrNoRows) {
		return identity.Account{}, identity.ErrSessionRefused
	}
	if err != nil {
		return identity.Account{}, fmt.Errorf("accounts: read session: %w", err)
	}
	now := s.now().UTC()
	issued, used := time.Unix(0, row.issued), time.Unix(0, row.used)
	if now.Before(issued) || now.Before(used) || !now.Before(issued.Add(identity.SessionAbsoluteLifetime)) || !now.Before(used.Add(identity.SessionIdleLifetime)) {
		return identity.Account{}, identity.ErrSessionRefused
	}
	return row.account, nil
}

func (s *Store) RevokeSession(ctx context.Context, digest string, allDevices bool) error {
	if !validSecretDigest(digest) {
		return identity.ErrSessionRefused
	}
	tx, err := s.pools.Write().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("accounts: begin revoke: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	account, err := s.readSession(ctx, tx, digest)
	if err != nil {
		return err
	}
	if allDevices {
		_, err = tx.ExecContext(ctx, `UPDATE account_sessions SET revoked = 1 WHERE account_id = ?`, account.ID)
	} else {
		_, err = tx.ExecContext(ctx, `UPDATE account_sessions SET revoked = 1 WHERE token_hash = ?`, digest)
	}
	if err != nil {
		return fmt.Errorf("accounts: revoke sessions: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("accounts: commit revoke: %w", err)
	}
	return nil
}
