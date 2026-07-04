// Package contracts holds the §9–§13 contract-op handlers that run BEHIND the B7a
// Coordinator: the per-player steps of a restructure (and, later, tag/extension/buyout)
// executed against a state.TxWriter inside the Coordinator's spanning transaction. Like
// the acquisitions/deadcap siblings it is implementation detail — depguard
// (transactions-only-through-coordinator) denies importing it from outside
// internal/transactions, so the ONLY way to run one of these is Coordinator.Execute. The
// handlers neither open nor commit a transaction; they sequence TxWriter calls and fail
// loud, and the enclosing WriteTx makes the set atomic.
//
// The math is FLAT (this league has no proration/acceleration — that NFL-model leakage was
// struck from OQ-014). CapUsed is the exact-cents SUM of a franchise's contracts, so a
// restructure that lowers a player's cap-counting salary by the owner's move lowers the cap
// by exactly that move — no spreading, no future-year bookkeeping.
package contracts

import (
	"context"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// restructureOpKind is the per-season op-counter key for §11's "one restructure per team
// per year" limit (a row in transaction_counts).
const restructureOpKind = "RESTRUCTURE"

// tagOpKind is the per-season op-counter key for §9's "one tag per team per year" limit
// (its own row in transaction_counts, independent of restructure).
const tagOpKind = "TAG"

// million is $1,000,000 in exact cents ($1M = 1e6 dollars × 100 cents = 1e8 cents) — the
// unit of the §11 restructure tier table.
const million = domain.Money(100_000_000)

// MaxMove returns the §11 maximum restructure "move" for a given Contract-Year Salary, and
// whether the contract is eligible to restructure at all. It is the rulebook §11 table
// encoded as a pure step function, so a move outside the allowed band is impossible by
// construction:
//
//	Contract-Year Salary  Max Move
//	≥ $12M                $3M
//	≥ $6M                 $2M
//	≥ $3M                 $1M
//	<  $3M                ineligible (ok=false)
//
// Every tier's max is strictly below its threshold, so subtracting a valid move from an
// eligible salary always leaves a positive cap-counting figure.
func MaxMove(contractYearSalary domain.Money) (maxMove domain.Money, eligible bool) {
	switch {
	case contractYearSalary >= 12*million:
		return 3 * million, true
	case contractYearSalary >= 6*million:
		return 2 * million, true
	case contractYearSalary >= 3*million:
		return 1 * million, true
	default:
		return 0, false
	}
}

// Restructure applies a §11 contract restructure against the shared tx writer: it lowers a
// player's cap-counting (adjusted) salary by the owner-chosen move and flags the contract
// restructured, so a later §8 cut charges 50% dead cap instead of 35%. It enforces both
// rulebook limits — one restructure per contract (is_restructured) and one per franchise
// per season (the durable op counter) — and validates the move against the §11 tier max
// read off the base Contract-Year Salary. The move amount is the owner's strategic choice
// WITHIN those bounds (Christopher, 2026-07-04): the rules ARE the boundary, so no rule
// violation is representable. Flat math — lowering the adjusted salary lowers CapUsed by
// exactly the move. Fails loud on an unknown player, an already-restructured or ineligible
// contract, an over-max move, or a spent per-season allowance.
func Restructure(ctx context.Context, w state.TxWriter, mflID string, move domain.Money) error {
	ps, ok := w.Player(mflID)
	if !ok {
		return fmt.Errorf("contracts: restructure %q: player not on any roster", mflID)
	}
	if ps.IsRestructured {
		return fmt.Errorf("contracts: restructure %q: contract already restructured (one per contract, §11)", mflID)
	}

	// §11 tiers on the base Contract-Year Salary (ps.Salary), never the adjusted figure.
	maxMove, eligible := MaxMove(ps.Salary)
	if !eligible {
		return fmt.Errorf("contracts: restructure %q: contract-year salary %s is below the $3M restructure floor (§11)", mflID, ps.Salary)
	}
	if move <= 0 {
		return fmt.Errorf("contracts: restructure %q: move must be positive, got %s", mflID, move)
	}
	if move > maxMove {
		return fmt.Errorf("contracts: restructure %q: move %s exceeds the §11 tier max %s for a %s contract-year salary", mflID, move, maxMove, ps.Salary)
	}

	// §11: one restructure per franchise per season.
	spent, err := w.OpCount(ctx, ps.FranchiseID, restructureOpKind)
	if err != nil {
		return fmt.Errorf("contracts: restructure %q: %w", mflID, err)
	}
	if spent >= 1 {
		return fmt.Errorf("contracts: restructure: franchise %q has already restructured a contract this season (one per team per year, §11)", ps.FranchiseID)
	}

	// Flat math: drop the cap-counting salary by the move. §11 tiers on the BASE salary, but
	// the money comes out of the cap-counting (effective) figure — equal in v1 (no prior
	// adjustments). Bound the move by the current effective salary so newAdjusted can NEVER
	// go negative even if a future downward adjustment made effective < base: the invariant
	// is asserted LOCALLY here, not left to ApplyContract's non-negative backstop. This only
	// adds a floor; it never widens the §11 tier band.
	eff := state.EffectiveSalary(ps)
	if move > eff {
		return fmt.Errorf("contracts: restructure %q: move %s exceeds the player's cap-counting salary %s", mflID, move, eff)
	}
	newAdjusted := eff - move
	change := state.ContractChange{
		AnnualSalary:   ps.Salary, // base salary is unchanged — only the cap-counting figure drops
		AdjustedSalary: newAdjusted,
		ContractYears:  ps.ContractYears,
		ExpirationYear: ps.ExpirationYear,
		ContractStatus: ps.ContractStatus,
		IsRestructured: true,
		IsTagged:       ps.IsTagged,
	}
	if err := w.ApplyContract(ctx, mflID, change); err != nil {
		return fmt.Errorf("contracts: restructure %q: %w", mflID, err)
	}
	if err := w.IncOpCount(ctx, ps.FranchiseID, restructureOpKind); err != nil {
		return fmt.Errorf("contracts: restructure %q: bump per-season counter: %w", mflID, err)
	}
	return nil
}

// Tag applies a §9 franchise tag against the shared tx writer: it sets the player's
// cap-counting salary to the already-resolved tag price and flags the contract tagged. The
// price (the §9 top-5-by-position average, floored at 120% of the prior year) is computed
// AUTHORITATIVELY by the Coordinator from committed state before this runs — the handler
// never trusts a caller figure and never re-reads the league, it just writes the box. Flat
// math: CapUsed is the sum of contracts, so setting the salary sets the cap contribution.
// Enforces §9's "one tag per franchise per season" via the durable op counter. Fails loud on
// an unknown player, an already-tagged contract, a non-positive price, or a spent allowance.
//
// v1 sets ONLY the salary + is_tagged (Christopher 2026-07-04, "lightest weight"): the
// "two consecutive years" / "second tag = 120% of first" mechanics need cross-season
// per-player history and are DEFERRED (handoff 30), so this does not touch contract length.
func Tag(ctx context.Context, w state.TxWriter, mflID string, price domain.Money) error {
	ps, ok := w.Player(mflID)
	if !ok {
		return fmt.Errorf("contracts: tag %q: player not on any roster", mflID)
	}
	if ps.IsTagged {
		return fmt.Errorf("contracts: tag %q: contract already tagged (§9)", mflID)
	}
	if price <= 0 {
		return fmt.Errorf("contracts: tag %q: resolved tag price must be positive, got %s", mflID, price)
	}

	// §9: one tag per franchise per season.
	spent, err := w.OpCount(ctx, ps.FranchiseID, tagOpKind)
	if err != nil {
		return fmt.Errorf("contracts: tag %q: %w", mflID, err)
	}
	if spent >= 1 {
		return fmt.Errorf("contracts: tag: franchise %q has already tagged a player this season (one per team per year, §9)", ps.FranchiseID)
	}

	change := state.ContractChange{
		AnnualSalary:   price, // the tag IS a new salary (the §9 box); base and cap-counting both become the price
		AdjustedSalary: price,
		ContractYears:  ps.ContractYears,
		ExpirationYear: ps.ExpirationYear,
		ContractStatus: ps.ContractStatus,
		IsRestructured: ps.IsRestructured,
		IsTagged:       true,
	}
	if err := w.ApplyContract(ctx, mflID, change); err != nil {
		return fmt.Errorf("contracts: tag %q: %w", mflID, err)
	}
	if err := w.IncOpCount(ctx, ps.FranchiseID, tagOpKind); err != nil {
		return fmt.Errorf("contracts: tag %q: bump per-season counter: %w", mflID, err)
	}
	return nil
}
