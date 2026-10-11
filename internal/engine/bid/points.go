// Package bid is the free-agent auction's arithmetic and rules (rulebook §6): bid points, the
// minimum salary by experience, and the named checks a bid plan must pass. Pure: every input
// arrives as a parameter, and money is exact cents.
package bid

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// Points are bid points in thousandths: the rulebook posts bid totals to three decimals, so
// integer math is exact where a float would drift.
type Points int64

const (
	pointsPerUnit = 1000
	// centsPerPointMilli turns cents times a hundredths multiplier into thousandths of a point:
	// $1M is 100,000,000 cents, and one point is $1M at 1.00x.
	centsPerPointMilli = 10_000_000
)

// String posts the points to three decimals: "7.000".
func (p Points) String() string {
	return fmt.Sprintf("%d.%03d", int64(p)/pointsPerUnit, int64(p)%pointsPerUnit)
}

// ParsePoints reads an owner-typed point total such as "7", "7.5" or "7.250". More than three
// decimals is refused, not rounded: the owner typed a figure the forum cannot show.
func ParsePoints(raw string) (Points, error) {
	s := strings.TrimSpace(raw)
	whole, frac, _ := strings.Cut(s, ".")
	if whole == "" && frac == "" || len(frac) > 3 || !digits(whole) || !digits(frac) {
		return 0, fmt.Errorf("bid: points %q must be a number with at most three decimals", raw)
	}
	n, err := strconv.ParseInt(whole+frac+strings.Repeat("0", 3-len(frac)), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("bid: points %q: %w", raw, err)
	}
	return Points(n), nil
}

func digits(s string) bool {
	return strings.IndexFunc(s, func(r rune) bool { return r < '0' || r > '9' }) < 0
}

// multiplier is the year multiplier in hundredths (§6): 1.00x, 1.20x, 1.40x, 1.60x.
func multiplier(years int) (int64, bool) {
	if years < 1 || years > MaxYears {
		return 0, false
	}
	return int64(100 + 20*(years-1)), true
}

// Score is the annual salary in millions times the year multiplier. The salary snaps to $10k
// first (§0), so the division is exact: 1YR/$5M is 5.000, 2YR/$5M 6.000, 3YR/$1.3M 1.820.
func Score(salary domain.Money, years int) (Points, error) {
	m, ok := multiplier(years)
	if !ok {
		return 0, fmt.Errorf("bid: a bid runs 1 to %d years, not %d", MaxYears, years)
	}
	return Points(domain.RoundToNearest10k(salary).Cents() * m / centsPerPointMilli), nil
}

// MoneyText writes cents as dollars, dropping ".00": "$5,000,000" or "$5,250,000.50".
func MoneyText(m domain.Money) string {
	return strings.TrimSuffix(m.String(), ".00")
}

// MinimumSalary is the §6 minimum by years of NFL experience (season minus draft year). With
// no real draft year the caller passes 0, the rookie minimum (Christopher's ruling).
func MinimumSalary(experienceYears int) domain.Money {
	dollars := int64(630_000)
	switch {
	case experienceYears <= 0:
		dollars = 330_000
	case experienceYears == 1:
		dollars = 380_000
	case experienceYears == 2:
		dollars = 430_000
	case experienceYears == 3:
		dollars = 480_000
	case experienceYears <= 6:
		dollars = 530_000
	case experienceYears <= 9:
		dollars = 580_000
	}
	return domain.Money(dollars * 100)
}

// maxPlausibleCareer rejects MFL's placeholder draft years (the "1970" sentinel), which would
// otherwise earn the highest minimum.
const maxPlausibleCareer = 30

// ExperienceFrom is the player's NFL years: the season minus his draft year. With no real draft
// year the result is not Known and the caller prices him at the rookie minimum (Christopher's
// ruling).
func ExperienceFrom(season, draftYear int, hasDraftYear bool) Experience {
	if hasDraftYear && draftYear >= season-maxPlausibleCareer && draftYear <= season {
		return Experience{Years: season - draftYear, Known: true}
	}
	return Experience{}
}

// Post is what the forum post says about one bid.
type Post struct {
	Player string // "Name (POS, TEAM)"
	Salary domain.Money
	Years  int
	Points Points
	Snipe  bool
}

// ContractText is the contract in words: "3 years at $5,000,000 a year".
func ContractText(salary domain.Money, years int) string {
	return fmt.Sprintf("%d %s at %s a year", years, plural(years, "year"), MoneyText(salary))
}

// Text is the exact ProBoards post to copy: player, contract, cap hit and points, and "Snipe"
// when it is one. Copying it is the hand-off; nothing is posted for the owner.
func (p Post) Text() string {
	lines := []string{
		"Player: " + p.Player,
		"Contract: " + ContractText(p.Salary, p.Years) + ", flat",
		fmt.Sprintf("Cap hit: %s a year", MoneyText(p.Salary)),
		"Points: " + p.Points.String(),
	}
	if p.Snipe {
		lines = append(lines, "Snipe")
	}
	return strings.Join(lines, "\n")
}
