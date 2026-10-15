package nativecheck

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// Input is everything one price reads: the league's rules, the contract, what has been done already, what
// the GM asked for and, for a tag, the base salaries of everyone at the position league-wide.
type Input struct {
	Rules    leaguerules.RuleSet
	Contract Contract
	Trail    Trail
	Ask      Ask
	Market   []domain.Money
}

// pricer holds one price while its rules are read. rem counts the seasons after this one that the
// contract still runs, the same count a release and a buyout charge by.
type pricer struct {
	in      Input
	rules   leaguerules.ContractsRules
	season  int
	rem     int
	quantum domain.Money
	p       Pricing
}

// PriceContract prices one action on one contract under the league's own rules, or says plainly why it
// cannot: the action is off, a limit is used up, the contract does not qualify, or a ruling is missing.
// It reads no calendar and no cap; the caller adds those. Every figure is exact cents, with percentage
// math taken to cents first and only then snapped to the league's increment.
func PriceContract(in Input, action ContractAction) (Pricing, error) {
	x := &pricer{in: in, rules: in.Rules.Contracts, season: in.Rules.Season}
	if !action.Known() {
		x.p.refuse("This league does not price that action.")
		return x.p, nil
	}
	if refusal, needs := Availability(x.rules, action); refusal != "" || needs != "" {
		x.p.Refusal = refusal
		if needs != "" {
			x.p.Needs = append(x.p.Needs, needs)
		}
		return x.p, nil
	}
	if in.Rules.Cap.Quantum.State != leaguerules.IsKnown || in.Rules.Cap.Quantum.Value <= 0 {
		x.p.need("the money rounding increment")
		return x.p, nil
	}
	x.quantum = domain.Money(in.Rules.Cap.Quantum.Value)
	c := in.Contract
	switch {
	case c.Salary <= 0 || c.Expiration == 0:
		x.p.refuse("No contract is on record for %s.", c.Name)
		return x.p, nil
	case c.Expiration < x.season:
		x.p.refuse("%s's contract ended after the %d season.", c.Name, c.Expiration)
		return x.p, nil
	}
	x.rem = c.Expiration - x.season
	return x.run(action)
}

func (x *pricer) run(action ContractAction) (Pricing, error) {
	var err error
	switch action {
	case ActionTag:
		err = x.tag()
	case ActionExtension:
		err = x.extension()
	case ActionRestructure:
		err = x.restructure()
	case ActionBuyout:
		err = x.buyout()
	case ActionTender:
		err = x.tender()
	case ActionResign:
		err = x.resign()
	case ActionCut:
		err = x.release()
	case ActionRetire:
		err = x.special(x.rules.Retirement.Value.Rate, "retirement")
	case ActionDeath:
		err = x.special(x.rules.Death.Value.Rate, "player death")
	case ActionCapRelief:
		err = x.capRelief()
	}
	if err != nil {
		return Pricing{}, fmt.Errorf("price the %s: %w", action.Name(), err)
	}
	return x.p, nil
}

// used refuses an action whose limit for the season is already reached. The limit is the league's.
func (x *pricer) used(action ContractAction, limit leaguerules.Value[int], one, many string) bool {
	if limit.State != leaguerules.IsKnown {
		x.p.need("how many " + many + " a franchise may use each season")
		return true
	}
	x.p.note("Yearly limits count only the contract actions recorded in this app league.")
	if n := x.in.Trail.Used[action]; n >= limit.Value {
		x.p.refuse("Your franchise has already used its %s this season.", plural(limit.Value, one, many))
		return true
	}
	return false
}

// remainingCells is the contract's remaining seasons after this one, each at its salary.
func (x *pricer) remainingCells() []domain.Money {
	cells := make([]domain.Money, x.rem)
	for i := range cells {
		cells[i] = x.in.Contract.Salary
	}
	return cells
}

// contractSchedule is the salary for every season from this one through the last, as it stands now.
func (x *pricer) contractSchedule() []SeasonAmount {
	out := make([]SeasonAmount, 0, x.rem+1)
	for season := x.season; season <= x.in.Contract.Expiration; season++ {
		out = append(out, SeasonAmount{Season: season, Cents: x.in.Contract.Salary})
	}
	return out
}

func (x *pricer) scale(amount domain.Money, rate leaguerules.Decimal) (domain.Money, error) {
	got, err := scaleMoney(amount, rate, x.quantum)
	if err != nil {
		return 0, fmt.Errorf("scale %s: %w", amount, err)
	}
	return got, nil
}

func (x *pricer) snap(amount domain.Money) (domain.Money, error) {
	return x.scale(amount, 10000)
}

// tag prices the franchise tag. Only a player in the last year of his deal can be tagged, and the new
// one-year contract starts the season after.
func (x *pricer) tag() error {
	r, c := x.rules.Tag.Value, x.in.Contract
	if x.used(ActionTag, r.AnnualLimit, "tag", "tags") {
		return nil
	}
	if x.rem != 0 {
		x.p.refuse("Only a player in the last year of his contract can be tagged; %s's runs through %d.",
			c.Name, c.Expiration)
		return nil
	}
	if r.ConsecutiveLimit.State != leaguerules.IsKnown {
		x.p.need("how many years in a row a player may be tagged")
		return nil
	}
	if x.in.Trail.ConsecutiveTags >= r.ConsecutiveLimit.Value {
		x.p.refuse("%s has already been tagged %s in a row, the most this league allows.", c.Name,
			plural(x.in.Trail.ConsecutiveTags, "year", "years"))
		return nil
	}
	price, ok, err := x.tagPrice(r)
	if err != nil || !ok {
		return err
	}
	x.p.Years, x.p.Salary = 1, &price
	x.p.Schedule = []SeasonAmount{{Season: x.season + 1, Cents: price}}
	if r.ExcludePlayoffContracts.State == leaguerules.IsKnown && r.ExcludePlayoffContracts.Value {
		x.p.note("The league leaves out contracts signed during the playoffs. This league's record does not " +
			"mark which were, so every salary at the position counts.")
	}
	return nil
}

func (x *pricer) tagPrice(r leaguerules.TagRules) (domain.Money, bool, error) {
	if x.in.Trail.ConsecutiveTags >= 1 {
		if r.SecondMultiplier.State != leaguerules.IsKnown {
			x.p.need("the price of a second tag in a row")
			return 0, false, nil
		}
		if x.in.Trail.FirstTag <= 0 {
			x.p.refuse("The first tag price is not on record, so a second tag cannot be priced.")
			return 0, false, nil
		}
		price, err := x.scale(x.in.Trail.FirstTag, r.SecondMultiplier.Value)
		x.p.basis("A second tag costs %s of the first tag, %s.", percentWords(r.SecondMultiplier.Value), x.in.Trail.FirstTag)
		return price, err == nil, err
	}
	if r.TopPlayers.State != leaguerules.IsKnown || r.FloorMultiplier.State != leaguerules.IsKnown {
		x.p.need("how many top salaries set the tag price, and the floor on it")
		return 0, false, nil
	}
	top, c := r.TopPlayers.Value, x.in.Contract
	if !c.PositionHeld {
		x.p.refuse("%s's position is not held, so the tag price cannot be set.", c.Name)
		return 0, false, nil
	}
	if top < 1 || len(x.in.Market) < top {
		x.p.refuse("Only %s at %s are on record; the tag price needs the top %d.",
			plural(len(x.in.Market), "salary", "salaries"), c.Position, top)
		return 0, false, nil
	}
	return x.topAverage(r, top)
}

func (x *pricer) topAverage(r leaguerules.TagRules, top int) (domain.Money, bool, error) {
	pool := slices.Clone(x.in.Market)
	slices.SortFunc(pool, func(a, b domain.Money) int { return cmp.Compare(b, a) })
	average, err := shareOfAverage(pool[:top], 10000, 1)
	if err != nil {
		return 0, false, fmt.Errorf("average the top salaries: %w", err)
	}
	c := x.in.Contract
	if average < c.Salary {
		price, err := x.scale(c.Salary, r.FloorMultiplier.Value)
		x.p.basis("The average of the top %d %s salaries, %s, is below his own %s, so the tag is %s of his salary.",
			top, c.Position, average, c.Salary, percentWords(r.FloorMultiplier.Value))
		return price, err == nil, err
	}
	price, err := x.snap(average)
	x.p.basis("The tag is the average of the top %d %s salaries in the league.", top, c.Position)
	return price, err == nil, err
}
