package fit

import (
	"math"

	"github.com/secureprospective/TheWarRoom/internal/domain"

	"github.com/secureprospective/TheWarRoom/internal/model"
)

// priorMaxExperience is how far into a career a season still tests the prior: the first three.
const priorMaxExperience = 3

// fitPrior is the joint regression of an early-career season's percentile on every pre-NFL fact
// at once, so correlated facts are not counted twice. It is weighted by games, ridge-penalized
// with the penalty chosen by cross-validation, and scored on the holdout season.
func fitPrior(sm samples, holdout int, rep *Report) (train, full model.Params) {
	var trainRows, testRows, allRows []season
	for _, s := range sm.seasons {
		if s.games < model.RegularGames || s.exp < 1 || s.exp > priorMaxExperience {
			continue
		}
		allRows = append(allRows, s)
		if s.year < holdout {
			trainRows = append(trainRows, s)
		} else {
			testRows = append(testRows, s)
		}
	}
	rep.PriorN, rep.PriorTestN = len(trainRows), len(testRows)
	if len(trainRows) < 20 {
		return emptyPrior(sm.pos), emptyPrior(sm.pos)
	}
	lambda := chooseLambda(sm.pos, trainRows)
	train = regressPrior(sm.pos, trainRows, lambda)
	rep.PriorR2 = r2(train, trainRows)
	rep.PriorR2Test = r2(train, testRows)
	return train, regressPrior(sm.pos, allRows, lambda)
}

func emptyPrior(pos domain.Position) model.Params {
	n := len(model.PriorFeatures(pos))
	return model.Params{Position: pos, Intercept: 0.5, Weight: make([]float64, n), Missing: make([]float64, n)}
}

// regressPrior fits on standardized inputs with a missing indicator per input, then folds the
// standardization into raw-unit weights so the stored prior needs no means.
func regressPrior(pos domain.Position, rows []season, lambda float64) model.Params {
	n := len(model.PriorFeatures(pos))
	mu, sd := standardize(rows, n)
	x := make([][]float64, len(rows))
	y := make([]float64, len(rows))
	w := make([]float64, len(rows))
	for i, s := range rows {
		x[i], y[i], w[i] = priorRow(s.player, mu, sd), s.pct, s.games
	}
	beta, err := ridge(x, y, w, lambda)
	if err != nil {
		return emptyPrior(pos)
	}
	p := emptyPrior(pos)
	p.Intercept = beta[0]
	for j := range n {
		if sd[j] == 0 {
			continue
		}
		p.Weight[j] = beta[1+j] / sd[j]
		p.Missing[j] = beta[1+n+j] + p.Weight[j]*mu[j]
		p.Intercept -= p.Weight[j] * mu[j]
	}
	return p
}

func standardize(rows []season, n int) (mu, sd []float64) {
	mu, sd = make([]float64, n), make([]float64, n)
	count := make([]float64, n)
	for _, s := range rows {
		x, known := s.player.PriorInputs()
		for j := range n {
			if known[j] {
				mu[j] += x[j]
				sd[j] += x[j] * x[j]
				count[j]++
			}
		}
	}
	for j := range n {
		if count[j] < 2 {
			mu[j], sd[j] = 0, 0
			continue
		}
		mu[j] /= count[j]
		sd[j] = math.Sqrt(math.Max(sd[j]/count[j]-mu[j]*mu[j], 0))
	}
	return mu, sd
}

func priorRow(p *model.Player, mu, sd []float64) []float64 {
	x, known := p.PriorInputs()
	n := len(x)
	row := make([]float64, 1+2*n)
	row[0] = 1
	for j := range n {
		switch {
		case known[j] && sd[j] > 0:
			row[1+j] = (x[j] - mu[j]) / sd[j]
		case !known[j]:
			row[1+n+j] = 1
		}
	}
	return row
}

// chooseLambda picks the ridge penalty with the lowest error over three folds of the training rows.
func chooseLambda(pos domain.Position, rows []season) float64 {
	best, bestErr := 10.0, math.Inf(1)
	for _, lambda := range []float64{0.3, 1, 3, 10, 30, 100} {
		var sse float64
		for fold := range 3 {
			var fitRows, valRows []season
			for i, s := range rows {
				if i%3 == fold {
					valRows = append(valRows, s)
				} else {
					fitRows = append(fitRows, s)
				}
			}
			p := regressPrior(pos, fitRows, lambda)
			for _, s := range valRows {
				d := p.Prior(s.player) - s.pct
				sse += s.games * d * d
			}
		}
		if sse < bestErr {
			best, bestErr = lambda, sse
		}
	}
	return best
}

// r2 is the games-weighted share of percentile variance the prior explains in rows.
func r2(p model.Params, rows []season) float64 {
	if len(rows) == 0 {
		return math.NaN()
	}
	var sw, swy float64
	for _, s := range rows {
		sw += s.games
		swy += s.games * s.pct
	}
	mean := swy / sw
	var sse, sst float64
	for _, s := range rows {
		d := p.Prior(s.player) - s.pct
		sse += s.games * d * d
		sst += s.games * (s.pct - mean) * (s.pct - mean)
	}
	return 1 - sse/sst
}
