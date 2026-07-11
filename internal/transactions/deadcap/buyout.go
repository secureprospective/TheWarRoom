package deadcap

import (
	"context"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// BuyoutReason is the audit string on every §12 dead-cap ledger row. Exported because the
// free-agency SIGN handler derives the §12 "no re-bid until the following offseason" lockout from
// the presence of a dead_cap row carrying this reason (Free_Agency_Design Q3).
const BuyoutReason = "buyout §12"

// buyoutOpKind is the transaction_counts op_kind for the §12 per-season buyout limit.
const buyoutOpKind = "BUYOUT"

// maxBuyoutsPerSeason is §12's "two buyouts per team per season". The offseason-only rule is
// enforced upstream by the Coordinator's phase gate, NOT re-checked here (single source of truth).
const maxBuyoutsPerSeason = 2

// buyoutRatePct returns §12's dead-cap rate (whole percent) for a contract with `remaining`
// years left, and whether §12 defines a rate at all. The rulebook enumerates ONLY 2/3/4 years
// (60/75/90%). 1 year, or 5–6 (reachable via a §10 extension), have NO defined rate — ok=false,
// so the handler fails loud and routes the case to the §13 commissioner path rather than
// inventing a number (expert-panel A6 / locked GQ3).
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

// buyoutCharge computes the §12 dead-cap charge: rate% × average remaining salary, in exact
// cents, snapped ONCE to the $10k grid (flat-$10k doctrine, matching §8's Charge). ok=false if
// `remaining` is outside §12's defined 2..4 range. A non-positive average yields a $0 charge (a
// valid, if degenerate, buyout — not an error).
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

// remainingAfter reduces a player's PAID cells to §12's inputs: the count of years strictly
// AFTER the current season (the "remaining years" convention shared with §8/§10 — exclusive of
// the current year) and the arithmetic mean of those cells' salaries (round half-up to the cent).
// Returns (0, 0) when the player has no remaining paid year.
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

// Buyout executes a §12 contract buyout against the shared tx: it reads the player's remaining
// paid years, computes the §12 dead-cap charge (rate by years remaining × average remaining
// salary), releases him, records the charge to the current season's cap, voids his cells, and
// bumps the per-season buyout counter — all in the Coordinator's one spanning transaction, so
// the release and its penalty land together or not at all. The offseason-only rule is enforced
// by the Coordinator phase gate before this runs (not re-checked here). Fails loud on an unknown
// player, a franchise that has used its two buyouts this season, or a remaining-year count
// outside §12's defined 2..4 range (routed to the §13 commissioner path). Returns the ledger
// entry it wrote so the caller can surface the charge.
func Buyout(ctx context.Context, w state.TxWriter, mflID string) (state.DeadCapEntry, error) {
	ps, ok := w.Player(mflID)
	if !ok {
		return state.DeadCapEntry{}, fmt.Errorf("deadcap: buyout %q: player not on any roster", mflID)
	}

	// §12 "two buyouts per team per season" — read, check, (mutate), bump, all atomic in the tx.
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
