package labcompile

import (
	"context"
	"fmt"
	"math"
	"slices"
)

type deliveryObservation struct {
	Expected, Realized float64
	Season             int64
}
type backtestRow struct {
	Expected, Realized float64
	Season             int64
}

func deliveredRatio(obs []deliveryObservation) *DeliveredRatio {
	expected, realized := make([]float64, len(obs)), make([]float64, len(obs))
	for i, o := range obs {
		expected[i] = o.Expected
		realized[i] = o.Realized
	}
	e := pythonSum(expected)
	if len(obs) < 8 || e < 0.5 {
		return nil
	}
	ratio := pythonSum(realized) / e
	residuals := make([]float64, len(obs))
	for i, o := range obs {
		d := o.Realized - ratio*o.Expected
		residuals[i] = d * d
	}
	se := math.Sqrt(pythonSum(residuals)) / e
	return &DeliveredRatio{pythonRound(ratio, 3), pythonRound(se, 3), int64(len(obs))}
}
func realizedCommon(item tradeValueItem, common []int) float64 {
	values := make([]float64, len(common))
	for i, k := range common {
		values[i] = *item.Real[k] * math.Pow(discount, float64(k))
	}
	return pythonSum(values)
}
func expectedCommon(item tradeValueItem, common []int) float64 {
	values := make([]float64, len(common))
	for i, k := range common {
		values[i] = item.Value[k] * math.Pow(discount, float64(k))
	}
	return pythonSum(values)
}

type backtestResult struct {
	Check     *BacktestCheck
	Delivered map[string]*MarketDelivered
	Records   []AnalysisDelivery
	Rows      []backtestRow
}

func backtest(ctx context.Context, l *Lab, trades []Trade, draft []Pick, known, estimated map[teamYear]int64, w *winsModel) (backtestResult, error) {
	last := w.lastComplete()
	pay := rookieSalaries(l, draft)
	rows := []backtestRow{}
	records := []AnalysisDelivery{}
	kinds := map[string][]deliveryObservation{}
	for season := int64(2018); season < last; season++ {
		if err := ctx.Err(); err != nil {
			return backtestResult{}, fmt.Errorf("backtest: %w", err)
		}
		through := season - 1
		f := newForecast(w, &through)
		v := newValuer(w, f, newPicks(w, draft, through), known, estimated, last)
		for _, row := range valueTrades(l, trades, v, pay) {
			if row.Season != season {
				continue
			}
			common := commonSeasons(row)
			if len(common) < 2 {
				continue
			}
			for _, side := range row.Sides {
				for _, it := range side {
					e, r := expectedCommon(it, common), realizedCommon(it, common)
					kinds[it.Class] = append(kinds[it.Class], deliveryObservation{e, r, season})
					records = append(records, AnalysisDelivery{row.Index, it.Class, e, r, int64(len(common))})
				}
			}
			rows = append(rows, backtestTrade(row, common))
		}
	}
	check := summarizeBacktest(rows, last)
	delivered := summarizeDelivered(kinds, last)
	return backtestResult{check, delivered, records, rows}, nil
}
func backtestTrade(row tradeValue, common []int) backtestRow {
	exp, actual := [2]float64{}, [2]float64{}
	for s, side := range row.Sides {
		es := make([]float64, len(side))
		rs := []float64{}
		for i, it := range side {
			es[i] = discounted(it.Value, discount)
			for _, k := range common {
				rs = append(rs, *it.Real[k]*math.Pow(discount, float64(k)))
			}
		}
		exp[s] = pythonSum(es)
		actual[s] = pythonSum(rs)
	}
	gain, got := exp[1]-exp[0], actual[1]-actual[0]
	if gain < 0 {
		got = -got
	}
	return backtestRow{math.Abs(gain), got, row.Season}
}
func summarizeBacktest(rows []backtestRow, last int64) *BacktestCheck {
	if len(rows) == 0 {
		return nil
	}
	slices.SortFunc(rows, func(a, b backtestRow) int {
		if a.Expected < b.Expected {
			return -1
		}
		if a.Expected > b.Expected {
			return 1
		}
		if a.Realized < b.Realized {
			return -1
		}
		if a.Realized > b.Realized {
			return 1
		}
		return int(a.Season - b.Season)
	})
	out := &BacktestCheck{Trades: int64(len(rows)), Seasons: [2]int64{2018, last - 1}, Bands: []BacktestBand{}}
	for q := range 5 {
		part := rows[q*len(rows)/5 : (q+1)*len(rows)/5]
		if len(part) == 0 {
			continue
		}
		xs, ys, won := backtestVectors(part)
		out.Bands = append(out.Bands, BacktestBand{pythonRound(preciseMean(xs), 3), pythonRound(preciseMean(ys), 3), pythonRound(won, 3), int64(len(part))})
	}
	xs, ys, won := backtestVectors(rows)
	mx, my := preciseMean(xs), preciseMean(ys)
	numerator, denominator := make([]float64, len(rows)), make([]float64, len(rows))
	for i, x := range xs {
		d := x - mx
		numerator[i] = d * (ys[i] - my)
		denominator[i] = d * d
	}
	den := pythonSum(denominator)
	if den == 0 {
		return nil
	}
	out.Slope = pythonRound(pythonSum(numerator)/den, 3)
	out.Won = pythonRound(won, 3)
	return out
}
func backtestVectors(rows []backtestRow) ([]float64, []float64, float64) {
	xs, ys := make([]float64, len(rows)), make([]float64, len(rows))
	won := 0
	for i, r := range rows {
		xs[i] = r.Expected
		ys[i] = r.Realized
		if r.Realized > 0 {
			won++
		}
	}
	return xs, ys, float64(won) / float64(len(rows))
}
func summarizeDelivered(kinds map[string][]deliveryObservation, last int64) map[string]*MarketDelivered {
	out := map[string]*MarketDelivered{}
	split := (2018 + last - 1) / 2
	for class, obs := range kinds {
		ratio := deliveredRatio(obs)
		if ratio == nil {
			continue
		}
		early, late := []deliveryObservation{}, []deliveryObservation{}
		for _, o := range obs {
			if o.Season <= split {
				early = append(early, o)
			} else {
				late = append(late, o)
			}
		}
		out[class] = &MarketDelivered{ratio.Ratio, ratio.SE, ratio.N, deliveredRatio(early), deliveredRatio(late), split}
	}
	return out
}
