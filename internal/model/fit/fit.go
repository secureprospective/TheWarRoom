// Package fit fits the model's parameters from stored seasons (plan Stage 6). Every parameter
// is fitted on seasons before the holdout season and scored on predicting it; the stored values
// are then refitted on every season.
package fit

import (
	"math"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/model"
)

// Result is one position's fitted params and the evidence behind them.
type Result struct {
	Position domain.Position
	Params   model.Params
	Train    model.Params // the params fitted before the holdout season, which score it
	Report   Report
}

// Report holds each fit's sample size and holdout score.
type Report struct {
	Seasons int // player-seasons with a game

	KNow                  float64 // games, against the league mean
	SplitN                int
	SplitRaw, SplitShrunk float64 // RMSE in points per game: odd-week mean vs shrunk, predicting even weeks
	SplitMean             float64

	PriorN, PriorTestN    int
	PriorR2, PriorR2Test  float64
	PriorR2True           float64 // the share of true-talent spread the prior explains
	Arc                   arcScore
	PairsTrain, PairsTest int
	DynCredibility        float64 // holdout RMSE, percentile, with Z = e/(e+k)
	DynExponential        float64 // with Z = 1 − exp(−e/k)
	DynLastSeason         float64 // last season's percentile alone
	DynPrior              float64 // the prior alone
	Recency               recencyScore
	Survival              survivalScore
	Debut                 survivalScore
}

// Run fits every position on seasons first..holdout.
func Run(d model.Data, first, holdout int) []Result {
	scales := d.Scales()
	var out []Result
	for _, pos := range model.Positions() {
		sm := collect(d, scales, pos, first, holdout)
		if len(sm.seasons) == 0 {
			continue
		}
		out = append(out, fitPosition(sm, holdout))
	}
	return out
}

func fitPosition(sm samples, holdout int) Result {
	r := Result{Position: sm.pos}
	r.Report.Seasons = len(sm.seasons)
	k := fitKNow(sm, holdout, &r.Report)
	trainPrior, prior := fitPrior(sm, holdout, &r.Report)
	r.Params.Intercept, r.Params.Weight, r.Params.Missing = prior.Intercept, prior.Weight, prior.Missing
	reliability := meanReliability(sm, k)
	r.Report.PriorR2True = clamp(r.Report.priorR2()/reliability, 0, 0.9)
	r.Params.KNow = k / (1 - r.Report.PriorR2True)

	trainSurv, fullSurv := fitSurvival(sm, holdout, &r.Report.Survival)
	r.Params.Survival = fullSurv
	trainDebut, fullDebut := fitDebut(sm, holdout, &r.Report.Debut)
	r.Params.Debut = fullDebut
	trainModel, fullModel := fitDynasty(sm, holdout, trainPrior, trainSurv, fullSurv, &r.Report)
	r.Params.KDynasty, r.Params.Exponential, r.Params.Arc = fullModel.KDynasty, fullModel.Exponential, fullModel.Arc
	r.Train = trainModel
	r.Train.Debut = trainDebut
	r.Train.Recency, r.Params.Recency = fitRecency(sm, holdout, trainModel, fullModel, &r.Report.Recency)
	return r
}

// fitKNow is the random-effects k for points per game: within-season noise per game over the
// spread of true per-game levels, from one-way analysis of variance on training seasons. The
// holdout season checks it by predicting each player's even weeks from his odd ones.
func fitKNow(sm samples, holdout int, rep *Report) float64 {
	var within, dfWithin, n, sumN2, sumNM, sumNM2 float64
	groups := 0.0
	for _, s := range sm.seasons {
		all, _, _ := weekly(s.s)
		if s.year >= holdout || len(all) < 2 {
			continue
		}
		m := avg(all)
		for _, v := range all {
			within += (v - m) * (v - m)
		}
		ni := float64(len(all))
		dfWithin += ni - 1
		n += ni
		sumN2 += ni * ni
		sumNM += ni * m
		sumNM2 += ni * m * m
		groups++
	}
	if groups < 10 {
		return math.NaN()
	}
	sigma2 := within / dfWithin
	grand := sumNM / n
	between := (sumNM2 - n*grand*grand) / (groups - 1)
	n0 := (n - sumN2/n) / (groups - 1)
	tau2 := (between - sigma2) / n0
	k := sigma2 / math.Max(tau2, sigma2/1000)
	rep.KNow = k
	splitHalf(sm, holdout, k, grand, rep)
	return k
}

func splitHalf(sm samples, holdout int, k, grand float64, rep *Report) {
	var raw, shrunk, flat, target []float64
	for _, s := range sm.seasons {
		_, odd, even := weekly(s.s)
		if s.year != holdout || len(odd) < 2 || len(even) < 2 {
			continue
		}
		o := avg(odd)
		z := float64(len(odd)) / (float64(len(odd)) + k)
		raw, shrunk, flat = append(raw, o), append(shrunk, grand+z*(o-grand)), append(flat, grand)
		target = append(target, avg(even))
	}
	rep.SplitN = len(target)
	rep.SplitRaw, rep.SplitShrunk, rep.SplitMean = rmse(raw, target), rmse(shrunk, target), rmse(flat, target)
}

// meanReliability is the average share of a regular season's percentile that is signal.
func meanReliability(sm samples, k float64) float64 {
	sum, n := 0.0, 0.0
	for _, s := range sm.seasons {
		if s.games >= model.RegularGames {
			sum += s.games / (s.games + k)
			n++
		}
	}
	if n == 0 || !finite(k) {
		return 1
	}
	return sum / n
}

// priorR2 is the holdout R² when the holdout is large enough to trust, else the lower of the
// training and holdout R²s.
func (r Report) priorR2() float64 {
	if r.PriorTestN >= 30 && finite(r.PriorR2Test) {
		return r.PriorR2Test
	}
	if finite(r.PriorR2Test) {
		return min(r.PriorR2, r.PriorR2Test)
	}
	return r.PriorR2
}

func clamp(v, lo, hi float64) float64 {
	if !finite(v) {
		return lo
	}
	return min(max(v, lo), hi)
}
