package moves

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/secureprospective/TheWarRoom/internal/envelope"
)

// Save stores a new envelope or appends the audit entries not yet stored; saving the same state
// twice is a no-op, and a history that diverges from the stored one is refused.
func (s *Store) Save(ctx context.Context, e envelope.Envelope) error {
	receipt := e.Receipt()
	if _, err := envelope.Restore(e.ID(), e.Created(), receipt.Spec, receipt.Audit); err != nil {
		return fmt.Errorf("moves: save %q validate: %w", e.ID(), err)
	}
	tx, err := s.pools.Write().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("moves: save %q begin: %w", e.ID(), err)
	}
	defer func() { _ = tx.Rollback() }()
	stored, err := load(ctx, tx, e.ID())
	start := 0
	switch {
	case errors.Is(err, ErrNotFound):
		if err := insertEnvelope(ctx, tx, e); err != nil {
			return err
		}
	case err != nil:
		return err
	default:
		if err := checkPrefix(stored, e); err != nil {
			return err
		}
		start = len(stored.Receipt().Audit)
	}
	if err := appendAudit(ctx, tx, e.ID(), start, receipt.Audit); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("moves: save %q commit: %w", e.ID(), err)
	}
	return nil
}

func checkPrefix(stored, incoming envelope.Envelope) error {
	old, next := stored.Receipt(), incoming.Receipt()
	if !stored.Created().Equal(incoming.Created()) || !reflect.DeepEqual(old.Spec, next.Spec) {
		return fmt.Errorf("moves: save %q: envelope differs", incoming.ID())
	}
	if len(next.Audit) < len(old.Audit) || !reflect.DeepEqual(old.Audit, next.Audit[:len(old.Audit)]) {
		return fmt.Errorf("moves: save %q: stored audit prefix differs", incoming.ID())
	}
	return nil
}

func insertEnvelope(ctx context.Context, tx *sql.Tx, e envelope.Envelope) error {
	body, err := json.Marshal(e.Receipt().Spec)
	if err != nil {
		return fmt.Errorf("moves: save %q spec: %w", e.ID(), err)
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO move_envelopes (correlation_id, created, spec) VALUES (?, ?, ?)`,
		e.ID(), e.Created().Format(timestampLayout), string(body)); err != nil {
		return fmt.Errorf("moves: save %q envelope: %w", e.ID(), err)
	}
	return nil
}

func appendAudit(ctx context.Context, tx *sql.Tx, id string, start int, audit []envelope.AuditEntry) error {
	for seq := start; seq < len(audit); seq++ {
		entry := audit[seq]
		if _, err := tx.ExecContext(ctx, `
INSERT INTO move_audit (correlation_id, sequence, at, from_state, event, to_state, note)
VALUES (?, ?, ?, ?, ?, ?, ?)`,
			id, seq, entry.At.Format(timestampLayout), entry.From, entry.Event, entry.To, entry.Note); err != nil {
			return fmt.Errorf("moves: save %q audit entry %d: %w", id, seq, err)
		}
	}
	return nil
}
