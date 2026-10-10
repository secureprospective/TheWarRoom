package lab

import (
	"math"
	"slices"
)

type MarketRow struct {
	Kind      string       `json:"kind"`
	N         int          `json:"n"`
	Price     Price        `json:"price"`
	Delivered *Delivered   `json:"delivered"`
	Range     *Range       `json:"range"`
	Cost      *Interval    `json:"cost"`
	Side      *string      `json:"side"`
	Paid      *Interval    `json:"paid"`
	Halves    []*MarketRow `json:"halves,omitempty"`
	Trend     *string      `json:"trend,omitempty"`
	Holds     bool         `json:"holds,omitempty"`
	Times     []MarketTime `json:"times,omitempty"`
	Cheapest  *string      `json:"cheapest,omitempty"`
	Dearest   *string      `json:"dearest,omitempty"`
}
type MarketTime struct {
	Window string   `json:"window"`
	Cost   *float64 `json:"cost"`
	SE     *float64 `json:"se,omitempty"`
	N      int      `json:"n"`
}
type Market struct {
	Rows    []MarketRow `json:"rows"`
	Going   *float64    `json:"going,omitempty"`
	Trades  int         `json:"trades"`
	Seasons []int       `json:"seasons,omitempty"`
	Mid     int         `json:"mid,omitempty"`
	From    int         `json:"from"`
	To      int         `json:"to"`
	Phases  []string    `json:"phases"`
}
type windowFit struct {
	Window string
	Fit    *Fit
	Median float64
	N      map[string]int
}

func basicRows(trades []ValuedTrade, records []Delivery, of func(string) string) []MarketRow {
	fit := FitPrices(regroup(trades, of))
	counts := legCounts(trades, of)
	out := []MarketRow{}
	for _, k := range fit.Kinds {
		if k == "player:unknown" {
			continue
		}
		recs := []Delivery{}
		for _, r := range records {
			if of(r.Class) == k {
				recs = append(recs, r)
			}
		}
		p := fit.Prices[k]
		d := DeliveredRatio(recs)
		out = append(out, MarketRow{Kind: k, N: counts[k], Price: p, Delivered: d, Range: PerDelivered(&p, d)})
	}
	return out
}
func goingRate(rows []MarketRow) *float64 {
	items := []MarketRow{}
	for _, r := range rows {
		if r.Range != nil {
			items = append(items, r)
		}
	}
	slices.SortStableFunc(items, func(a, b MarketRow) int {
		if a.Range.Value < b.Range.Value {
			return -1
		}
		if a.Range.Value > b.Range.Value {
			return 1
		}
		return 0
	})
	total := 0
	for _, r := range items {
		total += r.N
	}
	seen := 0
	for _, r := range items {
		seen += r.N
		if float64(seen) >= float64(total)/2 {
			return ptr(r.Range.Value)
		}
	}
	return nil
}
func costRows(trades []ValuedTrade, records []Delivery, byAge bool) ([]MarketRow, *float64) {
	base := basicRows(trades, records, GroupOf)
	going := goingRate(base)
	paying := typicalOf(FitPrices(regroup(trades, GroupOf)), legCounts(trades, GroupOf), true)
	rows := base
	if byAge {
		rows = basicRows(trades, records, AgeOf)
	}
	for i := range rows {
		r := &rows[i]
		if r.Range != nil && going != nil && *going != 0 {
			r.Cost = &Interval{r.Range.Value / *going, r.Range.Lo / *going, r.Range.Hi / *going}
			side := "fair"
			if r.Cost.Hi < 1 {
				side = "cheap"
			} else if r.Cost.Lo > 1 {
				side = "dear"
			}
			r.Side = ptr(side)
		}
		if paying != nil && *paying != 0 {
			r.Paid = relative(r.Price, *paying)
		}
	}
	return rows, going
}
func windowFits(d *Data, trades []ValuedTrade, byAge bool) []windowFit {
	out := []windowFit{}
	of := classifier(byAge)
	for _, w := range []string{"spring", "summer", "season"} {
		sub := []ValuedTrade{}
		for _, v := range trades {
			if t := d.Trades[v.ID]; t.Window != nil && *t.Window == w {
				sub = append(sub, v)
			}
		}
		if len(sub) == 0 {
			continue
		}
		wf := windowFit{Window: w}
		if len(sub) >= MinPart {
			fit := FitPrices(regroup(sub, of))
			wf.Fit = &fit
			wf.N = legCounts(sub, of)
			median := typicalOf(FitPrices(regroup(sub, GroupOf)), legCounts(sub, GroupOf), false)
			wf.Median = 1
			if median != nil {
				wf.Median = *median
			}
		}
		out = append(out, wf)
	}
	return out
}
func marketHalves(d *Data, trades []ValuedTrade, records []Delivery, mid int, byAge bool) [2][]MarketRow {
	var halves [2][]MarketRow
	for h := range 2 {
		sub := []ValuedTrade{}
		for _, v := range trades {
			if (*d.Trades[v.ID].Season <= mid) == (h == 0) {
				sub = append(sub, v)
			}
		}
		recs := []Delivery{}
		for _, r := range records {
			if (*d.Trades[r.Trade].Season <= mid) == (h == 0) {
				recs = append(recs, r)
			}
		}
		if len(sub) >= MinPart {
			halves[h], _ = costRows(sub, recs, byAge)
		}
	}
	return halves
}
func clearsHalf(r *MarketRow, side *string) bool {
	return r != nil && r.Cost != nil && side != nil && ((*side == "cheap" && r.Cost.Hi < 1) || (*side == "dear" && r.Cost.Lo > 1))
}
func decorateRow(r *MarketRow, halves [2][]MarketRow, windows []windowFit) {
	r.Halves = make([]*MarketRow, 2)
	for h := range halves {
		for _, row := range halves[h] {
			if row.Kind == r.Kind {
				r.Halves[h] = ptr(row)
				break
			}
		}
	}
	early := clearsHalf(r.Halves[0], r.Side)
	late := clearsHalf(r.Halves[1], r.Side)
	if r.Side != nil && *r.Side != "fair" {
		trend := "whole"
		switch {
		case early && late:
			trend = "holds"
		case late:
			trend = "new"
		case early:
			trend = "fading"
		}
		r.Trend = ptr(trend)
		r.Holds = trend == "holds"
	}
	r.Times = []MarketTime{}
	for _, w := range windows {
		t := MarketTime{Window: w.Window}
		if w.Fit != nil {
			t.N = w.N[r.Kind]
			if p, ok := w.Fit.Prices[r.Kind]; ok && t.N >= 25 && p.SE < 0.5*math.Max(p.Price, 0.2) {
				t.Cost = ptr(p.Price / w.Median)
				t.SE = ptr(p.SE / w.Median)
			}
		}
		r.Times = append(r.Times, t)
	}
	r.Cheapest, r.Dearest = bestTimes(r.Times)
}
func bestTimes(times []MarketTime) (*string, *string) {
	known := []MarketTime{}
	for _, t := range times {
		if t.Cost != nil {
			known = append(known, t)
		}
	}
	if len(known) < 2 {
		return nil, nil
	}
	lo, hi := 0, 0
	for i, t := range known {
		if *t.Cost < *known[lo].Cost {
			lo = i
		}
		if *t.Cost > *known[hi].Cost {
			hi = i
		}
	}
	beats := func(i int, sign float64) bool {
		for j, o := range known {
			if i != j && sign*(*o.Cost-*known[i].Cost) <= Z*math.Hypot(*o.SE, *known[i].SE) {
				return false
			}
		}
		return true
	}
	var cheap, dear *string
	if beats(lo, 1) {
		cheap = ptr(known[lo].Window)
	}
	if beats(hi, -1) {
		dear = ptr(known[hi].Window)
	}
	return cheap, dear
}
func LeagueMarket(d *Data, s Scope) Market {
	trades := ValuedTrades(d, InScope(s))
	m := Market{Trades: len(trades), From: s.From, To: s.To, Phases: s.Phases}
	if len(trades) < MinWhole {
		m.Phases = nil
		return m
	}
	records := recordsFor(d, InScope(s), GroupOf, "")
	m.Rows, m.Going = costRows(trades, records, s.ByAge)
	m.Seasons, m.Mid = seasonSplit(d, trades)
	halves := marketHalves(d, trades, records, m.Mid, s.ByAge)
	windows := windowFits(d, trades, s.ByAge)
	for i := range m.Rows {
		decorateRow(&m.Rows[i], halves, windows)
	}
	return m
}
