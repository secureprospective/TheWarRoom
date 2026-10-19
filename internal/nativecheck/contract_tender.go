package nativecheck

import (
	"fmt"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// tender prices a restricted free agent's tender: the level's price, or the league's share of his last
// salary if that is more. The one-year deal starts the season after.
func (x *pricer) tender() error {
	price, ok, err := x.tenderPrice()
	if err != nil || !ok {
		return err
	}
	x.p.Years, x.p.Salary = 1, &price
	x.p.Schedule = []SeasonAmount{{Season: x.season + 1, Cents: price}}
	return nil
}

func (x *pricer) tenderPrice() (domain.Money, bool, error) {
	r, c := x.rules.Tender.Value, x.in.Contract
	if c.Status != domain.CStatusRFA || x.rem != 0 {
		x.p.refuse("Only a restricted free agent in the last year of his contract can be tendered; "+
			"%s is not one.", c.Name)
		return 0, false, nil
	}
	if r.Prices.State != leaguerules.IsKnown || r.PriorSalaryMultiplier.State != leaguerules.IsKnown {
		x.p.need("the tender prices and the share of his last salary a tender must reach")
		return 0, false, nil
	}
	level := x.in.Ask.Level
	if level < 1 || level > len(r.Prices.Value) {
		x.p.refuse("Choose a tender level from 1 to %d.", len(r.Prices.Value))
		return 0, false, nil
	}
	tender := r.Prices.Value[level-1]
	if tender.Price.State != leaguerules.IsKnown {
		x.p.need(fmt.Sprintf("the price of tender level %d", level))
		return 0, false, nil
	}
	floor, err := x.scale(c.Salary, r.PriorSalaryMultiplier.Value)
	if err != nil {
		return 0, false, err
	}
	price := max(domain.Money(tender.Price.Value), floor)
	x.p.basis("Tender level %d is %s%s. %s of his %s salary is %s; the higher applies.", level,
		domain.Money(tender.Price.Value), compensationWords(tender), percentWords(r.PriorSalaryMultiplier.Value),
		c.Salary, floor)
	return price, true, nil
}

func compensationWords(t leaguerules.Tender) string {
	if t.Rounds.State != leaguerules.IsKnown || len(t.Rounds.Value) == 0 {
		return ""
	}
	rounds := make([]string, len(t.Rounds.Value))
	for i, round := range t.Rounds.Value {
		rounds[i] = fmt.Sprint(round)
	}
	word := "round"
	if len(rounds) > 1 {
		word = "rounds"
	}
	return ", with draft pick compensation in " + word + " " + strings.Join(rounds, " and ")
}

// resign prices keeping a restricted free agent for the years chosen: the tender plus the league's
// premium for that length, flat from the season after.
func (x *pricer) resign() error {
	r := x.rules.ReSign.Value
	tender, ok, err := x.tenderPrice()
	if err != nil || !ok {
		return err
	}
	years, ok := x.resignYears(r)
	if !ok {
		return nil
	}
	premium := r.Premiums.Value[years]
	price, err := x.scale(tender, 10000+premium.Value)
	if err != nil {
		return err
	}
	x.p.Years, x.p.Salary = years, &price
	for i := 1; i <= years; i++ {
		x.p.Schedule = append(x.p.Schedule, SeasonAmount{Season: x.season + i, Cents: price})
	}
	x.p.basis("A %d-year deal costs the tender plus %s: %s a year.", years, percentWords(premium.Value), price)
	return nil
}

func (x *pricer) resignYears(r leaguerules.ReSignRules) (int, bool) {
	years, longest := x.in.Ask.Years, x.rules.MaximumYears
	if r.Premiums.State != leaguerules.IsKnown || longest.State != leaguerules.IsKnown {
		x.p.need("the re-sign premiums and the longest contract")
		return 0, false
	}
	premium, listed := r.Premiums.Value[years]
	switch {
	case years < 1:
		x.p.refuse("Choose how many years to re-sign for.")
	case years > longest.Value:
		x.p.refuse("That is longer than the league's maximum contract of %s.", plural(longest.Value, "year", "years"))
	case !listed:
		x.p.refuse("This league has no re-sign price for %s.", plural(years, "year", "years"))
	case premium.State != leaguerules.IsKnown:
		x.p.need(fmt.Sprintf("the re-sign premium for %s", plural(years, "year", "years")))
	default:
		return years, true
	}
	return 0, false
}
