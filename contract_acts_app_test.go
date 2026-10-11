package main

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine/contractacts"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

func TestContractActsPreviewDraftStaleRebaseAndMirrorIsolation(t *testing.T) {
	a := releaseTestApp(t)
	idPlayer, err := playerid.New("0042")
	if err != nil {
		t.Fatal(err)
	}
	a.targetSnapshot.Players.Value = []snapshot.Player{{ID: idPlayer, Position: domain.PosQB, Name: "Example"}}
	before, stamp, _ := a.league.ContractsView()
	raw := RawContractAct{Mine: "0001", Player: "0042", Action: "preview", Intent: "contract.extend", Level: 1, Years: 2}
	preview := a.TargetContractActs(raw)
	if preview.Status != ReadingOK || len(preview.Value) != 1 || preview.Value[0].Price != "Price: $15,000,000" || len(preview.Value[0].Figures) != 5 || !strings.HasPrefix(preview.Value[0].Reasons[0], "Not checked:") {
		t.Fatalf("preview %+v", preview)
	}
	raw.Action = "draft"
	plans := a.TargetContractActs(raw)
	if plans.Status != ReadingOK || plans.Value[0].Receipt.State != envelope.Ready || !plans.Value[0].CanHandOff {
		t.Fatalf("draft %+v", plans)
	}
	id := plans.Value[0].ID
	if !strings.Contains(plans.Value[0].Post, "commissioner on ProBoards") || !strings.Contains(plans.Value[0].Post, "2029: $15,000,000") {
		t.Fatal(plans.Value[0].Post)
	}
	after, afterStamp, _ := a.league.ContractsView()
	if !reflect.DeepEqual(before, after) || stamp != afterStamp {
		t.Fatal("contract draft wrote mirror")
	}
	replaceWhatIfMirror(t, a, 9*100_000_000)
	stale := a.TargetContractActs(RawContractAct{Mine: "0001", Action: "read"})
	if stale.Status != ReadingOK || stale.Value[0].Receipt.State != envelope.Stale || !stale.Value[0].CanRebase {
		t.Fatalf("stale %+v", stale)
	}
	rebased := a.TargetContractActs(RawContractAct{Mine: "0001", Action: "rebase", ID: id})
	if rebased.Status != ReadingOK || rebased.Value[0].Receipt.State != envelope.Ready || rebased.Value[0].Figures[0].Before != "$9,000,000" {
		t.Fatalf("rebase %+v", rebased)
	}
	withdrawn := a.TargetContractActs(RawContractAct{Mine: "0001", Action: "withdraw", ID: id})
	if withdrawn.Status != ReadingOK || withdrawn.Value[0].Receipt.State != envelope.Withdrawn || !strings.Contains(withdrawn.Value[0].Post, "2029:") {
		t.Fatalf("withdrawn %+v", withdrawn)
	}
}
func TestContractActsWindowsAndHeldEligibility(t *testing.T) {
	a := releaseTestApp(t)
	id, err := playerid.New("0042")
	if err != nil {
		t.Fatal(err)
	}
	a.targetSnapshot.Players.Value = []snapshot.Player{{ID: id, Position: domain.PosQB}}
	for _, phase := range []domain.Phase{domain.PhaseOffseason, domain.PhaseRegularSeason, domain.PhasePlayoffs} {
		releasePhase(t, a, phase)
		for _, intent := range []string{"rfa.tender", "fa.resign", "contract.tag", "contract.extend", "contract.restructure"} {
			raw := RawContractAct{Mine: "0001", Player: "0042", Action: "preview", Intent: intent, Level: 1, Years: 1, Source: 2026, Destination: 2028, Move: "1"}
			r := a.TargetContractActs(raw)
			if r.Status != ReadingOK {
				t.Fatalf("%s %s: %+v", phase, intent, r)
			}
			result := (envelope.ContractCheck{Reasons: r.Value[0].Reasons}).Evaluate(envelope.Spec{Intent: intent}, snapshot.Snapshot{})
			wantBlocked := intent != "contract.extend" && intent != "contract.restructure"
			if result.Blocked != wantBlocked {
				t.Fatalf("%s %s: %+v", phase, intent, result)
			}
		}
	}
	options := a.TargetContractActs(RawContractAct{Mine: "0001", Player: "0042", Action: "options"})
	if len(options.Value) != 2 || options.Value[0].Intent != "contract.extend" || options.Value[1].Intent != "contract.restructure" {
		t.Fatal(options)
	}
}
func TestContractActsRawBoundaryAndOwnRosterOnly(t *testing.T) {
	a := releaseTestApp(t)
	for _, raw := range []RawContractAct{
		{Mine: "bad", Action: "read"}, {Mine: "0001", Action: "delete"},
		{Mine: "0001", Action: "preview", Player: "x", Intent: "rfa.tender", Level: 1, Years: 1},
		{Mine: "0001", Action: "draft", Player: "0042", Intent: "rfa.tender", Level: 7, Years: 1},
		{Mine: "0001", Action: "draft", Player: "0042", Intent: "contract.extend", Level: 1, Years: 5},
		{Mine: "0001", Action: "draft", Player: "0042", Intent: "contract.restructure", Level: 1, Years: 1, Move: "4"},
		{Mine: "0001", Action: "posted"},
		{Mine: "0002", Action: "preview", Player: "0042", Intent: "rfa.tender", Level: 1, Years: 1},
	} {
		if r := a.TargetContractActs(raw); r.Status != ReadingFailed {
			t.Errorf("bad request accepted %+v", raw)
		}
	}
	raw := RawContractAct{Mine: "0001", Action: "draft", Player: "0042", Intent: "rfa.tender", Level: 1, Years: 1}
	r := a.TargetContractActs(raw)
	id := r.Value[0].ID
	if r := a.TargetContractActs(RawContractAct{Mine: "0002", Action: "withdraw", ID: id}); r.Status != ReadingFailed {
		t.Fatal("cross-franchise mutation")
	}
	if _, err := a.TargetHandOff(id); err == nil {
		t.Fatal("generic handoff bypassed blocker")
	}
	if err := a.revalidateContractPlans(context.Background()); err != nil {
		t.Fatal(err)
	}
}
func TestBuyoutCommissionerChannelAndHandOffWords(t *testing.T) {
	a := releaseTestApp(t)
	plan := planRelease(t, a, "buyout", "0042")
	if plan.Channel != "Post this to the commissioner on ProBoards." {
		t.Fatal(plan.Channel)
	}
	handed := releaseCall(t, a, "handoff", "", plan.ID, false)
	if handed[0].Receipt.State != envelope.HandedOff || !strings.Contains(handed[0].Receipt.Audit[1].Note, "commissioner on ProBoards") || strings.Contains(handed[0].Receipt.Audit[1].Note, "must be confirmed") {
		t.Fatal(handed)
	}
}

func TestContractPostOnlyNamesOwnTerms(t *testing.T) {
	for _, tc := range []struct {
		intent string
		want   string
	}{
		{"contract.extend", "Adds 1 year: 2029 at $16,500,000"},
		{"contract.restructure", "Moves $16,500,000 from 2026 to 2028"},
		{"contract.tag", "Franchise tag for 2029 at $16,500,000"},
		{"rfa.tender", "Tender level 1 for 2029: $16,500,000"},
		{"fa.resign", "Re-sign for 1 years from 2029 at $16,500,000 a year"},
	} {
		t.Run(tc.intent, func(t *testing.T) {
			f := envelope.ContractFacts{Intent: tc.intent, Season: 2027, Years: 1, Level: 1, Source: 2026, Destination: 2028,
				Baseline: []contractacts.Cell{{Season: 2028, Salary: 1_100_000_000}},
				Terms:    contractacts.Terms{Price: 1_650_000_000, Cells: []contractacts.Cell{{Season: 2029, Salary: 1_650_000_000}}}}
			if got := contractPostTerms(f); got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
			post := contractActView(envelope.Receipt{}, f).Post
			if !strings.Contains(post, tc.want) || strings.Contains(post, "term/additional years") {
				t.Fatal(post)
			}
			if tc.intent != "rfa.tender" && strings.Contains(post, "Tender level") {
				t.Fatal(post)
			}
		})
	}
}

func TestContractActSubjectAndOneOpenPlanPerAct(t *testing.T) {
	a := releaseTestApp(t)
	id, err := playerid.New("0042")
	if err != nil {
		t.Fatal(err)
	}
	a.targetSnapshot.Players.Value = []snapshot.Player{{ID: id, Position: domain.PosQB, Name: "Example"}}
	raw := RawContractAct{Mine: "0001", Player: "0042", Action: "preview", Intent: "contract.extend", Level: 1, Years: 1}
	preview := a.TargetContractActs(raw)
	if preview.Status != ReadingOK || preview.Value[0].Subject != "Example · QB · $10,000,000 a year through 2028" {
		t.Fatalf("subject %+v", preview.Value)
	}
	raw.Action = "draft"
	first := a.TargetContractActs(raw)
	if first.Status != ReadingOK || len(first.Value) != 1 || first.Value[0].Subject != preview.Value[0].Subject {
		t.Fatalf("first draft %+v", first)
	}
	raw.Years = 2
	second := a.TargetContractActs(raw)
	if second.Status == ReadingOK || second.AbsenceReason == nil || !strings.Contains(*second.AbsenceReason, "Extend him · Example is already an open plan in My moves") {
		t.Fatalf("second open plan of the same Act was kept: %+v", second)
	}
	raw.Intent, raw.Years, raw.Source, raw.Destination, raw.Move = "contract.restructure", 1, 2026, 2028, "1"
	if other := a.TargetContractActs(raw); other.Status != ReadingOK || len(other.Value) != 2 {
		t.Fatalf("a different Act was refused: %+v", other)
	}
	a.TargetContractActs(RawContractAct{Mine: "0001", Action: "withdraw", ID: first.Value[0].ID})
	raw.Intent, raw.Years = "contract.extend", 2
	if again := a.TargetContractActs(raw); again.Status != ReadingOK || len(again.Value) != 3 {
		t.Fatalf("a removed plan still blocked a new one: %+v", again)
	}
	raw.Player, raw.Years = "0043", 1
	if another := a.TargetContractActs(raw); another.Status != ReadingOK || len(another.Value) != 4 {
		t.Fatalf("the same Act for a different player was refused: %+v", another)
	}
	held := envelope.ContractFacts{Name: "Oluokun, Foyesade", Position: "LB", Status: domain.CStatusUFA, Baseline: []contractacts.Cell{{Season: 2026, Salary: 11 * contractacts.Million}, {Season: 2028, Salary: 11 * contractacts.Million}}}
	if got := contractSubject(held); got != "Oluokun, Foyesade · LB · $11,000,000 a year through 2028 · UFA" {
		t.Fatal(got)
	}
}

func TestContractActChoicesComeFromTheRuleTable(t *testing.T) {
	want := []string{"5th · $0.60M", "4th · $1.00M", "3rd · $1.50M", "2nd · $2.50M", "1st · $5.00M", "1st + 2nd · $6.50M"}
	for _, intent := range []string{"rfa.tender", "fa.resign"} {
		levels, _ := contractActChoices(intent)
		if len(levels) != len(want) {
			t.Fatalf("%s levels %+v", intent, levels)
		}
		for i, l := range levels {
			price, err := domain.ParseMoneyMillions(strings.TrimSuffix(strings.SplitN(l.Label, "$", 2)[1], "M"))
			formula, ferr := contractacts.Tender(0, l.Value)
			if err != nil || ferr != nil || price != formula || l.Label != want[i] || l.Value != i+1 {
				t.Fatalf("level %d label %q price %d, formula %d", l.Value, l.Label, price, formula)
			}
		}
	}
	for intent, years := range map[string][]int{"rfa.tender": {}, "fa.resign": {1, 2, 3, 4}, "contract.tag": {}, "contract.extend": {1, 2, 3}, "contract.restructure": {}} {
		levels, got := contractActChoices(intent)
		if !reflect.DeepEqual(got, years) || (intent != "rfa.tender" && intent != "fa.resign" && len(levels) != 0) {
			t.Fatalf("%s years %v levels %v", intent, got, levels)
		}
	}
	if _, err := contractacts.Resign(contractacts.Million, contractacts.MaxResignYears+1); err == nil {
		t.Fatal("re-sign beyond the offered years accepted")
	}
	view := contractActView(envelope.Receipt{}, envelope.ContractFacts{Intent: "fa.resign", Name: "Example"})
	if view.Act != "Re-sign him" || len(view.Levels) != 6 || len(view.Years) != 4 {
		t.Fatalf("view choices %+v", view)
	}
}
