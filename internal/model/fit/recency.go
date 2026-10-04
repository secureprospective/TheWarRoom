package fit

import (
	"math"

	"github.com/secureprospective/TheWarRoom/internal/model"
)

type recencyScore struct {
	Train, Test          int
	RMSEFitted, RMSEMarc float64 // holdout: fitted weights vs Marcel's 1 / 0.8 / 0.6
}

// fitRecency weighs up to three past seasons, each first moved along the talent arc to the
// target season so recency measures lost information, not age. The evidence they carry is the
// effective count (Σw·g)² / Σw²·g. Weights for the two older seasons are fitted on a grid and
// compared with Marcel's on the holdout season.
func fitRecency(sm samples, holdout int, m model.Params, score *recencyScore) [2]float64 {
	var train, test []history
	for _, s := range sm.seasons {
		if s.games < model.RegularGames {
			continue
		}
		h := sm.history(s)
		if len(h.past) == 0 {
			continue
		}
		if s.year < holdout {
			train = append(train, h)
		} else {
			test = append(test, h)
		}
	}
	score.Train, score.Test = len(train), len(test)
	best, bestErr := [2]float64{0.8, 0.6}, math.Inf(1)
	for a := 0.0; a <= 1.0001; a += 0.1 {
		for b := 0.0; b <= a+0.0001; b += 0.1 {
			if e := historyError(m, [2]float64{a, b}, train); e < bestErr {
				best, bestErr = [2]float64{a, b}, e
			}
		}
	}
	score.RMSEFitted = historyError(m, best, test)
	score.RMSEMarc = historyError(m, [2]float64{0.8, 0.6}, test)
	return best
}

// history is a target season and the player's seasons in the three before it, latest first.
type history struct {
	target season
	past   []season // index 0 is the season before the target; a gap is a zero-games season
}

func (sm samples) history(target season) history {
	h := history{target: target}
	for back := 1; back <= 3; back++ {
		s, ok := sm.byYear[target.player.ID][target.year-back]
		if !ok {
			s = season{player: target.player, year: target.year - back}
		}
		h.past = append(h.past, s)
	}
	for len(h.past) > 0 && h.past[len(h.past)-1].games == 0 {
		h.past = h.past[:len(h.past)-1]
	}
	return h
}

// predict is the blended percentile for the target season from the player's past seasons.
func (h history) predict(m model.Params, weights [2]float64) float64 {
	w := []float64{1, weights[0], weights[1]}
	var sw, swx, sw2 float64
	for i, s := range h.past {
		if s.games == 0 || !finite(s.age) {
			continue
		}
		moved := s.pct
		for y := s.year; y < h.target.year; y++ {
			moved += m.ArcStep(s.age+float64(y-s.year), s.exp+y-s.year)
		}
		sw += w[i] * s.games
		swx += w[i] * s.games * moved
		sw2 += w[i] * w[i] * s.games
	}
	prior := m.Prior(h.target.player)
	if sw == 0 {
		return prior
	}
	z := m.Z(sw*sw/sw2, m.KDynasty)
	return z*swx/sw + (1-z)*prior
}

func historyError(m model.Params, weights [2]float64, hs []history) float64 {
	pred, target := make([]float64, len(hs)), make([]float64, len(hs))
	for i, h := range hs {
		pred[i], target[i] = h.predict(m, weights), h.target.pct
	}
	return rmse(pred, target)
}
