package contracts

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	mflwhatif "github.com/secureprospective/TheWarRoom/internal/mflref/whatif"

	"github.com/secureprospective/TheWarRoom/internal/app/mflref/contracts/contractstest"
	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine/contractacts"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/mflref/envelope"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
	"github.com/secureprospective/TheWarRoom/internal/transactions"
)

// mirrorHost is the held Arizona mirror behind the API: the release fixture with the frontend
// snapshot's rosters and cap adjustments replacing its mirror, at a $125M cap. The snapshot is
// held as the directory value but not marked loaded, as the root fixture held it.
func mirrorHost(t *testing.T) (*API, *fakeHost) {
	t.Helper()
	a, h := planHost(t)
	s := contractstest.HeldSnapshot(t)
	h.snap = s
	if err := h.Rulebook.Initialize(h.Ctx, capSource{}); err != nil {
		t.Fatal(err)
	}
	if err := h.Rulebook.SetOverride(h.Ctx, "cap", "salaryCapAmount", "125", "fixture"); err != nil {
		t.Fatal(err)
	}
	m := state.MirrorSnapshot{Season: s.League.Value.Season}
	for _, roster := range s.Rosters.Value {
		for _, p := range roster.Players {
			m.Players = append(m.Players, state.MirrorPlayer{MFLID: p.ID.String(), FranchiseID: roster.FranchiseID, Salary: p.Salary, ContractYear: m.Season + *p.YearsRemaining - 1, ContractStatus: p.ContractStatus, RosterStatus: p.RosterStatus})
		}
	}
	if _, err := h.League.Replace(h.Ctx, m); err != nil {
		t.Fatal(err)
	}
	_, _, used := h.League.ContractsView()
	for _, team := range s.Franchises.Value {
		if team.CapUsed != nil {
			m.Adjustments = append(m.Adjustments, state.Adjustment{ID: team.ID, FranchiseID: team.ID, Amount: *team.CapUsed - used[team.ID]})
		}
	}
	if _, err := h.League.Replace(h.Ctx, m); err != nil {
		t.Fatal(err)
	}
	return a, h
}

// capSource is the league configuration behind the rulebook: a $125M cap.
type capSource struct{}

func (capSource) Fetch(context.Context) (league.RawConfig, error) {
	return league.RawConfig{Source: "test", SalaryCapAmount: "125", RosterSize: "80", TaxiSquad: "8",
		ScoringRules: []league.PositionRuleSet{{Positions: "CB|S", Rules: []league.ScoringRule{
			{Event: "TK", Points: "*0.5", Range: "0-99"}}}},
		RosterLimits: []league.PositionLimit{{Name: "QB", Limit: "0-0"}}}, nil
}

func TestContractTeamCapMatchesCoordinatorForSameRestructure(t *testing.T) {
	t.Parallel()
	a, h := mirrorHost(t)
	raw := RawContractAct{Mine: "0025", Player: "13813", Action: "preview", Intent: "contract.restructure", Level: 1, Years: 1, Source: 2026, Destination: 2028, Move: "1"}
	plan := a.TargetContractActs(raw)
	if plan.Status != report.ReadingOK {
		t.Fatal(plan)
	}
	m, _, used := h.League.ContractsView()
	s := mflwhatif.WhatIfSession{Mine: raw.Mine, Snapshot: m, Used: used[raw.Mine], Generation: "comparison", TaxiPercent: h.Rulebook.TaxiCapPercent(), IRPercent: h.Rulebook.IRCapPercent()}
	capAmount := domain.Money(12500000000)
	s.Cap = &capAmount
	store, err := LoadWhatIf(h.Ctx, h.WhatifPools, s)
	if err != nil {
		t.Fatal(err)
	}
	s.SeedUsed, _ = store.CapUsed(raw.Mine)
	coord, err := transactions.New(store.Writer(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := coord.Execute(h.Ctx, transactions.Restructure{MFLID: raw.Player, Move: 100000000}); err != nil {
		t.Fatal(err)
	}
	view, err := a.whatIfViewThrough(h.Ctx, s, store, false, s.Snapshot.Season)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(plan.Value[0].TeamFigures, view.Figures) {
		t.Fatalf("plan=%+v engine=%+v", plan.Value[0].TeamFigures, view.Figures)
	}
	if _, found, err := mflwhatif.NewWhatIfSessions(h.WhatifPools).Load(context.Background(), raw.Mine); err != nil || found {
		t.Fatal("contract preview published a what-if session", err)
	}
}

func TestContractTagPoolUsesLeagueMirrorBASE(t *testing.T) {
	t.Parallel()
	a, h := mirrorHost(t)
	h.phase(t, domain.PhaseRegularSeason)
	m, _, _ := h.League.ContractsView()
	f, err := a.contractActFacts(h.Ctx, RawContractAct{Mine: "0025", Player: "13322", Intent: "contract.tag", Level: 1, Years: 1})
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
	if _, err := h.League.Replace(h.Ctx, m); err != nil {
		t.Fatal(err)
	}
	f, err = a.contractActFacts(h.Ctx, RawContractAct{Mine: "0025", Player: "13322", Intent: "contract.tag", Level: 1, Years: 1})
	if err != nil {
		t.Fatal(err)
	}
	result := (envelope.ContractCheck{Reasons: f.Reasons}).Evaluate(envelope.Spec{Intent: f.Intent, LeagueID: ingestion.LeagueID}, snapshot.Snapshot{})
	if !result.Blocked || !strings.Contains(result.Note, "two consecutive") {
		t.Fatal(result)
	}
}

func contractFigure(t *testing.T, rows []WhatIfFigure, label string) WhatIfFigure {
	t.Helper()
	for _, row := range rows {
		if row.Label == label {
			return row
		}
	}
	t.Fatalf("no %q row in %+v", label, rows)
	return WhatIfFigure{}
}

func TestContractNewContractStartsTheSeasonAfterExpiry(t *testing.T) {
	t.Parallel()
	a, h := mirrorHost(t)
	h.phase(t, domain.PhaseOffseason)
	resign, err := contractacts.Tender(83_000_000, 1)
	if err != nil {
		t.Fatal(err)
	}
	resign, err = contractacts.Resign(resign, 2)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		intent, player, held string
		years                int
		cells                []contractacts.Cell
	}{
		{"rfa.tender", "16846", "$330,000", 1, []contractacts.Cell{{Season: 2027, Salary: 60_000_000}}},
		{"fa.resign", "16692", "$830,000", 2, []contractacts.Cell{{Season: 2027, Salary: resign}, {Season: 2028, Salary: resign}}},
		{"contract.tag", "13322", "$950,000", 1, []contractacts.Cell{{Season: 2027, Salary: 710_000_000}}},
	} {
		t.Run(tc.intent, func(t *testing.T) {
			r := a.TargetContractActs(RawContractAct{Mine: "0025", Player: tc.player, Action: "preview", Intent: tc.intent, Level: 1, Years: tc.years})
			if r.Status != report.ReadingOK {
				t.Fatal(r)
			}
			f, err := a.contractActFacts(h.Ctx, RawContractAct{Mine: "0025", Player: tc.player, Intent: tc.intent, Level: 1, Years: tc.years})
			if err != nil || !reflect.DeepEqual(f.Terms.Cells, tc.cells) {
				t.Fatalf("cells %+v %v", f.Terms.Cells, err)
			}
			view := r.Value[0]
			if held := view.Figures[0]; held.Season != 2026 || held.Before != tc.held || held.After != tc.held || held.Change != "$0" {
				t.Fatalf("expiring cell changed %+v", view.Figures)
			}
			for _, label := range []string{"Used", "Room"} {
				if row := contractFigure(t, view.TeamFigures, label); row.Change != "" || row.Before != row.After {
					t.Fatalf("this season changed %+v", row)
				}
			}
			for _, c := range tc.cells {
				row := contractFigure(t, view.TeamFigures, fmt.Sprintf("%d commitments at today's salaries", c.Season))
				if row.Change != "+"+snapshot.ContractAmountOf(c.Salary).Text {
					t.Fatalf("%d not raised by the new cell %+v", c.Season, row)
				}
				if !strings.Contains(view.Post, fmt.Sprintf("%d: %s\n", c.Season, snapshot.ContractAmountOf(c.Salary).Text)) {
					t.Fatal(view.Post)
				}
			}
			if strings.Contains(view.Post, "2026: $") || !strings.Contains(view.Post, "2027") {
				t.Fatal(view.Post)
			}
		})
	}
}

func TestContractOverCapJudgesTheSeasonTheMoneyLandsIn(t *testing.T) {
	t.Parallel()
	a, h := mirrorHost(t)
	h.phase(t, domain.PhaseOffseason)
	// $110M: this season is already over the cap; 2027 holds $104.67M before any Act.
	if err := h.Rulebook.SetOverride(h.Ctx, "cap", "salaryCapAmount", "110", "fixture"); err != nil {
		t.Fatal(err)
	}
	reasons := func(intent, player string) string {
		f, err := a.contractActFacts(h.Ctx, RawContractAct{Mine: "0025", Player: player, Intent: intent, Level: 1, Years: 1})
		if err != nil {
			t.Fatal(err)
		}
		if room := contractFigure(t, contractActView(envelope.Receipt{}, f).TeamFigures, "Room"); !strings.HasPrefix(room.Before, "-") {
			t.Fatalf("fixture room %+v", room)
		}
		return strings.Join(f.Reasons, "\n")
	}
	if got := reasons("rfa.tender", "16846"); strings.Contains(got, "exceeds") || !strings.Contains(got, "Not checked: salary caps after 2026") {
		t.Fatalf("a 2027 tender was judged on this season's room:\n%s", got)
	}
	if got := reasons("contract.tag", "13322"); !strings.Contains(got, "This move exceeds the team's 2027 salary cap.") {
		t.Fatalf("a 2027 tag over the 2027 commitments passed:\n%s", got)
	}
}
