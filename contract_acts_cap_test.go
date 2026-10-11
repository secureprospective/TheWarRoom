package main

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
	"github.com/secureprospective/TheWarRoom/internal/transactions"
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
	a.targetSnapshot = s
	if err := a.rulebook.Initialize(a.ctx, capSource{}); err != nil {
		t.Fatal(err)
	}
	if err := a.rulebook.SetOverride(a.ctx, "cap", "salaryCapAmount", "125", "fixture"); err != nil {
		t.Fatal(err)
	}
	m := state.MirrorSnapshot{Season: s.League.Value.Season}
	for _, roster := range s.Rosters.Value {
		for _, p := range roster.Players {
			m.Players = append(m.Players, state.MirrorPlayer{MFLID: p.ID.String(), FranchiseID: roster.FranchiseID, Salary: p.Salary, ContractYear: m.Season + *p.YearsRemaining - 1, ContractStatus: p.ContractStatus, RosterStatus: p.RosterStatus})
		}
	}
	if _, err := a.league.Replace(a.ctx, m); err != nil {
		t.Fatal(err)
	}
	_, _, used := a.league.ContractsView()
	for _, team := range s.Franchises.Value {
		if team.CapUsed != nil {
			m.Adjustments = append(m.Adjustments, state.Adjustment{ID: team.ID, FranchiseID: team.ID, Amount: *team.CapUsed - used[team.ID]})
		}
	}
	if _, err := a.league.Replace(a.ctx, m); err != nil {
		t.Fatal(err)
	}
	return a
}

func TestContractActsArizonaHeldMirrorPricesAndReadiness(t *testing.T) {
	a := contractMirrorFixture(t)
	before, stamp, _ := a.league.ContractsView()
	for _, tc := range []struct {
		intent, player string
		phase          domain.Phase
		ready          bool
	}{
		{"rfa.tender", "16846", domain.PhaseOffseason, true},
		{"fa.resign", "16692", domain.PhaseOffseason, true},
		{"contract.tag", "13322", domain.PhaseRegularSeason, false},
		{"contract.extend", "13813", domain.PhaseRegularSeason, true},
		{"contract.restructure", "13813", domain.PhaseRegularSeason, true},
	} {
		releasePhase(t, a, tc.phase)
		raw := RawContractAct{Mine: "0025", Player: tc.player, Action: "draft", Intent: tc.intent, Level: 1, Years: 1, Source: 2026, Destination: 2028, Move: "1"}
		r := a.TargetContractActs(raw)
		if r.Status != ReadingOK {
			t.Fatalf("%s: %+v", tc.intent, r)
		}
		var plan ContractActView
		for _, p := range r.Value {
			if p.Receipt.Spec.Intent == tc.intent && p.Receipt.Spec.Expected.Player.String() == tc.player {
				plan = p
			}
		}
		if (plan.Receipt.State == envelope.Ready) != tc.ready {
			t.Fatalf("%s %+v", tc.intent, plan)
		}
		if tc.intent == "contract.tag" && !strings.Contains(strings.Join(plan.Reasons, " "), "exceeds the team's salary cap") {
			t.Fatal(plan)
		}
		if tc.ready && (!strings.Contains(plan.Post, "Not checked:") && tc.intent != "rfa.tender") {
			t.Fatal(plan.Post)
		}
		t.Logf("%s: %s %s %s, reasons=%v", tc.intent, plan.Player, plan.Price, plan.Receipt.State, plan.Reasons)
	}
	releasePhase(t, a, domain.PhaseRegularSeason)
	for _, intent := range []string{"rfa.tender", "fa.resign"} {
		r := a.TargetContractActs(RawContractAct{Mine: "0025", Player: "16846", Action: "preview", Intent: intent, Level: 1, Years: 1})
		if r.Status != ReadingOK || !strings.Contains(strings.Join(r.Value[0].Reasons, " "), "Offseason only:") {
			t.Fatal(r)
		}
	}
	options := a.TargetContractActs(RawContractAct{Mine: "0025", Player: "13813", Action: "options"})
	if len(options.Value) != 2 || options.Value[0].Intent != "contract.extend" || options.Value[1].Intent != "contract.restructure" {
		t.Fatal(options)
	}
	after, afterStamp, _ := a.league.ContractsView()
	if stamp != afterStamp || !reflect.DeepEqual(before, after) {
		t.Fatal("mirror changed")
	}
}

func TestContractTeamCapMatchesCoordinatorForSameRestructure(t *testing.T) {
	a := contractMirrorFixture(t)
	raw := RawContractAct{Mine: "0025", Player: "13813", Action: "preview", Intent: "contract.restructure", Level: 1, Years: 1, Source: 2026, Destination: 2028, Move: "1"}
	plan := a.TargetContractActs(raw)
	if plan.Status != ReadingOK {
		t.Fatal(plan)
	}
	m, _, used := a.league.ContractsView()
	s := state.WhatIfSession{Mine: raw.Mine, Snapshot: m, Used: used[raw.Mine], Generation: "comparison", TaxiPercent: a.rulebook.TaxiCapPercent(), IRPercent: a.rulebook.IRCapPercent()}
	capAmount := domain.Money(12500000000)
	s.Cap = &capAmount
	store, err := a.loadWhatIf(a.ctx, s)
	if err != nil {
		t.Fatal(err)
	}
	s.SeedUsed, _ = store.CapUsed(raw.Mine)
	coord, err := transactions.New(store.Writer(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := coord.Execute(a.ctx, transactions.Restructure{MFLID: raw.Player, Move: 100000000}); err != nil {
		t.Fatal(err)
	}
	view, err := a.whatIfView(a.ctx, s, store, false)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(plan.Value[0].TeamFigures, view.Figures) {
		t.Fatalf("plan=%+v engine=%+v", plan.Value[0].TeamFigures, view.Figures)
	}
	if _, found, err := state.NewWhatIfSessions(a.whatifPools).Load(context.Background(), raw.Mine); err != nil || found {
		t.Fatal("contract preview published a what-if session", err)
	}
}

func TestContractTagPoolUsesLeagueMirrorBASE(t *testing.T) {
	a := contractMirrorFixture(t)
	releasePhase(t, a, domain.PhaseRegularSeason)
	m, _, _ := a.league.ContractsView()
	f, err := a.contractActFacts(a.ctx, RawContractAct{Mine: "0025", Player: "13322", Intent: "contract.tag", Level: 1, Years: 1})
	if err != nil {
		t.Fatal(err)
	}
	// The five held S salaries are $8M, $8M, $7M, $6.5M and $6M.
	if f.Terms.Price != 710000000 {
		t.Fatalf("mirror tag price %s", f.Terms.Price)
	}
	for i := range m.Players {
		if m.Players[i].MFLID == "13322" {
			m.Players[i].ContractStatus = domain.CStatusFT2
		}
	}
	if _, err := a.league.Replace(a.ctx, m); err != nil {
		t.Fatal(err)
	}
	f, err = a.contractActFacts(a.ctx, RawContractAct{Mine: "0025", Player: "13322", Intent: "contract.tag", Level: 1, Years: 1})
	if err != nil {
		t.Fatal(err)
	}
	result := (envelope.ContractCheck{Reasons: f.Reasons}).Evaluate(envelope.Spec{Intent: f.Intent, LeagueID: ingestion.LeagueID}, snapshot.Snapshot{})
	if !result.Blocked || !strings.Contains(result.Note, "two consecutive") {
		t.Fatal(result)
	}
}
