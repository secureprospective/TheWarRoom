package deadcap

import (
	"context"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// §13 SPECIAL SITUATIONS — the retirement, death (Gaines Adams Rule), and cap-relief handlers.
// Retirement and Death reuse the §8 release/void path (release the player, void his cells);
// they differ only in the dead-cap charge they write. Cap Relief is a franchise-scoped CREDIT
// with no player release. All three run BEHIND the Coordinator depguard, like every deadcap op.

// retirementReason is the audit string on a §13 retirement dead-cap row.
const retirementReason = "retirement §13"

// gainesAdamsReason is the audit string on a §13 death (Gaines Adams Rule) row — a $0 dead-cap
// marker recording the no-penalty removal in the ledger.
const gainesAdamsReason = "gaines-adams §13"

// retirementPct is §13's retirement rate: the team carries 30% of the player's REMAINING
// contract (the salary of every year strictly after the current season). Whole percent, so the
// cents math stays exact bar the final round.
const retirementPct = 30

// retirementCharge computes the §13 retirement dead-cap charge in exact cents:
//
//	30% × Σ(salary of each PAID year strictly after the current season)
//
// It sums the ACTUAL remaining cells (the ledger is king — no annual×years approximation), then
// snaps ONCE to the $10k grid (flat-$10k doctrine, matching §8/§12). The current season's salary
// is excluded (already on the books, the §8/§12 exclusive-of-current convention). $0 when the
// player has no remaining paid year (a final-year/UFA contract — retirement then costs nothing).
// Pure — no I/O, no clock.
func retirementCharge(cells []state.LedgerCell, season int) domain.Money {
	var sum domain.Money
	for _, c := range cells {
		if c.Year > season {
			sum += c.Salary
		}
	}
	if sum <= 0 {
		return 0
	}
	cents := (int64(sum)*retirementPct + 50) / 100 // × percent ÷ 100, round half-up
	return domain.RoundToNearest10k(domain.Money(cents))
}

// Retire executes a §13 retirement against the shared tx: it reads the player's remaining paid
// years, computes the 30%-of-remaining dead-cap charge, releases him, records the charge to the
// current season's cap, and voids his cells — all in the Coordinator's one spanning transaction,
// so the removal and its penalty land together or not at all. Legal in every phase (a player may
// retire whenever) and unlimited per season. Fails loud on an unknown player. Returns the ledger
// entry it wrote so the caller can surface the charge.
func Retire(ctx context.Context, w state.TxWriter, mflID string) (state.DeadCapEntry, error) {
	ps, ok := w.Player(mflID)
	if !ok {
		return state.DeadCapEntry{}, fmt.Errorf("deadcap: retire %q: player not on any roster", mflID)
	}
	cells, err := w.PaidCells(ctx, mflID)
	if err != nil {
		return state.DeadCapEntry{}, fmt.Errorf("deadcap: retire %q: read cells: %w", mflID, err)
	}
	charge := retirementCharge(cells, w.Season())

	if err := w.ReleasePlayer(ctx, mflID); err != nil {
		return state.DeadCapEntry{}, fmt.Errorf("deadcap: retire %q: release: %w", mflID, err)
	}
	entry := state.DeadCapEntry{
		FranchiseID: ps.FranchiseID,
		MFLID:       mflID,
		LeagueYear:  w.Season(),
		DeadCap:     charge,
		Reason:      retirementReason,
	}
	if err := w.AddDeadCap(ctx, entry); err != nil {
		return state.DeadCapEntry{}, fmt.Errorf("deadcap: retire %q: charge: %w", mflID, err)
	}
	reason := fmt.Sprintf("%s: contract voided, dead cap %s", retirementReason, charge)
	if err := w.VoidCells(ctx, mflID, reason); err != nil {
		return state.DeadCapEntry{}, fmt.Errorf("deadcap: retire %q: void cells: %w", mflID, err)
	}
	return entry, nil
}

// Death executes the §13 Gaines Adams Rule against the shared tx: a player's death removes him
// from his roster with NO cap penalty — a cut with ZERO dead cap. It releases him, voids his
// cells (relieving every cap-bearing year), and records a $0 dead-cap row as the explicit audit
// marker that he was removed under Gaines Adams with no charge — all atomic in the Coordinator's
// one spanning transaction. Legal in every phase. Fails loud on an unknown player. Returns the
// ($0) ledger entry it wrote.
func Death(ctx context.Context, w state.TxWriter, mflID string) (state.DeadCapEntry, error) {
	ps, ok := w.Player(mflID)
	if !ok {
		return state.DeadCapEntry{}, fmt.Errorf("deadcap: death %q: player not on any roster", mflID)
	}
	if err := w.ReleasePlayer(ctx, mflID); err != nil {
		return state.DeadCapEntry{}, fmt.Errorf("deadcap: death %q: release: %w", mflID, err)
	}
	entry := state.DeadCapEntry{
		FranchiseID: ps.FranchiseID,
		MFLID:       mflID,
		LeagueYear:  w.Season(),
		DeadCap:     0, // Gaines Adams Rule: no cap penalty
		Reason:      gainesAdamsReason,
	}
	if err := w.AddDeadCap(ctx, entry); err != nil {
		return state.DeadCapEntry{}, fmt.Errorf("deadcap: death %q: audit row: %w", mflID, err)
	}
	reason := fmt.Sprintf("%s: contract voided, no cap penalty", gainesAdamsReason)
	if err := w.VoidCells(ctx, mflID, reason); err != nil {
		return state.DeadCapEntry{}, fmt.Errorf("deadcap: death %q: void cells: %w", mflID, err)
	}
	return entry, nil
}

// Relieve executes a §13 Cap Relief Appeal against the shared tx: the commissioner reduces a
// franchise's cap hit by `amount` (career-ending injury, recurring injury, behavioral
// suspension). It appends one positive credit to the cap_relief_ledger, which CapUsed subtracts —
// a franchise-scoped adjustment, no player release. Every figure is the commissioner's judgment,
// validated only for shape (positive amount, non-empty franchise/reason) by the store. Legal in
// every phase. It touches no players (returns nothing to release).
func Relieve(ctx context.Context, w state.TxWriter, franchiseID string, amount domain.Money, reason string) error {
	entry := state.CapReliefEntry{
		FranchiseID: franchiseID,
		LeagueYear:  w.Season(),
		// Snap the commissioner's discretionary figure to the $10k grid ONCE (the universal FLAT
		// $10k invariant, matching §8/§12/retirement): CapUsed subtracts this, so an off-grid relief
		// would push a franchise's cap off-grid. Snap before the write keeps the stored ledger row
		// and the derived cap in lockstep (GLM §13 review).
		Amount: domain.RoundToNearest10k(amount),
		Reason: reason,
	}
	if err := w.AddCapRelief(ctx, entry); err != nil {
		return fmt.Errorf("deadcap: cap relief %q: %w", franchiseID, err)
	}
	return nil
}
