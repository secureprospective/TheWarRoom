package lab

import (
	"math"
	"slices"
)

// Reference relations the JS Lab engine computes and no screen reads. They live with the tests
// so the parity goldens keep pinning every relation without shipping them in the app.

type Contenders struct {
	N        int      `json:"n"`
	Mean     *float64 `json:"mean"`
	Half     *float64 `json:"half"`
	RealN    int      `json:"realN"`
	RealMean *float64 `json:"realMean"`
	RealHalf *float64 `json:"realHalf"`
}

func ContenderTrades(d *Data, s Scope) Contenders {
	nets := []float64{}
	realized := []float64{}
	keep := InScope(s)
	for _, t := range d.Trades {
		if t.ANet == nil || !keep(t) {
			continue
		}
		sign := 0.0
		if t.ARank <= 10 && t.BRank >= 23 {
			sign = 1
		} else if t.BRank <= 10 && t.ARank >= 23 {
			sign = -1
		}
		if sign == 0 {
			continue
		}
		nets = append(nets, sign**t.ANet)
		if t.ARealNet != nil && t.RealSeasons != nil && *t.RealSeasons >= 2 {
			realized = append(realized, sign**t.ARealNet)
		}
	}
	m, r := MeanVar(nets), MeanVar(realized)
	out := Contenders{N: m.N, Mean: m.Mean, RealN: r.N, RealMean: r.Mean}
	if m.N > 1 {
		out.Half = ptr(Z * math.Sqrt(*m.Var/float64(m.N)))
	}
	if r.N > 1 {
		out.RealHalf = ptr(Z * math.Sqrt(*r.Var/float64(r.N)))
	}
	return out
}

type DraftMapRow struct {
	Group   string         `json:"group"`
	Overall *DraftRatio    `json:"overall"`
	N       int            `json:"n"`
	When    []DraftCell    `json:"when"`
	Where   []DraftMapCell `json:"where"`
}
type DraftMapCell struct {
	Segment string      `json:"segment"`
	Reading *DraftRatio `json:"reading"`
	N       int         `json:"n"`
}

func DraftMap(picks []JudgedPick, groups []string) []DraftMapRow {
	out := []DraftMapRow{}
	for _, g := range groups {
		mine := draftSelect(picks, func(p JudgedPick) bool { return *p.Group == g })
		r := DraftMapRow{Group: g, Overall: VsSlot(mine), N: judgedCount(mine), When: BySeason(mine), Where: []DraftMapCell{}}
		for _, seg := range DraftSegments() {
			xs := draftSelect(mine, func(p JudgedPick) bool { return p.Segment == seg })
			r.Where = append(r.Where, DraftMapCell{seg, VsSlot(xs), judgedCount(xs)})
		}
		out = append(out, r)
	}
	return out
}

func evidence(r *Range) string {
	if r == nil {
		return "none"
	}
	if r.Spread <= 2.5 {
		return "solid"
	}
	if r.Spread <= 6 {
		return "rough"
	}
	return "thin"
}

func TradeSides(d *Data) []TeamSide { return tradeSides(d, 2) }

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
