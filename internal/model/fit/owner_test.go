package fit

import (
	"math"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/model"
)

// Owner rules land on their position only, and a shifted arc steps at age a as the fitted arc
// did at a-shift.
func TestOwnerRulesSetTheirPositionsParams(t *testing.T) {
	fitted := model.Params{Arc: [4]float64{-0.1, -0.03, -0.002, 0.01}, Recency: [2]float64{0.3, 0.1}, KNow: 3}
	results := []Result{{Position: domain.PosTE, Params: fitted}, {Position: domain.PosCB, Params: fitted},
		{Position: domain.PosWR, Params: fitted}}
	ApplyOwnerRules(results, []OwnerRule{
		{Position: domain.PosTE, ArcShift: 3},
		{Position: domain.PosCB, Recency: &[2]float64{0.6, 0.4}, KNow: 6},
	})
	te, cb, wr := results[0].Params, results[1].Params, results[2].Params
	for _, age := range []float64{24, 27, 30, 33} {
		if got, want := te.ArcStep(age, 5), fitted.ArcStep(age-3, 5); math.Abs(got-want) > 1e-12 {
			t.Errorf("TE step at %v = %v, want the fitted step at %v, %v", age, got, age-3, want)
		}
	}
	if te.Recency != fitted.Recency || te.KNow != fitted.KNow {
		t.Errorf("TE weights moved: %+v", te)
	}
	if cb.Recency != [2]float64{0.6, 0.4} || cb.KNow != 6 || cb.Arc != fitted.Arc {
		t.Errorf("CB = %+v", cb)
	}
	if wr.Arc != fitted.Arc || wr.Recency != fitted.Recency || wr.KNow != fitted.KNow {
		t.Errorf("WR has no rule but changed: %+v", wr)
	}
}
