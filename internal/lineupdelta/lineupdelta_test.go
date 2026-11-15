package lineupdelta

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/powerrankings"
)

func p(id, pos string, ppg float64) Player {
	return Player{ID: id, Position: pos, PPG: ppg, Projected: true, Plays: true}
}

// 1 QB, 1-2 RB, 1-2 WR, 3 starters... total 4: QB + RB + WR + flex.
func testRules() powerrankings.LineupRules {
	return powerrankings.LineupRules{Total: 4, Slots: []powerrankings.Slot{
		{Position: "QB", Min: 1, Max: 1}, {Position: "RB", Min: 1, Max: 2}, {Position: "WR", Min: 1, Max: 2}}}
}

func kinds(cs []Change) map[string]string {
	out := map[string]string{}
	for _, c := range cs {
		out[c.Player] = c.Kind
	}
	return out
}

func TestCompute(t *testing.T) {
	roster := []Player{p("qb", "QB", 20), p("rb1", "RB", 15), p("rb2", "RB", 9), p("wr1", "WR", 14), p("wr2", "WR", 8)}
	tests := []struct {
		name     string
		give     []string
		get      []Player
		perWeek  float64
		thisWeek float64
		want     map[string]string
	}{
		{"upgrade benches the flex", []string{"wr2"}, []Player{p("wr9", "WR", 13)}, 4, 4,
			map[string]string{"wr9": "enters", "rb2": "benched"}},
		{"selling a starter promotes the bench", []string{"rb1"}, nil, -7, -7,
			map[string]string{"rb1": "leaves", "wr2": "promoted"}},
		{"bench for bench changes nothing", []string{"wr2"}, []Player{p("wr8", "WR", 7)}, 0, 0, map[string]string{}},
		{"bye only hits this week", []string{"wr2"},
			[]Player{{ID: "wr9", Position: "WR", PPG: 13, Projected: true, Plays: false}}, 4, 0,
			map[string]string{"wr9": "enters", "rb2": "benched"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Compute("F1", roster, tc.give, tc.get, testRules())
			if got.PerWeek != tc.perWeek || got.ThisWeek != tc.thisWeek {
				t.Fatalf("perWeek %v thisWeek %v, want %v %v", got.PerWeek, got.ThisWeek, tc.perWeek, tc.thisWeek)
			}
			if k := kinds(got.Changes); len(k) != len(tc.want) {
				t.Fatalf("changes %v, want %v", k, tc.want)
			} else {
				for id, kind := range tc.want {
					if k[id] != kind {
						t.Fatalf("%s: %q, want %q (all %v)", id, k[id], kind, k)
					}
				}
			}
		})
	}
}

func TestUnprojectedNamed(t *testing.T) {
	got := Compute("F1", []Player{p("qb", "QB", 20), {ID: "rook", Position: "RB"}}, nil, nil, testRules())
	if len(got.Unprojected) != 1 || got.Unprojected[0] != "rook" {
		t.Fatalf("unprojected %v", got.Unprojected)
	}
}

func TestRules(t *testing.T) {
	known := func(lo, hi int) leaguerules.Bounds {
		return leaguerules.Bounds{Min: leaguerules.Known(lo), Max: leaguerules.Known(hi)}
	}
	lr := leaguerules.LineupRules{
		Positions: leaguerules.Known(map[leaguerules.Position]leaguerules.Bounds{"QB": known(1, 1), "LB": known(2, 4)}),
		Groups: leaguerules.Known(map[leaguerules.LineupGroup]leaguerules.Bounds{
			leaguerules.Total: known(5, 5), leaguerules.Defense: known(3, 3)}),
	}
	got, ok := Rules(lr)
	if !ok || got.Total != 5 || got.Defense != 3 || len(got.Slots) != 2 || got.Slots[0].Position != "LB" {
		t.Fatalf("rules %+v ok %v", got, ok)
	}
	delete(lr.Groups.Value, leaguerules.Defense)
	if _, ok := Rules(lr); ok {
		t.Fatal("defenders without a defense count must not read")
	}
}
