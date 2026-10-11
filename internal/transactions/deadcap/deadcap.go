// Package deadcap holds the §8 waiver, §12 buyout and §13 handlers, run only through the
// Coordinator. Charges are flat integer math on exact cents, and the whole penalty lands in the cut
// year's cap: one number, not a per-year spread (Christopher, 2026-07-04).
package deadcap

import (
	"context"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

const waiverReason = "waiver-cut §8"

// §8 dead-cap rates: 35% per remaining year, 50% if restructured (§11).
const (
	baseCutPct         = 35
	restructuredCutPct = 50
)

// Charge sums the paid cells after the cut season before rounding once.
func Charge(cells []state.LedgerCell, season int, isRestructured, claimed bool) domain.Money {
	if claimed {
		return 0
	}
	var sum domain.Money
	for _, cell := range cells {
		if cell.Year > season {
			sum += cell.Salary
		}
	}
	if sum <= 0 {
		return 0
	}
	pct := int64(baseCutPct)
	if isRestructured {
		pct = restructuredCutPct
	}
	return domain.RoundToNearest10k(domain.Money((int64(sum)*pct + 50) / 100))
}

// Waive cuts a player: computes the §8 charge, releases him and records it, in one transaction.
// v1 models the unclaimed cut; Charge already handles a claim for when free agency adds it.
func Waive(ctx context.Context, w state.TxWriter, mflID string) (state.DeadCapEntry, error) {
	ps, ok := w.Player(mflID)
	if !ok {
		return state.DeadCapEntry{}, fmt.Errorf("deadcap: waive %q: player not on any roster", mflID)
	}
	cells, err := w.PaidCells(ctx, mflID)
	if err != nil {
		return state.DeadCapEntry{}, fmt.Errorf("deadcap: waive %q: read cells: %w", mflID, err)
	}
	charge := Charge(cells, w.Season(), ps.IsRestructured, false)

	if err := w.ReleasePlayer(ctx, mflID, domain.PlayerFreeAgent, waiverReason); err != nil {
		return state.DeadCapEntry{}, fmt.Errorf("deadcap: release %q: %w", mflID, err)
	}
	entry := state.DeadCapEntry{
		FranchiseID: ps.FranchiseID,
		MFLID:       mflID,
		LeagueYear:  w.Season(),
		DeadCap:     charge,
		Reason:      waiverReason,
	}
	if err := w.AddDeadCap(ctx, entry); err != nil {
		return state.DeadCapEntry{}, fmt.Errorf("deadcap: charge %q: %w", mflID, err)
	}
	// Void the remaining PAID cells so no orphan cell is re-counted if he re-rosters. Dead cap is its
	// own ledger, not a contract cell.
	reason := fmt.Sprintf("%s: contract voided, dead cap %s", waiverReason, charge)
	if err := w.VoidCells(ctx, mflID, reason); err != nil {
		return state.DeadCapEntry{}, fmt.Errorf("deadcap: void cells %q: %w", mflID, err)
	}
	return entry, nil
}
