package transactions

import (
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// CapDelta is one signed cap-impact line item a transaction produces — a piece of the pre-commit
// dollar breakdown a Preview surfaces so a commissioner sees the figure BEFORE confirming, not only
// on the post-commit refresh. Cents is SIGNED against cap USED: a positive value INCREASES the
// franchise's cap hit (a dead-cap charge), a negative value DECREASES it (a cap-relief credit),
// matching CapUsed = Σcells + Σdead_cap − Σcap_relief. Reason is the audit label ("waiver-cut §8",
// "cap relief §13") the store already stamps on the underlying ledger row, reused verbatim so the
// quote and the committed ledger never describe the same charge differently.
type CapDelta struct {
	FranchiseID string       `json:"franchiseID"`
	Cents       domain.Money `json:"cents"`
	Reason      string       `json:"reason"`
}

// applyResult is what a handler's apply returns: how many players it moved plus its cap-impact line
// items. It replaces the bare int so a Preview can surface the dollar figures a handler ALREADY
// computes without a post-commit re-read — the in-tx read wall (a TxWriter read reflects committed
// state, not this tx's own uncommitted writes) means the breakdown must travel OUT of apply, not be
// re-derived after it. Deltas is empty (nil) for an op not yet wired for a pre-commit breakdown; the
// quote then shows only will-succeed/rejection and the dollar lands on the post-commit refresh.
type applyResult struct {
	PlayersAffected int
	Deltas          []CapDelta
}

// deadCapDeltas projects a dead-cap ledger entry (the charge Waive/Buyout/Retire compute and return)
// onto the pre-commit breakdown: a single POSITIVE cap delta carrying the store's own audit reason.
// A zero charge — a §13 death (Gaines Adams: removal at $0 dead cap) — yields NO line, so the quote
// for a no-penalty removal shows an empty breakdown rather than a noise "$0" row.
func deadCapDeltas(e state.DeadCapEntry) []CapDelta {
	if e.DeadCap <= 0 {
		return nil
	}
	return []CapDelta{{FranchiseID: e.FranchiseID, Cents: e.DeadCap, Reason: e.Reason}}
}
