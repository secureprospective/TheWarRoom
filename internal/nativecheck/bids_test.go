package nativecheck

import (
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func legacyBidPolicy(t *testing.T) BidPolicy {
	t.Helper()
	policy, err := NewBidPolicy(leaguerules.LegacyNFL())
	if err != nil {
		t.Fatal(err)
	}
	return policy
}

const million = domain.Money(100_000_000)

func bidNow() time.Time { return time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC) }

func inSeason(week int) BidFacts {
	return BidFacts{Now: bidNow(), Season: BidSeason{Week: week, WeekKnown: true}, Playoff: PlayoffIn}
}

func TestBidPointsAreSalaryTimesTheYearMultiplier(t *testing.T) {
	p := legacyBidPolicy(t)
	for _, c := range []struct {
		salary domain.Money
		years  int
		want   string
	}{
		{5 * million, 1, "5.000"}, {5 * million, 2, "6.000"}, {130 * million / 100, 3, "1.820"},
		{5 * million, 4, "8.000"}, {330_000_00, 1, "0.330"},
	} {
		got, err := p.Points(c.salary, c.years)
		if err != nil || p.PointsText(got) != c.want {
			t.Errorf("%v for %d years: %v %v, want %s", c.salary, c.years, p.PointsText(got), err, c.want)
		}
	}
	if _, err := p.Points(5*million, 5); err == nil {
		t.Fatal("a fifth year was priced")
	}
}

func TestBidPointsFollowTheLeagueMultipliersAndDecimals(t *testing.T) {
	rules := leaguerules.LegacyNFL()
	rules.Bids.YearMultipliers = leaguerules.Known([]leaguerules.Value[leaguerules.Decimal]{
		leaguerules.Known(leaguerules.Decimal(10000)), leaguerules.Known(leaguerules.Decimal(15000))})
	rules.Bids.PointDecimals = leaguerules.Known(1)
	p, err := NewBidPolicy(rules)
	if err != nil {
		t.Fatal(err)
	}
	got, err := p.Points(5*million, 2)
	if err != nil || p.PointsText(got) != "7.5" || p.MaxYears() != 2 {
		t.Fatalf("league multipliers ignored: %v %v years %d", p.PointsText(got), err, p.MaxYears())
	}
	if _, err := p.Points(5*million, 3); err == nil {
		t.Fatal("a third year was priced under a two year table")
	}
}

func TestBidRulesRefuseInPlainWords(t *testing.T) {
	p := legacyBidPolicy(t)
	late := inSeason(5)
	late.Standing = &StandingBid{Points: 50000, PlacedAt: bidNow().Add(-21 * time.Hour)}
	late.LateRule = true
	justBefore := inSeason(5)
	justBefore.Standing = &StandingBid{Points: 50000, PlacedAt: bidNow().Add(-20 * time.Hour)}
	justBefore.LateRule = true
	step := inSeason(5)
	step.Standing = &StandingBid{Points: 50000, PlacedAt: bidNow().Add(-time.Hour)}
	step.LateRule = true
	full := inSeason(5)
	full.Active = 30
	unknownWeek := BidFacts{Now: bidNow()}
	week13 := inSeason(13)
	week13.Playoff = PlayoffOut
	week14 := inSeason(14)
	week14.Playoff = PlayoffUnknown
	for _, c := range []struct {
		name   string
		bid    BidProposal
		facts  BidFacts
		refuse string
	}{
		{"ordinary", BidProposal{5 * million, 1}, inSeason(5), ""},
		{"not on the money grid", BidProposal{5*million + 1, 1}, inSeason(5), "multiple of $10,000"},
		{"below the minimum", BidProposal{300_000_00, 1}, inSeason(5), "minimum salary for this player is $330,000"},
		{"one year at the limit", BidProposal{12 * million, 1}, inSeason(5), ""},
		{"one year over the limit", BidProposal{12*million + 1_000_000, 1}, inSeason(5), "no more than $12,000,000"},
		{"two years over the limit", BidProposal{13 * million, 2}, inSeason(5), ""},
		{"the top of the middle tier is still two years", BidProposal{24 * million, 2}, inSeason(5), ""},
		{"over the middle tier needs three", BidProposal{25 * million, 2}, inSeason(5), "needs at least 3 years"},
		{"three years over the middle tier", BidProposal{25 * million, 3}, inSeason(5), ""},
		{"too many active bids", BidProposal{5 * million, 1}, full, "already have 30 active bids"},
		{"no week held", BidProposal{5 * million, 1}, unknownWeek, "holds no NFL week"},
		{"multi year after week 13", BidProposal{5 * million, 2}, inSeason(14), "closed at the end of week 13"},
		{"a playoff team may not go long after week 12", BidProposal{5 * million, 2}, inSeason(13),
			"a playoff team may bid for 1 year at most"},
		{"a team out of the playoffs after week 12", BidProposal{5 * million, 1}, week13, "only playoff teams may bid"},
		{"a team the app cannot place", BidProposal{5 * million, 1}, week14, "cannot tell whether yours is one"},
		{"a playoff team bids one year", BidProposal{5 * million, 1}, inSeason(13), ""},
		{"the playoff cap lifted", BidProposal{30 * million, 1}, inSeason(14), ""},
		{"the cap holds before the playoffs", BidProposal{30 * million, 1}, inSeason(13), "no more than $12,000,000"},
		{"the standing bid stood past the mark", BidProposal{5 * million, 1}, late, "late bid must beat"},
		{"a bid at exactly the mark is not late", BidProposal{51 * million / 10, 1}, justBefore, ""},
		{"a hundredth short of the step", BidProposal{5 * million, 1}, step, "at least 5.100"},
	} {
		got := p.Check(c.bid, c.facts)
		if c.refuse == "" && got.Refusal != "" || c.refuse != "" && !strings.Contains(got.Refusal, c.refuse) {
			t.Errorf("%s: refusal %q, want %q", c.name, got.Refusal, c.refuse)
		}
	}
}

func TestBidStepsAreTheLeaguesAndLateBidsNeedMore(t *testing.T) {
	p := legacyBidPolicy(t)
	standing := &StandingBid{Points: 50000, PlacedAt: bidNow().Add(-time.Hour)}
	check := func(salary domain.Money, placedAgo time.Duration, lateRule bool) BidVerdict {
		f := inSeason(5)
		f.Standing = &StandingBid{Points: standing.Points, PlacedAt: bidNow().Add(-placedAgo)}
		f.LateRule = lateRule
		return p.Check(BidProposal{salary, 1}, f)
	}
	if v := check(51*million/10, time.Hour, true); v.Refusal != "" || v.Late {
		t.Fatalf("a 0.1 raise within the day refused: %+v", v)
	}
	if v := check(51*million/10, 21*time.Hour, true); !strings.Contains(v.Refusal, "late bid must beat") || !v.Late {
		t.Fatalf("a late bid cleared at +0.1: %+v", v)
	}
	if v := check(6*million, 21*time.Hour, true); v.Refusal != "" || !v.Late {
		t.Fatalf("a late bid at a full point refused: %+v", v)
	}
	if v := check(51*million/10, 21*time.Hour, false); v.Refusal != "" || v.Late {
		t.Fatalf("blind bids have no visible clock, yet were held to the late step: %+v", v)
	}
}

func TestBidPolicyNeedsEveryRuleSet(t *testing.T) {
	for name, edit := range map[string]func(*leaguerules.RuleSet){
		"multipliers": func(r *leaguerules.RuleSet) {
			r.Bids.YearMultipliers = leaguerules.Needs[[]leaguerules.Value[leaguerules.Decimal]]("")
		},
		"active":   func(r *leaguerules.RuleSet) { r.Bids.MaximumActive = leaguerules.Needs[int]("") },
		"one year": func(r *leaguerules.RuleSet) { r.Bids.OneYearMaximum = leaguerules.Needs[leaguerules.Money]("") },
		"tiers":    func(r *leaguerules.RuleSet) { r.Bids.Tiers = leaguerules.Needs[[]leaguerules.BidTier]("") },
		"step":     func(r *leaguerules.RuleSet) { r.Bids.MinimumIncrement = leaguerules.Needs[leaguerules.Decimal]("") },
		"snipe":    func(r *leaguerules.RuleSet) { r.Bids.SnipeAfter = leaguerules.Needs[time.Duration]("") },
		"late step": func(r *leaguerules.RuleSet) {
			r.Bids.SnipeIncrement = leaguerules.Needs[leaguerules.Decimal]("")
		},
		"multi year":  func(r *leaguerules.RuleSet) { r.Bids.MultiYearEndWeek = leaguerules.Needs[int]("") },
		"playoff":     func(r *leaguerules.RuleSet) { r.Bids.PlayoffOnlyAfterWeek = leaguerules.Needs[int]("") },
		"playoff max": func(r *leaguerules.RuleSet) { r.Bids.PlayoffMaximumYears = leaguerules.Needs[int]("") },
		"cap lifted":  func(r *leaguerules.RuleSet) { r.Bids.PlayoffCapLifted = leaguerules.Needs[bool]("") },
		"decimals":    func(r *leaguerules.RuleSet) { r.Bids.PointDecimals = leaguerules.Needs[int]("") },
		"minimums": func(r *leaguerules.RuleSet) {
			r.Contracts.MinimumSalaries = leaguerules.Needs[[]leaguerules.ExperienceSalary]("")
		},
		"quantum": func(r *leaguerules.RuleSet) { r.Cap.Quantum = leaguerules.Needs[leaguerules.Money]("") },
	} {
		rules := leaguerules.LegacyNFL()
		edit(&rules)
		if _, err := NewBidPolicy(rules); err == nil || !strings.Contains(err.Error(), "needs a ruling") {
			t.Errorf("%s: unset rule judged a bid: %v", name, err)
		}
	}
}

func TestBidModeIsPerProcessAndUDFAFollowsUFA(t *testing.T) {
	rules := leaguerules.LegacyNFL().Bids
	rules.UFA, rules.RFA = leaguerules.Known(leaguerules.Blind), leaguerules.Known(leaguerules.Open)
	for process, want := range map[leaguerules.BidProcess]leaguerules.BidMode{
		leaguerules.UFAProcess: leaguerules.Blind, leaguerules.UDFAProcess: leaguerules.Blind,
		leaguerules.RFAProcess: leaguerules.Open} {
		if got, err := BidMode(rules, process); err != nil || got != want {
			t.Errorf("process %d: %v %v, want %v", process, got, err, want)
		}
	}
	rules.UFA = leaguerules.Needs[leaguerules.BidMode]("")
	if _, err := BidMode(rules, leaguerules.UFAProcess); err == nil {
		t.Fatal("an unset mode was read as a mode")
	}
}

func TestChooseBidWinnerTakesTheMostPointsAndNeverBreaksATie(t *testing.T) {
	award := ChooseBidWinner([]AwardBid{{"a", "f1", 50000}, {"b", "f2", 60000}, {"c", "f3", 55000}})
	if !award.Won || award.Winner.ID != "b" {
		t.Fatalf("highest bid did not win: %+v", award)
	}
	award = ChooseBidWinner([]AwardBid{{"a", "f1", 60000}, {"b", "f2", 60000}, {"c", "f3", 10000}})
	if award.Won || len(award.Tied) != 2 {
		t.Fatalf("a tie was settled by arrival order: %+v", award)
	}
	if award = ChooseBidWinner(nil); award.Won || len(award.Tied) != 0 {
		t.Fatalf("no bids produced a winner: %+v", award)
	}
}

func TestMultiYearBidsCloseAtTheEndOfTheLeaguesWeek(t *testing.T) {
	rules := leaguerules.LegacyNFL()
	rules.Bids.PlayoffOnlyAfterWeek = leaguerules.Known(99)
	p, err := NewBidPolicy(rules)
	if err != nil {
		t.Fatal(err)
	}
	if v := p.Check(BidProposal{5 * million, 2}, inSeason(13)); v.Refusal != "" {
		t.Fatalf("a multi year bid in the last open week refused: %q", v.Refusal)
	}
	if v := p.Check(BidProposal{5 * million, 2}, inSeason(14)); !strings.Contains(v.Refusal, "closed at the end of week 13") {
		t.Fatalf("a multi year bid after the close accepted: %q", v.Refusal)
	}
	rules.Bids.MultiYearEndWeek = leaguerules.Known(10)
	p, _ = NewBidPolicy(rules)
	if v := p.Check(BidProposal{5 * million, 2}, inSeason(11)); !strings.Contains(v.Refusal, "end of week 10") {
		t.Fatalf("the close ignored the league's week: %q", v.Refusal)
	}
}
