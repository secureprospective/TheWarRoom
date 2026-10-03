package deadcap

import (
	"context"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

const retirementReason = "retirement §13"

// gainesAdamsReason labels the $0 dead-cap row that records a §13 death.
const gainesAdamsReason = "gaines-adams §13"

// retirementPct is §13's rate: 30% of the contract remaining after this season.
const retirementPct = 30

// retirementCharge is 30% of the summed PAID cells after this season (actual cells, not
// annual × years), snapped once to $10k. Zero with no remaining year. Pure.
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

// Retire releases the player, charges 30% of his remaining contract to this season and voids his
// cells, in one transaction. Any phase, no limit.
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

	if err := w.ReleasePlayer(ctx, mflID, domain.PlayerRetired, retirementReason); err != nil {
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

// Death applies §13's Gaines Adams Rule: release with no cap penalty, cells voided, and a $0
// dead-cap row as the audit marker. Any phase.
func Death(ctx context.Context, w state.TxWriter, mflID string) (state.DeadCapEntry, error) {
	ps, ok := w.Player(mflID)
	if !ok {
		return state.DeadCapEntry{}, fmt.Errorf("deadcap: death %q: player not on any roster", mflID)
	}
	if err := w.ReleasePlayer(ctx, mflID, domain.PlayerDeceased, gainesAdamsReason); err != nil {
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

// Relieve credits a franchise's cap by `amount` (§13 appeal) on the cap-relief ledger. No player is
// released; the amount is the commissioner's judgment. Any phase.
func Relieve(ctx context.Context, w state.TxWriter, franchiseID string, amount domain.Money, reason string) (state.CapReliefEntry, error) {
	entry := state.CapReliefEntry{
		FranchiseID: franchiseID,
		LeagueYear:  w.Season(),
		// Snap once to $10k so the relief can't push the cap off-grid.
		Amount: domain.RoundToNearest10k(amount),
		Reason: reason,
	}
	if err := w.AddCapRelief(ctx, entry); err != nil {
		return state.CapReliefEntry{}, fmt.Errorf("deadcap: cap relief %q: %w", franchiseID, err)
	}
	// Return the snapped entry so the quote shows what CapUsed will actually subtract.
	return entry, nil
}
