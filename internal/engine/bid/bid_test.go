package bid

import (
	"fmt"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

const tenK domain.Money = 1_000_000

func millions(m float64) domain.Money { return domain.Money(m * 100_000_000) }

func TestScoreRulebookExamples(t *testing.T) {
	for _, tc := range []struct {
		salary domain.Money
		years  int
		want   string
	}{
		{millions(5), 1, "5.000"},
		{millions(5), 2, "6.000"},
		{millions(1.3), 3, "1.820"},
		{millions(24.5), 4, "39.200"},
		{millions(0.33), 1, "0.330"},
		{millions(5) + 4*tenK/10, 1, "5.000"}, // $5,000,400 snaps down
		{millions(5) + 5*tenK/10, 1, "5.010"}, // $5,000,500 snaps up
	} {
		got, err := Score(tc.salary, tc.years)
		if err != nil || got.String() != tc.want {
			t.Errorf("Score(%v, %d) = %v, %v; want %s", tc.salary, tc.years, got, err, tc.want)
		}
	}
	for _, years := range []int{0, 5, -1} {
		if _, err := Score(millions(5), years); err == nil {
			t.Errorf("Score with %d years succeeded", years)
		}
	}
}

func TestParsePoints(t *testing.T) {
	if got, err := ParsePoints(" 7.250 "); err != nil || got.String() != "7.250" {
		t.Errorf("ParsePoints with spaces = %v, %v", got, err)
	}
	for raw, want := range map[string]string{"7": "7.000", "7.5": "7.500", ".5": "0.500", "0.001": "0.001"} {
		if got, err := ParsePoints(raw); err != nil || got.String() != want {
			t.Errorf("ParsePoints(%q) = %v, %v; want %s", raw, got, err, want)
		}
	}
	for _, raw := range []string{"", ".", "7.2501", "-1", "x", "1e3", "7,5"} {
		if _, err := ParsePoints(raw); err == nil {
			t.Errorf("ParsePoints(%q) succeeded", raw)
		}
	}
}

func TestMinimumSalaryTable(t *testing.T) {
	for exp, want := range map[int]int64{-1: 330_000, 0: 330_000, 1: 380_000, 2: 430_000, 3: 480_000,
		4: 530_000, 6: 530_000, 7: 580_000, 9: 580_000, 10: 630_000, 15: 630_000} {
		if got := MinimumSalary(exp); got != domain.Money(want*100) {
			t.Errorf("MinimumSalary(%d) = %v, want $%d", exp, got, want)
		}
	}
}

// passing is a 2-year $5M bid in the offseason that every rule accepts.
func passing() Input {
	room := millions(20)
	return Input{Salary: millions(5), Years: 2, Minimum: MinimumSalary(3),
		Experience: Experience{Years: 3, Known: true}, Room: &room, WeekKnown: true}
}

func verdictOf(t *testing.T, in Input, rule string) Verdict {
	t.Helper()
	for _, r := range Check(in) {
		if r.Rule == rule {
			return r.Verdict
		}
	}
	t.Fatalf("no result for rule %s", rule)
	return ""
}

func TestCheckPassesAndKeepsOrder(t *testing.T) {
	results := Check(passing())
	want := []string{RuleLength, RuleFlat, RuleSnap, RuleMinimum, RuleOneYear, RuleTierYears,
		RuleIncrement, RuleActive, RuleMultiClose, RuleLateSeason, RuleCapRoom}
	if len(results) != len(want) {
		t.Fatalf("%d results, want %d", len(results), len(want))
	}
	for i, r := range results {
		if r.Rule != want[i] || r.Label == "" || r.Reason == "" {
			t.Errorf("result %d = %+v, want rule %s with words", i, r, want[i])
		}
	}
	if Blocked(results) || Note(results) != "Bid rules checked; the plan is ready to hand off" {
		t.Errorf("a passing plan is blocked: %q", Note(results))
	}
}

func TestRuleBoundaries(t *testing.T) {
	beat := func(p string) *Points { v, _ := ParsePoints(p); return &v }
	for _, tc := range []struct {
		name string
		rule string
		edit func(*Input)
		want Verdict
	}{
		{"4 years", RuleLength, func(in *Input) { in.Years = 4 }, Pass},
		{"5 years", RuleLength, func(in *Input) { in.Years = 5 }, Fail},
		{"0 years", RuleLength, func(in *Input) { in.Years = 0 }, Fail},
		{"zero salary", RuleFlat, func(in *Input) { in.Salary = 0 }, Fail},
		{"off the 10k grid", RuleSnap, func(in *Input) { in.Salary++ }, Fail},
		{"at the minimum", RuleMinimum, func(in *Input) { in.Salary = in.Minimum }, Pass},
		{"a step under the minimum", RuleMinimum, func(in *Input) { in.Salary = in.Minimum - tenK }, Fail},
		{"1 year at $12M", RuleOneYear, func(in *Input) { in.Years, in.Salary = 1, millions(12) }, Pass},
		{"1 year over $12M", RuleOneYear, func(in *Input) { in.Years, in.Salary = 1, millions(12)+tenK }, Fail},
		{"2 years over $12M", RuleOneYear, func(in *Input) { in.Salary = millions(13) }, Pass},
		{"1 year over $12M in the playoffs", RuleOneYear, func(in *Input) {
			in.Years, in.Salary, in.Week = 1, millions(13), 14
		}, Pass},
		{"1 year over $12M in week 13", RuleOneYear, func(in *Input) {
			in.Years, in.Salary, in.Week = 1, millions(13), 13
		}, Fail},
		{"1 year at $12M needs 1", RuleTierYears, func(in *Input) { in.Years, in.Salary = 1, millions(12) }, Pass},
		{"1 year over $12M needs 2", RuleTierYears, func(in *Input) { in.Years, in.Salary = 1, millions(12)+tenK }, Fail},
		{"2 years at $24M", RuleTierYears, func(in *Input) { in.Years, in.Salary = 2, millions(24) }, Pass},
		{"2 years over $24M needs 3", RuleTierYears, func(in *Input) { in.Years, in.Salary = 2, millions(24)+tenK }, Fail},
		{"3 years over $24M", RuleTierYears, func(in *Input) { in.Years, in.Salary = 3, millions(30) }, Pass},
		{"tiers lifted in the playoffs", RuleTierYears, func(in *Input) {
			in.Years, in.Salary, in.Week = 1, millions(30), 14
		}, Pass},
		{"no bid to beat", RuleIncrement, func(*Input) {}, Pass},
		// $5M for 2 years is 6.000 points.
		{"exactly 0.1 over", RuleIncrement, func(in *Input) { in.Beat = beat("5.900") }, Pass},
		{"just under 0.1 over", RuleIncrement, func(in *Input) { in.Beat = beat("5.901") }, Fail},
		{"snipe exactly 1.0 over", RuleIncrement, func(in *Input) { in.Beat, in.Snipe = beat("5.000"), true }, Pass},
		{"snipe just under 1.0 over", RuleIncrement, func(in *Input) { in.Beat, in.Snipe = beat("5.001"), true }, Fail},
		{"not a snipe, 0.1 is enough", RuleIncrement, func(in *Input) { in.Beat = beat("5.001") }, Pass},
		{"29 other active bids", RuleActive, func(in *Input) { in.Active = 29 }, Pass},
		{"30 other active bids", RuleActive, func(in *Input) { in.Active = 30 }, Fail},
		{"multi-year in week 13", RuleMultiClose, func(in *Input) { in.Week = 13 }, Pass},
		{"multi-year in week 14", RuleMultiClose, func(in *Input) { in.Week = 14 }, Fail},
		{"1-year in week 14", RuleMultiClose, func(in *Input) { in.Years, in.Week = 1, 14 }, Pass},
		{"week 12, any team", RuleLateSeason, func(in *Input) { in.Week = 12 }, Pass},
		{"offseason", RuleLateSeason, func(in *Input) { in.Week = 0 }, Pass},
		{"week 13, 2 years", RuleLateSeason, func(in *Input) { in.Week, in.Standing = 13, PlayoffIn }, Fail},
		{"week 13, playoff team, 1 year", RuleLateSeason, func(in *Input) {
			in.Week, in.Years, in.Standing = 13, 1, PlayoffIn
		}, Pass},
		{"week 13, out of the playoffs", RuleLateSeason, func(in *Input) {
			in.Week, in.Years, in.Standing = 13, 1, PlayoffOut
		}, Fail},
		{"week 13, playoffs unknown", RuleLateSeason, func(in *Input) { in.Week, in.Years = 13, 1 }, Unknown},
		{"no schedule held", RuleLateSeason, func(in *Input) { in.WeekKnown = false }, Unknown},
		{"cap room is only shown", RuleCapRoom, func(in *Input) { r := millions(1); in.Room = &r }, Info},
		{"no cap room held", RuleCapRoom, func(in *Input) { in.Room = nil }, Info},
	} {
		in := passing()
		tc.edit(&in)
		if got := verdictOf(t, in, tc.rule); got != tc.want {
			t.Errorf("%s: %s = %s, want %s", tc.name, tc.rule, got, tc.want)
		}
	}
}

func TestUnknownPlayoffStatusBlocksWithItsReason(t *testing.T) {
	in := passing()
	in.Week, in.Years = 13, 1
	results := Check(in)
	if !Blocked(results) || !strings.Contains(Note(results), "cannot tell whether yours is one") {
		t.Errorf("unknown playoff status did not block with its reason: %q", Note(results))
	}
}

func TestCapRoomNeverBlocks(t *testing.T) {
	in := passing()
	room := millions(1)
	in.Room = &room
	results := Check(in)
	if Blocked(results) {
		t.Fatalf("a bid over the cap room blocked the plan: %q", Note(results))
	}
	for _, r := range results {
		if r.Rule == RuleCapRoom && !strings.Contains(r.Reason, "$4,000,000 over your $1,000,000 of room") {
			t.Errorf("cap room reason = %q", r.Reason)
		}
	}
}

func TestPostText(t *testing.T) {
	p := Post{Player: "Smith, John (WR, KCC)", Salary: millions(5), Years: 3, Points: 7000}
	want := "Player: Smith, John (WR, KCC)\nContract: 3 years at $5,000,000 a year, flat\n" +
		"Cap hit: $5,000,000 a year\nPoints: 7.000"
	if got := p.Text(); got != want {
		t.Errorf("post text:\n%s\nwant:\n%s", got, want)
	}
	p.Snipe, p.Years = true, 1
	if got := p.Text(); !strings.HasSuffix(got, "\nSnipe") || !strings.Contains(got, "1 year at") {
		t.Errorf("snipe post text:\n%s", got)
	}
}

func TestExperienceFrom(t *testing.T) {
	for _, tc := range []struct {
		name   string
		season int
		draft  int
		has    bool
		want   Experience
	}{
		{"drafted last year", 2026, 2025, true, Experience{Years: 1, Known: true}},
		{"drafted this year", 2026, 2026, true, Experience{Years: 0, Known: true}},
		{"thirty years back", 2026, 1996, true, Experience{Years: 30, Known: true}},
		{"the 1970 placeholder", 2026, 1970, true, Experience{}},
		{"a future draft year", 2026, 2027, true, Experience{}},
		{"undrafted", 2026, 0, false, Experience{}},
	} {
		if got := ExperienceFrom(tc.season, tc.draft, tc.has); got != tc.want {
			t.Errorf("%s: %+v, want %+v", tc.name, got, tc.want)
		}
	}
}

func TestRuleWordsForTheOwner(t *testing.T) {
	words := func(rule string, edit func(*Input)) string {
		in := passing()
		edit(&in)
		for _, r := range Check(in) {
			if r.Rule == rule {
				return r.Reason
			}
		}
		return ""
	}
	for _, tc := range []struct {
		name, rule, want string
		edit             func(*Input)
	}{
		{"one year", RuleFlat, "$5,000,000 a year for its 1 year", func(in *Input) { in.Years = 1 }},
		{"three years", RuleFlat, "$5,000,000 a year, the same in each of the 3 years", func(in *Input) { in.Years = 3 }},
		{"lowest tier", RuleTierYears, "Up to $12,000,000 needs at least 1 year", func(in *Input) { in.Years = 2 }},
		{"at $12M", RuleTierYears, "Up to $12,000,000 needs at least 1 year", func(in *Input) { in.Salary = millions(12) }},
		{"middle tier", RuleTierYears, "Over $12,000,000 up to $24,000,000 needs at least 2 years", func(in *Input) {
			in.Years, in.Salary = 2, millions(13)
		}},
		{"top tier", RuleTierYears, "Over $24,000,000 needs at least 3 years", func(in *Input) {
			in.Years, in.Salary = 3, millions(30)
		}},
		{"playoffs", RuleTierYears, "In the playoffs any salary needs at least 1 year", func(in *Input) {
			in.Years, in.Salary, in.Week = 1, millions(30), 14
		}},
	} {
		if got := words(tc.rule, tc.edit); got != tc.want {
			t.Errorf("%s: %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestSummaryCountsOnlyThePassingRules(t *testing.T) {
	all := Check(passing())
	pass := 0
	for _, r := range all {
		if r.Verdict == Pass {
			pass++
		}
	}
	if got, want := Summary(all), fmt.Sprintf("%d rules pass", pass); got != want || pass < 8 {
		t.Errorf("summary = %q, want %q", got, want)
	}
	in := passing()
	in.Years = 5
	failing := Check(in)
	if got := Summary(failing); got == Summary(all) || !strings.HasSuffix(got, "rules pass") {
		t.Errorf("a failing plan summarised as %q", got)
	}
	if Summary(nil) != "No rule passes yet" || Summary([]Result{{Verdict: Pass}}) != "1 rule passes" ||
		Summary([]Result{{Verdict: Info}, {Verdict: Fail}}) != "No rule passes yet" {
		t.Error("edge summaries")
	}
}
