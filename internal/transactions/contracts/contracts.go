// Package contracts holds the §9–§11 contract ops (tag, extension, restructure). They run only
// inside Coordinator.Execute (depguard enforces it) against its TxWriter; they never open or
// commit a transaction themselves.
//
// The math is flat: this league has no proration or acceleration, and CapUsed is the exact-cents
// sum of the current-season ledger cells.
package contracts

import (
	"context"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// Per-season op-counter keys (transaction_counts rows): each op is limited to one per franchise
// per season.
const (
	restructureOpKind = "RESTRUCTURE"
	tagOpKind         = "TAG"
	extensionOpKind   = "EXTENSION"
)

// §10: an extension adds 1..3 years and no contract may exceed 6 paid years.
const (
	maxExtensionYears     = 3
	maxTotalContractYears = 6
)

// seasonAllowance fails when the franchise has already used this season's one op of kind.
func seasonAllowance(ctx context.Context, w state.TxWriter, op, mflID, franchiseID, kind, spentMsg string) error {
	spent, err := w.OpCount(ctx, franchiseID, kind)
	if err != nil {
		return fmt.Errorf("contracts: %s %q: %w", op, mflID, err)
	}
	if spent >= 1 {
		return fmt.Errorf("contracts: %s: franchise %q %s", op, franchiseID, spentMsg)
	}
	return nil
}

// million is $1M in cents, the unit of the §11 tier table.
const million = domain.Money(100_000_000)

// MaxMove returns the §11 maximum restructure move for a contract-year salary, and whether the
// contract may restructure at all:
//
//	Contract-Year Salary  Max Move
//	≥ $12M                $3M
//	≥ $6M                 $2M
//	≥ $3M                 $1M
//	<  $3M                ineligible (ok=false)
//
// Every tier's max is below its threshold, so a valid move always leaves a positive salary.
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

// Restructure applies a §11 restructure: it moves the owner-chosen amount out of the current
// season's cell into the contract's last paid year (the total is conserved) and flags the
// contract, so a later §8 cut charges 50% dead cap instead of 35%. Limits: one per contract, one
// per franchise per season, and the move within the §11 tier max for the base salary.
func Restructure(ctx context.Context, w state.TxWriter, mflID string, move domain.Money) error {
	ps, ok := w.Player(mflID)
	if !ok {
		return fmt.Errorf("contracts: restructure %q: player not on any roster", mflID)
	}
	if ps.IsRestructured {
		return fmt.Errorf("contracts: restructure %q: contract already restructured (one per contract, §11)", mflID)
	}
	// A tag is a fixed one-year salary, so it cannot restructure. The reverse is allowed: Tag
	// mints a fresh contract and resets the flag.
	if ps.IsTagged {
		return fmt.Errorf("contracts: restructure %q: a franchise-tagged contract is a fixed one-year deal and cannot be restructured (§9/§11)", mflID)
	}

	// §11 tiers on the base salary, never the cap-counting figure.
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

	if err := seasonAllowance(ctx, w, "restructure", mflID, ps.FranchiseID, restructureOpKind,
		"has already restructured a contract this season (one per team per year, §11)"); err != nil {
		return err
	}

	// The move leaves the current-season cell, so it cannot exceed that cell.
	capSalary := ps.CapSalary
	if move > capSalary {
		return fmt.Errorf("contracts: restructure %q: move %s exceeds the player's cap-counting salary %s", mflID, move, capSalary)
	}
	change := state.ContractChange{
		AnnualSalary:   ps.Salary,
		ExpirationYear: ps.ExpirationYear,
		ContractStatus: ps.ContractStatus,
		IsRestructured: true,
		IsTagged:       ps.IsTagged,
	}
	if err := w.ApplyContract(ctx, mflID, change); err != nil {
		return fmt.Errorf("contracts: restructure %q: %w", mflID, err)
	}
	// The money lands in the last paid year (expiration_year) until the owner can pick the
	// destination. A contract ending this season has nowhere to move it, so it cannot restructure.
	dest := ps.ExpirationYear
	if dest <= w.Season() {
		return fmt.Errorf("contracts: restructure %q: no future paid year to move money into (contract ends in %d, this season is %d)", mflID, dest, w.Season())
	}
	reason := fmt.Sprintf("§11 restructure: moved %s from %d to %d", move, w.Season(), dest)
	if err := w.MoveCellMoney(ctx, mflID, w.Season(), dest, move, reason); err != nil {
		return fmt.Errorf("contracts: restructure %q: %w", mflID, err)
	}
	if err := w.IncOpCount(ctx, ps.FranchiseID, restructureOpKind); err != nil {
		return fmt.Errorf("contracts: restructure %q: bump per-season counter: %w", mflID, err)
	}
	return nil
}

// Tag applies a §9 franchise tag: it sets the current-season cell to the price the Coordinator
// resolved from committed state and flags the contract tagged. One per franchise per season.
//
// The "two consecutive years" and "second tag at 120%" rules need per-player history across
// seasons and are deferred, so a tag never changes contract length.
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

	if err := seasonAllowance(ctx, w, "tag", mflID, ps.FranchiseID, tagOpKind,
		"has already tagged a player this season (one per team per year, §9)"); err != nil {
		return err
	}

	change := state.ContractChange{
		AnnualSalary:   price,
		ExpirationYear: ps.ExpirationYear,
		ContractStatus: ps.ContractStatus,
		// A tag is a fresh contract: a later cut charges the standard 35%, not the
		// restructured 50%.
		IsRestructured: false,
		IsTagged:       true,
	}
	if err := w.ApplyContract(ctx, mflID, change); err != nil {
		return fmt.Errorf("contracts: tag %q: %w", mflID, err)
	}
	reason := fmt.Sprintf("§9 franchise tag: season salary set to %s", price)
	if err := w.SetCell(ctx, mflID, w.Season(), price, reason); err != nil {
		return fmt.Errorf("contracts: tag %q: %w", mflID, err)
	}
	if err := w.IncOpCount(ctx, ps.FranchiseID, tagOpKind); err != nil {
		return fmt.Errorf("contracts: tag %q: bump per-season counter: %w", mflID, err)
	}
	return nil
}

// extensionYearPrice is the §10 price of each added year: 150% of the highest remaining year,
// snapped to $10k (§1), raised to the position floor. Integer math (×3, ÷2 rounding half up) keeps
// floats out of money.
func extensionYearPrice(highestRemaining, floor domain.Money) domain.Money {
	scaled := domain.RoundToNearest10k((highestRemaining*3 + 1) / 2)
	if floor > scaled {
		return floor
	}
	return scaled
}

// extendEligible runs the §10 checks that need no ledger cells. A tagged player is rejected
// because the extension would price off the tag (a §9 rule not built yet). "Remaining" years exclude
// the current season, as in §8, so a contract ending this season is a UFA and ineligible.
func extendEligible(w state.TxWriter, mflID string, addedYears int, floor domain.Money) (state.PlayerState, error) {
	if addedYears < 1 || addedYears > maxExtensionYears {
		return state.PlayerState{}, fmt.Errorf("contracts: extend %q: added years %d out of range 1..%d (§10)", mflID, addedYears, maxExtensionYears)
	}
	if floor <= 0 {
		return state.PlayerState{}, fmt.Errorf("contracts: extend %q: position floor must be positive (the coordinator resolves it)", mflID)
	}
	ps, ok := w.Player(mflID)
	if !ok {
		return state.PlayerState{}, fmt.Errorf("contracts: extend %q: player not on any roster", mflID)
	}
	if ps.IsTagged {
		return state.PlayerState{}, fmt.Errorf("contracts: extend %q: a franchise-tagged player's extension prices at 120%% of the tag (§9) and is out of scope in v1", mflID)
	}
	if ps.ExpirationYear <= w.Season() {
		return state.PlayerState{}, fmt.Errorf("contracts: extend %q: no year remaining (contract ends %d, season is %d) — UFAs are ineligible (§10)", mflID, ps.ExpirationYear, w.Season())
	}
	return ps, nil
}

// scanExtensionCells returns, in one pass over the paid cells: the highest remaining year (after
// the current season, matching extendEligible), whether a prior extension wrote any cell, and the
// last paid year.
func scanExtensionCells(cells []state.LedgerCell, season int) (highestRemaining domain.Money, alreadyExtended bool, lastPaidYear int) {
	for _, c := range cells {
		if c.Source == state.SourceExtension {
			alreadyExtended = true
		}
		if c.Year > season && c.Salary > highestRemaining {
			highestRemaining = c.Salary
		}
		if c.Year > lastPaidYear {
			lastPaidYear = c.Year
		}
	}
	return highestRemaining, alreadyExtended, lastPaidYear
}

// Extend applies a §10 extension: it appends addedYears paid cells at extensionYearPrice,
// lengthens the term, and resets is_restructured (each extension re-allows one §11 restructure).
// Limits: a year must remain, no second extension on a contract, 1..3 added years, at most 6
// total, and one per franchise per season. The coordinator resolves floor.
func Extend(ctx context.Context, w state.TxWriter, mflID string, addedYears int, floor domain.Money) error {
	ps, err := extendEligible(w, mflID, addedYears, floor)
	if err != nil {
		return err
	}
	cells, err := w.PaidCells(ctx, mflID)
	if err != nil {
		return fmt.Errorf("contracts: extend %q: %w", mflID, err)
	}
	// A prior extension is marked on the cells themselves. Free agency mints a new contract_id,
	// so a re-signed player is eligible again.
	highestRemaining, alreadyExtended, lastPaidYear := scanExtensionCells(cells, w.Season())
	if alreadyExtended {
		return fmt.Errorf("contracts: extend %q: contract already carries an extension — no second extension off a prior one (§10)", mflID)
	}
	if highestRemaining <= 0 {
		return fmt.Errorf("contracts: extend %q: no remaining paid year to price the extension from (§10)", mflID)
	}
	// The cells grow from their tail and the term from expiration_year, so the two must agree
	// before both are lengthened.
	if lastPaidYear != ps.ExpirationYear {
		return fmt.Errorf("contracts: extend %q: term/ledger drift — expiration_year %d != last paid cell %d", mflID, ps.ExpirationYear, lastPaidYear)
	}
	if total := len(cells) + addedYears; total > maxTotalContractYears {
		return fmt.Errorf("contracts: extend %q: %d existing + %d added = %d exceeds the %d-year max (§10)",
			mflID, len(cells), addedYears, total, maxTotalContractYears)
	}
	if err := seasonAllowance(ctx, w, "extend", mflID, ps.FranchiseID, extensionOpKind,
		"has already extended a contract this season (one per team per year, §10)"); err != nil {
		return err
	}

	price := extensionYearPrice(highestRemaining, floor)
	reason := fmt.Sprintf("§10 extension: +%d years at %s (150%% of %s, floor %s)", addedYears, price, highestRemaining, floor)
	if err := w.AppendExtensionYears(ctx, mflID, addedYears, price, reason); err != nil {
		return fmt.Errorf("contracts: extend %q: %w", mflID, err)
	}
	change := state.ContractChange{
		AnnualSalary:   ps.Salary,
		ExpirationYear: ps.ExpirationYear + addedYears,
		ContractStatus: ps.ContractStatus,
		IsRestructured: false,
		IsTagged:       ps.IsTagged,
	}
	if err := w.ApplyContract(ctx, mflID, change); err != nil {
		return fmt.Errorf("contracts: extend %q: %w", mflID, err)
	}
	if err := w.IncOpCount(ctx, ps.FranchiseID, extensionOpKind); err != nil {
		return fmt.Errorf("contracts: extend %q: bump per-season counter: %w", mflID, err)
	}
	return nil
}
