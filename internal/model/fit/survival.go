package fit

import (
	"math"

	"github.com/secureprospective/TheWarRoom/internal/model"
)

type survivalScore struct {
	Train, Test               int
	LogLossFit, LogLossBase   float64 // holdout: the fitted model vs the training base rate
	BaseRate, HoldoutObserved float64
}

// fitSurvival is a logistic regression of playing at all next season on age, draft capital,
// games played and percentile. It conditions only on what is known at the season's end.
func fitSurvival(sm samples, holdout int, score *survivalScore) (trainFit, fullFit [6]float64) {
	var xTrain, xAll, xTest [][]float64
	var yTrain, yAll, yTest []float64
	for _, p := range sm.pairs(holdout) {
		if !finite(p.from.age) {
			continue
		}
		row := survivalRow(p.from)
		y := 0.0
		if p.next != nil {
			y = 1
		}
		xAll, yAll = append(xAll, row), append(yAll, y)
		if p.from.year+1 < holdout {
			xTrain, yTrain = append(xTrain, row), append(yTrain, y)
		} else {
			xTest, yTest = append(xTest, row), append(yTest, y)
		}
	}
	score.Train, score.Test = len(yTrain), len(yTest)
	if len(yTrain) < 30 {
		return trainFit, fullFit
	}
	train, err := logistic(xTrain, yTrain, 1)
	if err != nil {
		return trainFit, fullFit
	}
	copy(trainFit[:], train)
	score.BaseRate, score.HoldoutObserved = avg(yTrain), avg(yTest)
	score.LogLossFit = logLoss(xTest, yTest, func(x []float64) float64 { return sigmoid(dot(x, train)) })
	score.LogLossBase = logLoss(xTest, yTest, func([]float64) float64 { return score.BaseRate })
	if full, err := logistic(xAll, yAll, 1); err == nil {
		copy(fullFit[:], full)
	}
	return trainFit, fullFit
}

func survivalRow(s season) []float64 {
	a := s.age - model.ArcCenter
	pick := s.player.DraftPick
	if pick < 1 {
		pick = 260
	}
	return []float64{1, a, a * a, math.Log(pick), s.games / 17, s.pct}
}

func logLoss(x [][]float64, y []float64, p func([]float64) float64) float64 {
	if len(y) == 0 {
		return math.NaN()
	}
	sum := 0.0
	for i := range y {
		q := min(max(p(x[i]), 1e-6), 1-1e-6)
		sum -= y[i]*math.Log(q) + (1-y[i])*math.Log(1-q)
	}
	return sum / float64(len(y))
}
