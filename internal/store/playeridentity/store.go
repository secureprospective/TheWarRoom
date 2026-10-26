// Package playeridentity owns permanent player identities, provider links and
// side-qualified ownership. It shares the history file, not the history Store.
package playeridentity

import (
	"context"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

type Store struct{ pools *db.Pools }

// New uses the history-file migrations; it never independently creates schema.
func New(pools *db.Pools) *Store { return &Store{pools: pools} }

// Resolve returns our ID, not the provider value. Missing links remain errors.
func (s *Store) Resolve(ctx context.Context, provider, value string) (string, error) {
	if provider == "" || value == "" {
		return "", fmt.Errorf("player link requires provider and value")
	}
	var id string
	if err := s.pools.Read().QueryRowContext(ctx, `SELECT player FROM player_external_id WHERE provider=? AND value=?`, provider, value).Scan(&id); err != nil {
		return "", fmt.Errorf("resolve outside player ID: %w", err)
	}
	return id, nil
}

// Link attaches an additional provider identifier without guessing or merging
// people. Conflicting links fail; ingestion corrections use their checked adapter.
func (s *Store) Link(ctx context.Context, raw playerid.RawIdentity) error {
	if err := raw.Validate(); err != nil {
		return fmt.Errorf("validate player links: %w", err)
	}
	tx, err := s.pools.BeginWrite(ctx)
	if err != nil {
		return fmt.Errorf("begin player links: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var exists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM canonical_player WHERE id=?)`, raw.Canonical).Scan(&exists); err != nil {
		return fmt.Errorf("check canonical player: %w", err)
	}
	if !exists {
		return fmt.Errorf("choose an existing canonical player")
	}
	for _, link := range raw.Links {
		var conflict bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM player_external_id WHERE provider=? AND value=? AND player<>?)`, link.Provider, link.Value, raw.Canonical).Scan(&conflict); err != nil {
			return fmt.Errorf("check outside player link: %w", err)
		}
		if conflict {
			return fmt.Errorf("outside ID already belongs to another player")
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO player_external_id(provider,value,player) VALUES(?,?,?) ON CONFLICT(provider,value) DO NOTHING`, link.Provider, link.Value, raw.Canonical); err != nil {
			return fmt.Errorf("save outside player link: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit player links: %w", err)
	}
	return nil
}

// Create creates an identity even when no MFL link exists.
func (s *Store) Create(ctx context.Context, links []playerid.RawOutsideID) (string, error) {
	tx, err := s.pools.BeginWrite(ctx)
	if err != nil {
		return "", fmt.Errorf("begin player identity: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	id, err := db.NewPlayerIdentity(ctx, tx, links)
	if err != nil {
		return "", fmt.Errorf("create player identity: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("commit player identity: %w", err)
	}
	return id, nil
}
