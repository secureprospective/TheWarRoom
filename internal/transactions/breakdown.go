package transactions

import (
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// CapDelta is one signed cap-impact line a transaction produces, shown before the commissioner
// confirms. Positive raises cap used (dead cap); negative lowers it (relief). Reason is the same
// label the ledger row gets, so the quote and the ledger never describe a charge differently.
type CapDelta struct {
	FranchiseID string       `json:"franchiseID"`
	Cents       domain.Money `json:"cents"`
	Reason      string       `json:"reason"`
}

// applyResult is what a handler's apply returns: players moved plus cap-impact lines. The lines
// must come out of apply, because reads inside the transaction see only committed state. Nil
// deltas means the op isn't wired for a breakdown yet.
type applyResult struct {
	PlayersAffected int
	Deltas          []CapDelta
}

// deadCapDeltas turns a dead-cap entry into one positive line. A $0 charge (§13 death) gives no
// line rather than a "$0" row.
func deadCapDeltas(e state.DeadCapEntry) []CapDelta {
	if e.DeadCap <= 0 {
		return nil
	}
	return []CapDelta{{FranchiseID: e.FranchiseID, Cents: e.DeadCap, Reason: e.Reason}}
}
