package labcompile

import (
	"math"
	"slices"
)

const firstValued = 2014

type positionYear struct {
	Position string
	Year     int64
}

// winsModel measures production against the league's own free-player baseline.
type winsModel struct {
	lab         *Lab
	years       []int64
	played      map[int64][]int64
	spread      map[int64]float64
	replacement map[positionYear]float64
	week        map[playerYear]map[int64]float64
}

func newWins(l *Lab) *winsModel {
	w := &winsModel{lab: l, played: map[int64][]int64{}, spread: map[int64]float64{}, replacement: map[positionYear]float64{}, week: map[playerYear]map[int64]float64{}}
	for y := int64(firstYear); y <= lastYear; y++ {
		w.played[y] = []int64{}
		for week := int64(1); week <= l.Regular[y]; week++ {
			if l.Done[yearWeek{y, week}] {
				w.played[y] = append(w.played[y], week)
			}
		}
		if y >= firstValued {
			w.years = append(w.years, y)
		}
		w.buildBaselines(y)
	}
	for _, key := range l.WeeklyOrder {
		w.buildPlayerWeeks(key)
	}
	return w
}

func (w *winsModel) buildBaselines(year int64) {
	scores := []float64{}
	for _, key := range w.lab.ResultOrder {
		if key.Year == year && slices.Contains(w.played[year], key.Week) {
			scores = append(scores, w.lab.Results[key].PF)
		}
	}
	if len(scores) > 30 {
		w.spread[year] = populationSpread(scores)
	}
	byPosition := map[string][]float64{}
	for _, key := range w.lab.WeeklyOrder {
		if key.Year != year {
			continue
		}
		games := w.scoringGames(key)
		pos := w.lab.pos(key.Pid, year)
		if pos != nil && len(games) >= 4 {
			byPosition[*pos] = append(byPosition[*pos], pythonSum(games)/float64(len(games)))
		}
	}
	for pos, values := range byPosition {
		line := value(w.lab.StarterLine[year][pos])
		if replacement := replacementLevel(values, int(line)); replacement != nil {
			w.replacement[positionYear{pos, year}] = *replacement
		}
	}
}

func populationSpread(values []float64) float64 {
	// statistics.pstdev computes exact rational squared deviations before its square root.
	mean := preciseMean(values)
	squares := make([]float64, len(values))
	for i, v := range values {
		d := v - mean
		squares[i] = d * d
	}
	return math.Sqrt(preciseMean(squares))
}

func replacementLevel(values []float64, line int) *float64 {
	values = slices.Clone(values)
	slices.SortFunc(values, func(a, b float64) int {
		if a > b {
			return -1
		}
		if a < b {
			return 1
		}
		return 0
	})
	if line < 0 || line >= len(values) {
		return nil
	}
	band := values[line:min(len(values), line+max(3, line/3))]
	return ptr(preciseMean(band))
}

func (w *winsModel) scoringGames(key playerYear) []float64 {
	games := []float64{}
	for _, week := range w.played[key.Year] {
		if p := w.lab.WeeklyPts[key][week]; p != 0 {
			games = append(games, p)
		}
	}
	return games
}

func (w *winsModel) buildPlayerWeeks(key playerYear) {
	pos := w.lab.pos(key.Pid, key.Year)
	if pos == nil {
		return
	}
	replacement, ok := w.replacement[positionYear{*pos, key.Year}]
	spread, hasSpread := w.spread[key.Year]
	if !ok || !hasSpread || spread <= 0 {
		return
	}
	mine := map[int64]float64{}
	for _, week := range w.played[key.Year] {
		if points, scored := w.lab.WeeklyPts[key][week]; scored {
			mine[week] = weekWins(points, replacement, spread)
		}
	}
	w.week[key] = mine
}

func weekWins(points, replacement, spread float64) float64 {
	if points == 0 || spread <= 0 {
		return 0
	}
	x := (points - replacement) / (spread * math.Sqrt(2))
	return (1+math.Erf(x/math.Sqrt2))/2 - 0.5
}

func (w *winsModel) season(pid string, year int64) float64 {
	values := []float64{}
	for _, week := range w.played[year] {
		if v, ok := w.week[playerYear{pid, year}][week]; ok {
			values = append(values, v)
		}
	}
	return pythonSum(values)
}

func (w *winsModel) rate(pid string, year int64, weeks []int64) float64 {
	played := w.played[year]
	if weeks != nil {
		played = intersectWeeks(played, weeks)
	}
	if len(played) == 0 {
		return 0
	}
	values := make([]float64, len(played))
	for i, week := range played {
		values[i] = w.week[playerYear{pid, year}][week]
	}
	return pythonSum(values) / float64(len(played))
}

func intersectWeeks(played, weeks []int64) []int64 {
	out := []int64{}
	for _, week := range played {
		if slices.Contains(weeks, week) {
			out = append(out, week)
		}
	}
	return out
}

func (w *winsModel) games(pid string, year int64) int {
	return len(w.scoringGames(playerYear{pid, year}))
}
func (w *winsModel) seasonWeeks(year int64) int64 {
	for key := range w.lab.WeekStart {
		if key.Year == year && w.lab.Regular[year] != 0 {
			return w.lab.Regular[year]
		}
	}
	return 13
}

func (w *winsModel) lastComplete() int64 {
	last := int64(0)
	for _, year := range w.years {
		if len(w.played[year]) > 0 && int64(len(w.played[year])) == w.lab.Regular[year] {
			last = max(last, year)
		}
	}
	return last
}
