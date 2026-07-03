// Package acquisitions holds the roster-state transaction handlers that run BEHIND the
// B7a Coordinator: the actual per-player steps of a trade or a roster-status change,
// executed against a state.TxWriter inside the Coordinator's spanning transaction. It
// is implementation detail — depguard (transactions-only-through-coordinator) denies
// importing it from outside internal/transactions, so the ONLY way to run one of these
// is Coordinator.Execute. The handlers neither open nor commit a transaction; they
// sequence TxWriter calls and fail loud, and the enclosing WriteTx makes the set atomic.
//
// v1.0 houses trade + roster-status here (both mutate roster STATE only). Contract ops
// (tag/restructure) and dead-cap land in the contracts/deadcap sibling packages once
// the Money representation (OQ-014) is locked — see handoff 27.
package acquisitions

import (
	"context"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// Move is one player reassignment: MFLID to ToFranchiseID. A trade is an ordered set
// of Moves applied in one transaction.
type Move struct {
	MFLID         string
	ToFranchiseID string
}

// Trade applies every move of a trade against the shared tx writer, in order. A single
// step's failure returns immediately, and because all moves run in the Coordinator's
// one spanning transaction, that error rolls the WHOLE trade back — no half-executed
// swap. The step index rides in the error so a failure names which move broke.
func Trade(ctx context.Context, w state.TxWriter, moves []Move) error {
	for i, m := range moves {
		if err := w.MovePlayer(ctx, m.MFLID, m.ToFranchiseID); err != nil {
			return fmt.Errorf("acquisitions: trade move %d (player %q → %q): %w",
				i, m.MFLID, m.ToFranchiseID, err)
		}
	}
	return nil
}

// SetStatus applies a single roster-status change against the shared tx writer. Status
// validity is enforced by the state layer (one whitelist, not duplicated here).
func SetStatus(ctx context.Context, w state.TxWriter, mflID string, status domain.RosterStatus) error {
	if err := w.SetRosterStatus(ctx, mflID, status); err != nil {
		return fmt.Errorf("acquisitions: set roster status (player %q → %q): %w", mflID, status, err)
	}
	return nil
}
