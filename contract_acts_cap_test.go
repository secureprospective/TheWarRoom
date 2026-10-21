package main

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/app/report"

	contractsapp "github.com/secureprospective/TheWarRoom/internal/app/contracts"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

func contractMirrorFixture(t *testing.T) *App {
	t.Helper()
	a := releaseTestApp(t)
	data, err := os.ReadFile("frontend/src/app/data/fixtures/snapshot.json")
	if err != nil {
		t.Fatal(err)
	}
	var s snapshot.Snapshot
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatal(err)
	}
	a.core.TargetSnapshot = s
	if err := a.core.Rulebook.Initialize(a.core.Ctx, capSource{}); err != nil {
		t.Fatal(err)
	}
	if err := a.core.Rulebook.SetOverride(a.core.Ctx, "cap", "salaryCapAmount", "125", "fixture"); err != nil {
		t.Fatal(err)
	}
	m := state.MirrorSnapshot{Season: s.League.Value.Season}
	for _, roster := range s.Rosters.Value {
		for _, p := range roster.Players {
			m.Players = append(m.Players, state.MirrorPlayer{MFLID: p.ID.String(), FranchiseID: roster.FranchiseID, Salary: p.Salary, ContractYear: m.Season + *p.YearsRemaining - 1, ContractStatus: p.ContractStatus, RosterStatus: p.RosterStatus})
		}
	}
	if _, err := a.core.League.Replace(a.core.Ctx, m); err != nil {
		t.Fatal(err)
	}
	_, _, used := a.core.League.ContractsView()
	for _, team := range s.Franchises.Value {
		if team.CapUsed != nil {
			m.Adjustments = append(m.Adjustments, state.Adjustment{ID: team.ID, FranchiseID: team.ID, Amount: *team.CapUsed - used[team.ID]})
		}
	}
	if _, err := a.core.League.Replace(a.core.Ctx, m); err != nil {
		t.Fatal(err)
	}
	return a
}

func TestContractActsArizonaHeldMirrorPricesAndReadiness(t *testing.T) {
	t.Parallel()
	a := contractMirrorFixture(t)
	before, stamp, _ := a.core.League.ContractsView()
	for _, tc := range []struct {
		intent, player string
		phase          domain.Phase
		ready          bool
	}{
		{"rfa.tender", "16846", domain.PhaseOffseason, true},
		{"fa.resign", "16692", domain.PhaseOffseason, true},
		{"contract.tag", "13322", domain.PhaseRegularSeason, true},
		{"contract.extend", "13813", domain.PhaseRegularSeason, true},
		{"contract.restructure", "13813", domain.PhaseRegularSeason, true},
	} {
		releasePhase(t, a, tc.phase)
		raw := contractsapp.RawContractAct{Mine: "0025", Player: tc.player, Action: "draft", Intent: tc.intent, Level: 1, Years: 1, Source: 2026, Destination: 2028, Move: "1"}
		r := a.contracts().TargetContractActs(raw)
		if r.Status != report.ReadingOK {
			t.Fatalf("%s: %+v", tc.intent, r)
		}
		var plan contractsapp.ContractActView
		for _, p := range r.Value {
			if p.Receipt.Spec.Intent == tc.intent && p.Receipt.Spec.Expected.Player.String() == tc.player {
				plan = p
			}
		}
		if tc.intent == "rfa.tender" && !strings.Contains(plan.Post, "Priced from his 2026 salary, $330,000.") {
			t.Fatal(plan.Post)
		}
		if (plan.Receipt.State == envelope.Ready) != tc.ready {
			t.Fatalf("%s %+v", tc.intent, plan)
		}
		if tc.ready && (!strings.Contains(plan.Post, "Not checked:") && tc.intent != "rfa.tender") {
			t.Fatal(plan.Post)
		}
		t.Logf("%s: %s %s %s, reasons=%v", tc.intent, plan.Player, plan.Price, plan.Receipt.State, plan.Reasons)
	}
	releasePhase(t, a, domain.PhaseRegularSeason)
	for _, intent := range []string{"rfa.tender", "fa.resign"} {
		r := a.contracts().TargetContractActs(contractsapp.RawContractAct{Mine: "0025", Player: "16846", Action: "preview", Intent: intent, Level: 1, Years: 1})
		if r.Status != report.ReadingOK || !strings.Contains(strings.Join(r.Value[0].Reasons, " "), "Offseason only:") {
			t.Fatal(r)
		}
	}
	options := a.contracts().TargetContractActs(contractsapp.RawContractAct{Mine: "0025", Player: "13813", Action: "options"})
	if len(options.Value) != 2 || options.Value[0].Intent != "contract.extend" || options.Value[1].Intent != "contract.restructure" {
		t.Fatal(options)
	}
	after, afterStamp, _ := a.core.League.ContractsView()
	if stamp != afterStamp || !reflect.DeepEqual(before, after) {
		t.Fatal("mirror changed")
	}
}
