package accounts

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	identity "github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/governance"
)

func addHomeLayouts(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `CREATE TABLE account_home_layouts (
 account_id TEXT PRIMARY KEY NOT NULL REFERENCES accounts(id),
 layout TEXT NOT NULL CHECK(length(CAST(layout AS BLOB)) <= 8192),
 revision INTEGER NOT NULL CHECK(revision > 0)
) STRICT`)
	if err != nil {
		return fmt.Errorf("accounts: home layout schema: %w", err)
	}
	return nil
}

// ReadHomeLayout returns only the preference for a trusted authenticated account ID.
func (s *Store) ReadHomeLayout(ctx context.Context, accountID string) (identity.HomeLayoutState, error) {
	if _, err := governance.AccountPrincipal(accountID); err != nil {
		return identity.HomeLayoutState{}, fmt.Errorf("accounts: home layout identity: %w", err)
	}
	state, err := readHomeLayout(ctx, s.pools.Read(), accountID)
	if err != nil {
		return identity.HomeLayoutState{}, fmt.Errorf("accounts: read home layout: %w", err)
	}
	return state, nil
}

// SetHomeLayout atomically replaces one account's layout only at the expected revision.
func (s *Store) SetHomeLayout(ctx context.Context, accountID string, raw identity.RawHomeLayoutSet) (identity.HomeLayoutState, error) {
	if _, err := governance.AccountPrincipal(accountID); err != nil {
		return identity.HomeLayoutState{}, fmt.Errorf("accounts: home layout identity: %w", err)
	}
	if err := raw.Validate(); err != nil {
		return identity.HomeLayoutState{}, fmt.Errorf("accounts: home layout input: %w", err)
	}
	layout, err := identity.ValidateHomeLayout(raw.Layout)
	if err != nil {
		return identity.HomeLayoutState{}, fmt.Errorf("accounts: home layout input: %w", err)
	}
	tx, err := s.pools.BeginWrite(ctx)
	if err != nil {
		return identity.HomeLayoutState{}, fmt.Errorf("accounts: begin home layout write: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	result, err := tx.ExecContext(ctx, `INSERT INTO account_home_layouts(account_id, layout, revision)
 SELECT ?, ?, 1 WHERE ? = 0
 ON CONFLICT(account_id) DO UPDATE SET layout = excluded.layout, revision = account_home_layouts.revision + 1
 WHERE account_home_layouts.revision = ?`, accountID, string(layout), raw.ExpectedRevision, raw.ExpectedRevision)
	if err != nil {
		return identity.HomeLayoutState{}, fmt.Errorf("accounts: write home layout: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return identity.HomeLayoutState{}, fmt.Errorf("accounts: home layout result: %w", err)
	}
	if changed == 0 {
		current, err := readHomeLayout(ctx, tx, accountID)
		if err != nil {
			return identity.HomeLayoutState{}, fmt.Errorf("accounts: read conflicting home layout: %w", err)
		}
		current.Conflict = true
		return current, nil
	}
	if err := tx.Commit(); err != nil {
		return identity.HomeLayoutState{}, fmt.Errorf("accounts: commit home layout: %w", err)
	}
	return identity.HomeLayoutState{Layout: layout, Revision: raw.ExpectedRevision + 1}, nil
}

type homeLayoutQuerier interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func readHomeLayout(ctx context.Context, query homeLayoutQuerier, accountID string) (identity.HomeLayoutState, error) {
	var raw string
	var revision int64
	err := query.QueryRowContext(ctx, `SELECT layout, revision FROM account_home_layouts WHERE account_id = ?`, accountID).
		Scan(&raw, &revision)
	if errors.Is(err, sql.ErrNoRows) {
		return identity.HomeLayoutState{Layout: []byte(`{"mode":"default","tiles":[],"over":[],"pins":{}}`)}, nil
	}
	if err != nil {
		return identity.HomeLayoutState{}, fmt.Errorf("read stored preference: %w", err)
	}
	layout, err := identity.ValidateHomeLayout([]byte(raw))
	if err != nil {
		return identity.HomeLayoutState{}, fmt.Errorf("stored preference is invalid: %w", err)
	}
	if revision < 1 {
		return identity.HomeLayoutState{}, fmt.Errorf("stored preference has invalid revision")
	}
	return identity.HomeLayoutState{Layout: layout, Revision: revision}, nil
}
