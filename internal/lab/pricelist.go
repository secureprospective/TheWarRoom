package lab

import (
	"math"
	"slices"
)

type PriceHalf struct {
	Price     Price      `json:"price"`
	Delivered *Delivered `json:"delivered"`
	Range     *Range     `json:"range"`
}
type PriceTime struct {
	Window string  `json:"window"`
	Price  float64 `json:"price"`
	SE     float64 `json:"se"`
}
type PriceRow struct {
	Kind      string       `json:"kind"`
	N         int          `json:"n"`
	AvgValue  float64      `json:"avgValue"`
	Price     Price        `json:"price"`
	Delivered *Delivered   `json:"delivered"`
	Range     *Range       `json:"range"`
	Halves    []*PriceHalf `json:"halves"`
	Times     []PriceTime  `json:"times"`
}
type List struct {
	Rows     []PriceRow `json:"rows"`
	Trades   int        `json:"trades"`
	PerAsset float64    `json:"perAsset"`
	From     int        `json:"from"`
	To       int        `json:"to"`
	Mid      int        `json:"mid"`
	Seasons  []int      `json:"seasons"`
	Records  int        `json:"records"`
}
type listHalf struct {
	fit     Fit
	records []Delivery
}

func windowScope(s Scope) func(Trade) bool {
	return func(t Trade) bool {
		return t.Season != nil && *t.Season >= s.From && *t.Season <= s.To && (s.Windows == nil || (t.Window != nil && slices.Contains(s.Windows, *t.Window)))
	}
}
func listHalves(d *Data, trades []ValuedTrade, records []Delivery, mid int) [2]*listHalf {
	var out [2]*listHalf
	for h := range 2 {
		sub := []ValuedTrade{}
		recs := []Delivery{}
		for _, v := range trades {
			if (*d.Trades[v.ID].Season <= mid) == (h == 0) {
				sub = append(sub, v)
			}
		}
		for _, r := range records {
			if (*d.Trades[r.Trade].Season <= mid) == (h == 0) {
				recs = append(recs, r)
			}
		}
		if len(sub) >= 40 {
			out[h] = &listHalf{FitPrices(sub), recs}
		}
	}
	return out
}
func listRow(kind string, price Price, records []Delivery, halves [2]*listHalf, windows []windowFit, n int, value float64) PriceRow {
	recs := []Delivery{}
	for _, r := range records {
		if r.Class == kind {
			recs = append(recs, r)
		}
	}
	d := DeliveredRatio(recs)
	row := PriceRow{Kind: kind, N: n, AvgValue: value / float64(max(n, 1)), Price: price, Delivered: d, Range: PerDelivered(&price, d), Halves: make([]*PriceHalf, 2), Times: []PriceTime{}}
	for h, hv := range halves {
		if hv == nil {
			continue
		}
		p, ok := hv.fit.Prices[kind]
		if !ok {
			continue
		}
		rs := []Delivery{}
		for _, r := range hv.records {
			if r.Class == kind {
				rs = append(rs, r)
			}
		}
		dh := DeliveredRatio(rs)
		row.Halves[h] = &PriceHalf{p, dh, PerDelivered(&p, dh)}
	}
	for _, w := range windows {
		if w.Fit == nil {
			continue
		}
		if p, ok := w.Fit.Prices[kind]; ok && p.SE < 0.5*math.Max(p.Price, 0.2) {
			row.Times = append(row.Times, PriceTime{w.Window, p.Price, p.SE})
		}
	}
	return row
}
func PriceList(d *Data, s Scope) *List {
	keep := windowScope(s)
	trades := ValuedTrades(d, keep)
	if len(trades) < 40 {
		return nil
	}
	fit := FitPrices(trades)
	records := recordsFor(d, keep, AgeOf, "")
	seasons, mid := seasonSplit(d, trades)
	halves := listHalves(d, trades, records, mid)
	windows := windowFits(d, trades, true)
	counts := legCounts(trades, AgeOf)
	values := make(map[string]float64)
	for _, v := range trades {
		for _, side := range v.Sides {
			for _, a := range side {
				values[a.Kind] += a.Value
			}
		}
	}
	out := &List{Rows: []PriceRow{}, Trades: len(trades), PerAsset: fit.PerAsset, From: s.From, To: s.To, Mid: mid, Seasons: seasons, Records: len(records)}
	for _, k := range fit.Kinds {
		if k != "player:unknown" {
			out.Rows = append(out.Rows, listRow(k, fit.Prices[k], records, halves, windows, counts[k], values[k]))
		}
	}
	return out
}
