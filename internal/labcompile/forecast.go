package labcompile

import (
	"slices"
	"time"
)

const horizon = 5
const allFolds = 5

type forecastRow struct {
	forecastFeatures
	Pid    string
	Year   int64
	Group  string
	Fold   int
	Future [horizon]*float64
}
type coefficientKey struct {
	Group         string
	Horizon, Fold int
}
type calibrationKey struct {
	Group, Band string
	Horizon     int
}
type rookieKey struct {
	Group string
	Round int64
}
type forecastModel struct {
	wins        *winsModel
	last        int64
	pointInTime bool
	rows        []forecastRow
	coef        map[coefficientKey][]float64
	calibration map[calibrationKey]float64
	rookie      map[rookieKey][]float64
	pids        map[string]bool
}

func groups() []string { return []string{"QB", "RB", "WR", "TE", "PK", "DL", "LB", "DB"} }
func ageBand(age float64) string {
	switch {
	case age >= 0 && age < 24.5:
		return "young"
	case age >= 24.5 && age < 27.5:
		return "prime"
	case age >= 27.5 && age < 99:
		return "older"
	default:
		return ""
	}
}
func playerAge(l *Lab, pid string, year int64) *float64 {
	birth, ok := l.Birth[pid]
	if !ok {
		return nil
	}
	date := time.Date(int(year), time.September, 1, 0, 0, 0, 0, time.UTC)
	return ptr(float64(easternEpoch(date)-birth) / (365.25 * 86400))
}

func newForecast(w *winsModel, through *int64) *forecastModel {
	f := &forecastModel{wins: w, last: w.lastComplete(), pointInTime: through != nil, coef: map[coefficientKey][]float64{}, calibration: map[calibrationKey]float64{}, rookie: map[rookieKey][]float64{}, pids: map[string]bool{}}
	if through != nil {
		f.last = min(f.last, *through)
	}
	pids := []string{}
	for key := range w.week {
		if !slices.Contains(pids, key.Pid) {
			pids = append(pids, key.Pid)
		}
	}
	slices.Sort(pids)
	for _, pid := range pids {
		for _, year := range w.years {
			if year > f.last {
				continue
			}
			row := f.features(pid, year)
			if row == nil {
				continue
			}
			for k := range horizon {
				if year+int64(k)+1 <= f.last {
					row.Future[k] = ptr(w.rate(pid, year+int64(k)+1, nil))
				}
			}
			f.rows = append(f.rows, *row)
			f.pids[pid] = true
		}
	}
	f.fit()
	f.calibrate()
	f.rookieClasses()
	return f
}

func (f *forecastModel) features(pid string, year int64) *forecastRow {
	pos := f.wins.lab.pos(pid, year)
	if pos == nil {
		return nil
	}
	g0, g1 := f.wins.games(pid, year), f.wins.games(pid, year-1)
	if g0 < 1 && g1 < 1 {
		return nil
	}
	age := playerAge(f.wins.lab, pid, year)
	if age == nil {
		return nil
	}
	features := forecastFeatures{Age: *age, R0: f.wins.rate(pid, year, nil), R1: f.wins.rate(pid, year-1, nil), GM: float64(g0) / float64(max(1, len(f.wins.played[year])))}
	return &forecastRow{forecastFeatures: features, Pid: pid, Year: year, Group: value(positionGroup(pos)), Fold: foldOf(pid)}
}

func (f *forecastModel) fit() {
	for _, group := range groups() {
		for k := range horizon {
			firstFold := 0
			if f.pointInTime {
				firstFold = allFolds
			}
			for fold := firstFold; fold <= allFolds; fold++ {
				f.fitHorizon(group, k, fold)
			}
		}
	}
}
func (f *forecastModel) fitHorizon(group string, k, fold int) {
	x, y := [][]float64{}, []float64{}
	for _, r := range f.rows {
		if r.Group == group && r.Future[k] != nil && (fold == allFolds || r.Fold != fold) {
			x = append(x, forecastX(r.forecastFeatures))
			y = append(y, *r.Future[k])
		}
	}
	key := coefficientKey{group, k, fold}
	if len(x) < 30 {
		f.coef[key] = slices.Clone(f.coef[coefficientKey{group, k - 1, fold}])
		if f.coef[key] == nil {
			f.coef[key] = make([]float64, 7)
		}
		return
	}
	beta, _ := solve(x, y, 1, 1, nil)
	f.coef[key] = beta
}
func (f *forecastModel) predict(r forecastRow, k, fold int) float64 {
	terms := make([]float64, 7)
	x := forecastX(r.forecastFeatures)
	for i, b := range f.coef[coefficientKey{r.Group, k, fold}] {
		terms[i] = b * x[i]
	}
	return max(0, pythonSum(terms))
}
func (f *forecastModel) calibrate() {
	acc := map[calibrationKey][2]float64{}
	for _, r := range f.rows {
		band := ageBand(r.Age + 1)
		fold := r.Fold
		if f.pointInTime {
			fold = allFolds
		}
		for k, v := range r.Future {
			if v == nil {
				continue
			}
			key := calibrationKey{r.Group, band, k}
			a := acc[key]
			a[0] += *v
			a[1] += f.predict(r, k, fold)
			acc[key] = a
		}
	}
	for key, a := range acc {
		if a[1] > 0 {
			f.calibration[key] = a[0] / a[1]
		}
	}
}
func (f *forecastModel) scale(group, band string, k int) float64 {
	if v, ok := f.calibration[calibrationKey{group, band, k}]; ok {
		return v
	}
	return 1
}
func (f *forecastModel) rates(pid string, baseYear int64) []float64 {
	if row := f.features(pid, baseYear); row != nil {
		fold := row.Fold
		if f.pointInTime || !f.pids[pid] {
			fold = allFolds
		}
		out := make([]float64, horizon)
		for k := range out {
			out[k] = f.predict(*row, k, fold) * f.scale(row.Group, ageBand(row.Age+1), k)
		}
		return out
	}
	return f.rookieRates(pid, baseYear)
}
func (f *forecastModel) rookieRates(pid string, baseYear int64) []float64 {
	nfl, ok := f.wins.lab.NFLDraft[pid]
	pos := f.wins.lab.pos(pid, baseYear+1)
	if !ok || pos == nil || nfl.Year < baseYear {
		return nil
	}
	seasons := f.rookie[rookieKey{value(positionGroup(pos)), nflRound(nfl.Round)}]
	if seasons == nil {
		return nil
	}
	return shiftRates(seasons, int(nfl.Year-(baseYear+1)))
}
func shiftRates(seasons []float64, lead int) []float64 {
	out := make([]float64, horizon)
	for k := range out {
		if index := k - lead; index >= 0 && index < horizon {
			out[k] = seasons[index]
		}
	}
	return out
}
func nflRound(round *int64) int64 {
	if value(round) == 0 {
		return 8
	}
	return min(*round, 8)
}
