package db

import (
	"context"
	"database/sql"
	"fmt"
)

// PreparedWrite holds a store's writer lock until Release. Persist changes only the transaction;
// Publish cannot fail and changes only memory while all participants' read locks are held.
type PreparedWrite struct {
	Persist                        func(context.Context, *sql.Tx) error
	Lock, Publish, Unlock, Release func()
}

// CommitPrepared commits related store writes before publishing any of their staged views.
func (p *Pools) CommitPrepared(ctx context.Context, writes ...PreparedWrite) error {
	defer func() {
		for i := len(writes) - 1; i >= 0; i-- {
			writes[i].Release()
		}
	}()
	tx, err := p.write.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("db: publication begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	for _, write := range writes {
		if err := write.Persist(ctx, tx); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("db: publication commit: %w", err)
	}
	for _, write := range writes {
		write.Lock()
	}
	for _, write := range writes {
		write.Publish()
	}
	for i := len(writes) - 1; i >= 0; i-- {
		writes[i].Unlock()
	}
	return nil
}
