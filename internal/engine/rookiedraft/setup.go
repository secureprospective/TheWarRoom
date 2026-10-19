// Package rookiedraft plans and replays a native rookie draft from supplied league rules and facts.
// It performs no I/O and never substitutes Legacy values for an unset commissioner setting.
package rookiedraft

import (
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// Setup is the resolved competitive policy needed before the clock can run.
type Setup struct {
	Mode              leaguerules.DraftMode
	Clock             time.Duration
	OnExpire          leaguerules.ExpiryAction
	UnlimitedTimeouts bool
	Timeouts          int
	Rounds            int
	MaximumPicks      int
	ForfeitExtra      bool
}

func ResolveSetup(r leaguerules.DraftRules) (Setup, error) {
	if r.Mode.State != leaguerules.IsKnown || r.Clock.State != leaguerules.IsKnown ||
		r.OnExpire.State != leaguerules.IsKnown ||
		r.Rounds.State != leaguerules.IsKnown || r.MaximumPicks.State != leaguerules.IsKnown ||
		r.ForfeitExtra.State != leaguerules.IsKnown {
		return Setup{}, fmt.Errorf("needs a ruling: draft setup")
	}
	if (r.Mode.Value != leaguerules.SlowDraft && r.Mode.Value != leaguerules.LiveDraft) ||
		r.Clock.Value <= 0 || r.Rounds.Value < 1 || r.MaximumPicks.Value < 1 {
		return Setup{}, fmt.Errorf("set a draft pace, positive pick clock, round count and pick limit")
	}
	if r.OnExpire.Value != leaguerules.SkipPick {
		return Setup{}, fmt.Errorf("automatic player selection is not built yet; choose skip on expiry")
	}
	policy, err := timeoutPolicy(r.Timeouts)
	if err != nil {
		return Setup{}, err
	}
	return Setup{Mode: r.Mode.Value, Clock: r.Clock.Value, OnExpire: r.OnExpire.Value,
		UnlimitedTimeouts: policy.Unlimited.Value, Timeouts: policy.Count.Value, Rounds: r.Rounds.Value,
		MaximumPicks: r.MaximumPicks.Value, ForfeitExtra: r.ForfeitExtra.Value}, nil
}

func timeoutPolicy(value leaguerules.Value[leaguerules.TimeoutPolicy]) (leaguerules.TimeoutPolicy, error) {
	policy := value.Value
	if value.State != leaguerules.IsKnown || policy.Unlimited.State != leaguerules.IsKnown ||
		(!policy.Unlimited.Value && (policy.Count.State != leaguerules.IsKnown || policy.Count.Value < 0)) {
		return leaguerules.TimeoutPolicy{}, fmt.Errorf("needs a ruling: draft timeouts")
	}
	return policy, nil
}

// Terms is a slot's actual rookie contract, in exact cents, not a player valuation.
type Terms struct {
	Salary   domain.Money `json:"salaryCents"`
	Years    int          `json:"years"`
	RFAAfter bool         `json:"rfaAfter"`
}

func SlotTerms(r leaguerules.DraftRules, round, slot int) (Terms, error) {
	if r.Rounds.State != leaguerules.IsKnown || round < 1 || round > r.Rounds.Value || slot < 1 {
		return Terms{}, fmt.Errorf("choose a pick in this draft's rounds")
	}
	if r.Pricing.State != leaguerules.IsKnown || r.Lengths.State != leaguerules.IsKnown ||
		r.RFAAfter.State != leaguerules.IsKnown {
		return Terms{}, fmt.Errorf("needs a ruling: rookie salaries, contract years and restricted free agency")
	}
	years, rfa := r.Lengths.Value[round], r.RFAAfter.Value[round]
	if years.State != leaguerules.IsKnown || years.Value < 1 || rfa.State != leaguerules.IsKnown {
		return Terms{}, fmt.Errorf("needs a ruling: rookie contract terms for round %d", round)
	}
	salary, err := slotSalary(r.Pricing.Value[round], round, slot)
	if err != nil {
		return Terms{}, err
	}
	return Terms{Salary: salary, Years: years.Value, RFAAfter: rfa.Value}, nil
}

func slotSalary(prices []leaguerules.SlotPrice, round, slot int) (domain.Money, error) {
	matches := 0
	var salary domain.Money
	for _, price := range prices {
		if price.Slots.Min.State != leaguerules.IsKnown || price.Slots.Max.State != leaguerules.IsKnown {
			return 0, fmt.Errorf("needs a ruling: salary slot boundaries for round %d", round)
		}
		if slot < price.Slots.Min.Value || slot > price.Slots.Max.Value {
			continue
		}
		if price.Salary.State != leaguerules.IsKnown || price.Salary.Value < 0 {
			return 0, fmt.Errorf("needs a ruling: salary for round %d pick %d", round, slot)
		}
		matches++
		salary = domain.Money(price.Salary.Value)
	}
	if matches != 1 {
		return 0, fmt.Errorf("set exactly one salary for round %d pick %d", round, slot)
	}
	return salary, nil
}
