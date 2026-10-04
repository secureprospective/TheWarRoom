package fit

import (
	"math"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/model"
)

type survivalScore struct {
	Train, Test               int
	LogLossFit, LogLossBase   float64 // holdout: the fitted model vs the training base rate
	BaseRate, HoldoutObserved float64
	Kept                      bool // the fitted model beat the base rate; otherwise the base rate is stored
	// LogLossNoContract is the holdout log loss without the contract terms, at a position that
	// has them; Contract is whether they beat it and were kept.
	LogLossNoContract float64
	Contract          bool
}

// fitSurvival is a logistic regression of playing at all next season on age, draft capital,
// games played and percentile, and at a defensive position the contract he is on. It conditions
// only on what is known at the season's end. The contract terms are kept only when they beat the
// same regression without them on the holdout; the model only when it beats the base rate.
func fitSurvival(sm samples, holdout int, score *survivalScore) (trainFit, fullFit []float64) {
	m := model.Params{Position: sm.pos}
	n := len(model.SurvivalTerms(sm.pos))
	trainFit, fullFit = make([]float64, n), make([]float64, n)
	var train, test, all survivalRows
	for _, p := range sm.pairs(holdout) {
		if !finite(p.from.age) {
			continue
		}
		row := survivalRow(m, p.from)
		y := 0.0
		if p.next != nil {
			y = 1
		}
		all.add(row, y)
		if p.from.year+1 < holdout {
			train.add(row, y)
		} else {
			test.add(row, y)
		}
	}
	score.Train, score.Test = len(train.y), len(test.y)
	if len(train.y) < 30 {
		return trainFit, fullFit
	}
	terms, fitted, ok := n, []float64(nil), false
	if fitted, score.LogLossFit, ok = train.fit(test, terms); !ok {
		return trainFit, fullFit
	}
	if base := len(model.SurvivalTerms(domain.PosQB)); n > base {
		without, loss, ok := train.fit(test, base)
		score.LogLossNoContract = loss
		score.Contract = !ok || score.LogLossFit < loss
		if !score.Contract {
			terms, fitted, score.LogLossFit = base, without, loss
		}
	}
	copy(trainFit, fitted)
	score.BaseRate, score.HoldoutObserved = avg(train.y), avg(test.y)
	score.LogLossBase = logLoss(test.x, test.y, func([]float64) float64 { return score.BaseRate })
	score.Kept = score.LogLossFit < score.LogLossBase
	if !score.Kept {
		return baseRate(n, score.BaseRate), baseRate(n, avg(all.y))
	}
	if full, err := logistic(all.columns(terms), all.y, 1); err == nil {
		copy(fullFit, full)
	}
	return trainFit, fullFit
}

// survivalRows are survival inputs and whether each player played the next season.
type survivalRows struct {
	x [][]float64
	y []float64
}

func (r *survivalRows) add(x []float64, y float64) { r.x, r.y = append(r.x, x), append(r.y, y) }

// columns is every row's first n inputs.
func (r survivalRows) columns(n int) [][]float64 {
	out := make([][]float64, len(r.x))
	for i, x := range r.x {
		out[i] = x[:n]
	}
	return out
}

// fit regresses on the first n inputs and scores the fit's log loss on test.
func (r survivalRows) fit(test survivalRows, n int) (beta []float64, loss float64, ok bool) {
	beta, err := logistic(r.columns(n), r.y, 1)
	if err != nil {
		return nil, math.NaN(), false
	}
	return beta, logLoss(test.columns(n), test.y, func(x []float64) float64 { return sigmoid(dot(x, beta)) }), true
}

// baseRate is the survival fit of n terms that gives every player the same chance p.
func baseRate(n int, p float64) []float64 {
	p = min(max(p, 1e-3), 1-1e-3)
	out := make([]float64, n)
	out[0] = math.Log(p / (1 - p))
	return out
}

// survivalRow is the season's survival inputs, with the contract he was on that season.
func survivalRow(m model.Params, s season) []float64 {
	return m.SurvivalRow(s.age, s.player.DraftPick, s.games, s.pct, s.player.TenureAt(s.year))
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

// fitDebut is a logistic regression of a rookie becoming a regular (4+ games) in his first season
// on his draft slot. Like survival, it falls back to the base rate when it does not beat it on
// the holdout season's rookies.
func fitDebut(sm samples, holdout int, score *survivalScore) (trainFit, fullFit [2]float64) {
	var xTrain, xAll, xTest [][]float64
	var yTrain, yAll, yTest []float64
	for _, pl := range sm.rookies {
		row := []float64{1, math.Log(debutPick(pl.DraftPick))}
		y := 0.0
		if s, ok := sm.byYear[pl.ID][pl.Rookie]; ok && s.games >= model.RegularGames {
			y = 1
		}
		xAll, yAll = append(xAll, row), append(yAll, y)
		if pl.Rookie < holdout {
			xTrain, yTrain = append(xTrain, row), append(yTrain, y)
		} else {
			xTest, yTest = append(xTest, row), append(yTest, y)
		}
	}
	score.Train, score.Test = len(yTrain), len(yTest)
	if len(yTrain) < 30 {
		return trainFit, fullFit
	}
	score.BaseRate, score.HoldoutObserved = avg(yTrain), avg(yTest)
	train, err := logistic(xTrain, yTrain, 1)
	if err != nil {
		return logOdds(score.BaseRate), logOdds(avg(yAll))
	}
	score.LogLossFit = logLoss(xTest, yTest, func(x []float64) float64 { return sigmoid(dot(x, train)) })
	score.LogLossBase = logLoss(xTest, yTest, func([]float64) float64 { return score.BaseRate })
	score.Kept = score.LogLossFit < score.LogLossBase
	if !score.Kept {
		return logOdds(score.BaseRate), logOdds(avg(yAll))
	}
	copy(trainFit[:], train)
	if full, err := logistic(xAll, yAll, 1); err == nil {
		copy(fullFit[:], full)
	}
	return trainFit, fullFit
}

func debutPick(pick float64) float64 {
	if pick < 1 {
		return 260
	}
	return pick
}

// logOdds is the debut fit that gives every rookie the same chance p.
func logOdds(p float64) [2]float64 {
	p = min(max(p, 1e-3), 1-1e-3)
	return [2]float64{math.Log(p / (1 - p))}
}
