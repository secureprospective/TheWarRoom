package main

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

func whatIfTestApp(t *testing.T) *App {
	t.Helper()
	a := targetTestApp(t)
	p, err := db.Open(context.Background(), filepath.Join(t.TempDir(), "whatif.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := p.Close(); err != nil {
			t.Error(err)
		}
	})
	a.whatifPools = p
	replaceWhatIfMirror(t, a, 10*100_000_000)
	return a
}
func replaceWhatIfMirror(t *testing.T, a *App, salary domain.Money) {
	t.Helper()
	players := []state.MirrorPlayer{}
	for i, id := range []string{"0042", "0043", "0044", "0045", "0046"} {
		expiration := 2028
		if i == 3 {
			expiration = 2027
		}
		if i == 4 {
			expiration = 2026
		}
		players = append(players, state.MirrorPlayer{MFLID: id, FranchiseID: "0001", Salary: salary, ContractYear: expiration, RosterStatus: domain.RosterActive})
	}
	if _, err := a.league.Replace(a.ctx, state.MirrorSnapshot{Season: 2026, Players: players, Adjustments: []state.Adjustment{{ID: "a", FranchiseID: "0001", Amount: -12345}}}); err != nil {
		t.Fatal(err)
	}
}
func whatIfCall(t *testing.T, a *App, action, player, move string) Reading[WhatIfView] {
	t.Helper()
	r := a.TargetWhatIf(RawWhatIfRequest{Mine: "0001", Action: action, Player: player, Move: move})
	if r.Status != ReadingOK {
		t.Fatalf("%s: %+v", action, r)
	}
	return r
}
func TestWhatIfComposeUndoResetRestartAndMirrorIsolation(t *testing.T) {
	a := whatIfTestApp(t)
	before, stamp, _ := a.league.ContractsView()
	empty := whatIfCall(t, a, "read", "", "")
	if len(empty.Value.Moves) != 0 || len(empty.Value.Choices) != 5 || empty.Baseline == "" || empty.HowCounted == "" {
		t.Fatalf("empty %+v", empty)
	}
	cut := whatIfCall(t, a, "cut", "0042", "")
	if len(cut.Value.Moves) != 1 || cut.Value.Moves[0].Charge.Cents != 700_000_000 {
		t.Fatalf("cut %+v", cut)
	}
	bought := whatIfCall(t, a, "buyout", "0043", "")
	if len(bought.Value.Moves) != 2 || bought.Value.Moves[1].Charge.Cents != 600_000_000 || !strings.Contains(bought.Value.Moves[1].Explanation, "Offseason only") {
		t.Fatalf("buyout %+v", bought)
	}
	if bought.Value.Figures[1].After != "$42,999,876.55" {
		t.Fatalf("used %+v", bought.Value.Figures)
	}
	// New composition root over the same pools proves no in-memory move dependency.
	restart := NewApp()
	restart.ctx = a.ctx
	restart.whatifPools = a.whatifPools
	restart.league = a.league
	restart.rulebook = a.rulebook
	close(restart.started)
	saved := whatIfCall(t, restart, "read", "", "")
	if len(saved.Value.Moves) != 2 || saved.Value.Moves[0].ID != bought.Value.Moves[0].ID {
		t.Fatal("restart lost moves")
	}
	undone := whatIfCall(t, restart, "undo", "", saved.Value.Moves[0].ID)
	if len(undone.Value.Moves) != 1 || undone.Value.Moves[0].Charge.Cents != 600_000_000 {
		t.Fatalf("undo %+v", undone)
	}
	reset := whatIfCall(t, restart, "reset", "", "")
	if len(reset.Value.Moves) != 0 || reset.Value.Figures[1].Before != reset.Value.Figures[1].After {
		t.Fatalf("reset %+v", reset)
	}
	after, afterStamp, _ := a.league.ContractsView()
	if stamp != afterStamp || len(before.Players) != len(after.Players) || before.Players[0].Salary != after.Players[0].Salary {
		t.Fatal("simulation wrote mirror")
	}
}
func TestWhatIfStaleBlocksAllChangesUntilReset(t *testing.T) {
	a := whatIfTestApp(t)
	start := whatIfCall(t, a, "cut", "0042", "")
	replaceWhatIfMirror(t, a, 11*100_000_000)
	for _, action := range []string{"read", "buyout", "undo"} {
		r := whatIfCall(t, a, action, "0043", start.Value.Moves[0].ID)
		if !r.Value.Stale || r.Value.Banner != "MFL changed since this what-if started" || len(r.Value.Moves) != 1 || r.Value.Figures[1].Before != start.Value.Figures[1].Before {
			t.Fatalf("stale %s %+v", action, r)
		}
	}
	r := whatIfCall(t, a, "reset", "", "")
	if r.Value.Stale || len(r.Value.Moves) != 0 || r.Value.Figures[1].Before == start.Value.Figures[1].Before {
		t.Fatalf("reset %+v", r)
	}
}
func TestWhatIfRulesFailuresAndValidation(t *testing.T) {
	a := whatIfTestApp(t)
	start := whatIfCall(t, a, "read", "", "")
	if start.Value.Choices[3].BuyoutAllowed || start.Value.Choices[4].BuyoutAllowed {
		t.Fatal("one/final year offered")
	}
	rejected := whatIfCall(t, a, "buyout", "0045", "")
	if rejected.Value.Notice == "" || len(rejected.Value.Moves) != 0 {
		t.Fatal("one-year bought out")
	}
	final := whatIfCall(t, a, "cut", "0046", "")
	if final.Value.Moves[0].Charge.Cents != 0 || !strings.Contains(final.Value.Moves[0].Explanation, "Final paid year") {
		t.Fatalf("final %+v", final)
	}
	first := whatIfCall(t, a, "buyout", "0042", "")
	if !strings.Contains(first.HowCounted, "35%") || !strings.Contains(first.HowCounted, "inferred") {
		t.Fatal("inference absent")
	}
	whatIfCall(t, a, "buyout", "0043", "")
	third := whatIfCall(t, a, "buyout", "0044", "")
	if len(third.Value.Moves) != 3 || !strings.Contains(third.Value.Notice, "2 buyouts") {
		t.Fatalf("third %+v", third)
	}
	invalid := a.TargetWhatIf(RawWhatIfRequest{Mine: "0001", Action: "cut", Player: "bad"})
	if invalid.Status != ReadingFailed {
		t.Fatal("invalid id accepted")
	}
	if a.TargetWhatIf(RawWhatIfRequest{Mine: "0001", Action: "unknown"}).Status != ReadingFailed {
		t.Fatal("invalid action accepted")
	}
	if !strings.Contains(whatIfRule("WAIVER", state.PlayerState{IsRestructured: true, ExpirationYear: 2028}, 2026), "50%") {
		t.Fatal("restructured label")
	}
}

func TestWhatIfPresentation(t *testing.T) {
	a := whatIfTestApp(t)
	empty := whatIfCall(t, a, "read", "", "")
	if empty.Baseline == "" || strings.Contains(empty.Baseline, "revision") || !strings.Contains(empty.HowCounted, "Separate contract-cell update time: not held by this mirror") || !strings.Contains(empty.HowCounted, "whole modelled charge lands in the cut year") {
		t.Fatalf("baseline/disclosure %+v", empty)
	}
	if _, err := time.Parse(time.RFC3339Nano, empty.Baseline); err != nil {
		t.Fatal(err)
	}
	for _, choice := range empty.Value.Choices {
		if choice.CutLabel == "" || choice.BuyoutLabel == "" || choice.Reason == "" {
			t.Fatalf("menu %+v", choice)
		}
	}
	cut := whatIfCall(t, a, "cut", "0042", "")
	move := cut.Value.Moves[0]
	if move.Player != "0042" || move.Tag != "Cut in what-if" || move.Status != "Modelled: cut Player 0042, $7,000,000; see What-if at the top" {
		t.Fatalf("cut row %+v", move)
	}
	bought := whatIfCall(t, a, "buyout", "0043", "")
	if bought.Value.Moves[1].Tag != "Bought out in what-if" || !strings.Contains(bought.Value.Moves[1].Status, "Modelled: buyout") {
		t.Fatalf("buyout row %+v", bought.Value.Moves[1])
	}
	if len(bought.Value.Figures) != 6 {
		t.Fatalf("result rows %+v", bought.Value.Figures)
	}
	for i, figure := range bought.Value.Figures {
		if i > 3 && (!strings.Contains(figure.Label, "gross commitments") || strings.Contains(figure.Label, "dead cap")) {
			t.Fatalf("future row %+v", figure)
		}
	}
	if bought.Value.Figures[0].Change != "" || bought.Value.Figures[1].Change != "−$7,000,000" || bought.Value.Figures[3].Change != "+$13,000,000" || bought.Value.Figures[4].Change != "−$20,000,000" {
		t.Fatalf("changes %+v", bought.Value.Figures)
	}
	for _, delta := range []struct {
		cents domain.Money
		want  string
	}{{0, ""}, {-3_000_000, "−$30,000"}, {3_000_000, "+$30,000"}} {
		if got := whatIfChange(delta.cents); got != delta.want {
			t.Fatalf("change %d: %s", delta.cents, got)
		}
	}
}
