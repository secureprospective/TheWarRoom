package fit

import (
	"maps"
	"math"
	"slices"
	"sort"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/model"
)

// Today is today's board's rule for a player's next season: last season's total league points
// and the engine's age pull at the season's age.
type Today func(pl *model.Player, lastTotal, age float64) float64

// BoardCheck is one position's holdout against today's board: Spearman rank correlations with
// the holdout season, for the model (trained before it) and for today's rule (plan Stage 7).
type BoardCheck struct {
	Position domain.Position
	// Talent: among the holdout season's regulars, against points per game.
	TalentN                  int
	TalentModel, TalentToday float64
	// Value: among everyone who played the season before, against the holdout season's total
	// points; the model's level is weighted by its chance he plays.
	ValueN                 int
	ValueModel, ValueToday float64
	// Rookies: the holdout season's rookies who became regulars; today's board scores them all 0.
	RookieN     int
	RookieModel float64
}

// CheckBoard scores the trained model against today's rule on predicting season holdout.
func CheckBoard(d model.Data, results []Result, holdout int, today Today) []BoardCheck {
	scales := d.Scales()
	var out []BoardCheck
	for _, r := range results {
		c := BoardCheck{Position: r.Position}
		var talent, value, rookie [][2]float64 // model, target
		var talentToday, valueToday [][2]float64
		for _, id := range slices.Sorted(maps.Keys(d.Seasons)) {
			seasons, pl := d.Seasons[id], d.Players[id]
			if pl == nil || pl.Position != r.Position || math.IsNaN(pl.AgeAt(holdout)) {
				continue
			}
			past := pastSeasons(seasons, scales[r.Position], holdout)
			proj, _ := r.Train.Project(pl, past, holdout)
			next, last := seasons[holdout], seasons[holdout-1]
			lastTotal := total(last)
			now := today(pl, lastTotal, pl.AgeAt(holdout))
			if next != nil && next.Games() >= model.RegularGames {
				talent = append(talent, [2]float64{proj, next.PPG()})
				talentToday = append(talentToday, [2]float64{now, next.PPG()})
				if pl.Rookie == holdout {
					rookie = append(rookie, [2]float64{proj, next.PPG()})
				}
			}
			if last != nil && last.Games() > 0 {
				lastPct := scales[r.Position][holdout-1].Pct(last.PPG())
				plays := r.Train.Survives(pl.AgeAt(holdout-1), pl.DraftPick, float64(last.Games()), lastPct)
				value = append(value, [2]float64{proj * plays, total(next)})
				valueToday = append(valueToday, [2]float64{now, total(next)})
			}
		}
		c.TalentN, c.TalentModel, c.TalentToday = len(talent), spearman(talent), spearman(talentToday)
		c.ValueN, c.ValueModel, c.ValueToday = len(value), spearman(value), spearman(valueToday)
		c.RookieN, c.RookieModel = len(rookie), spearman(rookie)
		out = append(out, c)
	}
	return out
}

// pastSeasons are a player's seasons before holdout as the model reads them.
func pastSeasons(seasons map[int]*model.Season, scales map[int]model.Scale, holdout int) []model.Past {
	var out []model.Past
	for y, s := range seasons {
		if y < holdout && s.Games() > 0 {
			out = append(out, model.Past{Year: y, Games: float64(s.Games()), Pct: scales[y].Pct(s.PPG())})
		}
	}
	slices.SortFunc(out, func(a, b model.Past) int { return a.Year - b.Year })
	return out
}

func total(s *model.Season) float64 {
	if s == nil || s.Games() == 0 {
		return 0
	}
	return s.PPG() * float64(s.Games())
}

// spearman is the rank correlation of the pairs' two columns, ties ranked by their mean; NaN
// with fewer than three pairs or a column without spread.
func spearman(pairs [][2]float64) float64 {
	if len(pairs) < 3 {
		return math.NaN()
	}
	a, b := make([]float64, len(pairs)), make([]float64, len(pairs))
	for i, p := range pairs {
		a[i], b[i] = p[0], p[1]
	}
	return pearson(ranks(a), ranks(b))
}

func ranks(v []float64) []float64 {
	idx := make([]int, len(v))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(i, j int) bool { return v[idx[i]] < v[idx[j]] })
	out := make([]float64, len(v))
	for i := 0; i < len(idx); {
		j := i
		for j+1 < len(idx) && v[idx[j+1]] == v[idx[i]] {
			j++
		}
		for k := i; k <= j; k++ {
			out[idx[k]] = float64(i+j) / 2
		}
		i = j + 1
	}
	return out
}

func pearson(a, b []float64) float64 {
	ma, mb := avg(a), avg(b)
	var sab, saa, sbb float64
	for i := range a {
		sab += (a[i] - ma) * (b[i] - mb)
		saa += (a[i] - ma) * (a[i] - ma)
		sbb += (b[i] - mb) * (b[i] - mb)
	}
	if saa == 0 || sbb == 0 {
		return math.NaN()
	}
	return sab / math.Sqrt(saa*sbb)
}
