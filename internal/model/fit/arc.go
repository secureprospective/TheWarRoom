package fit

import (
	"github.com/secureprospective/TheWarRoom/internal/model"
)

// maxSurvivalWeight caps a survivor's weight in the arc fit, so one unlikely survivor cannot
// dominate it.
const maxSurvivalWeight = 5.0

type arcScore struct {
	Exits             int     // players who played no game the next season
	RMSEArc, RMSEFlat float64 // holdout: the dynasty prediction with the arc vs without it
}

func arcRow(age float64, exp int) []float64 {
	a := age - model.ArcCenter
	second := 0.0
	if exp == 1 {
		second = 1
	}
	return []float64{1, a, a * a, second}
}

func solveArc(x [][]float64, y, w []float64) [4]float64 {
	var out [4]float64
	beta, err := ridge(x, y, w, 1e-3)
	if err != nil {
		return out
	}
	copy(out[:], beta)
	return out
}
