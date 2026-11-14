package history

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"

	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// DecideDestination is only called while the source's durable sealed decision is locked.
// It is deliberately not a transport endpoint or a portable activation-token consumer.
// Activation commits the move's reference data (identities, blobs) first, after the same refusals
// are read, then runs the league transaction with every player id already durable.
func (s *Store) DecideDestination(ctx context.Context, f TransferFence, activate bool, events []RawTransferEvent, blobs map[string][]byte, identities []playerid.RawIdentity) error {
	if activate {
		if done, err := s.destinationPrecheck(ctx, f); done || err != nil {
			return err
		}
		if err := s.commitReference(ctx, identities, blobs, "receive"); err != nil {
			return err
		}
	}
	_, err := identityFirst(ctx, s, func() (struct{}, error) {
		return struct{}{}, s.decideDestinationOnce(ctx, f, activate, events)
	})
	return err
}

func (s *Store) destinationPrecheck(ctx context.Context, f TransferFence) (bool, error) {
	tx, err := s.pools.Read().BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return true, fmt.Errorf("begin destination decision: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	_, done, err := destinationChecks(ctx, tx, f, true)
	return done || err != nil, err
}

// destinationChecks are the decision's refusals. done reports a move already active here.
func destinationChecks(ctx context.Context, tx *sql.Tx, f TransferFence, activate bool) (TransferFence, bool, error) {
	var id string
	if err := tx.QueryRowContext(ctx, `SELECT id FROM transfer_identity`).Scan(&id); err != nil {
		return TransferFence{}, true, fmt.Errorf("check move destination: %w", err)
	}
	if id != f.Peer {
		return TransferFence{}, true, fmt.Errorf("this is not the confirmed destination")
	}
	held, err := transferCurrent(ctx, tx, f.League)
	if err != nil {
		return TransferFence{}, true, err
	}
	if err := checkDestinationDecision(ctx, tx, f, held, activate); err != nil {
		return TransferFence{}, true, err
	}
	return held, held.Move == f.Move && held.State == "active", nil
}

func (s *Store) decideDestinationOnce(ctx context.Context, f TransferFence, activate bool, events []RawTransferEvent) error {
	tx, err := s.pools.Write().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin destination decision: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	held, done, err := destinationChecks(ctx, tx, f, activate)
	if done || err != nil {
		return err
	}
	if activate {
		if err := s.receiveTransfer(ctx, tx, f, held, events); err != nil {
			return err
		}
		f.State = "active"
		f.Epoch = f.Move
	} else {
		f.State = "cancelled"
	}
	if err := transferTransition(ctx, tx, f); err != nil {
		return err
	}
	// Cancelling a reverse must preserve the retired destination's lineage for a later move.
	if !activate && held.State == "moved" {
		if err := transferTransition(ctx, tx, held); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit destination decision: %w", err)
	}
	return nil
}

func checkDestinationDecision(ctx context.Context, tx *sql.Tx, f, held TransferFence, activate bool) error {
	var cancelled bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM transfer_transition WHERE league=? AND move=? AND state='cancelled')`, f.League, f.Move).Scan(&cancelled); err != nil {
		return fmt.Errorf("check cancelled move: %w", err)
	}
	if activate && cancelled {
		return fmt.Errorf("this move was cancelled; seal a new move")
	}
	if held.State == "active" && (held.Move != f.Move || !activate) {
		return fmt.Errorf("the destination is already writable; use a reverse transfer instead")
	}
	if held.State == "moved" && held.Move != f.Epoch {
		return fmt.Errorf("reverse transfer must come from the authority that received this league")
	}
	if held.State == "sealed" || held.State == "receiving" {
		return fmt.Errorf("the destination is moving elsewhere")
	}
	return checkInactiveDestination(ctx, tx, f, held)
}
func checkInactiveDestination(ctx context.Context, tx *sql.Tx, f, held TransferFence) error {
	if held.State == "" || held.State == "cancelled" {
		receipt, err := importReceipt(ctx, tx, f.League)
		if err != nil {
			return err
		}
		if receipt.Content != f.Content || receipt.SourceEpoch != f.Epoch {
			return fmt.Errorf("the inactive copy does not match the sealed export")
		}
	}
	return nil
}

func (s *Store) receiveTransfer(ctx context.Context, tx *sql.Tx, f, held TransferFence, events []RawTransferEvent) error {
	rows, err := transferEvents(ctx, tx, f.League)
	if err != nil {
		return err
	}
	wanted := make(map[string]RawTransferEvent, len(events))
	for _, e := range events {
		if e.League != f.League {
			return fmt.Errorf("the move includes another league")
		}
		wanted[e.ID] = e
	}
	for _, e := range rows {
		if !reflect.DeepEqual(e, wanted[e.ID]) {
			return fmt.Errorf("the destination is not an unchanged prefix of the sealed history")
		}
		delete(wanted, e.ID)
	}
	if held.State != "moved" && len(wanted) != 0 {
		return fmt.Errorf("the inactive copy is missing sealed events")
	}
	f.State = "receiving"
	if err := transferTransition(ctx, tx, f); err != nil {
		return err
	}
	for _, e := range events {
		if _, ok := wanted[e.ID]; ok {
			if err := stageEvent(ctx, tx, e); err != nil {
				return err
			}
		}
	}
	return s.captureRosterRestorePoint(ctx, tx, f.League)
}

// TransferStatus is shared wording for desktop and hosted services. IDs stay in the protocol,
// never in these owner-facing labels. A missing/unreadable status must not imply permission.
type TransferStatus struct {
	Writable bool
	Label    string
}

func (s *Store) TransferStatus(ctx context.Context, league string) (TransferStatus, error) {
	tx, err := s.pools.Read().BeginTx(ctx, nil)
	if err != nil {
		return TransferStatus{}, fmt.Errorf("read league location: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	f, err := transferCurrent(ctx, tx, league)
	if err != nil {
		return TransferStatus{}, err
	}
	switch f.State {
	case "active":
		return TransferStatus{Writable: true, Label: "This league is managed here"}, nil
	case "sealed":
		return TransferStatus{Label: "This league is paused while the commissioner moves it"}, nil
	case "moved":
		return TransferStatus{Label: "This league now lives on the other computer. This is a read-only copy"}, nil
	case "cancelled", "receiving":
		return TransferStatus{Label: "Read-only copy"}, nil
	}
	var imported bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM league_import WHERE league=?)`, league).Scan(&imported); err != nil {
		return TransferStatus{}, fmt.Errorf("check league location: %w", err)
	}
	if imported {
		return TransferStatus{Label: "Read-only copy. The commissioner has not activated this league here"}, nil
	}
	var exists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM league_event WHERE league=?)`, league).Scan(&exists); err != nil {
		return TransferStatus{}, fmt.Errorf("check league history: %w", err)
	}
	if !exists {
		return TransferStatus{}, fmt.Errorf("choose an existing league")
	}
	return TransferStatus{Writable: true, Label: "This league is managed here"}, nil
}

// CheckTransferDestination prevents pausing the source for an unprepared or wrong target.
// Activation repeats these checks under its own writer; this preflight is not permission.
func (s *Store) CheckTransferDestination(ctx context.Context, f TransferFence) error {
	tx, err := s.pools.Read().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("check move destination: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	held, err := transferCurrent(ctx, tx, f.League)
	if err != nil {
		return err
	}
	return checkDestinationDecision(ctx, tx, f, held, true)
}
