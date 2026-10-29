package contracts

import (
	"context"
	"errors"
	"strings"
	"testing"

	mflwhatif "github.com/secureprospective/TheWarRoom/internal/mflref/whatif"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
	"github.com/secureprospective/TheWarRoom/internal/transactions"
)

func TestRuleNamesARestructuredContract(t *testing.T) {
	t.Parallel()
	if !strings.Contains(Rule("WAIVER", state.PlayerState{IsRestructured: true, ExpirationYear: 2028}, 2026), "50%") {
		t.Fatal("restructured label")
	}
}

func TestChangeIsSignedAndEmptyAtZero(t *testing.T) {
	t.Parallel()
	for _, delta := range []struct {
		cents domain.Money
		want  string
	}{{0, ""}, {-3_000_000, "−$30,000"}, {3_000_000, "+$30,000"}} {
		if got := Change(delta.cents); got != delta.want {
			t.Fatalf("change %d: %s", delta.cents, got)
		}
	}
}

func TestRosterMissingPointsSayNoScore(t *testing.T) {
	t.Parallel()
	if got := scoreText(map[string]string{}, "0042"); got != "No score" {
		t.Fatalf("missing score = %q", got)
	}
	if got := scoreText(map[string]string{"0042": "0.00"}, "0042"); got != "0.00" {
		t.Fatalf("real zero score = %q", got)
	}
}

// ledger is a replayed what-if ledger: cap used per franchise and each player's paid cells.
type ledger struct {
	used  domain.Money
	cells map[string]map[int]domain.Money
	err   error
}

func (l ledger) CapUsed(string) (domain.Money, bool) { return l.used, true }

func (l ledger) LedgerCells(_ context.Context, id string) (map[int]domain.Money, error) {
	return l.cells[id], l.err
}

func TestWhatIfValidateRefusesEachMalformedRequest(t *testing.T) {
	t.Parallel()
	for want, r := range map[string]WhatIf{
		"what-if scope":         {Mine: "bad", Action: "read"},
		"what-if: choose your":  {Action: "read"},
		`unknown action "save"`: {Mine: "0001", Action: "save"},
		"what-if player":        {Mine: "0001", Action: "buyout", Player: "x"},
		"what-if: missing move": {Mine: "0001", Action: "undo"},
	} {
		if err := r.Validate(); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%+v: %v, want %q", r, err, want)
		}
	}
	for _, r := range []WhatIf{{Mine: "0001", Action: "reset"}, {Mine: "0001", Action: "cut", Player: "0042"}, {Mine: "0001", Action: "undo", Move: "1"}} {
		if err := r.Validate(); err != nil {
			t.Errorf("%+v refused: %v", r, err)
		}
	}
}

func TestDescribeWhatIfFiguresTheModelledMoves(t *testing.T) {
	t.Parallel()
	capAmount := domain.Money(120 * 100_000_000)
	s := mflwhatif.WhatIfSession{Mine: "0001", Snapshot: heldMirror(), Used: 30 * 100_000_000, SeedUsed: 30 * 100_000_000, Cap: &capAmount,
		Moves: []mflwhatif.WhatIfMove{{ID: "m1", Kind: string(transactions.KindBuyout), Player: "0042", Name: "Example", Charge: 6 * 100_000_000}}}
	l := ledger{used: 26 * 100_000_000, cells: map[string]map[int]domain.Money{"0046": {2026: 10 * 100_000_000}}}
	labelled := false
	v, err := DescribeWhatIf(context.Background(), s, l, true, 2029, func(moves []WhatIfMove) error {
		labelled = len(moves) == 1
		moves[0].Plan = "Plan: Ready"
		return nil
	})
	if err != nil || !labelled || !v.Stale || v.Banner == "" || v.Moves[0].Plan != "Plan: Ready" || v.Moves[0].Tag != "Bought out in what-if" {
		t.Fatalf("view %+v %v", v, err)
	}
	if len(v.Choices) != 1 || v.Choices[0].Player != "0046" || v.Choices[0].BuyoutAllowed {
		t.Fatalf("choices %+v", v.Choices)
	}
	// This season's cap, used, room and modelled dead cap, then 2027 through the 2029 horizon.
	if len(v.Figures) != 7 || v.Figures[1].After != "$26,000,000" || v.Figures[2].Change != "+$4,000,000" || v.Figures[3].After != "$6,000,000" ||
		v.Figures[4].Before != "$10,000,000" || v.Figures[4].After != "$0" || v.Figures[6].Label != "2029 commitments at today's salaries" {
		t.Fatalf("figures %+v", v.Figures)
	}
	open, err := DescribeWhatIf(context.Background(), mflwhatif.WhatIfSession{Mine: "0001", Snapshot: heldMirror()}, l, false, 0, func([]WhatIfMove) error { return nil })
	if err != nil || open.Banner != "" || open.Figures[0].Before != "Unavailable" || open.Figures[2].Change != "" || len(open.Choices) != 2 {
		t.Fatalf("no cap %+v %v", open, err)
	}
	if _, err := DescribeWhatIf(context.Background(), s, nil, false, 0, nil); err == nil || err.Error() != "what-if: current ledger unavailable" {
		t.Fatal(err)
	}
	refused := errors.New("plans unreadable")
	if v, err := DescribeWhatIf(context.Background(), s, l, false, 0, func([]WhatIfMove) error { return refused }); !errors.Is(err, refused) || len(v.Moves) != 1 || len(v.Figures) != 0 {
		t.Fatalf("label failure %+v %v", v, err)
	}
	if _, err := DescribeWhatIf(context.Background(), mflwhatif.WhatIfSession{Mine: "0001", Snapshot: heldMirror()}, ledger{err: refused}, false, 0, func([]WhatIfMove) error { return nil }); !errors.Is(err, refused) || !strings.HasPrefix(err.Error(), "what-if future cells") {
		t.Fatal(err)
	}
}

func TestCommittedKeepsTheHeldSalarysCents(t *testing.T) {
	t.Parallel()
	into := map[int]domain.Money{}
	l := ledger{cells: map[string]map[int]domain.Money{"0042": {2026: 100, 2027: 1_000_000, 2028: 2_000_000}}}
	if err := Committed(context.Background(), l, state.MirrorPlayer{MFLID: "0042", Salary: 1_000_123}, 2026, into); err != nil {
		t.Fatal(err)
	}
	if len(into) != 2 || into[2027] != 1_000_123 || into[2028] != 2_000_123 {
		t.Fatal(into)
	}
}

func TestRuleAndChoiceFollowTheContract(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		kind string
		p    state.PlayerState
		want string
	}{
		{string(transactions.KindBuyout), state.PlayerState{}, "Offseason only"},
		{"WAIVER", state.PlayerState{ExpirationYear: 2026}, "Final paid year"},
		{"WAIVER", state.PlayerState{ExpirationYear: 2028}, "35% assumed"},
	} {
		if got := Rule(tc.kind, tc.p, 2026); !strings.HasPrefix(got, tc.want) {
			t.Errorf("%s %+v: %s", tc.kind, tc.p, got)
		}
	}
	for years, allowed := range map[int]bool{1: false, 2: true, 4: true, 5: false} {
		c := Choice(state.MirrorPlayer{MFLID: "0042", ContractYear: 2026 + years}, 2026)
		if c.BuyoutAllowed != allowed || (allowed != strings.HasPrefix(c.Reason, "Offseason only")) || c.CutLabel != "What if I cut him" {
			t.Errorf("%d years: %+v", years, c)
		}
	}
	moves, removed, charge := Moves(mflwhatif.WhatIfSession{Moves: []mflwhatif.WhatIfMove{{ID: "a", Kind: "WAIVER", Player: "0042", Name: "Example", Charge: 7}, {ID: "b", Kind: string(transactions.KindBuyout), Player: "0043", Charge: 3}}})
	if len(moves) != 2 || moves[0].Label != "Cut Example" || moves[1].Label != "Buy out " || !removed["0043"] || charge != 10 ||
		moves[0].Status != "Modelled: cut Example, $0.07; see What-if at the top" {
		t.Fatalf("moves %+v %v %d", moves, removed, charge)
	}
	if f := Figure("Used", 100_000_000, 50_000_000); f.Before != "$1,000,000" || f.After != "$500,000" || f.Change != "−$500,000" {
		t.Fatal(f)
	}
}
