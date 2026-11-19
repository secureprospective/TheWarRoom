package accounts

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	identity "github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/governance"
)

func addJobSnoozes(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `CREATE TABLE account_job_snoozes (
 account_id TEXT NOT NULL REFERENCES accounts(id),
 league TEXT NOT NULL CHECK(length(CAST(league AS BLOB)) BETWEEN 1 AND 128),
 job_id TEXT NOT NULL CHECK(length(CAST(job_id AS BLOB)) BETWEEN 1 AND 256),
 until TEXT NOT NULL CHECK(length(CAST(until AS BLOB)) <= 40),
 PRIMARY KEY(account_id, league, job_id)
) STRICT`)
	if err != nil {
		return fmt.Errorf("accounts: job snooze schema: %w", err)
	}
	return nil
}

// ReadJobSnoozes reads only the private marks for a trusted authenticated account.
func (s *Store) ReadJobSnoozes(ctx context.Context, accountID string) (identity.JobSnoozesState, error) {
	if _, err := governance.AccountPrincipal(accountID); err != nil {
		return identity.JobSnoozesState{}, fmt.Errorf("accounts: job snooze identity: %w", err)
	}
	state, err := readJobSnoozes(ctx, s.pools.Read(), accountID)
	if err != nil {
		return identity.JobSnoozesState{}, fmt.Errorf("accounts: read job snoozes: %w", err)
	}
	return state, nil
}

// SetJobSnooze replaces one account's reminder without touching the shared league job.
func (s *Store) SetJobSnooze(ctx context.Context, accountID string,
	raw identity.RawJobSnoozeSet) (identity.JobSnoozesState, error) {
	if _, err := governance.AccountPrincipal(accountID); err != nil {
		return identity.JobSnoozesState{}, fmt.Errorf("accounts: job snooze identity: %w", err)
	}
	if err := raw.Validate(); err != nil {
		return identity.JobSnoozesState{}, fmt.Errorf("accounts: job snooze input: %w", err)
	}
	until, err := time.Parse(time.RFC3339Nano, raw.Until)
	if err != nil {
		return identity.JobSnoozesState{}, fmt.Errorf("accounts: job snooze time: %w", err)
	}
	now := s.now().UTC()
	if !until.After(now) {
		return identity.JobSnoozesState{}, fmt.Errorf("accounts: snooze time must be in the future")
	}
	tx, err := s.pools.BeginWrite(ctx)
	if err != nil {
		return identity.JobSnoozesState{}, fmt.Errorf("accounts: begin job snooze write: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM account_job_snoozes
 WHERE account_id = ? AND julianday(until) <= julianday(?)`, accountID, now.Format(time.RFC3339Nano)); err != nil {
		return identity.JobSnoozesState{}, fmt.Errorf("accounts: clear expired job snoozes: %w", err)
	}
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM account_job_snoozes WHERE account_id = ?`, accountID).
		Scan(&count); err != nil {
		return identity.JobSnoozesState{}, fmt.Errorf("accounts: count job snoozes: %w", err)
	}
	if count >= identity.MaxAccountJobSnoozes {
		var exists int
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM account_job_snoozes
 WHERE account_id = ? AND league = ? AND job_id = ?)`, accountID, raw.League, raw.JobID).Scan(&exists); err != nil {
			return identity.JobSnoozesState{}, fmt.Errorf("accounts: check existing job snooze: %w", err)
		}
		if exists == 0 {
			return identity.JobSnoozesState{}, fmt.Errorf("accounts: account has too many active job snoozes")
		}
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO account_job_snoozes(account_id, league, job_id, until)
 VALUES (?, ?, ?, ?) ON CONFLICT(account_id, league, job_id) DO UPDATE SET until=excluded.until`,
		accountID, raw.League, raw.JobID, until.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return identity.JobSnoozesState{}, fmt.Errorf("accounts: write job snooze: %w", err)
	}
	state, err := readJobSnoozes(ctx, tx, accountID)
	if err != nil {
		return identity.JobSnoozesState{}, fmt.Errorf("accounts: read written job snoozes: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return identity.JobSnoozesState{}, fmt.Errorf("accounts: commit job snooze: %w", err)
	}
	return state, nil
}

// WakeJobSnooze clears one account's reminder and leaves the shared job intact.
func (s *Store) WakeJobSnooze(ctx context.Context, accountID string,
	raw identity.RawJobSnoozeWake) (identity.JobSnoozesState, error) {
	if _, err := governance.AccountPrincipal(accountID); err != nil {
		return identity.JobSnoozesState{}, fmt.Errorf("accounts: job snooze identity: %w", err)
	}
	if err := raw.Validate(); err != nil {
		return identity.JobSnoozesState{}, fmt.Errorf("accounts: job snooze input: %w", err)
	}
	tx, err := s.pools.BeginWrite(ctx)
	if err != nil {
		return identity.JobSnoozesState{}, fmt.Errorf("accounts: begin job wake: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM account_job_snoozes
 WHERE account_id = ? AND league = ? AND job_id = ?`, accountID, raw.League, raw.JobID); err != nil {
		return identity.JobSnoozesState{}, fmt.Errorf("accounts: wake job snooze: %w", err)
	}
	state, err := readJobSnoozes(ctx, tx, accountID)
	if err != nil {
		return identity.JobSnoozesState{}, fmt.Errorf("accounts: read job snoozes after wake: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return identity.JobSnoozesState{}, fmt.Errorf("accounts: commit job wake: %w", err)
	}
	return state, nil
}

type jobSnoozeQuerier interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func readJobSnoozes(ctx context.Context, query jobSnoozeQuerier, accountID string) (identity.JobSnoozesState, error) {
	rows, err := query.QueryContext(ctx, `SELECT league, job_id, until FROM account_job_snoozes
 WHERE account_id = ? ORDER BY league, job_id`, accountID)
	if err != nil {
		return identity.JobSnoozesState{}, fmt.Errorf("query private job snoozes: %w", err)
	}
	defer func() { _ = rows.Close() }()
	state := identity.JobSnoozesState{Snoozes: []identity.JobSnooze{}}
	for rows.Next() {
		var snooze identity.JobSnooze
		if err := rows.Scan(&snooze.League, &snooze.JobID, &snooze.Until); err != nil {
			return identity.JobSnoozesState{}, fmt.Errorf("read private job snooze: %w", err)
		}
		if err := (identity.RawJobSnoozeSet(snooze)).Validate(); err != nil {
			return identity.JobSnoozesState{}, fmt.Errorf("stored job snooze is invalid: %w", err)
		}
		state.Snoozes = append(state.Snoozes, snooze)
	}
	if err := rows.Err(); err != nil {
		return identity.JobSnoozesState{}, fmt.Errorf("read private job snoozes: %w", err)
	}
	return state, nil
}
