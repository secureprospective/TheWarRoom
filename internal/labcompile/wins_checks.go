package labcompile

import "math"

type allPlayTally struct {
	AllPlay, Won       float64
	Comparisons, Games int64
}

func (w *winsModel) allPlay() []AnalysisTeam {
	out := []AnalysisTeam{}
	for _, year := range w.years {
		tally := map[string]allPlayTally{}
		order := []string{}
		for _, week := range w.played[year] {
			keys := []teamWeek{}
			for _, key := range w.lab.ResultOrder {
				if key.Year == year && key.Week == week {
					keys = append(keys, key)
				}
			}
			for _, key := range keys {
				if _, ok := tally[key.Fid]; !ok {
					order = append(order, key.Fid)
				}
				t := tally[key.Fid]
				score := w.lab.Results[key].PF
				for _, other := range keys {
					if other == key {
						continue
					}
					switch p := w.lab.Results[other].PF; {
					case p < score:
						t.AllPlay++
					case p == score:
						t.AllPlay += 0.5
					}
				}
				t.Comparisons += int64(len(keys) - 1)
				switch w.lab.Results[key].Result {
				case "W":
					t.Won++
				case "T":
					t.Won += 0.5
				}
				t.Games++
				tally[key.Fid] = t
			}
		}
		for _, fid := range order {
			out = append(out, allPlayRow(year, fid, tally[fid]))
		}
	}
	return out
}

func allPlayRow(year int64, fid string, t allPlayTally) AnalysisTeam {
	expected := 0.0
	var rate *float64
	if t.Comparisons != 0 {
		r := t.AllPlay / float64(t.Comparisons)
		rate = ptr(pythonRound(r, 3))
		expected = r * float64(t.Games)
	}
	return AnalysisTeam{Year: year, Fid: fid, AllPlay: rate, Wins: t.Won, Games: t.Games, ExpectedWins: ptr(pythonRound(expected, 2)), Luck: ptr(pythonRound(t.Won-expected, 2))}
}

func (w *winsModel) currencyCheck() *CurrencyCheck {
	last := w.lastComplete()
	xs, ys := []float64{}, []float64{}
	for _, rec := range w.allPlay() {
		if rec.Year > last || rec.AllPlay == nil {
			continue
		}
		total := 0.0
		for _, week := range w.played[rec.Year] {
			for _, pid := range w.lab.Starts[teamWeek{rec.Year, week, rec.Fid}] {
				total += w.week[playerYear{pid, rec.Year}][week]
			}
		}
		xs = append(xs, total)
		ys = append(ys, *rec.AllPlay)
	}
	correlation := pearson(xs, ys)
	if correlation == nil {
		return nil
	}
	return &CurrencyCheck{TeamSeasons: int64(len(xs)), Correlation: pythonRound(*correlation, 3)}
}

func pearson(xs, ys []float64) *float64 {
	if len(xs) < 2 || len(xs) != len(ys) {
		return nil
	}
	mx, my := preciseMean(xs), preciseMean(ys)
	xx, yy, xy := make([]float64, len(xs)), make([]float64, len(xs)), make([]float64, len(xs))
	for i, x := range xs {
		a, b := x-mx, ys[i]-my
		xx[i] = a * a
		yy[i] = b * b
		xy[i] = a * b
	}
	sx, sy := pythonSum(xx), pythonSum(yy)
	if sx <= 0 || sy <= 0 {
		return nil
	}
	return ptr(pythonSum(xy) / math.Sqrt(sx*sy))
}
