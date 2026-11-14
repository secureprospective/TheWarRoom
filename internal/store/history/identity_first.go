package history

import (
	"context"
	"errors"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// identityFirst runs a league write that only reads player identity. When the write stops on players
// this file has no identity for, it has rolled back; the identities are then minted and committed on
// their own (reference data first) and the write runs once more against durable ids. A crash between
// the two commits leaves only unused identity rows, which the next attempt reuses.
func identityFirst[T any](ctx context.Context, s *Store, write func() (T, error)) (T, error) {
	out, err := write()
	var missing *db.UnresolvedPlayersError
	if !errors.As(err, &missing) {
		return out, err
	}
	if err := s.mintPlayers(ctx, missing.MFL); err != nil {
		var zero T
		return zero, err
	}
	return write()
}

func (s *Store) mintPlayers(ctx context.Context, mfl []string) error {
	s.wmu.Lock()
	defer s.wmu.Unlock()
	tx, err := s.pools.BeginWrite(ctx)
	if err != nil {
		return fmt.Errorf("begin player identities: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := db.EnsureMFLPlayers(ctx, tx, mfl); err != nil {
		return fmt.Errorf("save league ownership: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit player identities: %w", err)
	}
	return nil
}

// commitReference is a transfer's first commit: its permanent identities and content-addressed
// evidence, before any league row.
func (s *Store) commitReference(ctx context.Context, identities []playerid.RawIdentity, blobs map[string][]byte, wrap string) error {
	tx, err := s.pools.BeginWrite(ctx)
	if err != nil {
		return fmt.Errorf("begin %s: %w", wrap, err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := db.MergePlayerIdentities(ctx, tx, identities); err != nil {
		return fmt.Errorf("%s permanent player identities: %w", wrap, err)
	}
	for key, body := range blobs {
		if err := stageArchive(ctx, tx, key, body); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit %s reference data: %w", wrap, err)
	}
	return nil
}
