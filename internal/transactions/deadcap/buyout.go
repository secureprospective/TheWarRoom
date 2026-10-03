package deadcap

import (
	"context"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// BuyoutReason labels §12 dead-cap rows. Exported because SIGN derives the buyout lockout from a
// row carrying it.
const BuyoutReason = "buyout §12"

const buyoutOpKind = "BUYOUT"

// maxBuyoutsPerSeason is §12's two per team per season. Offseason-only is the phase gate's job.
const maxBuyoutsPerSeason = 2

// buyoutRatePct is §12's rate for 2, 3 or 4 remaining years (60/75/90%). Any other count has no
// rate in the rulebook, so it fails and goes to the §13 commissioner path rather than inventing one.
func buyoutRatePct(remaining int) (int64, bool) {
	switch remaining {
	case 2:
		return 60, true
	case 3:
		return 75, true
	case 4:
		return 90, true
	default:
		return 0, false
	}
}

// buyoutCharge is rate × average remaining salary, snapped once to $10k. ok=false outside 2-4
// years.
func buyoutCharge(avgRemaining domain.Money, remaining int) (domain.Money, bool) {
	pct, ok := buyoutRatePct(remaining)
	if !ok {
		return 0, false
	}
	if avgRemaining <= 0 {
		return 0, true
	}
	cents := (int64(avgRemaining)*pct + 50) / 100 // × percent ÷ 100, round half-up
	return domain.RoundToNearest10k(domain.Money(cents)), true
}

// remainingAfter returns the PAID years after the current season and their average salary.
func remainingAfter(cells []state.LedgerCell, season int) (int, domain.Money) {
	var sum domain.Money
	var n int
	for _, c := range cells {
		if c.Year > season {
			sum += c.Salary
			n++
		}
	}
	if n == 0 {
		return 0, 0
	}
	return n, (sum + domain.Money(n)/2) / domain.Money(n)
}

// Buyout releases the player, charges §12 dead cap to this season, voids his cells and bumps the
// season's buyout count, in one transaction. Fails on an unknown player, a franchise that has used
// both buyouts, or a remaining-year count outside 2-4. Returns the ledger entry.
func Buyout(ctx context.Context, w state.TxWriter, mflID string) (state.DeadCapEntry, error) {
	ps, ok := w.Player(mflID)
	if !ok {
		return state.DeadCapEntry{}, fmt.Errorf("deadcap: buyout %q: player not on any roster", mflID)
	}

	used, err := w.OpCount(ctx, ps.FranchiseID, buyoutOpKind)
	if err != nil {
		return state.DeadCapEntry{}, fmt.Errorf("deadcap: buyout %q: op count: %w", mflID, err)
	}
	if used >= maxBuyoutsPerSeason {
		return state.DeadCapEntry{}, fmt.Errorf("deadcap: buyout %q: franchise %s has used its %d buyouts this season (§12)", mflID, ps.FranchiseID, maxBuyoutsPerSeason)
	}

	cells, err := w.PaidCells(ctx, mflID)
	if err != nil {
		return state.DeadCapEntry{}, fmt.Errorf("deadcap: buyout %q: read cells: %w", mflID, err)
	}
	remaining, avg := remainingAfter(cells, w.Season())
	charge, ok := buyoutCharge(avg, remaining)
	if !ok {
		return state.DeadCapEntry{}, fmt.Errorf("deadcap: buyout %q: %d remaining year(s) is outside §12's 2..4 range — route to the §13 commissioner cap-relief path", mflID, remaining)
	}

	if err := w.ReleasePlayer(ctx, mflID, domain.PlayerFreeAgent, BuyoutReason); err != nil {
		return state.DeadCapEntry{}, fmt.Errorf("deadcap: buyout %q: release: %w", mflID, err)
	}
	entry := state.DeadCapEntry{
		FranchiseID: ps.FranchiseID,
		MFLID:       mflID,
		LeagueYear:  w.Season(),
		DeadCap:     charge,
		Reason:      BuyoutReason,
	}
	if err := w.AddDeadCap(ctx, entry); err != nil {
		return state.DeadCapEntry{}, fmt.Errorf("deadcap: buyout %q: charge: %w", mflID, err)
	}
	reason := fmt.Sprintf("%s: contract bought out, dead cap %s", BuyoutReason, charge)
	if err := w.VoidCells(ctx, mflID, reason); err != nil {
		return state.DeadCapEntry{}, fmt.Errorf("deadcap: buyout %q: void cells: %w", mflID, err)
	}
	if err := w.IncOpCount(ctx, ps.FranchiseID, buyoutOpKind); err != nil {
		return state.DeadCapEntry{}, fmt.Errorf("deadcap: buyout %q: bump count: %w", mflID, err)
	}
	return entry, nil
}
