package contracts

import (
	"reflect"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine/contractacts"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

func TestEveryPlanOffersAnExit(t *testing.T) {
	t.Parallel()
	for _, state := range []envelope.State{envelope.Draft, envelope.Blocked, envelope.Ready, envelope.HandedOff, envelope.NotYetDone, envelope.NotVerified, envelope.Stale, envelope.BidPending} {
		if !DescribeAct(envelope.Receipt{State: state}, envelope.ContractFacts{}).CanWithdraw {
			t.Fatal("contract", state)
		}
		if state != envelope.BidPending {
			for _, cut := range []bool{false, true} {
				if withdrawLabel(state, cut) == "" {
					t.Fatal("release", cut, state)
				}
			}
		}
	}
}

func TestSubjectNamesTheHeldContract(t *testing.T) {
	t.Parallel()
	held := envelope.ContractFacts{Name: "Oluokun, Foyesade", Position: "LB", Status: domain.CStatusUFA, Baseline: []contractacts.Cell{{Season: 2026, Salary: 11 * contractacts.Million}, {Season: 2028, Salary: 11 * contractacts.Million}}}
	if got := Subject(held); got != "Oluokun, Foyesade · LB · $11,000,000 a year through 2028 · UFA" {
		t.Fatal(got)
	}
}

func TestContractFigureWordsPerAct(t *testing.T) {
	t.Parallel()
	for _, intent := range []string{"contract.restructure", "contract.extend", "contract.tag", "fa.resign", "rfa.tender"} {
		f := envelope.ContractFacts{Intent: intent, Source: 2026, Destination: 2028}
		want := "Price: $1,000,000"
		if intent == "contract.restructure" {
			want = "Moves: $1,000,000 from 2026 to 2028"
		}
		if got := figureSentence(f, "$1,000,000"); got != want {
			t.Fatal(intent, got)
		}
	}
}

func TestTenderPostUsesLeagueRoundAtEveryLevel(t *testing.T) {
	t.Parallel()
	for i, level := range contractacts.TenderLevels() {
		facts := envelope.ContractFacts{Intent: "rfa.tender", Level: i + 1, Season: 2026,
			Terms: contractacts.Flat(2027, 1, level.Price, "")}
		got := DescribeAct(envelope.Receipt{}, facts).Post
		want := level.Round + "-round RFA tender for 2027: " + snapshot.ContractAmountOf(level.Price).Text
		if !strings.Contains(got, want) || strings.Contains(got, "Tender level") {
			t.Fatal(got, want)
		}
	}
}

func TestContractPostOnlyNamesOwnTerms(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		intent string
		want   string
	}{
		{"contract.extend", "Adds 1 year: 2029 at $16,500,000"},
		{"contract.restructure", "Moves $16,500,000 from 2026 to 2028"},
		{"contract.tag", "Franchise tag for 2029 at $16,500,000"},
		{"rfa.tender", "5th-round RFA tender for 2029: $16,500,000"},
		{"fa.resign", "Re-sign for 1 year from 2029 at $16,500,000 a year"},
	} {
		t.Run(tc.intent, func(t *testing.T) {
			f := envelope.ContractFacts{Intent: tc.intent, Season: 2027, Years: 1, Level: 1, Source: 2026, Destination: 2028,
				Baseline: []contractacts.Cell{{Season: 2028, Salary: 1_100_000_000}},
				Terms:    contractacts.Terms{Price: 1_650_000_000, Cells: []contractacts.Cell{{Season: 2029, Salary: 1_650_000_000}}}}
			if got := postTerms(f); got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
			post := DescribeAct(envelope.Receipt{}, f).Post
			if !strings.Contains(post, tc.want) || strings.Contains(post, "term/additional years") {
				t.Fatal(post)
			}
			if tc.intent != "rfa.tender" && strings.Contains(post, "Tender level") {
				t.Fatal(post)
			}
		})
	}
}

func TestContractActChoicesComeFromTheRuleTable(t *testing.T) {
	t.Parallel()
	want := []string{"5th · $0.60M", "4th · $1.00M", "3rd · $1.50M", "2nd · $2.50M", "1st · $5.00M", "1st + 2nd · $6.50M"}
	for _, intent := range []string{"rfa.tender", "fa.resign"} {
		levels, _ := Choices(intent)
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
		levels, got := Choices(intent)
		if !reflect.DeepEqual(got, years) || (intent != "rfa.tender" && intent != "fa.resign" && len(levels) != 0) {
			t.Fatalf("%s years %v levels %v", intent, got, levels)
		}
	}
	if _, err := contractacts.Resign(contractacts.Million, contractacts.MaxResignYears+1); err == nil {
		t.Fatal("re-sign beyond the offered years accepted")
	}
	view := DescribeAct(envelope.Receipt{}, envelope.ContractFacts{Intent: "fa.resign", Name: "Example"})
	if view.Act != "Re-sign him" || len(view.Levels) != 6 || len(view.Years) != 4 {
		t.Fatalf("view choices %+v", view)
	}
}

func TestDescribeActCarriesTheTeamFiguresAndAnUnknownAct(t *testing.T) {
	t.Parallel()
	v := DescribeAct(envelope.Receipt{}, envelope.ContractFacts{Intent: "rfa.tender", TeamFigures: []envelope.ContractFigure{{Label: "Room", Before: "$1", After: "$2", Change: "+$1"}}})
	if len(v.TeamFigures) != 1 || v.TeamFigures[0] != (WhatIfFigure{"Room", "$1", "$2", "+$1"}) || v.Price != "Price: Not priced" || len(v.Figures) != 0 ||
		!strings.Contains(v.Post, "RFA tender: level not selected") {
		t.Fatalf("view %+v", v)
	}
	if got := postTerms(envelope.ContractFacts{Intent: "contract.void"}); got != "" {
		t.Fatal(got)
	}
}
