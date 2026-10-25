package accounts

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
	"unicode"
)

// ResetSignIns is the operator-only incident port. A new auth epoch ends every device
// session, helper token and recent second-factor proof issued before it; passwords and
// authenticators are kept. Never expose it as a league command or an HTTP route.
func (s *Store) ResetSignIns(ctx context.Context, reason string) (int, error) {
	return s.resetAuth(ctx, reason, "sign_ins_reset", false)
}

// ResetAfterRestore runs on a restored copy before any listener opens. Besides the new
// epoch it removes what the copy may have revived: a printed recovery code or invitation
// used after the snapshot reads as unused in it, so every printed code and unredeemed
// invitation goes. Owners print new codes after signing in with password and authenticator.
func (s *Store) ResetAfterRestore(ctx context.Context, reason string) (int, error) {
	return s.resetAuth(ctx, reason, "restored_sign_ins_reset", true)
}

func (s *Store) resetAuth(ctx context.Context, reason, operation string, restored bool) (int, error) {
	if reason == "" || len(reason) > 256 || strings.IndexFunc(reason, func(r rune) bool { return !unicode.IsPrint(r) }) >= 0 {
		return 0, fmt.Errorf("accounts: a printable reason of up to 256 characters is required")
	}
	epoch, err := newAuthEpoch()
	if err != nil {
		return 0, err
	}
	tx, err := s.pools.BeginWrite(ctx)
	if err != nil {
		return 0, fmt.Errorf("accounts: begin sign-in reset: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	statements := []string{`UPDATE account_auth_epoch SET epoch = ? WHERE singleton = 1`,
		`UPDATE account_totp SET proof_hash = '', authenticated_at = 0`}
	if restored {
		statements = append(statements, `DELETE FROM account_recovery_codes`,
			`DELETE FROM account_enrollments WHERE redeemed_account IS NULL`)
	}
	for i, statement := range statements {
		args := []any{}
		if i == 0 {
			args = append(args, epoch)
		}
		if _, err := tx.ExecContext(ctx, statement, args...); err != nil {
			return 0, fmt.Errorf("accounts: reset sign-ins: %w", err)
		}
	}
	ids, err := accountIDs(ctx, tx)
	if err != nil {
		return 0, err
	}
	now := s.now().UTC().UnixNano()
	for _, id := range ids {
		if err := recoveryAudit(ctx, tx, id, operation, reason, now); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("accounts: commit sign-in reset: %w", err)
	}
	return len(ids), nil
}

func accountIDs(ctx context.Context, tx *sql.Tx) ([]string, error) {
	rows, err := tx.QueryContext(ctx, `SELECT id FROM accounts ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("accounts: list accounts: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("accounts: list accounts: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("accounts: list accounts: %w", err)
	}
	return ids, nil
}

// Reset is the newest sign-in reset of one kind, read back from the sign-in history.
// The zero value means none has happened on this league's history.
type Reset struct {
	At       time.Time
	Reason   string
	Accounts int
}

// LastResets reads the newest spare-host restore and the newest incident reset for the
// private dashboard. Both are hash-chained history entries, so they survive restarts
// and ride along in every off-site backup.
func (s *Store) LastResets(ctx context.Context) (restore, incident Reset, err error) {
	if restore, err = s.lastReset(ctx, "restored_sign_ins_reset"); err != nil {
		return Reset{}, Reset{}, err
	}
	if incident, err = s.lastReset(ctx, "sign_ins_reset"); err != nil {
		return Reset{}, Reset{}, err
	}
	return restore, incident, nil
}

func (s *Store) lastReset(ctx context.Context, operation string) (Reset, error) {
	var reason sql.NullString
	var at sql.NullInt64
	var count int
	err := s.pools.Read().QueryRowContext(ctx, `SELECT MAX(reason), MAX(occurred_at), COUNT(*) FROM account_security_audit
 WHERE operation = ?1 AND occurred_at = (SELECT MAX(occurred_at) FROM account_security_audit WHERE operation = ?1)`, operation).Scan(&reason, &at, &count)
	if err != nil {
		return Reset{}, fmt.Errorf("accounts: read last sign-in reset: %w", err)
	}
	if count == 0 {
		return Reset{}, nil
	}
	return Reset{At: time.Unix(0, at.Int64).UTC(), Reason: reason.String, Accounts: count}, nil
}
