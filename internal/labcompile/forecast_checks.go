package labcompile

import (
	"fmt"
	"slices"
)

func carryRates(seasons []*float64) []float64 {
	out := make([]float64, len(seasons))
	for k, v := range seasons {
		if v != nil {
			out[k] = *v
		} else if k > 0 {
			out[k] = out[k-1]
		}
	}
	return out
}
func (f *forecastModel) rookieClasses() {
	acc := map[rookieKey][horizon][2]float64{}
	pids := make([]string, 0, len(f.wins.lab.NFLDraft))
	for pid := range f.wins.lab.NFLDraft {
		pids = append(pids, pid)
	}
	slices.Sort(pids)
	for _, pid := range pids {
		nfl := f.wins.lab.NFLDraft[pid]
		if nfl.Year < firstValued || nfl.Year > f.last {
			continue
		}
		pos := f.wins.lab.pos(pid, nfl.Year)
		if pos == nil {
			continue
		}
		key := rookieKey{value(positionGroup(pos)), nflRound(nfl.Round)}
		a := acc[key]
		for k := range horizon {
			if nfl.Year+int64(k) <= f.last {
				a[k][0] += f.wins.rate(pid, nfl.Year+int64(k), nil)
				a[k][1]++
			}
		}
		acc[key] = a
	}
	for key, a := range acc {
		seasons := make([]*float64, horizon)
		for k, pair := range a {
			if pair[1] != 0 {
				seasons[k] = ptr(pair[0] / pair[1])
			}
		}
		f.rookie[key] = carryRates(seasons)
	}
}
func (f *forecastModel) check() map[string]ForecastCheck {
	out := map[string]ForecastCheck{}
	for _, group := range groups() {
		for k := range horizon {
			pairs := [][2]float64{}
			for _, r := range f.rows {
				if r.Group == group && r.Future[k] != nil {
					pairs = append(pairs, [2]float64{*r.Future[k], f.predict(r, k, r.Fold) * f.scale(group, ageBand(r.Age+1), k)})
				}
			}
			if accuracy := forecastCheck(pairs); accuracy != nil {
				out[fmt.Sprintf("%s+%d", group, k+1)] = ForecastCheck{N: int64(accuracy.N), Explained: accuracy.Explained}
			}
		}
	}
	return out
}
