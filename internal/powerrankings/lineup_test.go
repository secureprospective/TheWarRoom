package powerrankings

import (
	"math"
	"testing"
)

func TestLineupFollowsTheLeaguesRules(t *testing.T) {
	rules := LineupRules{Total: 4, Defense: 2, Slots: []Slot{
		{Position: "QB", Min: 1, Max: 1}, {Position: "WR", Min: 1, Max: 2},
		{Position: "LB", Min: 1, Max: 2}, {Position: "CB", Min: 0, Max: 1},
	}}
	cands := []Candidate{
		{Position: "QB", Value: 20}, {Position: "QB", Value: 19}, // the second QB cannot start
		{Position: "WR", Value: 5}, {Position: "WR", Value: 4}, {Position: "WR", Value: 3},
		{Position: "LB", Value: 9}, {Position: "LB", Value: 8}, {Position: "LB", Value: 7},
		{Position: "CB", Value: 1}, {Position: "K", Value: 99}, // K is not in the rules
	}
	got, started := Lineup(cands, rules)
	// The two offensive slots are the QB and WR minimums (QB 20, WR 5); the two defensive ones the
	// best LBs (9, 8), since CB has no minimum and LB 7 and CB 1 are worse.
	if got != 20+5+9+8 {
		t.Fatalf("lineup value %v, want 42 (started %v)", got, started)
	}
	if len(started) != 4 {
		t.Fatalf("started %d players, want the 4 the rules allow", len(started))
	}
}

func TestLineupFillsMinimumsBeforeTheFlex(t *testing.T) {
	rules := LineupRules{Total: 3, Defense: 3, Slots: []Slot{
		{Position: "DT", Min: 1, Max: 3}, {Position: "S", Min: 1, Max: 3},
	}}
	cands := []Candidate{{Position: "DT", Value: 10}, {Position: "DT", Value: 9}, {Position: "DT", Value: 8},
		{Position: "S", Value: 1}}
	if got, _ := Lineup(cands, rules); got != 10+9+1 {
		t.Fatalf("lineup %v, want 20: a safety must start before the third DT", got)
	}
}

func TestAutoRosterWeightShrinksWithWeeks(t *testing.T) {
	for _, c := range []struct {
		weeks int
		want  float64
	}{{0, 1}, {1, 0.8}, {4, 0.5}, {-2, 1}} {
		if got := AutoRosterWeight(c.weeks); math.Abs(got-c.want) > 1e-12 {
			t.Fatalf("weeks %d: %v, want %v", c.weeks, got, c.want)
		}
	}
}

func TestExpectedWinsSplitsEachGame(t *testing.T) {
	exp := map[string]float64{"0001": 250, "0002": 200, "0003": 200}
	got := ExpectedWins([]Game{{Home: "0001", Away: "0002"}, {Home: "0002", Away: "0003"},
		{Home: "0001", Away: "0099"}}, exp)
	if math.Abs(got["0001"]+got["0002"]+got["0003"]-2) > 1e-12 {
		t.Fatalf("two games must hand out two wins, got %v", got)
	}
	if p := got["0001"]; p < 0.8 || p > 0.85 {
		t.Fatalf("a 50-point favourite wins %.3f, want about 0.84", p)
	}
	if math.Abs(got["0003"]-0.5) > 1e-12 {
		t.Fatalf("evenly matched teams split, got %v", got["0003"])
	}
	if got := ExpectedPoints(210, 600, 3); math.Abs(got-(4.0/7*210+3.0/7*200)) > 1e-9 {
		t.Fatalf("expected points %v", got)
	}
}
