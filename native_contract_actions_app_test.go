package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func contractActionApp(t *testing.T) *legalityFixture {
	t.Helper()
	a := commissionerTestApp(t)
	path := filepath.Join(t.TempDir(), "contract-actions.db")
	store, pools := openDirectoryHistory(t, path)
	a.core.History = store
	executeCommissionerTest(t, a, commissioner.RawCommand{League: legalityLeague, Command: commissioner.Create,
		Args:   commissioner.RawArgs{Name: "Native contracts", Preset: commissioner.PresetLegacy, Season: 2026},
		Reason: "Start this league", Request: "create", Effective: "now", Visibility: "league"})
	return &legalityFixture{app: a, path: path, pools: pools}
}

func contractActionRow(t *testing.T, f *legalityFixture, field string) commissioner.RuleRow {
	t.Helper()
	rules, err := f.app.TargetCommissionerRules(legalityLeague)
	if err != nil {
		t.Fatal(err)
	}
	for _, section := range rules.Sections {
		for _, row := range section.Rows {
			if row.Field == field {
				return row
			}
		}
	}
	t.Fatalf("no rule reading for %s", field)
	return commissioner.RuleRow{}
}

func nativeActionCommand(t *testing.T, f *legalityFixture, action, params, effective string,
	on bool) commissioner.RawCommand {
	t.Helper()
	rules, err := f.app.TargetCommissionerRules(legalityLeague)
	if err != nil {
		t.Fatal(err)
	}
	return commissioner.RawCommand{League: legalityLeague, Command: commissioner.ContractActionSet,
		Args: commissioner.RawArgs{ContractAction: &commissioner.RawContractAction{Action: action,
			On: &on, Params: json.RawMessage(params)}}, Expected: rules.Revision,
		Request: fmt.Sprintf("action.%d", rules.Revision), Effective: effective,
		Visibility: "league", Reason: "Our league's contract ruling"}
}

func assertActionCurrent(t *testing.T, f *legalityFixture, field, value string) commissioner.RuleRow {
	t.Helper()
	row := contractActionRow(t, f, field)
	var got, want map[string]json.RawMessage
	if err := json.Unmarshal(row.Current, &got); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(value), &want); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) || row.Citation != "Changed in the league ledger" || row.ChangedAt == "" {
		t.Fatalf("contract action and rule reading diverged: %+v; want %s", row, value)
	}
	return row
}

func TestNativeContractActionCommandsReplayAndCompensate(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ action, field, params string }{
		{"tag", "contracts.tag", `{"AnnualLimit":3}`},
		{"extension", "contracts.extension", `{"AddedYears":2}`},
		{"restructure", "contracts.restructure", `{"AnnualLimit":3}`},
		{"buyout", "contracts.buyout", `{"AnnualLimit":3}`},
		{"rfa_tender", "contracts.tender", `{"PriorSalaryMultiplier":11500}`},
		{"resign", "contracts.reSign", `{"Premiums":{"2":2000}}`},
		{"retire", "contracts.retirement", `{"Rate":2500}`},
		{"death", "contracts.death", `{"Rate":1000}`},
		{"cap_relief", "contracts.capRelief", `{"Reasons":["Career-ending injury"]}`},
	} {
		t.Run(tc.action, func(t *testing.T) {
			t.Parallel()
			f := contractActionApp(t)
			before := contractActionRow(t, f, tc.field).Current
			for pass := 0; pass < 2; pass++ {
				on := nativeActionCommand(t, f, tc.action, tc.params, "now", true)
				receipt := executeCommissionerTest(t, f.app, on)
				if again := executeCommissionerTest(t, f.app, on); again != receipt {
					t.Fatal("a contract action retry appended twice")
				}
				assertActionCurrent(t, f, tc.field, actionNatural(t, tc.params, true))
				f.reopen(t)
				assertActionCurrent(t, f, tc.field, actionNatural(t, tc.params, true))
				off := executeCommissionerTest(t, f.app,
					nativeActionCommand(t, f, tc.action, tc.params, "now", false))
				assertActionCurrent(t, f, tc.field, actionNatural(t, tc.params, false))
				f.revert(t, off)
				assertActionCurrent(t, f, tc.field, actionNatural(t, tc.params, true))
				f.revert(t, receipt)
				f.reopen(t)
				if got := contractActionRow(t, f, tc.field).Current; !reflect.DeepEqual(got, before) {
					t.Fatalf("compensated contract action survived replay: %s / %s", got, before)
				}
			}
		})
	}
}

func actionNatural(t *testing.T, params string, on bool) string {
	t.Helper()
	var values map[string]json.RawMessage
	if err := json.Unmarshal([]byte(params), &values); err != nil {
		t.Fatal(err)
	}
	values["Enabled"] = json.RawMessage(fmt.Sprintf("%t", on))
	body, err := json.Marshal(values)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func TestNativeContractActionAndRulesSetCannotDisagree(t *testing.T) {
	t.Parallel()
	f := contractActionApp(t)
	for pass := 0; pass < 2; pass++ {
		first := executeCommissionerTest(t, f.app,
			nativeActionCommand(t, f, "buyout", `{"AnnualLimit":3}`, "now", true))
		second := f.set(t, "contracts.buyout", `{"Enabled":false,"AnnualLimit":4}`, "now")
		assertActionCurrent(t, f, "contracts.buyout", `{"Enabled":false,"AnnualLimit":4}`)
		third := executeCommissionerTest(t, f.app,
			nativeActionCommand(t, f, "buyout", `{"AnnualLimit":5}`, "now", true))
		f.reopen(t)
		assertActionCurrent(t, f, "contracts.buyout", `{"Enabled":true,"AnnualLimit":5}`)
		f.revert(t, third)
		assertActionCurrent(t, f, "contracts.buyout", `{"Enabled":false,"AnnualLimit":4}`)
		f.revert(t, second)
		assertActionCurrent(t, f, "contracts.buyout", `{"Enabled":true,"AnnualLimit":3}`)
		f.revert(t, first)
		f.reopen(t)
	}
}

func TestNativeContractActionRefusalsAreAtomic(t *testing.T) {
	t.Parallel()
	f := contractActionApp(t)
	raw := nativeActionCommand(t, f, "buyout", `{"AnnualLimit":3}`, "now", true)
	receipt := executeCommissionerTest(t, f.app, raw)
	for _, change := range []func(*commissioner.RawCommand){
		func(r *commissioner.RawCommand) { r.Request = "stale" },
		func(r *commissioner.RawCommand) { r.Reason = "Changed request identity" },
		func(r *commissioner.RawCommand) { r.Args.ContractAction.Params = json.RawMessage(`{"Unknown":1}`) },
	} {
		bad := raw
		action := *raw.Args.ContractAction
		bad.Args.ContractAction = &action
		change(&bad)
		body, err := json.Marshal(bad)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.app.TargetCommissionerExecute(string(body)); err == nil {
			t.Fatalf("invalid contract action appended: %s", body)
		}
	}
	rules, err := f.app.TargetCommissionerRules(legalityLeague)
	if err != nil || rules.Revision != receipt.Revision {
		t.Fatalf("refused action changed revision: %+v %v", rules, err)
	}
	assertActionCurrent(t, f, "contracts.buyout", `{"Enabled":true,"AnnualLimit":3}`)
}

func TestNativeContractActionPendingHistoryAndWire(t *testing.T) {
	t.Parallel()
	f := contractActionApp(t)
	first := executeCommissionerTest(t, f.app,
		nativeActionCommand(t, f, "buyout", `{"AnnualLimit":3}`, "next_rollover", true))
	f.set(t, "contracts.buyout", `{"Enabled":false,"AnnualLimit":4}`, "next_rollover")
	f.reopen(t)
	row := contractActionRow(t, f, "contracts.buyout")
	if len(row.Pending) != 2 || row.Pending[0].Entry != first.Entry || row.Pending[0].ReplacedAt == "" ||
		!strings.Contains(row.Pending[0].Value, "3") {
		t.Fatalf("named command missing from waiting history: %+v", row)
	}
	f.set(t, "calendar.rollover", `"2027-03-01T05:00:00Z"`, "now")
	f.app.core.History.SetClock(func() time.Time { return time.Date(2027, 3, 1, 5, 0, 0, 0, time.UTC) })
	row = assertActionCurrent(t, f, "contracts.buyout", `{"Enabled":false,"AnnualLimit":4}`)
	if len(row.Pending) != 0 {
		t.Fatal("applied action still waiting")
	}
	ledger, err := f.app.TargetCommissionerLedger(legalityLeague)
	if err != nil {
		t.Fatal(err)
	}
	var named commissioner.Entry
	for _, entry := range ledger.Entries {
		if entry.ID == first.Entry {
			named = entry
		}
	}
	if !strings.Contains(named.What, "Contract buyout rules") ||
		!strings.Contains(named.What, "3") || named.ReplacedAt == "" {
		t.Fatalf("named contract action not labelled as a rule change: %+v", named)
	}
	body, err := json.MarshalIndent(struct {
		Rule   commissioner.RuleRow `json:"rule"`
		Ledger commissioner.Ledger  `json:"ledger"`
	}{row, ledger}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	checkWire(t, "internal/commissioner/testdata/contract-actions.wire.json", body, "contract action rules")
}
