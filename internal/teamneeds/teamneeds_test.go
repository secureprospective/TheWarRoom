package teamneeds

import (
	"fmt"
	"testing"
)

func roster(prefix string, shape map[string][]float64) []Player {
	var out []Player
	for pos, vals := range shape {
		for i, v := range vals {
			out = append(out, Player{ID: fmt.Sprintf("%s-%s%d", prefix, pos, i), Position: pos, Value: v, Valued: true})
		}
	}
	return out
}

func TestNeedsAndSurplusAgainstTheLeagueTemplate(t *testing.T) {
	typical := map[string][]float64{"QB": {9}, "WR": {8, 7, 6}, "RB": {5, 4}}
	rosters := map[string][]Player{
		"a": roster("a", typical),
		"b": roster("b", typical),
		// c starts no WR in its best 6: three RBs and two QBs crowd them out, one WR left over.
		"c": roster("c", map[string][]float64{"QB": {9, 8.5}, "RB": {8, 7, 6, 5}, "WR": {1}}),
	}
	rosters["c"] = append(rosters["c"], Player{ID: "c-x", Position: "TE"})
	lg := Compute(rosters, 6)
	if lg.Template["WR"] != 3 || lg.Template["RB"] != 2 || lg.Template["QB"] != 1 {
		t.Fatalf("template = %v", lg.Template)
	}
	c := lg.Teams["c"]
	if len(c.Needs) != 1 || c.Needs[0] != (Need{Position: "WR", Short: 3, Grade: Critical}) {
		t.Fatalf("c needs = %+v", c.Needs)
	}
	if c.SurplusAt("WR") != 1 || len(c.Unvalued) != 1 || c.Unvalued[0] != "c-x" {
		t.Fatalf("c surplus %+v unvalued %v", c.Surplus, c.Unvalued)
	}
	if len(lg.Teams["a"].Needs) != 0 || len(lg.Teams["a"].Surplus) != 0 {
		t.Fatalf("a = %+v", lg.Teams["a"])
	}
}

func TestSplitLeagueManufacturesNoNeed(t *testing.T) {
	rosters := map[string][]Player{
		"a": roster("a", map[string][]float64{"TE": {5}, "WR": {4}}),
		"b": roster("b", map[string][]float64{"WR": {5, 4}}),
	}
	lg := Compute(rosters, 2)
	if _, ok := lg.Template["TE"]; ok {
		t.Fatalf("lower median of {0,1} TEs must be 0: %v", lg.Template)
	}
	if len(lg.Teams["b"].Needs) != 0 {
		t.Fatalf("b needs = %+v", lg.Teams["b"].Needs)
	}
	if len(lg.Teams["a"].Needs) != 0 {
		t.Fatalf("a needs = %+v (template WR is lower median 1)", lg.Teams["a"].Needs)
	}
}

func TestZeroValueDepthIsNotSurplus(t *testing.T) {
	lg := Compute(map[string][]Player{"a": roster("a", map[string][]float64{"WR": {3, 0}})}, 1)
	if len(lg.Teams["a"].Surplus) != 0 {
		t.Fatalf("surplus = %+v", lg.Teams["a"].Surplus)
	}
}
