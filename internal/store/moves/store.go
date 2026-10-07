// Package moves persists envelopes and their replayable, append-only audit histories.
package moves

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
)

var ErrNotFound = errors.New("moves: not found")

const timestampLayout = "2006-01-02T15:04:05.000000000Z"

// Store keeps envelopes in thewarroom.db; triggers make both tables append-only.
type Store struct{ pools *db.Pools }

func New(pools *db.Pools) *Store { return &Store{pools: pools} }

func (s *Store) Initialize(ctx context.Context) error {
	const ddl = `
CREATE TABLE IF NOT EXISTS move_envelopes (
 correlation_id TEXT PRIMARY KEY,
 created TEXT NOT NULL,
 spec TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS move_audit (
 correlation_id TEXT NOT NULL REFERENCES move_envelopes(correlation_id),
 sequence INTEGER NOT NULL CHECK (sequence >= 0),
 at TEXT NOT NULL,
 from_state TEXT NOT NULL,
 event TEXT NOT NULL,
 to_state TEXT NOT NULL,
 note TEXT NOT NULL,
 PRIMARY KEY (correlation_id, sequence)
);
CREATE TRIGGER IF NOT EXISTS move_envelopes_no_update
BEFORE UPDATE ON move_envelopes
BEGIN SELECT RAISE(ABORT, 'move_envelopes is append-only'); END;
CREATE TRIGGER IF NOT EXISTS move_envelopes_no_delete
BEFORE DELETE ON move_envelopes
BEGIN SELECT RAISE(ABORT, 'move_envelopes is append-only'); END;
CREATE TRIGGER IF NOT EXISTS move_audit_no_update
BEFORE UPDATE ON move_audit
BEGIN SELECT RAISE(ABORT, 'move_audit is append-only'); END;
CREATE TRIGGER IF NOT EXISTS move_audit_no_delete
BEFORE DELETE ON move_audit
BEGIN SELECT RAISE(ABORT, 'move_audit is append-only'); END;`
	if _, err := s.pools.Write().ExecContext(ctx, ddl); err != nil {
		return fmt.Errorf("moves: initialize: %w", err)
	}
	return nil
}

// Get restores one envelope; a stored history the transition table rejects is an error.
func (s *Store) Get(ctx context.Context, id string) (envelope.Envelope, error) {
	tx, err := s.pools.Read().BeginTx(ctx, nil)
	if err != nil {
		return envelope.Envelope{}, fmt.Errorf("moves: get %q begin: %w", id, err)
	}
	defer func() { _ = tx.Rollback() }()
	e, err := load(ctx, tx, id)
	if err != nil {
		return envelope.Envelope{}, err
	}
	if err := tx.Commit(); err != nil {
		return envelope.Envelope{}, fmt.Errorf("moves: get %q commit: %w", id, err)
	}
	return e, nil
}

func load(ctx context.Context, tx *sql.Tx, id string) (envelope.Envelope, error) {
	var created, body string
	err := tx.QueryRowContext(ctx,
		`SELECT created, spec FROM move_envelopes WHERE correlation_id = ?`, id).Scan(&created, &body)
	if errors.Is(err, sql.ErrNoRows) {
		return envelope.Envelope{}, fmt.Errorf("moves: get %q: %w", id, ErrNotFound)
	}
	if err != nil {
		return envelope.Envelope{}, fmt.Errorf("moves: get %q row: %w", id, err)
	}
	at, err := time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return envelope.Envelope{}, fmt.Errorf("moves: get %q created: %w", id, err)
	}
	var spec envelope.Spec
	if err := json.Unmarshal([]byte(body), &spec); err != nil {
		return envelope.Envelope{}, fmt.Errorf("moves: get %q spec: %w", id, err)
	}
	audit, err := loadAudit(ctx, tx, id)
	if err != nil {
		return envelope.Envelope{}, err
	}
	e, err := envelope.Restore(id, at, spec, audit)
	if err != nil {
		return envelope.Envelope{}, fmt.Errorf("moves: get %q restore: %w", id, err)
	}
	return e, nil
}

func loadAudit(ctx context.Context, tx *sql.Tx, id string) ([]envelope.AuditEntry, error) {
	rows, err := tx.QueryContext(ctx, `
SELECT sequence, at, from_state, event, to_state, note FROM move_audit
WHERE correlation_id = ? ORDER BY sequence`, id)
	if err != nil {
		return nil, fmt.Errorf("moves: get %q audit: %w", id, err)
	}
	defer func() { _ = rows.Close() }()
	audit := []envelope.AuditEntry{}
	for rows.Next() {
		var seq int
		var stamp string
		var entry envelope.AuditEntry
		if err := rows.Scan(&seq, &stamp, &entry.From, &entry.Event, &entry.To, &entry.Note); err != nil {
			return nil, fmt.Errorf("moves: get %q audit scan: %w", id, err)
		}
		if seq != len(audit) {
			return nil, fmt.Errorf("moves: get %q audit entry %d: sequence %d", id, len(audit), seq)
		}
		entry.At, err = time.Parse(time.RFC3339Nano, stamp)
		if err != nil {
			return nil, fmt.Errorf("moves: get %q audit entry %d time: %w", id, seq, err)
		}
		audit = append(audit, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("moves: get %q audit rows: %w", id, err)
	}
	return audit, nil
}

// List returns each envelope's latest receipt, newest first; empty filters match all.
func (s *Store) List(ctx context.Context, league, franchise string) ([]envelope.Receipt, error) {
	entries, err := s.selectEnvelopes(ctx, league, franchise, false)
	if err != nil {
		return nil, err
	}
	receipts := make([]envelope.Receipt, 0, len(entries))
	for _, entry := range entries {
		receipts = append(receipts, entry.Receipt())
	}
	return receipts, nil
}

// Awaiting returns envelopes whose state waits on MFL evidence: the landing watcher's work.
func (s *Store) Awaiting(ctx context.Context) ([]envelope.Envelope, error) {
	return s.selectEnvelopes(ctx, "", "", true)
}

func (s *Store) selectEnvelopes(
	ctx context.Context, league, franchise string, awaiting bool,
) ([]envelope.Envelope, error) {
	tx, err := s.pools.Read().BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("moves: list begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	ids, err := selectIDs(ctx, tx, league, franchise, awaiting)
	if err != nil {
		return nil, err
	}
	entries := make([]envelope.Envelope, 0, len(ids))
	for _, id := range ids {
		e, err := load(ctx, tx, id)
		if err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("moves: list commit: %w", err)
	}
	return entries, nil
}

func selectIDs(ctx context.Context, tx *sql.Tx, league, franchise string, awaiting bool) ([]string, error) {
	rows, err := tx.QueryContext(ctx, `
SELECT correlation_id FROM move_envelopes AS e
WHERE (? = '' OR json_extract(spec, '$.leagueId') = ?)
AND (? = '' OR json_extract(spec, '$.franchiseId') = ?)
AND (NOT ? OR (
 SELECT to_state FROM move_audit AS a WHERE a.correlation_id = e.correlation_id
 ORDER BY sequence DESC LIMIT 1
) IN (?, ?, ?, ?, ?, ?))
ORDER BY created DESC, correlation_id DESC`,
		league, league, franchise, franchise, awaiting,
		envelope.HandedOff, envelope.NotYetDone, envelope.NotVerified,
		envelope.DOTReview, envelope.BidPending, envelope.WaiverPending)
	if err != nil {
		return nil, fmt.Errorf("moves: list IDs: %w", err)
	}
	defer func() { _ = rows.Close() }()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("moves: list ID scan: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("moves: list ID rows: %w", err)
	}
	return ids, nil
}
