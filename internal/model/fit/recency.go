package fit

import (
	"math"

	"github.com/secureprospective/TheWarRoom/internal/model"
)

type recencyScore struct {
	Train, Test          int
	RMSEFitted, RMSEMarc float64 // holdout: fitted weights vs Marcel's 1 / 0.8 / 0.6
	Kept                 bool
}

// fitRecency weighs up to three past seasons, each first moved along the talent arc to the
// target season so recency measures lost information, not age. The evidence they carry is the
// effective count (Σw·g)² / Σw²·g. Weights for the two older seasons are fitted on a grid with
// the training model and compared with Marcel's on the holdout season; when they win they are
// refitted on every season with the full model, and otherwise Marcel's are stored.
func fitRecency(sm samples, holdout int, train, full model.Params, score *recencyScore) (trainW, fullW [2]float64) {
	var trainH, testH []history
	for _, s := range sm.seasons {
		if s.games < model.RegularGames {
			continue
		}
		h := sm.history(s)
		if len(h.past) == 0 {
			continue
		}
		if s.year < holdout {
			trainH = append(trainH, h)
		} else {
			testH = append(testH, h)
		}
	}
	score.Train, score.Test = len(trainH), len(testH)
	best := bestRecency(train, trainH)
	score.RMSEFitted = historyError(train, best, testH)
	score.RMSEMarc = historyError(train, marcel, testH)
	score.Kept = score.RMSEFitted < score.RMSEMarc
	if !score.Kept {
		return marcel, marcel
	}
	return best, bestRecency(full, append(trainH, testH...))
}

// marcel is Marcel's weights for the two older seasons, the latest weighing 1.
var marcel = [2]float64{0.8, 0.6} //nolint:gochecknoglobals // a fixed reference array

func bestRecency(m model.Params, hs []history) [2]float64 {
	best, bestErr := marcel, math.Inf(1)
	for a := 0.0; a <= 1.0001; a += 0.1 {
		for b := 0.0; b <= a+0.0001; b += 0.1 {
			if e := historyError(m, [2]float64{a, b}, hs); e < bestErr {
				best, bestErr = [2]float64{a, b}, e
			}
		}
	}
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

// predict is the projected percentile for the target season from the player's past seasons.
func (h history) predict(m model.Params, weights [2]float64) float64 {
	m.Recency = weights
	past := make([]model.Past, 0, len(h.past))
	for _, s := range h.past {
		if s.games > 0 && finite(s.age) {
			past = append(past, model.Past{Year: s.year, Games: s.games, Pct: s.pct})
		}
	}
	pct, _ := m.Project(h.target.player, past, h.target.year)
	return pct
}

func historyError(m model.Params, weights [2]float64, hs []history) float64 {
	pred, target := make([]float64, len(hs)), make([]float64, len(hs))
	for i, h := range hs {
		pred[i], target[i] = h.predict(m, weights), h.target.pct
	}
	return rmse(pred, target)
}
