package commissioner

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func actionCommand(action, params, effective string, on bool) RawCommand {
	return RawCommand{League: "native.test", Command: ContractActionSet,
		Args:     RawArgs{ContractAction: &RawContractAction{Action: action, On: &on, Params: json.RawMessage(params)}},
		Expected: 1, Request: "action", Reason: "Owners agreed", Effective: effective, Visibility: "league"}
}

func TestContractActionBoundary(t *testing.T) {
	good := actionCommand("buyout", `{"AnnualLimit":3}`, "next_rollover", true)
	for _, timing := range []struct{ effective, gravity string }{{"next_rollover", "G2"}, {"now", "G3"}} {
		raw := good
		raw.Effective = timing.effective
		definition, err := raw.EffectiveDefinition()
		if err != nil || definition.Gravity != timing.gravity {
			t.Fatalf("timing safeguards: %+v %v", definition, err)
		}
		body, err := json.Marshal(raw)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Decode(string(body)); err != nil {
			t.Fatal(err)
		}
	}
	for _, change := range []func(*RawCommand){
		func(r *RawCommand) { r.Args.ContractAction = nil },
		func(r *RawCommand) { r.Args.Field = "contracts.buyout" },
		func(r *RawCommand) { r.Args.Override = &RawNativeOverride{} },
		func(r *RawCommand) { r.Args.People = &RawPeople{} },
		func(r *RawCommand) { r.Visibility = "private" },
		func(r *RawCommand) { r.Effective = "2027-03-01T05:00:00Z" },
		func(r *RawCommand) { r.Effective, r.Reason = "now", " " },
		func(r *RawCommand) { r.Effective, r.Gravity = "now", "G2" },
	} {
		bad := good
		change(&bad)
		if err := bad.Validate(); err == nil {
			t.Fatalf("accepted foreign arguments or lowered safeguards: %+v", bad)
		}
	}
}

func TestContractActionParametersUseTheCatalogue(t *testing.T) {
	for _, tc := range []struct{ action, params string }{
		{"unknown", `{}`}, {"tag", `null`}, {"tag", `[]`}, {"tag", `{"AnnualLimit":-1}`},
		{"tag", `{"AnnualLimit":1.1}`}, {"tag", `{"AnnualLimit":"one"}`},
		{"tag", `{"AnnualLimit":null}`}, {"tag", `{"Unknown":2}`},
		{"tag", `{"Enabled":true}`}, {"tag", `{"enabled":false}`},
		{"tag", `{"AnnualLimit":1,"AnnualLimit":2}`},
		{"tag", `{"AnnualLimit":1,"annualLimit":2}`}, {"tag", `{} {}`},
		{"extension", `{"Floors":{"WR":-1}}`}, {"extension", `{"Floors":{"invalid":1}}`},
		{"rfa_tender", `{"MatchWindow":-1}`}, {"restructure", `{"Tiers":[{"Amount":-1}]}`},
	} {
		raw := actionCommand(tc.action, tc.params, "next_rollover", true)
		if err := raw.Validate(); err == nil {
			t.Fatalf("accepted invalid action parameters: %s %s", tc.action, tc.params)
		}
	}
	raw := actionCommand("tag", `{}`, "next_rollover", true)
	raw.Args.ContractAction.On = nil
	if err := raw.Validate(); err == nil {
		t.Fatal("missing switch became off")
	}
	for _, command := range []string{RulesSet, Revert, WindowRemove, ContractEdit, Create} {
		raw := actionCommand("tag", `{}`, "now", true)
		raw.Command = command
		if err := raw.Validate(); err == nil {
			t.Fatalf("contract arguments leaked into %s", command)
		}
	}
}

func TestContractActionPermissionsIncludeReversal(t *testing.T) {
	b := newLedger(t, PresetLegacy)
	state, err := governance.Replay([]governance.Event{
		governance.LeagueCreated{Creator: "creator", Franchises: []governance.FranchiseID{"f1"}},
		governance.SeatReplaced{Franchise: "f1", Principal: "owner"},
		governance.RoleGranted{Principal: "cc", Role: governance.CoCommissioner},
		governance.RoleGranted{Principal: "dot", Role: governance.DOT},
	})
	if err != nil {
		t.Fatal(err)
	}
	auth := GovernanceAuthorizer{State: state}
	entry := b.add(actionCommand("tag", `{}`, "now", false), seasonMoment(3))
	for _, principal := range []string{"creator", "cc", "dot", "owner", "stranger"} {
		_, err := auth.Allow("native.test", principal, ContractActionSet)
		if (err == nil) != (principal == "creator") {
			t.Fatalf("permission for %s: %v", principal, err)
		}
		err = authorizeReversal(auth, entry, principal, b.records)
		if (err == nil) != (principal == "creator") {
			t.Fatalf("reversal permission for %s: %v", principal, err)
		}
	}
}

func TestContractActionUsesEveryCatalogueParameter(t *testing.T) {
	fields, current := leaguerules.Catalogue(), leaguerules.Current(leaguerules.LegacyNFL())
	covered := map[leaguerules.FieldPath]bool{}
	for action, field := range contractActionFields() {
		for i, candidate := range fields {
			if candidate.Path != field {
				continue
			}
			covered[field] = true
			var params map[string]json.RawMessage
			if err := json.Unmarshal(current[i], &params); err != nil {
				t.Fatal(err)
			}
			delete(params, "Enabled")
			body, err := json.Marshal(params)
			if err != nil {
				t.Fatal(err)
			}
			raw := actionCommand(action, string(body), "now", true)
			b := newLedger(t, PresetEmpty)
			b.add(raw, seasonMoment(2))
			rules, err := RulesNow(b.records, seasonMoment(3))
			if err != nil || !reflect.DeepEqual(current[i], leaguerules.Current(rules)[i]) {
				t.Fatalf("%s parameters diverged from catalogue: %s / %s (%v)", action,
					current[i], leaguerules.Current(rules)[i], err)
			}
		}
	}
	if len(covered) != 9 || len(covered) != len(contractActionFields()) {
		t.Fatalf("missing or duplicate contract action field: %v", covered)
	}
}

func TestContractActionAndRulesSetShareSeasonOrdering(t *testing.T) {
	b := newLedger(t, PresetLegacy)
	b.set(rolloverField, rolloverAt, "now", seasonMoment(2))
	first := b.add(actionCommand("buyout", `{"AnnualLimit":3}`, "next_rollover", true), seasonMoment(3))
	b.set("contracts.buyout", `{"Enabled":false,"AnnualLimit":4}`, "next_rollover", seasonMoment(4))
	reading, err := RulesReading("native.test", b.records, seasonMoment(5))
	if err != nil {
		t.Fatal(err)
	}
	assertActionPending(t, reading, first)
	assertBuyoutLimit(t, b.records, seasonMoment(5), 2, true)
	rollover := seasonMoment(5).AddDate(1, 0, 0)
	assertBuyoutLimit(t, b.records, rollover, 4, false)
	b.revert(b.records[len(b.records)-1].Receipt.Entry, seasonMoment(6))
	assertBuyoutLimit(t, b.records, rollover, 3, true)
	b.revert(first, seasonMoment(7))
	assertBuyoutLimit(t, b.records, rollover, 2, true)
}

func assertActionPending(t *testing.T, reading Rules, entry string) {
	t.Helper()
	for _, section := range reading.Sections {
		for _, row := range section.Rows {
			if row.Field == "contracts.buyout" && len(row.Pending) == 2 && row.Pending[0].Entry == entry &&
				row.Pending[0].ReplacedAt != "" && strings.Contains(row.Pending[0].Value, "3") {
				return
			}
		}
	}
	t.Fatal("named and general changes did not share pending/replacement history")
}

func assertBuyoutLimit(t *testing.T, records []Record, at time.Time, limit int, on bool) {
	t.Helper()
	rules, err := RulesNow(records, at)
	if err != nil {
		t.Fatal(err)
	}
	buyout := rules.Contracts.Buyout
	if buyout.State != leaguerules.IsKnown || buyout.Value.AnnualLimit.State != leaguerules.IsKnown ||
		buyout.Value.AnnualLimit.Value != limit || buyout.Value.Enabled.Value != on {
		t.Fatalf("named and general changes disagreed: %+v; want %d, %t", buyout, limit, on)
	}
}

func TestContractActionOnOffAndUnknown(t *testing.T) {
	b := newLedger(t, PresetLegacy)
	b.add(actionCommand("buyout", `{"AnnualLimit":3}`, "now", false), seasonMoment(2))
	assertBuyoutLimit(t, b.records, seasonMoment(3), 3, false)
	rules, err := RulesNow(b.records, seasonMoment(3))
	if err != nil {
		t.Fatal(err)
	}
	if rules.Contracts.Buyout.Value.Rates.State != leaguerules.NeedsRuling {
		t.Fatal("omitted parameters borrowed the preset's price table")
	}
	b.add(actionCommand("buyout", `{"AnnualLimit":4}`, "now", true), seasonMoment(4))
	assertBuyoutLimit(t, b.records, seasonMoment(5), 4, true)
}

func TestContractActionKeepsEndedSeasonAndCompensatedCompensation(t *testing.T) {
	b := newLedger(t, PresetLegacy)
	b.set(rolloverField, `"2026-10-10T12:00:00Z"`, "now", seasonMoment(2))
	entry := b.add(actionCommand("buyout", `{"AnnualLimit":3}`, "now", false), seasonMoment(3))
	b.revert(entry, seasonMoment(11))
	prior, err := RulesInForce(b.records, 2026, seasonMoment(12))
	if err != nil {
		t.Fatal(err)
	}
	if prior.Contracts.Buyout.Value.AnnualLimit.Value != 3 || prior.Contracts.Buyout.Value.Enabled.Value {
		t.Fatal("later compensation rewrote an ended season's contract rule")
	}
	assertBuyoutLimit(t, b.records, seasonMoment(12), 2, true)
	b.revert(b.records[len(b.records)-1].Receipt.Entry, seasonMoment(13))
	assertBuyoutLimit(t, b.records, seasonMoment(14), 3, false)
}
