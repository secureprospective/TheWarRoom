package nativecheck

import (
	"fmt"
	"math/big"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// BidMode reads how one bidding process shows its bids. UDFA has no process of its own and follows UFA.
func BidMode(rules leaguerules.BidsRules, process leaguerules.BidProcess) (leaguerules.BidMode, error) {
	mode := rules.Mode(process)
	if mode.State != leaguerules.IsKnown || (mode.Value != leaguerules.Blind && mode.Value != leaguerules.Open) {
		return "", fmt.Errorf("whether free-agent bids are blind or open needs a ruling")
	}
	return mode.Value, nil
}

// PlayoffStanding is whether a franchise is in the playoffs. The zero value is unknown: a check that needs
// it says so and never guesses.
type PlayoffStanding int

const (
	PlayoffUnknown PlayoffStanding = iota
	PlayoffIn
	PlayoffOut
)

// BidSeason is where the NFL season stands when a bid is made. Week is 0 before the first game.
type BidSeason struct {
	Week      int
	WeekKnown bool
	Offseason bool
}

// StandingBid is the bid a new bid must beat.
type StandingBid struct {
	Points   leaguerules.Decimal
	PlacedAt time.Time
}

// BidProposal is the contract an owner bids: one annual salary, flat across the years.
type BidProposal struct {
	Salary domain.Money
	Years  int
}

// BidFacts is everything a check reads besides the league's rules and the proposal.
type BidFacts struct {
	Now             time.Time
	ExperienceYears int
	Active          int // the franchise's other active bids
	Season          BidSeason
	Playoff         PlayoffStanding
	Standing        *StandingBid
	// LateRule is true when the bid to beat is public: open bidding. A blind bid has no visible clock.
	LateRule bool
}

// BidVerdict is a bid's points and, when the rules turn it down, the one sentence that says why.
type BidVerdict struct {
	Points  leaguerules.Decimal
	Late    bool
	Refusal string
}

// BidPolicy holds the league's resolved bid rules. Its zero value cannot judge a bid.
type BidPolicy struct {
	multipliers   []leaguerules.Decimal
	maxActive     int
	oneYearMax    domain.Money
	tiers         []leaguerules.BidTier
	increment     leaguerules.Decimal
	lateAfter     time.Duration
	lateIncrement leaguerules.Decimal
	multiYearEnd  int
	playoffOnly   int
	playoffYears  int
	capLifted     bool
	decimals      int
	playoffStart  int
	minimumSalary []leaguerules.ExperienceSalary
	quantum       domain.Money
}

// NewBidPolicy consumes the resolved rules and refuses any bid value still waiting for a ruling.
func NewBidPolicy(rules leaguerules.RuleSet) (BidPolicy, error) {
	b := rules.Bids
	if what := unsetBidRule(rules); what != "" {
		return BidPolicy{}, fmt.Errorf("%s needs a ruling before bids can be judged", what)
	}
	multipliers := make([]leaguerules.Decimal, 0, len(b.YearMultipliers.Value))
	for _, m := range b.YearMultipliers.Value {
		multipliers = append(multipliers, m.Value)
	}
	return BidPolicy{multipliers: multipliers, maxActive: b.MaximumActive.Value,
		oneYearMax: domain.Money(b.OneYearMaximum.Value), tiers: slices.Clone(b.Tiers.Value),
		increment: b.MinimumIncrement.Value, lateAfter: b.SnipeAfter.Value, lateIncrement: b.SnipeIncrement.Value,
		multiYearEnd: b.MultiYearEndWeek.Value, playoffOnly: b.PlayoffOnlyAfterWeek.Value,
		playoffYears: b.PlayoffMaximumYears.Value, capLifted: b.PlayoffCapLifted.Value,
		decimals: b.PointDecimals.Value, playoffStart: rules.Playoffs.StartWeek.Value,
		minimumSalary: slices.Clone(rules.Contracts.MinimumSalaries.Value),
		quantum:       domain.Money(rules.Cap.Quantum.Value)}, nil
}

// unsetBidRule names the first bid value that is missing or cannot be used, or is empty when all are set.
func unsetBidRule(rules leaguerules.RuleSet) string {
	b := rules.Bids
	for _, c := range []struct {
		usable bool
		what   string
	}{
		{multipliersUsable(b.YearMultipliers), "the bid year multipliers"},
		{known(b.MaximumActive) && b.MaximumActive.Value >= 1, "the most active bids"},
		{known(b.OneYearMaximum) && b.OneYearMaximum.Value > 0, "the one year bid limit"},
		{known(b.Tiers), "the salary tiers that require longer bids"},
		{known(b.MinimumIncrement) && b.MinimumIncrement.Value >= 0, "the minimum bid increase"},
		{known(b.SnipeAfter) && b.SnipeAfter.Value > 0, "the late bid threshold"},
		{known(b.SnipeIncrement) && b.SnipeIncrement.Value >= 0, "the late bid minimum increase"},
		{known(b.MultiYearEndWeek), "the last week for multi year bids"},
		{known(b.PlayoffOnlyAfterWeek), "the week after which only playoff teams bid"},
		{known(b.PlayoffMaximumYears) && b.PlayoffMaximumYears.Value >= 1, "the longest bid a playoff team may make"},
		{known(b.PlayoffCapLifted), "whether the one year limit lifts in the playoffs"},
		{known(b.PointDecimals) && b.PointDecimals.Value >= 0 && b.PointDecimals.Value <= 4, "the bid point decimal places"},
		{known(rules.Playoffs.StartWeek), "the first playoff week"},
		{known(rules.Contracts.MinimumSalaries), "the minimum bidding salaries"},
		{known(rules.Cap.Quantum) && rules.Cap.Quantum.Value > 0, "the money rounding increment"},
	} {
		if !c.usable {
			return c.what
		}
	}
	return ""
}

func known[T any](v leaguerules.Value[T]) bool { return v.State == leaguerules.IsKnown }

func multipliersUsable(list leaguerules.Value[[]leaguerules.Value[leaguerules.Decimal]]) bool {
	if !known(list) || len(list.Value) == 0 {
		return false
	}
	return !slices.ContainsFunc(list.Value, func(m leaguerules.Value[leaguerules.Decimal]) bool {
		return !known(m) || m.Value <= 0
	})
}

// LateAfter is how long after the standing bid a new bid counts as late.
func (p BidPolicy) LateAfter() time.Duration { return p.lateAfter }

// LateIncrement is the step a late bid must add to the bid it beats.
func (p BidPolicy) LateIncrement() leaguerules.Decimal { return p.lateIncrement }

// Points is the annual salary in millions times the year's multiplier, rounded half up to the league's
// decimal places. Points are exact ten-thousandths, like every rate in the rules.
func (p BidPolicy) Points(salary domain.Money, years int) (leaguerules.Decimal, error) {
	if years < 1 || years > len(p.multipliers) {
		return 0, fmt.Errorf("a bid runs 1 to %d years, not %d", len(p.multipliers), years)
	}
	const centsPerMillion = 100_000_000
	var points big.Int
	points.Mul(big.NewInt(int64(salary)), big.NewInt(int64(p.multipliers[years-1])))
	roundBigQuotient(&points, big.NewInt(centsPerMillion))
	unit := int64(1)
	for range 4 - p.decimals {
		unit *= 10
	}
	roundBigQuotient(&points, big.NewInt(unit))
	points.Mul(&points, big.NewInt(unit))
	if !points.IsInt64() {
		return 0, fmt.Errorf("that bid is too large to score")
	}
	return leaguerules.Decimal(points.Int64()), nil
}

// roundBigQuotient divides value by divisor, rounding half up; salaries and points are never negative.
func roundBigQuotient(value, divisor *big.Int) {
	half := new(big.Int).Rsh(divisor, 1)
	value.Add(value, half).Quo(value, divisor)
}

// Check judges one bid against the league's rules in a fixed order and stops at the first that turns it
// down. The cap and the roster are judged by the caller, who knows the rest of the franchise.
func (p BidPolicy) Check(proposal BidProposal, f BidFacts) BidVerdict {
	var verdict BidVerdict
	points, err := p.Points(proposal.Salary, proposal.Years)
	if err != nil {
		verdict.Refusal = sentence(err.Error())
		return verdict
	}
	verdict.Points = points
	for _, rule := range []func(BidProposal, BidFacts, *BidVerdict) string{
		p.salaryRefusal, p.seasonRefusal, p.lengthRefusal, p.countRefusal, p.increaseRefusal} {
		if verdict.Refusal = rule(proposal, f, &verdict); verdict.Refusal != "" {
			return verdict
		}
	}
	return verdict
}

func (p BidPolicy) salaryRefusal(in BidProposal, f BidFacts, _ *BidVerdict) string {
	if in.Salary <= 0 {
		return "Enter one annual salary above $0; it is the same in every year."
	}
	if in.Salary%p.quantum != 0 {
		return fmt.Sprintf("A salary must be a multiple of %s.", dollars(p.quantum))
	}
	minimum, ok := p.minimumFor(f.ExperienceYears)
	if !ok {
		return "The minimum bidding salary for this player's experience needs a ruling."
	}
	if in.Salary < minimum {
		return fmt.Sprintf("The minimum salary for this player is %s a year.", dollars(minimum))
	}
	return ""
}

func (p BidPolicy) minimumFor(years int) (domain.Money, bool) {
	for _, row := range p.minimumSalary {
		if row.Minimum.State != leaguerules.IsKnown || row.Years.Min.State != leaguerules.IsKnown ||
			years < row.Years.Min.Value {
			continue
		}
		unlimited := row.MaximumUnlimited.State == leaguerules.IsKnown && row.MaximumUnlimited.Value
		if unlimited || (row.Years.Max.State == leaguerules.IsKnown && years <= row.Years.Max.Value) {
			return domain.Money(row.Minimum.Value), true
		}
	}
	return 0, false
}

// seasonRefusal applies the phase rules: when multi year bids stop, and who may bid late in the season.
func (p BidPolicy) seasonRefusal(in BidProposal, f BidFacts, _ *BidVerdict) string {
	if f.Season.Offseason {
		return ""
	}
	if !f.Season.WeekKnown {
		return "The app holds no NFL week for this season, so it cannot tell which bidding limits apply."
	}
	if f.Season.Week > p.multiYearEnd && in.Years > 1 {
		return fmt.Sprintf("Bids for more than 1 year closed at the end of week %d.", p.multiYearEnd)
	}
	if f.Season.Week <= p.playoffOnly {
		return ""
	}
	switch {
	case f.Playoff == PlayoffOut:
		return fmt.Sprintf("After week %d only playoff teams may bid, and yours is not one.", p.playoffOnly)
	case f.Playoff == PlayoffUnknown:
		return fmt.Sprintf("After week %d only playoff teams may bid, and the app cannot tell whether yours is one.",
			p.playoffOnly)
	case in.Years > p.playoffYears:
		return fmt.Sprintf("After week %d a playoff team may bid for %s at most.", p.playoffOnly,
			yearsWords(p.playoffYears))
	}
	return ""
}

// lengthRefusal holds a bid to the one year limit and to the years its salary tier requires. In the
// playoffs the league may lift both, because only short bids are left.
func (p BidPolicy) lengthRefusal(in BidProposal, f BidFacts, _ *BidVerdict) string {
	if !f.Season.Offseason && p.capLifted && f.Season.Week >= p.playoffStart {
		return ""
	}
	if in.Years == 1 && in.Salary > p.oneYearMax {
		return fmt.Sprintf("A 1 year bid can be no more than %s a year.", dollars(p.oneYearMax))
	}
	if need := p.yearsRequired(in.Salary); in.Years < need {
		return fmt.Sprintf("A bid of %s a year needs at least %s.", dollars(in.Salary), yearsWords(need))
	}
	return ""
}

func (p BidPolicy) yearsRequired(salary domain.Money) int {
	need := 1
	for _, tier := range p.tiers {
		low := domain.Money(tier.Salary.Min.Value)
		above := salary > low
		if tier.LowerInclusive.State == leaguerules.IsKnown && tier.LowerInclusive.Value {
			above = salary >= low
		}
		unlimited := tier.Salary.MaximumUnlimited.State == leaguerules.IsKnown && tier.Salary.MaximumUnlimited.Value
		if above && (unlimited || salary <= domain.Money(tier.Salary.Max.Value)) && tier.MinimumYears.Value > need {
			need = tier.MinimumYears.Value
		}
	}
	return need
}

func (p BidPolicy) countRefusal(_ BidProposal, f BidFacts, _ *BidVerdict) string {
	if f.Active >= p.maxActive {
		return fmt.Sprintf("You already have %s, which is the most allowed.", plural(f.Active, "active bid", "active bids"))
	}
	return ""
}

// increaseRefusal makes a bid beat the standing bid by the league's step, and by the larger late step when
// the standing bid has stood past the late threshold.
func (p BidPolicy) increaseRefusal(_ BidProposal, f BidFacts, v *BidVerdict) string {
	if f.Standing == nil {
		return ""
	}
	step := p.increment
	late := f.LateRule && f.Now.After(f.Standing.PlacedAt.Add(p.lateAfter))
	if late {
		step = p.lateIncrement
	}
	v.Late = late
	need := f.Standing.Points + step
	if v.Points >= need {
		return ""
	}
	if late {
		return fmt.Sprintf("The standing bid is more than %s old, so a late bid must beat its %s points by %s: "+
			"at least %s points.", durationWords(p.lateAfter), p.PointsText(f.Standing.Points), p.PointsText(step),
			p.PointsText(need))
	}
	return fmt.Sprintf("A bid must beat the %s points to beat by at least %s: at least %s points.",
		p.PointsText(f.Standing.Points), p.PointsText(step), p.PointsText(need))
}

// PointsText writes points to the league's decimal places: "7.000".
func (p BidPolicy) PointsText(d leaguerules.Decimal) string {
	return FormatPoints(d, p.decimals)
}

// FormatPoints writes ten-thousandths to the given decimal places.
func FormatPoints(d leaguerules.Decimal, decimals int) string {
	whole, frac := int64(d)/10000, fmt.Sprintf("%04d", int64(d)%10000)
	if decimals <= 0 {
		return fmt.Sprintf("%d", whole)
	}
	return fmt.Sprintf("%d.%s", whole, frac[:min(decimals, 4)])
}

// AwardBid is one bid that is still allowed to win.
type AwardBid struct {
	ID        string
	Franchise string
	Points    leaguerules.Decimal
}

// BidAward is the winner of a player's bids, or the bids tied at the top when the rules cannot choose.
type BidAward struct {
	Winner AwardBid
	Won    bool
	Tied   []AwardBid
}

// ChooseBidWinner takes the highest points. Equal top bids are a tie: the rules name no tie-break, so the
// commissioner decides and nothing is invented.
func ChooseBidWinner(eligible []AwardBid) BidAward {
	var top []AwardBid
	for _, bid := range eligible {
		switch {
		case len(top) == 0 || bid.Points > top[0].Points:
			top = []AwardBid{bid}
		case bid.Points == top[0].Points:
			top = append(top, bid)
		}
	}
	switch len(top) {
	case 0:
		return BidAward{}
	case 1:
		return BidAward{Winner: top[0], Won: true}
	}
	return BidAward{Tied: top}
}

func dollars(m domain.Money) string {
	return strings.TrimSuffix(m.String(), ".00")
}

// ContractWords writes a flat contract: "3 years at $5,000,000 a year".
func ContractWords(salary domain.Money, years int) string {
	return fmt.Sprintf("%s at %s a year", yearsWords(years), dollars(salary))
}

func yearsWords(n int) string {
	if n == 1 {
		return "1 year"
	}
	return fmt.Sprintf("%d years", n)
}

func durationWords(d time.Duration) string {
	if d%time.Hour == 0 {
		hours := int(d / time.Hour)
		if hours == 1 {
			return "1 hour"
		}
		return fmt.Sprintf("%d hours", hours)
	}
	return d.String()
}

func sentence(text string) string {
	if text == "" {
		return text
	}
	text = strings.ToUpper(text[:1]) + text[1:]
	if !strings.HasSuffix(text, ".") {
		text += "."
	}
	return text
}
