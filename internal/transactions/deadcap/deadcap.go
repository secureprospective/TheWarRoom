// Package deadcap holds the §8 waiver-cut handler that runs BEHIND the B7a Coordinator.
// Like the acquisitions handlers, it is implementation detail: depguard
// (transactions-only-through-coordinator) denies importing it from outside
// internal/transactions, so the ONLY way to cut a player is Coordinator.Execute(Waiver).
// It neither opens nor commits a transaction — it reads current terms, computes the flat
// §8 charge, and sequences ReleasePlayer + AddDeadCap on the shared TxWriter; the
// enclosing WriteTx makes the release and its penalty atomic.
//
// The §8 charge is FLAT INTEGER MATH on exact cents (OQ-014) — there is no fractional
// distribution in this league (the earlier signing-bonus/proration model was NFL-model
// leakage, struck from OQ-014). The whole penalty lands in the CUT year's cap
// (LeagueYear = the current season): the formula yields one total, and the league's dead
// cap is a single flat number, not a per-year spread. CONFIRMED by Christopher 2026-07-04
// (a locked money-model decision, not an open choice).
package deadcap

import (
	"context"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// waiverReason is the audit string on every §8 ledger row.
const waiverReason = "waiver-cut §8"

// §8 dead-cap percentages: 35% of salary per remaining year, or 50% if the contract was
// restructured (§11). Whole percents, so the cents math stays exact bar the final round.
const (
	baseCutPct         = 35
	restructuredCutPct = 50
)

// Charge computes the §8 dead-cap penalty in exact cents:
//
//	pct% × annual salary × remaining years
//
// where pct is 50 for a restructured contract (§11) else 35. It is ZERO when the player
// was claimed off waivers (§8: the claim ends the obligation), has no remaining years
// (an expiring/UFA deal — remaining years = expiration_year − season), or carries no
// salary. Pure — no I/O, no clock. The final cent is rounded half-up (pct is a whole
// percent, so this only matters for salaries that aren't a round multiple of a cent-per-
// percent — the parser can produce any cent value).
func Charge(annualSalary domain.Money, remainingYears int, isRestructured, claimed bool) domain.Money {
	if claimed || remainingYears <= 0 || annualSalary <= 0 {
		return 0
	}
	pct := int64(baseCutPct)
	if isRestructured {
		pct = restructuredCutPct
	}
	num := int64(annualSalary) * pct * int64(remainingYears) // cents × whole-percent × years
	return domain.Money((num + 50) / 100)                    // ÷100 for the percent, round half-up
}

// Waive executes a §8 cut against the shared tx writer: it reads the player's current
// terms, computes the flat dead-cap charge, releases him from his franchise, and records
// the charge to the ledger — all in the Coordinator's one spanning transaction, so the
// cut and its penalty land together or not at all. It models the UNCLAIMED cut (v1); the
// claim path (dead cap 0, player moves to the claimer) arrives with free agency, and the
// claimed=0 rule already lives in Charge for when it does. Returns the ledger entry it
// wrote so the caller can surface the charge. Fails loud on an unknown player.
func Waive(ctx context.Context, w state.TxWriter, mflID string) (state.DeadCapEntry, error) {
	ps, ok := w.Player(mflID)
	if !ok {
		return state.DeadCapEntry{}, fmt.Errorf("deadcap: waive %q: player not on any roster", mflID)
	}
	remaining := ps.ExpirationYear - w.Season()
	if remaining < 0 {
		remaining = 0
	}
	// §8 reads "annual salary"; EffectiveSalary is that base salary until a restructure
	// (§11, B7c) sets an adjusted figure, at which point the restructured cap hit is the
	// right base — one definition, forward-consistent (v1 has no restructures, so equal).
	charge := Charge(state.EffectiveSalary(ps), remaining, ps.IsRestructured, false)

	if err := w.ReleasePlayer(ctx, mflID); err != nil {
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
	// Ledger dual-write: a cut VOIDs the player's remaining PAID cells (relieving every
	// cap-bearing year while preserving history — the ledger is king), so the ledger-derived
	// cap drops with the legacy release and no orphan cell survives to be re-counted if the
	// player later re-rosters via free agency. The dead-cap charge is a SEPARATE ledger
	// (dead_cap_ledger), not a contract cell — the two are distinct cap components.
	reason := fmt.Sprintf("%s: contract voided, dead cap %s", waiverReason, charge)
	if err := w.VoidCells(ctx, mflID, reason); err != nil {
		return state.DeadCapEntry{}, fmt.Errorf("deadcap: void cells %q: %w", mflID, err)
	}
	return entry, nil
}
