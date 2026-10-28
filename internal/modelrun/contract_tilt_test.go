package modelrun

import (
	"fmt"
	"math"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/model"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// Nine equal players paid 10 each set a going rate of 10. A tenth paid 5 on a three-year deal is
// worth 5% more (0.10 × half under the rate), the same deal for one year a third of that, and
// one paid 40 for three years falls by the cap, not by 30%. A free agent is left as valued.
func TestContractTiltMovesValueByPriceAndLength(t *testing.T) {
	var scores []history.ModelScore
	var ids []string
	roster := map[string]state.PlayerState{}
	add := func(id string, salary domain.Money, through int) {
		scores = append(scores, history.ModelScore{MFLID: id, Position: "WR", Value: model.Value{DynastyPPG: 10}})
		ids = append(ids, id)
		if salary > 0 {
			roster[id] = state.PlayerState{MFLID: id, Salary: salary, ExpirationYear: through}
		}
	}
	for i := range 9 {
		add(fmt.Sprint(i), 10, 2028)
	}
	add("cheap3", 5, 2028)
	add("cheap1", 5, 2026)
	add("dear3", 40, 2028)
	add("free", 0, 0)
	tilts := contractTilt(scores, ids, roster, 2026, 0.06, func(domain.Position) func(float64) float64 { return nil })
	want := map[string]float64{"cheap3": 0.05, "cheap1": 0.05 / 3, "dear3": -0.06, "0": 0}
	for id, w := range want {
		if math.Abs(tilts[id]-w) > 1e-12 {
			t.Errorf("%s tilt %v, want %v", id, tilts[id], w)
		}
	}
	for _, sc := range scores {
		if w, ok := want[sc.MFLID]; ok && math.Abs(sc.DynastyPPG-10*(1+w)) > 1e-9 {
			t.Errorf("%s value %v, want %v", sc.MFLID, sc.DynastyPPG, 10*(1+w))
		}
	}
	if _, ok := tilts["free"]; ok || scores[len(scores)-1].DynastyPPG != 10 {
		t.Error("a free agent was tilted")
	}
	if contractTilt(scores, ids, roster, 2026, 0, nil) != nil {
		t.Error("a zero cap must turn the tilt off")
	}
}
