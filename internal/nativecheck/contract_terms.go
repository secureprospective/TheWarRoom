package nativecheck

import (
	"fmt"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// extension prices added years on a contract with years left, or on a tagged player's one-year deal.
func (x *pricer) extension() error {
	r, c := x.rules.Extension.Value, x.in.Contract
	if x.used(ActionExtension, r.AnnualLimit, "extension", "extensions") {
		return nil
	}
	if r.UFAEligible.State != leaguerules.IsKnown || r.MinimumRemaining.State != leaguerules.IsKnown ||
		r.RepeatBeforeFreeAgency.State != leaguerules.IsKnown {
		x.p.need("who may be extended: free agents, years left, and a second extension")
		return nil
	}
	tagged := c.Status == domain.CStatusFT1 || c.Status == domain.CStatusFT2
	switch {
	case !tagged && x.rem == 0 && !r.UFAEligible.Value:
		x.p.refuse("%s is in the last year of his contract and would become an unrestricted free agent; "+
			"this league does not extend those.", c.Name)
	case !tagged && x.rem > 0 && x.rem < r.MinimumRemaining.Value:
		x.p.refuse("A contract needs at least %s left after this season to be extended; %s's runs through %d.",
			plural(r.MinimumRemaining.Value, "year", "years"), c.Name, c.Expiration)
	case x.in.Trail.Extensions > 0 && !r.RepeatBeforeFreeAgency.Value:
		x.p.refuse("%s has already been extended and must reach free agency before another extension.", c.Name)
	default:
		return x.priceExtension(r, tagged)
	}
	return nil
}

func (x *pricer) priceExtension(r leaguerules.ExtensionRules, tagged bool) error {
	added, ok := x.addedYears(r)
	if !ok {
		return nil
	}
	price, ok, err := x.extensionPrice(r, tagged)
	if err != nil || !ok {
		return err
	}
	last := x.in.Contract.Expiration
	x.p.Years, x.p.Salary = added, &price
	x.p.Schedule = x.contractSchedule()
	for year := last + 1; year <= last+added; year++ {
		x.p.Schedule = append(x.p.Schedule, SeasonAmount{Season: year, Cents: price})
	}
	return nil
}

// addedYears is how many years the extension adds: the most the league allows, unless the GM asked for fewer.
func (x *pricer) addedYears(r leaguerules.ExtensionRules) (int, bool) {
	if r.AddedYears.State != leaguerules.IsKnown || r.TotalYears.State != leaguerules.IsKnown {
		x.p.need("how many years an extension may add, and the longest total contract")
		return 0, false
	}
	total := x.rem + 1
	most := min(r.AddedYears.Value, r.TotalYears.Value-total)
	added := x.in.Ask.Years
	switch {
	case most < 1:
		x.p.refuse("%s's contract is already %s long, the most this league allows is %d.",
			x.in.Contract.Name, plural(total, "year", "years"), r.TotalYears.Value)
	case added > r.AddedYears.Value:
		x.p.refuse("An extension adds at most %s.", plural(r.AddedYears.Value, "year", "years"))
	case added > most:
		x.p.refuse("That would make %s, and the longest a contract can run is %d.",
			plural(total+added, "contract year", "contract years"), r.TotalYears.Value)
	case added < 0:
		x.p.refuse("Choose how many years to add.")
	case added == 0:
		return most, true
	default:
		return added, true
	}
	return 0, false
}

func (x *pricer) extensionPrice(r leaguerules.ExtensionRules, tagged bool) (domain.Money, bool, error) {
	c := x.in.Contract
	if tagged {
		tag := x.rules.Tag.Value
		if x.rules.Tag.State != leaguerules.IsKnown || tag.ExtensionMultiplier.State != leaguerules.IsKnown {
			x.p.need("the price of extension years for a tagged player")
			return 0, false, nil
		}
		price, err := x.scale(c.Salary, tag.ExtensionMultiplier.Value)
		x.p.basis("Each added year of a tagged player costs %s of his tag price, %s.",
			percentWords(tag.ExtensionMultiplier.Value), c.Salary)
		return price, err == nil, err
	}
	if r.Multiplier.State != leaguerules.IsKnown || r.Floors.State != leaguerules.IsKnown {
		x.p.need("the extension price and the yearly floor for each position")
		return 0, false, nil
	}
	if !c.PositionHeld {
		x.p.refuse("%s's position is not held, so the extension floor cannot be set.", c.Name)
		return 0, false, nil
	}
	floor, ok := r.Floors.Value[c.Position]
	if !ok || floor.State != leaguerules.IsKnown {
		x.p.need(fmt.Sprintf("the yearly floor for an extended %s", c.Position))
		return 0, false, nil
	}
	scaled, err := x.scale(c.Salary, r.Multiplier.Value)
	if err != nil {
		return 0, false, err
	}
	price, err := x.snap(max(scaled, domain.Money(floor.Value)))
	x.p.basis("Each added year costs %s of his highest remaining salary, %s, but not less than the %s floor of %s.",
		percentWords(r.Multiplier.Value), c.Salary, c.Position, domain.Money(floor.Value))
	return price, err == nil, err
}

// restructure moves salary out of this season into the contract's later years. The total stays the same.
func (x *pricer) restructure() error {
	r, c := x.rules.Restructure.Value, x.in.Contract
	if x.used(ActionRestructure, r.AnnualLimit, "restructure", "restructures") {
		return nil
	}
	if r.PerContract.State != leaguerules.IsKnown || r.ExtraPerExtension.State != leaguerules.IsKnown ||
		r.Tiers.State != leaguerules.IsKnown || r.ConserveTotal.State != leaguerules.IsKnown ||
		r.ExistingYearsOnly.State != leaguerules.IsKnown {
		x.p.need("how often a contract may be restructured, how much may move, and where it may go")
		return nil
	}
	allowed := r.PerContract.Value + r.ExtraPerExtension.Value*x.in.Trail.Extensions
	switch {
	case !r.ConserveTotal.Value || !r.ExistingYearsOnly.Value:
		x.p.need("how a restructure may change a contract's total or add years")
	case x.in.Trail.Restructures >= allowed:
		x.p.refuse("%s's contract has already been restructured %s; an extension unlocks one more.", c.Name,
			plural(x.in.Trail.Restructures, "time", "times"))
	case x.rem < 1:
		x.p.refuse("A restructure moves salary into later years, and %s's contract has none left.", c.Name)
	default:
		return x.priceRestructure(r)
	}
	return nil
}

func (x *pricer) priceRestructure(r leaguerules.RestructureRules) error {
	c := x.in.Contract
	most, ok := x.moveLimit(r)
	if !ok {
		return nil
	}
	move := most
	if x.in.Ask.Move != 0 {
		var err error
		if move, err = x.snap(x.in.Ask.Move); err != nil {
			return err
		}
	}
	switch {
	case move <= 0:
		x.p.refuse("Choose an amount of salary to move.")
		return nil
	case move > most:
		x.p.refuse("At most %s can move out of a %s season.", most, c.Salary)
		return nil
	}
	x.p.Schedule = x.spread(move)
	x.p.basis("%s moves out of %d into the %s after it, spread evenly; the contract's total stays the same.",
		move, x.season, plural(x.rem, "later year", "later years"))
	x.p.basis("The most that can move is %s, set by this season's salary of %s.", most, c.Salary)
	x.restructuredNote()
	return nil
}

// moveLimit is the most this season's salary can give up: the amount of the highest tier it reaches.
func (x *pricer) moveLimit(r leaguerules.RestructureRules) (domain.Money, bool) {
	var most, reached domain.Money
	for _, tier := range r.Tiers.Value {
		if tier.At.State != leaguerules.IsKnown || tier.Amount.State != leaguerules.IsKnown {
			x.p.need("each salary level and the most that may move at it")
			return 0, false
		}
		at := domain.Money(tier.At.Value)
		if x.in.Contract.Salary >= at && at >= reached {
			reached, most = at, domain.Money(tier.Amount.Value)
		}
	}
	if most <= 0 {
		x.p.refuse("A restructure needs a yearly salary of at least %s; %s's is %s.",
			lowestTier(r), x.in.Contract.Name, x.in.Contract.Salary)
		return 0, false
	}
	return most, true
}

func lowestTier(r leaguerules.RestructureRules) domain.Money {
	var lowest domain.Money
	for i, tier := range r.Tiers.Value {
		if at := domain.Money(tier.At.Value); i == 0 || at < lowest {
			lowest = at
		}
	}
	return lowest
}

// spread leaves this season smaller by the move and adds it to the later years in equal increments of the
// league's money step; whatever does not divide goes to the last year.
func (x *pricer) spread(move domain.Money) []SeasonAmount {
	steps := move / x.quantum / domain.Money(x.rem)
	share := steps * x.quantum
	out := []SeasonAmount{{Season: x.season, Cents: x.in.Contract.Salary - move}}
	given := domain.Money(0)
	for i := 1; i <= x.rem; i++ {
		part := share
		if i == x.rem {
			part = move - given
		}
		given += part
		out = append(out, SeasonAmount{Season: x.season + i, Cents: x.in.Contract.Salary + part})
	}
	return out
}

func (x *pricer) restructuredNote() {
	r := x.rules
	if r.DeadCap.State == leaguerules.IsKnown && r.RestructuredDeadCap.State == leaguerules.IsKnown {
		x.p.note("If a restructured contract is released later, the charge is %s of the remaining salaries "+
			"instead of %s.", percentWords(r.RestructuredDeadCap.Value), percentWords(r.DeadCap.Value))
	}
}

// buyout charges a share of the average remaining salary, all to this season, and ends the contract.
func (x *pricer) buyout() error {
	r := x.rules.Buyout.Value
	if x.used(ActionBuyout, r.AnnualLimit, "buyout", "buyouts") {
		return nil
	}
	if r.OneYearAllowed.State != leaguerules.IsKnown || r.Rates.State != leaguerules.IsKnown {
		x.p.need("the buyout rates and whether a contract with one year left may be bought out")
		return nil
	}
	c := x.in.Contract
	switch {
	case x.rem == 0:
		x.p.refuse("%s has no years left after this season, so there is nothing to buy out.", c.Name)
		return nil
	case x.rem == 1 && !r.OneYearAllowed.Value:
		x.p.refuse("A contract with one year left cannot be bought out; release %s instead.", c.Name)
		return nil
	}
	rate, ok := r.Rates.Value[x.rem]
	switch {
	case !ok && x.rem == 1:
		x.p.need("the buyout rate for a contract with one year left")
		return nil
	case !ok:
		x.p.refuse("This league's buyout table has no rate for %s left.", plural(x.rem, "year", "years"))
		return nil
	case rate.State != leaguerules.IsKnown:
		x.p.need(fmt.Sprintf("the buyout rate for %s left", plural(x.rem, "year", "years")))
		return nil
	}
	cells := x.remainingCells()
	charge, err := shareOfAverage(cells, rate.Value, x.quantum)
	if err != nil {
		return err
	}
	x.p.Removed = true
	x.p.Dead = []SeasonAmount{{Season: x.season, Cents: charge}}
	x.p.basis("%s of the average of his %s, %s each, all charged to the %d season.",
		percentWords(rate.Value), plural(x.rem, "remaining salary", "remaining salaries"), c.Salary, x.season)
	if r.RebidNextOffseason.State == leaguerules.IsKnown && r.RebidNextOffseason.Value {
		x.p.note("%s cannot be bid on again until the next offseason.", c.Name)
	}
	return nil
}

// release charges the league's dead-money rate on every salary after this season, all to this season.
func (x *pricer) release() error {
	rate, which := x.rules.DeadCap, "release"
	if x.in.Trail.Restructures > 0 {
		rate, which = x.rules.RestructuredDeadCap, "release of a restructured contract"
	}
	if rate.State != leaguerules.IsKnown {
		x.p.need("the dead money rate for a " + which)
		return nil
	}
	if err := x.charge(rate.Value, "of each of his remaining salaries"); err != nil {
		return err
	}
	if claim := x.rules.ClaimClearsDeadCap; claim.State == leaguerules.IsKnown {
		if claim.Value {
			x.p.note("If another team claims %s on waivers, this charge ends.", x.in.Contract.Name)
		} else {
			x.p.note("This charge stays even if another team claims %s on waivers.", x.in.Contract.Name)
		}
	}
	return nil
}

// special prices a retirement or a death: a rate of the salaries left, charged to this season.
func (x *pricer) special(rate leaguerules.Value[leaguerules.Decimal], what string) error {
	if rate.State != leaguerules.IsKnown {
		x.p.need("the cap charge for " + what)
		return nil
	}
	return x.charge(rate.Value, "of each of his remaining salaries")
}

// charge removes the player and lands rate × the sum of the salaries after this season in this season.
func (x *pricer) charge(rate leaguerules.Decimal, words string) error {
	sum, err := sumMoney(x.remainingCells())
	if err != nil {
		return err
	}
	amount, err := x.scale(sum, rate)
	if err != nil {
		return err
	}
	x.p.Removed = true
	if amount > 0 {
		x.p.Dead = []SeasonAmount{{Season: x.season, Cents: amount}}
		x.p.basis("%s %s, %s in all, charged to the %d season.", percentWords(rate), words, sum, x.season)
		return nil
	}
	x.p.basis("No cap charge: %s of %s is nothing.", percentWords(rate), sum)
	return nil
}

// capRelief is the commissioner returning part of a cap hit for one of the league's listed reasons.
func (x *pricer) capRelief() error {
	r, c := x.rules.CapRelief.Value, x.in.Contract
	if r.Reasons.State != leaguerules.IsKnown {
		x.p.need("the reasons that earn cap relief")
		return nil
	}
	reason := strings.ToLower(strings.TrimSpace(x.in.Ask.Reason))
	found := false
	for _, allowed := range r.Reasons.Value {
		found = found || strings.ToLower(allowed) == reason
	}
	switch {
	case !found:
		x.p.refuse("Cap relief is for one of: %s.", strings.Join(r.Reasons.Value, ", "))
	case x.in.Ask.Amount <= 0:
		x.p.refuse("Say how much of %s's cap hit the commissioner would relieve.", c.Name)
	default:
		return x.priceRelief()
	}
	return nil
}

func (x *pricer) priceRelief() error {
	amount, err := x.snap(x.in.Ask.Amount)
	if err != nil {
		return err
	}
	if amount > x.in.Contract.Salary {
		x.p.refuse("Relief cannot be more than %s's %s salary.", x.in.Contract.Name, x.in.Contract.Salary)
		return nil
	}
	x.p.Relief = amount
	x.p.basis("The commissioner returns %s of this season's cap hit; the salary on the contract does not change.", amount)
	return nil
}
