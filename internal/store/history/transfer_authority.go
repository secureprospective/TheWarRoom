package history

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"

	"github.com/google/uuid"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/governance"
)

func addTransferAuthority(ctx context.Context, tx *sql.Tx) error {
	const ddl = `
CREATE TABLE transfer_identity (id TEXT PRIMARY KEY);
CREATE TABLE transfer_transition (
 sequence INTEGER PRIMARY KEY, league TEXT NOT NULL, move TEXT NOT NULL,
 epoch TEXT NOT NULL, content TEXT NOT NULL, peer TEXT NOT NULL,
 state TEXT NOT NULL CHECK(state IN ('active','sealed','moved','cancelled','receiving'))
);
CREATE INDEX transfer_transition_league ON transfer_transition(league,sequence);
CREATE VIEW transfer_current AS SELECT * FROM transfer_transition t
 WHERE sequence=(SELECT max(sequence) FROM transfer_transition WHERE league=t.league);
DROP TRIGGER league_import_event_guard;
DROP TRIGGER league_import_authority_insert;
DROP TRIGGER league_import_authority_update;
CREATE TRIGGER league_import_event_guard BEFORE INSERT ON league_event
WHEN EXISTS(SELECT 1 FROM transfer_current WHERE league=NEW.league AND state NOT IN ('active','receiving'))
 OR (EXISTS(SELECT 1 FROM league_import WHERE league=NEW.league)
 AND NOT EXISTS(SELECT 1 FROM transfer_current WHERE league=NEW.league AND state IN ('active','receiving')))
BEGIN SELECT RAISE(ABORT,'this league is a read-only copy'); END;
CREATE TRIGGER league_import_authority_insert BEFORE INSERT ON authority_policy
WHEN EXISTS(SELECT 1 FROM transfer_current WHERE league=NEW.league AND state!='active')
 OR (EXISTS(SELECT 1 FROM league_import WHERE league=NEW.league)
 AND NOT EXISTS(SELECT 1 FROM transfer_current WHERE league=NEW.league AND state='active'))
BEGIN SELECT RAISE(ABORT,'this league is a read-only copy'); END;
CREATE TRIGGER league_import_authority_update BEFORE UPDATE ON authority_policy
WHEN EXISTS(SELECT 1 FROM transfer_current WHERE league=NEW.league AND state!='active')
 OR (EXISTS(SELECT 1 FROM league_import WHERE league=NEW.league)
 AND NOT EXISTS(SELECT 1 FROM transfer_current WHERE league=NEW.league AND state='active'))
BEGIN SELECT RAISE(ABORT,'this league is a read-only copy'); END;`
	if _, err := tx.ExecContext(ctx, ddl+appendOnlyDDL("transfer_transition")+appendOnlyDDL("transfer_identity")); err != nil {
		return fmt.Errorf("prepare league move authority: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO transfer_identity VALUES(?)`, uuid.NewString()); err != nil {
		return fmt.Errorf("identify league database: %w", err)
	}
	return nil
}

// TransferFence binds a single move to one destination and one exact replayed document.
type TransferFence struct{ League, Move, Epoch, Content, Peer, State string }

func transferCurrent(ctx context.Context, tx *sql.Tx, league string) (TransferFence, error) {
	var f TransferFence
	err := tx.QueryRowContext(ctx, `SELECT league,move,epoch,content,peer,state FROM transfer_current WHERE league=?`, league).Scan(&f.League, &f.Move, &f.Epoch, &f.Content, &f.Peer, &f.State)
	if errors.Is(err, sql.ErrNoRows) {
		return TransferFence{League: league}, nil
	}
	if err != nil {
		return f, fmt.Errorf("read league move: %w", err)
	}
	return f, nil
}
func transferTransition(ctx context.Context, tx *sql.Tx, f TransferFence) error {
	if _, err := tx.ExecContext(ctx, `INSERT INTO transfer_transition(league,move,epoch,content,peer,state) VALUES(?,?,?,?,?,?)`, f.League, f.Move, f.Epoch, f.Content, f.Peer, f.State); err != nil {
		return fmt.Errorf("record league move: %w", err)
	}
	return nil
}

func (s *Store) TransferIdentity(ctx context.Context) (string, error) {
	var id string
	if err := s.pools.Read().QueryRowContext(ctx, `SELECT id FROM transfer_identity`).Scan(&id); err != nil {
		return "", fmt.Errorf("read destination identity: %w", err)
	}
	return id, nil
}

func transferEvents(ctx context.Context, tx *sql.Tx, league string) ([]RawTransferEvent, error) {
	rows, err := tx.QueryContext(ctx, `SELECT id,league,stream,revision,effective_at,recorded_at,actor,scope,kind,payload,input_hash,correlation_id,supersedes FROM league_event WHERE league=? ORDER BY stream,revision`, league)
	if err != nil {
		return nil, fmt.Errorf("read move history: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := []RawTransferEvent{}
	for rows.Next() {
		var e RawTransferEvent
		if err := rows.Scan(&e.ID, &e.League, &e.Stream, &e.Revision, &e.EffectiveAt, &e.RecordedAt, &e.Actor, &e.Scope, &e.Kind, &e.Payload, &e.InputHash, &e.CorrelationID, &e.Supersedes); err != nil {
			return nil, fmt.Errorf("read move event: %w", err)
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read move history: %w", err)
	}
	return out, nil
}

// SealTransfer compares the exported rows under the same writer reservation that installs the fence.
func (s *Store) SealTransfer(ctx context.Context, f TransferFence, events []RawTransferEvent, anchor db.AuditHead) (TransferFence, error) {
	tx, err := s.pools.Write().BeginTx(ctx, nil)
	if err != nil {
		return f, fmt.Errorf("begin source seal: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	held, err := transferCurrent(ctx, tx, f.League)
	if err != nil {
		return f, err
	}
	if held.State == "sealed" && held.Epoch == f.Epoch && held.Content == f.Content && held.Peer == f.Peer {
		return held, nil
	}
	if err := checkTransferAnchor(ctx, tx, anchor); err != nil {
		return f, err
	}
	if err := checkSealSource(ctx, tx, f, held); err != nil {
		return f, err
	}
	rows, err := transferEvents(ctx, tx, f.League)
	if err != nil {
		return f, err
	}
	if !reflect.DeepEqual(rows, events) {
		return f, fmt.Errorf("the source changed after export; export it again before sealing")
	}
	f.Move = uuid.NewString()
	f.State = "sealed"
	if err := transferTransition(ctx, tx, f); err != nil {
		return f, err
	}
	if err := tx.Commit(); err != nil {
		return f, fmt.Errorf("commit source seal: %w", err)
	}
	return f, nil
}

// DecideTransfer serializes activation against abort across processes. The source stays sealed
// if the destination commits but this transaction crashes. Retry observes the target decision.
func (s *Store) DecideTransfer(ctx context.Context, f TransferFence, activate bool, decide func() error) error {
	tx, err := s.pools.Write().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin move decision: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	held, err := transferCurrent(ctx, tx, f.League)
	if err != nil {
		return err
	}
	final := "active"
	if activate {
		final = "moved"
	}
	if held.Move == f.Move && held.State == final {
		return nil
	}
	if held != f || held.State != "sealed" {
		return fmt.Errorf("this move is no longer awaiting a decision")
	}
	if err := decide(); err != nil {
		return err
	}
	f.State = final
	if err := transferTransition(ctx, tx, f); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit source decision: %w", err)
	}
	return nil
}

// TransferFence returns the durable decision for recovery, never independent activation permission.
func (s *Store) TransferFence(ctx context.Context, league string) (TransferFence, error) {
	tx, err := s.pools.Read().BeginTx(ctx, nil)
	if err != nil {
		return TransferFence{}, fmt.Errorf("read move decision: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	return transferCurrent(ctx, tx, league)
}

func checkSealSource(ctx context.Context, tx *sql.Tx, f, held TransferFence) error {
	if held.State != "" && (held.State != "active" || held.Epoch != f.Epoch) {
		return fmt.Errorf("this copy cannot be sealed for that source generation")
	}
	var inactive bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM league_import WHERE league=?)`, f.League).Scan(&inactive); err != nil {
		return fmt.Errorf("check source copy: %w", err)
	}
	if held.State == "" && inactive {
		return fmt.Errorf("an inactive copy cannot be the source")
	}
	return nil
}

// CheckTransferCommissioner uses the owning store's resolved current policy, not imported claims.
func (s *Store) CheckTransferCommissioner(ctx context.Context, league, principal string) error {
	records, err := s.leagueRecords(ctx, league)
	if err != nil {
		return err
	}
	state, err := commissioner.GovernanceState(records)
	if err != nil {
		return fmt.Errorf("read move policy: %w", err)
	}
	if !state.HasRole(governance.Principal(commissioner.PrincipalFor(records, principal)), governance.Commissioner) {
		return fmt.Errorf("only the league commissioner can move this league")
	}
	return nil
}

func checkTransferAnchor(ctx context.Context, tx *sql.Tx, anchor db.AuditHead) error {
	current, err := db.VerifyAudit(ctx, tx, ledgerAuditSource, ledgerAuditQuery)
	if err != nil {
		return fmt.Errorf("verify source before sealing: %w", err)
	}
	if err := db.CheckAuditAnchor(ctx, tx, current, anchor); err != nil {
		return fmt.Errorf("compare source checkpoint: %w", err)
	}
	return nil
}
