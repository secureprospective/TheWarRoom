// Package bid checks the frozen MFL local bid-plan policy without I/O.
package bid

import (
	"fmt"
	"strings"

	bidmath "github.com/secureprospective/TheWarRoom/internal/engine/bid"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// The §6 limits. Money is cents.
const (
	MaxActive = 30

	oneYearCap domain.Money = 1_200_000_000 // $12M
	midTierCap domain.Money = 2_400_000_000 // $24M

	minStep   bidmath.Points = 100  // each new bid beats the last by at least 0.1
	snipeStep bidmath.Points = 1000 // after the 20-hour mark, by a full point

	lateSeasonWeek   = 12 // after it only playoff teams bid, 1-year only
	multiYearCloseWk = 13 // multi-year bids close at its end; later weeks are the playoffs
)

// Standing is whether the owner's team is in the playoffs. The zero value is unknown: the app
// reads no playoff field, so it never guesses.
type Standing int

const (
	PlayoffUnknown Standing = iota
	PlayoffIn
	PlayoffOut
)

// Verdict is one rule's outcome. Fail and Unknown block the plan; Info only informs.
type Verdict string

const (
	Pass    Verdict = "pass"
	Fail    Verdict = "fail"
	Unknown Verdict = "unknown"
	Info    Verdict = "info"
)

// The rule ids, in the order Check returns them.
const (
	RuleLength     = "max-years"
	RuleFlat       = "flat-salary"
	RuleSnap       = "ten-k-grid"
	RuleMinimum    = "minimum-salary"
	RuleOneYear    = "one-year-cap"
	RuleTierYears  = "tier-years"
	RuleIncrement  = "increment"
	RuleActive     = "active-bids"
	RuleMultiClose = "multi-year-close"
	RuleLateSeason = "late-season"
	RuleCapRoom    = "cap-room"
)

// Result is one named rule on a plan: the reason is Go's words for the owner, and the mark is
// the short word beside it.
type Result struct {
	Rule    string  `json:"rule"`
	Label   string  `json:"label"`
	Verdict Verdict `json:"verdict"`
	Reason  string  `json:"reason"`
	Mark    string  `json:"mark"`
}

// finding is a rule's outcome before Check adds the mark.
type finding struct {
	rule, label string
	verdict     Verdict
	reason      string
}

func (f finding) result() Result {
	marks := map[Verdict]string{Pass: "OK", Fail: "Fails", Unknown: "Can't tell", Info: "Note"}
	return Result{f.rule, f.label, f.verdict, f.reason, marks[f.verdict]}
}

// Input is everything the rules read about one bid plan.
type Input struct {
	Salary      domain.Money // annual, flat across the years
	Years       int
	Minimum     domain.Money // the §6 minimum for the player's experience
	Experience  bidmath.Experience
	Snipe       bool            // the owner says the bid lands after the 20-hour mark
	Beat        *bidmath.Points // the bid to beat, as the owner typed it
	Active      int             // the owner's other active bid plans
	Conditional *ConditionalDrop
	Room        *domain.Money
	Week        int  // the NFL week in play; 0 outside the season
	WeekKnown   bool // false when the app holds no schedule, so the week limits cannot be checked
	Standing    Standing
}

// ConditionalDrop is a held roster cut priced by the shared transaction rule at the app boundary.
type ConditionalDrop struct {
	Name    string
	Salary  domain.Money
	DeadCap domain.Money
}

// Check runs every rule and returns one result each, always in the same order.
func Check(in Input) []Result {
	rules := []func(Input) finding{
		length, flat, snap, minimum, oneYear, tierYears, increment, active, multiClose, lateSeason, capRoom,
	}
	out := make([]Result, len(rules))
	for i, rule := range rules {
		out[i] = rule(in).result()
	}
	return out
}

// Blocked reports whether any rule fails or cannot be told.
func Blocked(results []Result) bool {
	for _, r := range results {
		if r.Verdict == Fail || r.Verdict == Unknown {
			return true
		}
	}
	return false
}

// Note is the plan's one-line status: the reasons that block it, or that all rules pass.
func Note(results []Result) string {
	var reasons []string
	for _, r := range results {
		if r.Verdict == Fail || r.Verdict == Unknown {
			reasons = append(reasons, strings.TrimRight(r.Reason, ".; "))
		}
	}
	if len(reasons) == 0 {
		return "Bid rules checked; the plan is ready to hand off"
	}
	return strings.Join(reasons, "; ")
}

// Summary is the one line that stands in for the rules that pass: "9 rules pass". Rules that
// fail, cannot be told or only inform are shown in full beside it.
func Summary(results []Result) string {
	n := 0
	for _, r := range results {
		if r.Verdict == Pass {
			n++
		}
	}
	switch n {
	case 0:
		return "No rule passes yet"
	case 1:
		return "1 rule passes"
	}
	return fmt.Sprintf("%d rules pass", n)
}

func verdict(ok bool) Verdict {
	if ok {
		return Pass
	}
	return Fail
}

func length(in Input) finding {
	ok := in.Years >= 1 && in.Years <= bidmath.MaxYears
	reason := fmt.Sprintf("%d-year bid; the longest allowed is %d years", in.Years, bidmath.MaxYears)
	if !ok {
		reason = fmt.Sprintf("A bid runs 1 to %d years, not %d", bidmath.MaxYears, in.Years)
	}
	return finding{RuleLength, "Length", verdict(ok), reason}
}

func flat(in Input) finding {
	ok := in.Salary > 0
	reason := fmt.Sprintf("%s a year, the same in each of the %d years", bidmath.MoneyText(in.Salary), in.Years)
	if in.Years == 1 {
		reason = fmt.Sprintf("%s a year for its 1 year", bidmath.MoneyText(in.Salary))
	}
	if !ok {
		reason = "Enter one annual salary above $0; it is the same every year"
	}
	return finding{RuleFlat, "Flat salary", verdict(ok), reason}
}

func snap(in Input) finding {
	ok := domain.RoundToNearest10k(in.Salary) == in.Salary
	reason := "Salary is on the $10,000 grid"
	if !ok {
		reason = fmt.Sprintf("%s is not on the $10,000 grid", bidmath.MoneyText(in.Salary))
	}
	return finding{RuleSnap, "$10k grid", verdict(ok), reason}
}

func minimum(in Input) finding {
	ok := in.Salary >= in.Minimum
	basis := fmt.Sprintf("%d years of experience", in.Experience.Years)
	if !in.Experience.Known {
		basis = "no draft year on file, so the rookie minimum"
	}
	reason := fmt.Sprintf("At or above the %s minimum (%s)", bidmath.MoneyText(in.Minimum), basis)
	if !ok {
		reason = fmt.Sprintf("Below the %s minimum salary (%s)", bidmath.MoneyText(in.Minimum), basis)
	}
	return finding{RuleMinimum, "Minimum salary", verdict(ok), reason}
}

func oneYear(in Input) finding {
	switch {
	case in.Years != 1:
		return finding{RuleOneYear, "1-year limit", Pass, "Applies to 1-year bids only"}
	case in.Salary > oneYearCap:
		return finding{RuleOneYear, "1-year limit", Fail,
			fmt.Sprintf("A 1-year bid is capped at %s", bidmath.MoneyText(oneYearCap))}
	}
	return finding{RuleOneYear, "1-year limit", Pass, fmt.Sprintf("At or under %s", bidmath.MoneyText(oneYearCap))}
}

func tierYears(in Input) finding {
	need := 1
	switch {
	case in.Salary > midTierCap:
		need = 3
	case in.Salary > oneYearCap:
		need = 2
	}
	if in.Years < need {
		return finding{RuleTierYears, "Years for the salary", Fail,
			fmt.Sprintf("A %s bid needs at least %d years", bidmath.MoneyText(in.Salary), need)}
	}
	return finding{RuleTierYears, "Years for the salary", Pass,
		fmt.Sprintf("%s needs at least %d %s", tierText(need), need, plural(need, "year"))}
}

// tierText names the salary band the length rule puts this bid in.
func tierText(need int) string {
	switch need {
	case 3:
		return "Over " + bidmath.MoneyText(midTierCap)
	case 2:
		return "Over " + bidmath.MoneyText(oneYearCap) + " up to " + bidmath.MoneyText(midTierCap)
	}
	return "Up to " + bidmath.MoneyText(oneYearCap)
}

func plural(n int, noun string) string {
	if n == 1 {
		return noun
	}
	return noun + "s"
}

func increment(in Input) finding {
	label := "Beats the bid to beat"
	if in.Beat == nil {
		return finding{RuleIncrement, label, Pass, "No bid to beat entered, so this is not checked"}
	}
	step, name := minStep, "0.1"
	if in.Snipe {
		step, name = snipeStep, "a full point (Snipe)"
	}
	points, err := bidmath.Score(in.Salary, in.Years)
	need := *in.Beat + step
	if err != nil || points < need {
		return finding{RuleIncrement, label, Fail,
			fmt.Sprintf("Must beat %s by %s: at least %s", *in.Beat, name, need)}
	}
	return finding{RuleIncrement, label, Pass, fmt.Sprintf("%s beats %s by %s or more", points, *in.Beat, name)}
}

func active(in Input) finding {
	ok := in.Active < MaxActive
	reason := fmt.Sprintf("%d of %d active bid plans, counting this one", in.Active+1, MaxActive)
	if !ok {
		reason = fmt.Sprintf("You already have %d active bid plans; %d is the limit", in.Active, MaxActive)
	}
	return finding{RuleActive, "Active bids", verdict(ok), reason}
}

func multiClose(in Input) finding {
	if !in.WeekKnown {
		return finding{RuleMultiClose, "Multi-year window", Info, "No NFL week is held, so the Week 13 close is not checked"}
	}
	if in.Week > multiYearCloseWk && in.Years > 1 {
		return finding{RuleMultiClose, "Multi-year window", Fail, "Multi-year bids closed at the end of Week 13"}
	}
	return finding{RuleMultiClose, "Multi-year window", Pass, "Multi-year bids close at the end of Week 13"}
}

func lateSeason(in Input) finding {
	label := "After Week 12"
	switch {
	case !in.WeekKnown:
		return finding{RuleLateSeason, label, Unknown, "The app holds no NFL week, so it cannot tell whether the Week 12 limits apply"}
	case in.Week <= lateSeasonWeek:
		return finding{RuleLateSeason, label, Pass, "Every team may bid through Week 12"}
	case in.Years != 1:
		return finding{RuleLateSeason, label, Fail, "After Week 12 only 1-year bids are allowed"}
	case in.Standing == PlayoffOut:
		return finding{RuleLateSeason, label, Fail, "After Week 12 only playoff teams may bid, and yours is out"}
	case in.Standing == PlayoffUnknown:
		return finding{RuleLateSeason, label, Unknown,
			"After Week 12 only playoff teams may bid, and this app cannot tell whether yours is one"}
	}
	return finding{RuleLateSeason, label, Pass, "A playoff team may place 1-year bids"}
}

func capRoom(in Input) finding {
	label := "Cap room"
	if in.Room == nil {
		return finding{RuleCapRoom, label, Info, "Cap room is not available; the commissioner's cap rules decide"}
	}
	after := *in.Room - in.Salary
	reason := fmt.Sprintf("%s a year against %s of room leaves %s", bidmath.MoneyText(in.Salary),
		bidmath.MoneyText(*in.Room), bidmath.MoneyText(after))
	if after >= 0 {
		return finding{RuleCapRoom, label, Pass, reason}
	}
	if in.Conditional == nil {
		return finding{RuleCapRoom, label, Fail, fmt.Sprintf(
			"%s a year is %s over your %s of room. Name a conditional drop to cover it.",
			bidmath.MoneyText(in.Salary), bidmath.MoneyText(-after), bidmath.MoneyText(*in.Room))}
	}
	drop := in.Conditional
	freed := drop.Salary - drop.DeadCap
	if after+freed < 0 {
		return finding{RuleCapRoom, label, Fail, fmt.Sprintf(
			"Dropping %s frees %s after %s dead cap; still %s short. Choose a drop that covers it.",
			drop.Name, bidmath.MoneyText(freed), bidmath.MoneyText(drop.DeadCap), bidmath.MoneyText(-after-freed))}
	}
	return finding{RuleCapRoom, label, Pass, fmt.Sprintf(
		"Covered if you win: the commissioner drops %s (%s freed after %s dead cap)",
		drop.Name, bidmath.MoneyText(freed), bidmath.MoneyText(drop.DeadCap))}
}
