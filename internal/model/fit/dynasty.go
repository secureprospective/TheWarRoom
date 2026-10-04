package fit

import (
	"math"

	"github.com/secureprospective/TheWarRoom/internal/model"
)

// fitDynasty fits the dynasty k and the talent arc together from season pairs. A player's next
// percentile is predicted as Z·(this season) + (1−Z)·(his prior), plus one step along the arc.
// Shrinking toward his own prior makes k a measure against the prior; fitting the arc on what
// the blend leaves keeps regression to the mean out of it. Survivors are weighted by the inverse
// of their chance of playing on, so the arc is not fitted to the players who aged well; leaving
// the league is the survival arc's, not counted again here. Both shapes of Z are fitted on
// training pairs; the one that predicts the holdout season better is kept, and the arc is kept
// only if it beats the same blend refitted without one. trainSurv and fullSurv are the survival
// fits on the same seasons.
func fitDynasty(sm samples, holdout int, prior model.Params, trainSurv, fullSurv []float64,
	rep *Report) (train, full model.Params) {
	var trainRows, testRows, allRows []pair
	for _, p := range sm.pairs(holdout) {
		if !finite(p.from.age) || p.from.games < 1 {
			continue
		}
		if p.next == nil {
			rep.Arc.Exits++
			continue
		}
		if p.next.games < model.RegularGames {
			continue
		}
		allRows = append(allRows, p)
		if p.from.year+1 < holdout {
			trainRows = append(trainRows, p)
		} else {
			testRows = append(testRows, p)
		}
	}
	rep.PairsTrain, rep.PairsTest = len(trainRows), len(testRows)
	if rep.PairsTrain < 20 {
		return prior, prior
	}
	trainPrior, fullPrior := prior, prior
	trainPrior.Survival, fullPrior.Survival = trainSurv, fullSurv
	credibility, exponential := bestK(trainPrior, false, true, trainRows), bestK(trainPrior, true, true, trainRows)
	rep.DynCredibility, rep.DynExponential = pairError(credibility, testRows), pairError(exponential, testRows)
	train = credibility
	if rep.DynExponential < rep.DynCredibility {
		train = exponential
	}
	flat := bestK(trainPrior, train.Exponential, false, trainRows)
	rep.Arc.RMSEArc, rep.Arc.RMSEFlat = pairError(train, testRows), pairError(flat, testRows)
	rep.Arc.Kept = rep.Arc.RMSEArc < rep.Arc.RMSEFlat
	if !rep.Arc.Kept {
		train = flat
	}
	rep.DynLastSeason, rep.DynPrior = baselines(prior, testRows)
	return train, bestK(fullPrior, train.Exponential, rep.Arc.Kept, allRows)
}

// bestK searches k on a grid; for each k the arc, when there is one, is the least-squares fit of
// what the blend leaves, and k is scored on the observed pairs.
func bestK(m model.Params, exponential, arc bool, rows []pair) model.Params {
	m.Exponential = exponential
	best, bestErr := m, math.Inf(1)
	for _, k := range grid(0.3, 300, 60) {
		m.KDynasty = k
		if arc {
			m.Arc = fitArcGiven(m, rows)
		}
		if e := pairError(m, rows); e < bestErr {
			best, bestErr = m, e
		}
	}
	return best
}

func fitArcGiven(m model.Params, rows []pair) [4]float64 {
	x := make([][]float64, len(rows))
	y := make([]float64, len(rows))
	w := make([]float64, len(rows))
	for i, p := range rows {
		s := p.from
		z := m.Z(s.games, m.KDynasty)
		blend := z*s.pct + (1-z)*m.Prior(s.player)
		stay := m.Survives(s.age, s.player.DraftPick, s.games, s.pct, s.player.TenureAt(s.year))
		x[i], y[i], w[i] = arcRow(s.age, s.exp), p.next.pct-blend, min(1/max(stay, 1e-3), maxSurvivalWeight)
	}
	return solveArc(x, y, w)
}

// nextPct predicts a player's next-season percentile from one season.
func nextPct(m model.Params, s season) float64 {
	pct, _ := m.Project(s.player, []model.Past{{Year: s.year, Games: s.games, Pct: s.pct}}, s.year+1)
	return pct
}

func pairError(m model.Params, pairs []pair) float64 {
	pred, target := make([]float64, len(pairs)), make([]float64, len(pairs))
	for i, p := range pairs {
		pred[i], target[i] = nextPct(m, p.from), p.next.pct
	}
	return rmse(pred, target)
}

func baselines(m model.Params, pairs []pair) (last, prior float64) {
	l, pr, target := make([]float64, len(pairs)), make([]float64, len(pairs)), make([]float64, len(pairs))
	for i, p := range pairs {
		l[i], pr[i], target[i] = p.from.pct, m.Prior(p.from.player), p.next.pct
	}
	return rmse(l, target), rmse(pr, target)
}

// maxSurvivalWeight caps a survivor's weight in the arc fit, so one unlikely survivor cannot
// dominate it.
const maxSurvivalWeight = 5.0

type arcScore struct {
	Exits             int     // players who played no game the next season
	RMSEArc, RMSEFlat float64 // holdout: the dynasty prediction with the arc vs refitted without one
	Kept              bool
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
