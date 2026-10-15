package commissioner

import (
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func TestRolloverAdvancesCrossYearSeasonDates(t *testing.T) {
	state, rules := rolloverInputs()
	february := time.Date(2027, 2, 15, 5, 0, 0, 0, time.UTC)
	january := time.Date(2027, 1, 10, 5, 0, 0, 0, time.UTC)
	rules.Calendar.Rollover = leaguerules.Known(february)
	window := rules.Calendar.Windows.Value["rookie_draft"]
	window.DatedStart = leaguerules.Known(january)
	window.DatedEnd = leaguerules.Known(january.AddDate(0, 0, 7))
	rules.Calendar.Windows.Value["playoffs"] = window
	before := rolloverJSON(t, rules)
	plan := requireRollover(t, state, rules)
	got := plan.Rules.Calendar.Windows.Value["playoffs"]
	if plan.Blocked || !plan.Rules.Calendar.Rollover.Value.Equal(february.AddDate(1, 0, 0)) ||
		!got.DatedStart.Value.Equal(january.AddDate(1, 0, 0)) ||
		!got.DatedEnd.Value.Equal(january.AddDate(1, 0, 7)) {
		t.Fatalf("dates of the old season were left behind: %+v", plan)
	}
	if string(before) != string(rolloverJSON(t, rules)) {
		t.Fatal("advancing the calendar mutated the old season")
	}
}

func TestRolloverWaitingDatesAreRecognizedByTheirSource(t *testing.T) {
	state, rules := rolloverInputs()
	for _, year := range []int{2026, 2027, 2028} {
		deadline := time.Date(year, 11, 7, 18, 0, 0, 0, time.UTC)
		change := waitingRolloverRule(rules, 0, 2)
		change.Field, change.New = "calendar.tradeDeadline", leaguerules.Known(deadline)
		state.Waiting = []leaguerules.Change{change}
		plan := requireRollover(t, state, rules)
		if plan.Blocked || plan.Calendar.TradeDeadline == nil || !plan.Calendar.TradeDeadline.Equal(deadline) {
			t.Fatalf("waiting deadline changed based on its calendar year %d", year)
		}
	}
	window := rules.Calendar.Windows.Value["rookie_draft"]
	opening := time.Date(2028, 1, 14, 5, 0, 0, 0, time.UTC)
	window.DatedStart = leaguerules.Known(opening)
	change := waitingRolloverRule(rules, 0, 2)
	change.Field = "calendar.windows"
	change.New = leaguerules.Known(map[string]leaguerules.Window{"new_playoffs": window})
	state.Waiting = []leaguerules.Change{change}
	plan := requireRollover(t, state, rules)
	if plan.Blocked || !plan.Rules.Calendar.Windows.Value["new_playoffs"].DatedStart.Value.Equal(opening) {
		t.Fatal("waiting window was advanced instead of used as recorded")
	}
}

func TestRolloverUsesHeldPlayerNamesAndWindowLabels(t *testing.T) {
	state, rules := rolloverInputs()
	state.PlayerNames = map[string]string{"0001": "Held rookie", "0002": "Held veteran"}
	state.WindowLabels = map[string]string{"rookie_draft": "The summer rookie draft"}
	plan := requireRollover(t, state, rules)
	lines := strings.Join(plan.Lines, "\n")
	for _, want := range []string{"Held rookie returns from taxi", "Held veteran returns from injured reserve",
		"Held rookie is pending", "A player with no name recorded (0003)", "The summer rookie draft opens"} {
		if !strings.Contains(lines, want) {
			t.Fatalf("missing plain label %q", want)
		}
	}
	for _, internal := range []string{"Player 0001", "Window rookie_draft", "Window rfa_tenders"} {
		if strings.Contains(lines, internal) {
			t.Fatalf("internal label reached preview: %q", internal)
		}
	}
	if !strings.Contains(lines, "The rfa tenders window opens") {
		t.Fatal("unlabelled window did not get readable words")
	}
}

func TestRolloverExplainsMissingRookieRoundForEveryExpiry(t *testing.T) {
	state, rules := rolloverInputs()
	state.Assets.Players[1].ExpirationYear = 2026
	state.Assets.Players[0].ContractInfo = "Round 2 rookie RFA"
	state.Assets.Players[1].ContractInfo = "Veteran UFA"
	state.PlayerNames = map[string]string{"0001": "Named rookie", "0002": "Named veteran"}
	plan := requireRollover(t, state, rules)
	if len(plan.Pending) != 2 {
		t.Fatal("not every expiring player has a route explanation")
	}
	for _, pending := range plan.Pending {
		if pending.Route != "RFA or UFA cannot be determined: the rookie contract's draft round is not recorded." {
			t.Fatalf("display-only contract notes were guessed into a route: %+v", pending)
		}
		want := state.PlayerNames[pending.Player] + " is pending a contract decision, not released. " + pending.Route
		if !strings.Contains(strings.Join(plan.Lines, "\n"), want) {
			t.Fatalf("missing per-player route explanation: %s", pending.Player)
		}
	}
}

func TestRolloverDigestCoversWaitingRulesAndCalendar(t *testing.T) {
	state, rules := rolloverInputs()
	assetsBefore := rolloverJSON(t, state.Assets)
	original := requireRollover(t, state, rules)
	state.Waiting = []leaguerules.Change{waitingRolloverRule(rules, 9000000000, 3)}
	waiting := requireRollover(t, state, rules)
	if original.BeforeDigest == waiting.BeforeDigest {
		t.Fatal("waiting rules omitted from the before digest")
	}
	rules.Calendar.Rollover = leaguerules.Known(time.Date(2027, 2, 15, 5, 0, 0, 0, time.UTC))
	calendar := requireRollover(t, state, rules)
	if waiting.BeforeDigest == calendar.BeforeDigest {
		t.Fatal("calendar omitted from the before digest")
	}
	rules.Roster.Limits.Value.Max.Value++
	changedRules := requireRollover(t, state, rules)
	if calendar.BeforeDigest == changedRules.BeforeDigest {
		t.Fatal("non-calendar rules omitted from the before digest")
	}
	if string(assetsBefore) != string(rolloverJSON(t, state.Assets)) {
		t.Fatal("digest calculation changed the before assets")
	}
}

func TestRolloverWaitingCalendarRulingStillBlocks(t *testing.T) {
	state, rules := rolloverInputs()
	change := waitingRolloverRule(rules, 0, 2)
	change.Field, change.New = "calendar.timezone", leaguerules.Needs[string]("choose the timezone")
	state.Waiting = []leaguerules.Change{change}
	plan := requireRollover(t, state, rules)
	if !plan.Blocked || len(plan.Reasons) != 1 || !strings.Contains(plan.Reasons[0], "League timezone") {
		t.Fatalf("new season's unset timezone did not block: %+v", plan.Reasons)
	}
	state.Waiting = nil
	rules.Calendar.Timezone = leaguerules.Needs[string]("choose the timezone")
	plan = requireRollover(t, state, rules)
	if !plan.Blocked || len(plan.Reasons) != 1 {
		t.Fatalf("the same missing timezone was reported more than once: %+v", plan.Reasons)
	}
}

func TestRolloverWaitingWindowsReplaceImpossibleInheritedDates(t *testing.T) {
	state, rules := rolloverInputs()
	window := rules.Calendar.Windows.Value["rookie_draft"]
	window.DatedStart = leaguerules.Known(time.Date(2024, 2, 29, 5, 0, 0, 0, time.UTC))
	rules.Calendar.Windows.Value["rookie_draft"] = window
	window.DatedStart = leaguerules.Known(time.Date(2027, 7, 14, 4, 0, 0, 0, time.UTC))
	change := waitingRolloverRule(rules, 0, 2)
	change.Field = "calendar.windows"
	change.New = leaguerules.Known(map[string]leaguerules.Window{"rookie_draft": window})
	state.Waiting = []leaguerules.Change{change}
	plan := requireRollover(t, state, rules)
	if plan.Blocked || !plan.Rules.Calendar.Windows.Value["rookie_draft"].DatedStart.Value.Equal(window.DatedStart.Value) {
		t.Fatal("a replaced old date blocked the new calendar")
	}
}
