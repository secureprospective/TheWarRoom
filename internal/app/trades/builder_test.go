package trades

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

func TestRosterFigureShowsBeforeAfterAndChange(t *testing.T) {
	t.Parallel()
	if f := RosterFigure("Active roster · 35–48", 37, 36); f.Before != "37" || f.After != "36" || f.Change != "-1" {
		t.Fatalf("one out: %+v", f)
	}
	if f := RosterFigure("Taxi squad · maximum 8", 6, 6); f.Change != "" {
		t.Fatalf("no change shows none: %+v", f)
	}
}

func TestPlayerAssetValuesAndExactCents(t *testing.T) {
	t.Parallel()
	value := PlayerAsset(state.MirrorPlayer{MFLID: "13593", Salary: 123456789, ContractYear: 2028},
		snapshot.Player{Position: domain.PosWR, Team: "DAL"}, "Name", Value{ID: "13593",
			Now: Cell{Text: "12.00 PPG", Held: true}, Dynasty: Cell{Text: "0.3 wins", Held: true}}, 2026)
	if value.Contract != "$1,234,567.89 · 3 years" || value.Now.Text != "12.00 PPG" || value.Dynasty.Text != "0.3 wins" {
		t.Fatal(value)
	}
}

func TestTradeBuilderBoundaryRejectsInvalidInputs(t *testing.T) {
	t.Parallel()
	for _, r := range []Input{{Mine: "x", Action: "read"}, {Mine: "0001", Action: "submit"},
		{Mine: "0001", Action: "revoke"}, {Mine: "0001", Other: "0001", Action: "save"},
		{Mine: "0001", Other: "0002", Action: "save", Give: []string{"99", "0099"}, Get: []string{"12345"}},
		{Mine: "0001", Other: "0002", Action: "save", Give: []string{"BB_2"}, Get: []string{"12345"}}} {
		if err := r.Validate(); err == nil {
			t.Fatalf("accepted %+v", r)
		}
	}
}
