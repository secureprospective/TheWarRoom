package market

import (
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// addCap counts the franchise's salary before and after the action, with the cap reading's own rules, and
// attaches what changed. It reports whether the cap itself refuses the move. A move that does not raise
// this season's counted salary is never refused for the cap.
func (in contractCheck) addCap(q *ContractQuote, p nativecheck.Pricing) (bool, error) {
	rules := in.base.Rules
	before, after := in.capCounts(p)
	was, err := nativecheck.CheckCap(rules.Cap, before.players, before.charges, in.zone, commissioner.ScopedDecisionExceptions(in.base.Records, commissioner.ExceptionScope{Franchise: in.player.Franchise, Player: in.player.ID}, in.base.Now)...)
	if err != nil {
		return false, fmt.Errorf("count the salary cap before the action: %w", err)
	}
	now, err := nativecheck.CheckCap(rules.Cap, after.players, after.charges, in.zone, commissioner.ScopedDecisionExceptions(in.base.Records, commissioner.ExceptionScope{Franchise: in.player.Franchise, Player: in.player.ID}, in.base.Now)...)
	if err != nil {
		return false, fmt.Errorf("count the salary cap after the action: %w", err)
	}
	change, counted, err := nativecheck.CountedChange(was, now)
	if err != nil {
		return false, fmt.Errorf("compare the salary cap counts: %w", err)
	}
	if !counted || was.CapCents == nil {
		q.Needs = append(q.Needs, "Needs a ruling: the salary cap and the salary shares of injured reserve and taxi")
		return false, nil
	}
	effect := ContractCapEffect{BeforeCents: was.CommittedCents, AfterCents: now.CommittedCents, ChangeCents: &change,
		RoomCents: now.RoomCents}
	q.Cap = &effect
	if change <= 0 {
		effect.Words = fmt.Sprintf("Counted salary this season goes from %s to %s; the cap is no tighter.",
			*was.CommittedCents, *now.CommittedCents)
		return false, nil
	}
	allowed, words := nativecheck.CapAllows(was, change, in.zone)
	effect.Words = fmt.Sprintf("Counted salary this season rises by %s, from %s to %s. %s.", change,
		*was.CommittedCents, *now.CommittedCents, words)
	return !allowed, nil
}

// capCount is one franchise's counted contracts and signed charges for the season.
type capCount struct {
	players []nativecheck.CapPlayer
	charges []domain.Money
}

// capCounts is the franchise's contracts and charges as they stand, and as they would after the action.
func (in contractCheck) capCounts(p nativecheck.Pricing) (before, after capCount) {
	seed, season := in.base.Assets.Snapshot, in.base.Rules.Season
	for _, held := range seed.Players {
		if held.Franchise != in.player.Franchise {
			continue
		}
		before.players = append(before.players, nativecheck.CapPlayer{Salary: held.Salary, Status: held.Status})
		if held.ID == in.player.ID {
			if p.Removed {
				continue
			}
			held.Salary = salaryIn(p.Schedule, season, held.Salary)
		}
		after.players = append(after.players, nativecheck.CapPlayer{Salary: held.Salary, Status: held.Status})
	}
	for _, c := range seed.Adjustments {
		if c.Franchise == in.player.Franchise && (c.Season == 0 || c.Season == season) {
			before.charges = append(before.charges, c.Amount)
		}
	}
	after.charges = append([]domain.Money{}, before.charges...)
	for _, dead := range p.Dead {
		if dead.Season == season {
			after.charges = append(after.charges, dead.Cents)
		}
	}
	if p.Relief > 0 {
		after.charges = append(after.charges, -p.Relief)
	}
	return before, after
}

// salaryIn is the contract's salary in one season of a schedule, or the current salary if it has none there.
func salaryIn(schedule []nativecheck.SeasonAmount, season int, current domain.Money) domain.Money {
	for _, cell := range schedule {
		if cell.Season == season {
			return cell.Cents
		}
	}
	return current
}
