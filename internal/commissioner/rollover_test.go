package commissioner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func rolloverInputs() (RolloverState, leaguerules.RuleSet) {
	at := time.Date(2027, 3, 1, 5, 0, 0, 0, time.UTC)
	seed := RawNativeSnapshot{Season: 2026, RosterAt: at.AddDate(-1, 0, 0), DraftAt: at, FutureAt: at,
		Players: []NativePlayer{
			{ID: "0001", Franchise: "f01", Status: domain.RosterTaxi, Salary: 100, ExpirationYear: 2026},
			{ID: "0002", Franchise: "f02", Status: domain.RosterIR, Salary: 200, ExpirationYear: 2028},
			{ID: "0003", Franchise: "f01", Status: domain.RosterActive, Salary: 300},
		},
		Picks:       []NativePick{{Season: 2028, Round: 1, Original: "f01", Owner: "f02"}},
		Adjustments: []NativeCharge{{ID: "held", Franchise: "f01", Amount: 123, Season: 2026}},
	}
	for i := 1; i <= 32; i++ {
		id := fmt.Sprintf("f%02d", i)
		seed.Franchises = append(seed.Franchises, NativeFranchise{ID: id, HeldID: id, Name: "Team " + id})
	}
	rules := leaguerules.LegacyNFL()
	rules.LeagueID, rules.Season = "native.rollover", 2026
	rules.Calendar.Boundary = leaguerules.Known(leaguerules.StartOfDay)
	return RolloverState{Assets: seed, At: at}, rules
}

func requireRollover(t *testing.T, state RolloverState, rules leaguerules.RuleSet) RolloverPlan {
	t.Helper()
	plan, err := Rollover(state, rules, 2026, 2027)
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

func TestRolloverReturnsReservesAndKeepsPendingPlayers(t *testing.T) {
	state, rules := rolloverInputs()
	plan := requireRollover(t, state, rules)
	if plan.Blocked || plan.After.Season != 2027 || len(plan.After.Players) != 3 {
		t.Fatalf("unexpected after state: %+v", plan)
	}
	for _, player := range plan.After.Players {
		if player.Status != domain.RosterActive || player.Franchise == "" {
			t.Fatalf("player released or left on reserve: %+v", player)
		}
	}
	if len(plan.Pending) != 1 || plan.Pending[0].Player != "0001" || plan.Pending[0].Franchise != "f01" {
		t.Fatalf("expiration must be pending on its franchise: %+v", plan.Pending)
	}
	if !strings.Contains(strings.Join(plan.Lines, "\n"), "contract years remaining change from 3 to 2") {
		t.Fatal("contract did not age one year")
	}
	if !reflect.DeepEqual(plan.After.Picks, state.Assets.Picks) {
		t.Fatal("picks were issued or ownership changed at rollover")
	}
	if !reflect.DeepEqual(plan.After.Adjustments, state.Assets.Adjustments) {
		t.Fatal("recorded charges were changed or dropped")
	}
}

func waitingRolloverRule(rules leaguerules.RuleSet, value leaguerules.Money, day int) leaguerules.Change {
	return leaguerules.Change{LeagueID: rules.LeagueID, Season: rules.Season, Field: "cap.amount",
		New: leaguerules.Known(value), Effective: leaguerules.Effective{Kind: leaguerules.NextRollover},
		RecordedAt: time.Date(2026, 10, day, 12, 0, 0, 0, time.UTC)}
}

func TestRolloverOnlyNewestWaitingRuleApplies(t *testing.T) {
	state, rules := rolloverInputs()
	state.Waiting = []leaguerules.Change{waitingRolloverRule(rules, 9000000000, 3),
		waitingRolloverRule(rules, 8000000000, 2)}
	plan := requireRollover(t, state, rules)
	if plan.Rules.Cap.Amount.Value != 9000000000 || plan.Rules.Season != 2027 {
		t.Fatal("older waiting change won over newest")
	}
	if rules.Cap.Amount.Value == plan.Rules.Cap.Amount.Value {
		t.Fatal("input season rules were overwritten")
	}
	lines := strings.Join(plan.Lines, "\n")
	if strings.Contains(lines, "$80,000,000") || !strings.Contains(lines, "$90,000,000") {
		t.Fatal("preview includes a replaced change")
	}
}

func TestRolloverRedatesCivilWindowsWithoutChangingInputs(t *testing.T) {
	state, rules := rolloverInputs()
	window := rules.Calendar.Windows.Value["rookie_draft"]
	window.DatedStart = leaguerules.Known(time.Date(2026, 7, 14, 4, 0, 0, 0, time.UTC))
	rules.Calendar.Windows.Value["rookie_draft"] = window
	before := rolloverJSON(t, state)
	ruleBefore := rolloverJSON(t, rules)
	plan := requireRollover(t, state, rules)
	for _, dated := range plan.Calendar.Windows {
		if dated.Start.Year() != 2027 || dated.Close != nil && dated.Close.Year() != 2027 {
			t.Fatalf("old window retained: %+v", dated)
		}
	}
	if !bytes.Equal(before, rolloverJSON(t, state)) || !bytes.Equal(ruleBefore, rolloverJSON(t, rules)) {
		t.Fatal("preview changed its inputs")
	}
	plan.After.Players[0].Salary = 999
	plan.Rules.Calendar.Windows.Value["rookie_draft"] = leaguerules.Window{}
	if !bytes.Equal(before, rolloverJSON(t, state)) || !bytes.Equal(ruleBefore, rolloverJSON(t, rules)) {
		t.Fatal("plan aliases its inputs")
	}
}

func rolloverJSON[T any](t *testing.T, value T) []byte {
	t.Helper()
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return append(body, '\n')
}

func TestRolloverBlocksNeededRulingsAndDisclosesUnbuiltFacts(t *testing.T) {
	state, rules := rolloverInputs()
	plan := requireRollover(t, state, rules)
	lines := strings.Join(plan.Lines, "\n")
	for _, missing := range []string{"cannot say yet whether he becomes a restricted or unrestricted free agent", "Salaries stay as recorded.",
		"No champion is decided for season 2026", "no contract end year is recorded, so the contract is unchanged"} {
		if !strings.Contains(lines, missing) {
			t.Fatalf("missing disclosure %q", missing)
		}
	}
	rules.Cap.Amount = leaguerules.Needs[leaguerules.Money]("choose the cap")
	rules.Calendar.Boundary = leaguerules.Needs[leaguerules.BoundarySide]("choose the shared date side")
	plan = requireRollover(t, state, rules)
	if !plan.Blocked || len(plan.Reasons) != 2 || !strings.Contains(strings.Join(plan.Reasons, " "), "Salary cap") {
		t.Fatalf("unset necessary rules did not block: %+v", plan.Reasons)
	}
}

func TestRolloverRefusesNonNextSeasonAndForeignWaitingRules(t *testing.T) {
	state, rules := rolloverInputs()
	for _, to := range []int{2025, 2026, 2028, 10000} {
		if _, err := Rollover(state, rules, 2026, to); err == nil {
			t.Fatalf("accepted destination %d", to)
		}
	}
	change := waitingRolloverRule(rules, 100, 2)
	change.LeagueID = "native.foreign"
	state.Waiting = []leaguerules.Change{change}
	if _, err := Rollover(state, rules, 2026, 2027); err == nil {
		t.Fatal("accepted a different league's waiting rule")
	}
}

func TestRolloverWire(t *testing.T) {
	state, rules := rolloverInputs()
	plan := requireRollover(t, state, rules)
	body := rolloverJSON(t, plan)
	path := "testdata/rollover-preview.wire.json"
	if os.Getenv("TWR_WRITE_WIRE") == "1" {
		if err := os.WriteFile(path, body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(body, want) {
		t.Fatal("rollover wire differs from Go fixture")
	}
}

func TestRolloverKeepsNewlyWaitingDatesAndBlocksImpossibleDates(t *testing.T) {
	state, rules := rolloverInputs()
	deadline := time.Date(2027, 11, 7, 18, 0, 0, 0, time.UTC)
	change := waitingRolloverRule(rules, 0, 2)
	change.Field, change.New = "calendar.tradeDeadline", leaguerules.Known(deadline)
	state.Waiting = []leaguerules.Change{change}
	plan := requireRollover(t, state, rules)
	if plan.Blocked || plan.Calendar.TradeDeadline == nil || !plan.Calendar.TradeDeadline.Equal(deadline) {
		t.Fatal("newly waiting exact deadline was advanced again")
	}
	state, rules = rolloverInputs()
	state.Assets.Season, rules.Season = 2024, 2024
	state.At = time.Date(2025, 3, 1, 5, 0, 0, 0, time.UTC)
	window := rules.Calendar.Windows.Value["rookie_draft"]
	window.DatedStart = leaguerules.Known(time.Date(2024, 2, 29, 5, 0, 0, 0, time.UTC))
	rules.Calendar.Windows.Value["rookie_draft"] = window
	plan, err := Rollover(state, rules, 2024, 2025)
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Blocked || !strings.Contains(strings.Join(plan.Reasons, " "), "day does not exist") {
		t.Fatal("leap-day opening silently moved to a different day")
	}
}

func TestRolloverSnapshotDigestIsStableAndSensitive(t *testing.T) {
	state, rules := rolloverInputs()
	plan := requireRollover(t, state, rules)
	again := requireRollover(t, state, rules)
	if plan.BeforeDigest != again.BeforeDigest || !bytes.Equal(rolloverJSON(t, plan), rolloverJSON(t, again)) {
		t.Fatal("same inputs produced different previews")
	}
	state.Assets.Players[0].Salary++
	changed := requireRollover(t, state, rules)
	if changed.BeforeDigest == plan.BeforeDigest {
		t.Fatal("before digest does not snapshot contract changes")
	}
}
