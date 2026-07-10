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

// extensionOpKind is the per-season op-counter key for §10's "one extension per GM per season"
// limit (its own row in transaction_counts, independent of tag/restructure).
const extensionOpKind = "EXTENSION"

// §10 extension bounds: an extension adds 1..maxExtensionYears years, and no contract may
// exceed maxTotalContractYears total years (existing PAID cells + the added years).
const (
	maxExtensionYears     = 3
	maxTotalContractYears = 6
)

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
	// A franchise tag is a fixed one-year salary, not a multi-year contract — restructuring it
	// is meaningless, so reject it (GLM M4a, Christopher-confirmed). The reverse (tagging a
	// previously-restructured player) IS allowed: the tag is a fresh contract and Tag resets
	// the restructure flag.
	if ps.IsTagged {
		return fmt.Errorf("contracts: restructure %q: a franchise-tagged contract is a fixed one-year deal and cannot be restructured (§9/§11)", mflID)
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

	// Flat math: the move comes out of the CAP-COUNTING salary (the current-season cell,
	// CapSalary). Bound it by that figure so the cell can NEVER go negative — the invariant
	// is asserted LOCALLY here, in addition to MoveCellMoney's own non-negativity guard. This
	// only adds a floor; it never widens the §11 tier band (which bands on the base Salary).
	capSalary := ps.CapSalary
	if move > capSalary {
		return fmt.Errorf("contracts: restructure %q: move %s exceeds the player's cap-counting salary %s", mflID, move, capSalary)
	}
	// Ship 3: the cap drop is realized SOLELY by moving money between cells (below) — the
	// ledger cell is the KING, so there is no legacy adjusted figure to write. ApplyContract
	// only flips the restructure flag (and re-persists the unchanged base terms).
	change := state.ContractChange{
		AnnualSalary:   ps.Salary, // base salary is unchanged — only the cap-counting cell drops
		ContractYears:  ps.ContractYears,
		ExpirationYear: ps.ExpirationYear,
		ContractStatus: ps.ContractStatus,
		IsRestructured: true,
		IsTagged:       ps.IsTagged,
	}
	if err := w.ApplyContract(ctx, mflID, change); err != nil {
		return fmt.Errorf("contracts: restructure %q: %w", mflID, err)
	}
	// Ledger dual-write: a restructure is money MOVEMENT between the player's year cells,
	// conserving the contract total (the real §11, which the single-salary model could not
	// express). The relief comes out of the CURRENT season cell (matching the legacy adjusted
	// drop) and lands in the last PAID year — the v1 destination default until the owner-pick
	// money-mover UI ships. The last paid year is the contract's expiration year (the seed and
	// every op keep expiration_year == the last PAID cell). A contract that ends this season
	// has no future paid year to absorb the move, so it cannot restructure — a correctness
	// tightening the old single-number model silently swallowed (the relief just vanished).
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
		AnnualSalary:   price, // the tag IS a new base salary (the §9 box); SetCell writes the cap-counting cell
		ContractYears:  ps.ContractYears,
		ExpirationYear: ps.ExpirationYear,
		ContractStatus: ps.ContractStatus,
		// A franchise tag is a FRESH one-year contract, not a restructured deal — reset the
		// restructure flag (GLM M4). Otherwise cutting a restructured-then-tagged player would
		// charge §8 dead cap at the restructured 50% on the tag figure; a tag contract that has
		// not itself been restructured must charge the standard 35%.
		IsRestructured: false,
		IsTagged:       true,
	}
	if err := w.ApplyContract(ctx, mflID, change); err != nil {
		return fmt.Errorf("contracts: tag %q: %w", mflID, err)
	}
	// Ledger dual-write: a tag SETS the current-season cell to the resolved price (a fresh
	// one-year §9 figure — non-conserving, unlike restructure's money MOVEMENT). Parity
	// checks only the current-season derived cap, so setting the season cell = the same price
	// the legacy column just took keeps parity green by construction. v1 does not touch
	// contract length, so only the season cell changes; the seed guarantees it exists.
	reason := fmt.Sprintf("§9 franchise tag: season salary set to %s", price)
	if err := w.SetCell(ctx, mflID, w.Season(), price, reason); err != nil {
		return fmt.Errorf("contracts: tag %q: %w", mflID, err)
	}
	if err := w.IncOpCount(ctx, ps.FranchiseID, tagOpKind); err != nil {
		return fmt.Errorf("contracts: tag %q: bump per-season counter: %w", mflID, err)
	}
	return nil
}

// extensionYearPrice is the §10 price of each added year: 150% of the player's highest-paid
// remaining year, snapped to $10k (§1), then raised to the position floor if that is greater
// (the resolved floor from the Coordinator). The 150% is exact integer math — ×3 then ÷2,
// rounding half up to the cent — so no float money enters. Both inputs are $10k multiples, so
// the result is too.
func extensionYearPrice(highestRemaining, floor domain.Money) domain.Money {
	scaled := domain.RoundToNearest10k((highestRemaining*3 + 1) / 2)
	if floor > scaled {
		return floor
	}
	return scaled
}

// extendEligible runs the §10 pre-cell eligibility gates (cheap argument + single-player
// checks) and returns the player's current state for the caller to price against. It rejects a
// bad added-year count, an unresolved floor, an unknown player, a franchise-tagged player (his
// extension prices at 120% of the tag — a §9 cross-rule out of scope in v1), or a player with
// no year remaining (UFAs ineligible; remaining is EXCLUSIVE of the elapsed count, so
// expiration_year must be strictly beyond the current season — the §8 convention).
func extendEligible(w state.TxWriter, mflID string, addedYears int, floor domain.Money) (state.PlayerState, error) {
	if addedYears < 1 || addedYears > maxExtensionYears {
		return state.PlayerState{}, fmt.Errorf("contracts: extend %q: added years %d out of range 1..%d (§10)", mflID, addedYears, maxExtensionYears)
	}
	if floor <= 0 {
		return state.PlayerState{}, fmt.Errorf("contracts: extend %q: position floor must be positive (resolve via ExecuteExtension)", mflID)
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

// scanExtensionCells makes one pass over a player's PAID cells, returning: the highest-paid
// REMAINING year (the §10 pricing base — "remaining" is EXCLUSIVE of the current season, year
// > season, matching the §8 remaining-years convention AND this handler's own eligibility gate,
// so the two definitions of "remaining" agree — GLM M1); whether any cell was written by a
// prior extension (source "extension" — the no-second-extension marker); and the last PAID year
// (for the term-vs-ledger drift guard in Extend).
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

// Extend applies a §10 contract extension against the shared tx writer: it appends addedYears
// new PAID contract-year cells priced at 150% of the player's highest-paid remaining year
// (raised to the position floor, resolved by the Coordinator and passed as floor), lengthens
// the contract term, and RESETS is_restructured — the §10 unlock, where each extension
// re-allows one §11 restructure. Every §10 limit is enforced against authoritative state so a
// rule violation is unrepresentable: at least one year remaining (UFAs ineligible), no prior
// extension on this contract, 1..3 added years, no more than 6 total contract years, and one
// extension per franchise per season. A franchise-tagged player is OUT OF SCOPE in v1 (his
// extension years price at 120% of the tag — a §9 cross-rule, deferred) and is rejected. Fails
// loud on an unknown player, any limit breach, or a non-positive floor (an Extension not
// resolved through Coordinator.ExecuteExtension).
func Extend(ctx context.Context, w state.TxWriter, mflID string, addedYears int, floor domain.Money) error {
	ps, err := extendEligible(w, mflID, addedYears, floor)
	if err != nil {
		return err
	}
	cells, err := w.PaidCells(ctx, mflID)
	if err != nil {
		return fmt.Errorf("contracts: extend %q: %w", mflID, err)
	}
	// §10: no second extension based on a prior extension (must reach free agency first). A
	// prior extension is recorded ON THE CELLS (source "extension"), the ledger's own marker —
	// no is_extended flag. The whole-contract check is correct because free agency mints a fresh
	// contract_id whose cells are seed-sourced, so a re-signed player is eligible again. The scan
	// also returns the highest-paid remaining year, the §10 pricing base.
	highestRemaining, alreadyExtended, lastPaidYear := scanExtensionCells(cells, w.Season())
	if alreadyExtended {
		return fmt.Errorf("contracts: extend %q: contract already carries an extension — no second extension off a prior one (§10)", mflID)
	}
	if highestRemaining <= 0 {
		return fmt.Errorf("contracts: extend %q: no remaining paid year to price the extension from (§10)", mflID)
	}
	// Drift guard (GLM M3): the term (expiration_year, in the contracts table) and the ledger
	// tail (last PAID cell) must agree before we lengthen BOTH from them independently —
	// AppendExtensionYears extends from the cell tail, this handler bumps expiration_year. The
	// domain invariant keeps them equal; assert it rather than silently write a term that
	// diverges from the cells (fail loud, the house style).
	if lastPaidYear != ps.ExpirationYear {
		return fmt.Errorf("contracts: extend %q: term/ledger drift — expiration_year %d != last paid cell %d", mflID, ps.ExpirationYear, lastPaidYear)
	}
	// §10: no more than 6 total contract years (existing PAID cells + the added years).
	if total := len(cells) + addedYears; total > maxTotalContractYears {
		return fmt.Errorf("contracts: extend %q: %d existing + %d added = %d exceeds the %d-year max (§10)",
			mflID, len(cells), addedYears, total, maxTotalContractYears)
	}
	// §10: one extension per franchise per season.
	spent, err := w.OpCount(ctx, ps.FranchiseID, extensionOpKind)
	if err != nil {
		return fmt.Errorf("contracts: extend %q: %w", mflID, err)
	}
	if spent >= 1 {
		return fmt.Errorf("contracts: extend: franchise %q has already extended a contract this season (one per team per year, §10)", ps.FranchiseID)
	}

	price := extensionYearPrice(highestRemaining, floor)
	reason := fmt.Sprintf("§10 extension: +%d years at %s (150%% of %s, floor %s)", addedYears, price, highestRemaining, floor)
	if err := w.AppendExtensionYears(ctx, mflID, addedYears, price, reason); err != nil {
		return fmt.Errorf("contracts: extend %q: %w", mflID, err)
	}
	// Lengthen the term and RESET is_restructured (the §10 unlock). The base annual salary is
	// UNCHANGED — an extension adds future years at their own price; the existing deal's face
	// figure stands and the cells carry the per-year truth. ContractYears is left as-is (it is
	// not the store's computed value — expiration_year is the load-bearing term field, and the
	// cells are the source of truth for total length; tag/restructure leave it untouched too).
	change := state.ContractChange{
		AnnualSalary:   ps.Salary,
		ContractYears:  ps.ContractYears,
		ExpirationYear: ps.ExpirationYear + addedYears,
		ContractStatus: ps.ContractStatus,
		IsRestructured: false,
		IsTagged:       ps.IsTagged, // false here — a tagged player was rejected above
	}
	if err := w.ApplyContract(ctx, mflID, change); err != nil {
		return fmt.Errorf("contracts: extend %q: %w", mflID, err)
	}
	if err := w.IncOpCount(ctx, ps.FranchiseID, extensionOpKind); err != nil {
		return fmt.Errorf("contracts: extend %q: bump per-season counter: %w", mflID, err)
	}
	return nil
}
