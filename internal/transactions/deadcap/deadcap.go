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

// Charge is the §8 penalty: pct% × annual salary × remaining years, with pct 50 if restructured,
// else 35. Zero if claimed off waivers, with no remaining years, or with no salary. Snapped to $10k,
// like every cap figure (Christopher, 2026-07-05). Pure.
func Charge(annualSalary domain.Money, remainingYears int, isRestructured, claimed bool) domain.Money {
	if claimed || remainingYears <= 0 || annualSalary <= 0 {
		return 0
	}
	pct := int64(baseCutPct)
	if isRestructured {
		pct = restructuredCutPct
	}
	num := int64(annualSalary) * pct * int64(remainingYears) // cents × whole-percent × years
	cents := domain.Money((num + 50) / 100)                  // ÷100 for the percent, round half-up
	return domain.RoundToNearest10k(cents)                   // land on the universal $10k grid
}

// Waive cuts a player: computes the §8 charge, releases him and records it, in one transaction.
// v1 models the unclaimed cut; Charge already handles a claim for when free agency adds it.
func Waive(ctx context.Context, w state.TxWriter, mflID string) (state.DeadCapEntry, error) {
	ps, ok := w.Player(mflID)
	if !ok {
		return state.DeadCapEntry{}, fmt.Errorf("deadcap: waive %q: player not on any roster", mflID)
	}
	remaining := ps.ExpirationYear - w.Season()
	if remaining < 0 {
		remaining = 0
	}
	// Charge on the cap salary: after a §11 restructure that is the reduced figure.
	charge := Charge(ps.CapSalary, remaining, ps.IsRestructured, false)

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
