// Package acquisitions holds the roster handlers (trade, roster-status change) that run inside the
// Coordinator's transaction. Only internal/transactions may import it, so Coordinator.Execute is
// the only way to run them.
package acquisitions

import (
	"context"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// Move is one player reassignment.
type Move struct {
	MFLID         string
	ToFranchiseID string
}

// Trade applies every move in order. A failing step returns at once and rolls back the whole
// trade; the error names the step.
func Trade(ctx context.Context, w state.TxWriter, moves []Move) error {
	for i, m := range moves {
		if err := w.MovePlayer(ctx, m.MFLID, m.ToFranchiseID); err != nil {
			return fmt.Errorf("acquisitions: trade move %d (player %q → %q): %w",
				i, m.MFLID, m.ToFranchiseID, err)
		}
	}
	return nil
}

// SetStatus applies one roster-status change; the state layer validates the status.
func SetStatus(ctx context.Context, w state.TxWriter, mflID string, status domain.RosterStatus) error {
	if err := w.SetRosterStatus(ctx, mflID, status); err != nil {
		return fmt.Errorf("acquisitions: set roster status (player %q → %q): %w", mflID, status, err)
	}
	return nil
}
