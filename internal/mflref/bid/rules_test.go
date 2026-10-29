package bid

import (
	"fmt"
	"strings"
	"testing"

	bidmath "github.com/secureprospective/TheWarRoom/internal/engine/bid"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

const tenK domain.Money = 1_000_000

func millions(m float64) domain.Money { return domain.Money(m * 100_000_000) }

func passing() Input {
	room := millions(20)
	return Input{Salary: millions(5), Years: 2, Minimum: bidmath.MinimumSalary(3),
		Experience: bidmath.Experience{Years: 3, Known: true}, Room: &room, WeekKnown: true}
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
	beat := func(p string) *bidmath.Points { v, _ := bidmath.ParsePoints(p); return &v }
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
		}, Fail},
		{"1 year over $12M in week 13", RuleOneYear, func(in *Input) {
			in.Years, in.Salary, in.Week = 1, millions(13), 13
		}, Fail},
		{"1 year at $12M needs 1", RuleTierYears, func(in *Input) { in.Years, in.Salary = 1, millions(12) }, Pass},
		{"1 year over $12M needs 2", RuleTierYears, func(in *Input) { in.Years, in.Salary = 1, millions(12)+tenK }, Fail},
		{"2 years at $24M", RuleTierYears, func(in *Input) { in.Years, in.Salary = 2, millions(24) }, Pass},
		{"2 years over $24M needs 3", RuleTierYears, func(in *Input) { in.Years, in.Salary = 2, millions(24)+tenK }, Fail},
		{"3 years over $24M", RuleTierYears, func(in *Input) { in.Years, in.Salary = 3, millions(30) }, Pass},
		{"tiers remain in the playoffs", RuleTierYears, func(in *Input) {
			in.Years, in.Salary, in.Week = 1, millions(30), 14
		}, Fail},
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
		{"over cap blocks", RuleCapRoom, func(in *Input) { r := millions(1); in.Room = &r }, Fail},
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

func TestCapRoomBlocks(t *testing.T) {
	in := passing()
	room := millions(1)
	in.Room = &room
	results := Check(in)
	if !Blocked(results) {
		t.Fatalf("a bid over the cap room did not block the plan: %q", Note(results))
	}
	for _, r := range results {
		if r.Rule == RuleCapRoom && !strings.Contains(r.Reason, "$4,000,000 over your $1,000,000 of room") {
			t.Errorf("cap room reason = %q", r.Reason)
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
		{"playoffs", RuleTierYears, "A $30,000,000 bid needs at least 3 years", func(in *Input) {
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

func TestPlayoffsKeepSalaryTiersAndRequirePlayoffTeam(t *testing.T) {
	for _, tc := range []struct {
		name     string
		standing Standing
		salary   domain.Money
		years    int
		rule     string
		want     Verdict
	}{
		{"playoff team may bid", PlayoffIn, millions(12), 1, RuleLateSeason, Pass},
		{"non-playoff team may not bid", PlayoffOut, millions(12), 1, RuleLateSeason, Fail},
		{"playoff one-year ceiling", PlayoffIn, millions(12), 1, RuleOneYear, Pass},
		{"playoff one-year above ceiling", PlayoffIn, millions(12) + tenK, 1, RuleOneYear, Fail},
		{"playoff middle-tier boundary", PlayoffIn, millions(24), 2, RuleTierYears, Pass},
		{"playoff above middle tier", PlayoffIn, millions(24) + tenK, 2, RuleTierYears, Fail},
		{"playoff three-year tier", PlayoffIn, millions(24) + tenK, 3, RuleTierYears, Pass},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := passing()
			in.Week = 14
			in.Standing = tc.standing
			in.Salary = tc.salary
			in.Years = tc.years
			if got := verdictOf(t, in, tc.rule); got != tc.want {
				t.Fatalf("%s: %s want %s", tc.rule, got, tc.want)
			}
			if tc.standing == PlayoffOut && !strings.Contains(Note(Check(in)), "only playoff teams may bid") {
				t.Fatal("non-playoff blocker is not Go-written", Note(Check(in)))
			}
		})
	}
}

func TestNoteDoesNotJoinFullStopsToSemicolons(t *testing.T) {
	got := Note([]Result{{Verdict: Fail, Reason: "First."}, {Verdict: Unknown, Reason: "Second."}})
	if strings.Contains(got, ".;") {
		t.Fatal(got)
	}
}

func TestConditionalCapRoomCases(t *testing.T) {
	room := domain.Money(14_950_000)
	salary := domain.Money(60_000_000)
	for _, tc := range []struct {
		name   string
		room   *domain.Money
		salary domain.Money
		drop   *ConditionalDrop
		want   Verdict
		reason string
	}{
		{"fits", &room, room, nil, Pass, "leaves $0"},
		{"over without drop", &room, salary, nil, Fail,
			"$600,000 a year is $450,500 over your $149,500 of room. Name a conditional drop to cover it."},
		{"covered", &room, salary, &ConditionalDrop{"Brooks, Jordyn", 90_000_000, 30_000_000}, Pass,
			"Covered if you win: the commissioner drops Brooks, Jordyn ($600,000 freed after $300,000 dead cap)"},
		{"exactly covered", &room, salary, &ConditionalDrop{"Brooks, Jordyn", 75_050_000, 30_000_000}, Pass,
			"$450,500 freed"},
		{"still short", &room, salary, &ConditionalDrop{"Brooks, Jordyn", 50_000_000, 30_000_000}, Fail,
			"still $250,500 short"},
		{"negative savings", &room, salary, &ConditionalDrop{"Brooks, Jordyn", 10_000_000, 30_000_000}, Fail,
			"still $650,500 short"},
		{"unknown", nil, salary, nil, Info, "Cap room is not available"},
		{"unknown with drop", nil, salary, &ConditionalDrop{"Brooks, Jordyn", 90_000_000, 30_000_000}, Info,
			"Cap room is not available"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := capRoom(Input{Room: tc.room, Salary: tc.salary, Conditional: tc.drop})
			if got.verdict != tc.want || !strings.Contains(got.reason, tc.reason) {
				t.Fatalf("cap room: %+v, want %s containing %q", got, tc.want, tc.reason)
			}
		})
	}
}
