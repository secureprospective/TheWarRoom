package lab

import (
	"math"
	"slices"
	"strings"
)

type Scope struct {
	From    int      `json:"from"`
	To      int      `json:"to"`
	Phases  []string `json:"phases"`
	Windows []string `json:"windows,omitempty"`
	Assets  []string `json:"assets,omitempty"`
	ByAge   bool     `json:"byAge,omitempty"`
}
type Delivered struct {
	Ratio float64 `json:"ratio"`
	SE    float64 `json:"se"`
	N     int     `json:"n"`
}
type Range struct {
	Value  float64 `json:"value"`
	Lo     float64 `json:"lo"`
	Hi     float64 `json:"hi"`
	Spread float64 `json:"spread"`
}
type Interval struct {
	Value float64 `json:"value"`
	Lo    float64 `json:"lo"`
	Hi    float64 `json:"hi"`
}
type Stats struct {
	N    int      `json:"n"`
	Mean *float64 `json:"mean"`
	Var  *float64 `json:"var"`
}

func ptr[T any](v T) *T { return &v }
func GroupOf(kind string) string {
	if strings.HasPrefix(kind, "pick") || strings.HasPrefix(kind, "player:") {
		return kind
	}
	return strings.SplitN(kind, ":", 2)[0]
}
func AgeOf(kind string) string { return kind }
func AssetOf(kind string) string {
	if strings.HasPrefix(kind, "pick") {
		return kind
	}
	return strings.SplitN(kind, ":", 2)[0]
}
func classifier(byAge bool) func(string) string {
	if byAge {
		return AgeOf
	}
	return GroupOf
}
func InScope(s Scope) func(Trade) bool {
	return func(t Trade) bool {
		return t.Season != nil && *t.Season >= s.From && *t.Season <= s.To && (s.Phases == nil || slices.Contains(s.Phases, t.Phase))
	}
}

func DeliveredRatio(records []Delivery) *Delivered { return deliveredRatio(records, 8) }
func deliveredRatio(records []Delivery, threshold int) *Delivered {
	e := 0.0
	for _, r := range records {
		e += r.Expected
	}
	if len(records) < threshold || e < 0.5 {
		return nil
	}
	d := 0.0
	for _, r := range records {
		d += r.Delivered
	}
	ratio := d / e
	v := 0.0
	for _, r := range records {
		x := r.Delivered - ratio*r.Expected
		v += x * x
	}
	return &Delivered{ratio, math.Sqrt(v) / e, len(records)}
}

func PerDelivered(p *Price, d *Delivered) *Range {
	if p == nil || d == nil || d.Ratio <= 0 || p.Price <= 0 {
		return nil
	}
	value := p.Price / d.Ratio
	a := p.SE / p.Price
	b := d.SE / d.Ratio
	rel := math.Sqrt(a*a + b*b)
	return &Range{value, value * math.Exp(-Z*rel), value * math.Exp(Z*rel), math.Exp(2 * Z * rel)}
}

func MeanVar(values []float64) Stats {
	n := len(values)
	if n == 0 {
		return Stats{}
	}
	mean := 0.0
	for _, v := range values {
		mean += v
	}
	mean /= float64(n)
	v := 0.0
	for _, x := range values {
		v += (x - mean) * (x - mean)
	}
	if n > 1 {
		v /= float64(n - 1)
	}
	return Stats{n, ptr(mean), ptr(v)}
}
func regroup(trades []ValuedTrade, of func(string) string) []ValuedTrade {
	out := make([]ValuedTrade, len(trades))
	for i, v := range trades {
		out[i] = ValuedTrade{ID: v.ID, Sides: [2][]Asset{{}, {}}}
		for side, assets := range v.Sides {
			for _, a := range assets {
				out[i].Sides[side] = append(out[i].Sides[side], Asset{of(a.Kind), a.Value})
			}
		}
	}
	return out
}
func recordsFor(d *Data, keep func(Trade) bool, of func(string) string, kind string) []Delivery {
	out := []Delivery{}
	for _, r := range d.Delivered {
		if keep(d.Trades[r.Trade]) && (kind == "" || of(r.Class) == kind) {
			out = append(out, r)
		}
	}
	return out
}
func legCounts(trades []ValuedTrade, of func(string) string) map[string]int {
	n := make(map[string]int)
	for _, v := range trades {
		for _, side := range v.Sides {
			for _, a := range side {
				n[of(a.Kind)]++
			}
		}
	}
	return n
}
func seasonSplit(d *Data, trades []ValuedTrade) ([]int, int) {
	seasons := []int{}
	for _, v := range trades {
		s := d.Trades[v.ID].Season
		if s != nil && !slices.Contains(seasons, *s) {
			seasons = append(seasons, *s)
		}
	}
	slices.Sort(seasons)
	return seasons, seasons[(len(seasons)-1)/2]
}
func relative(p Price, base float64) *Interval {
	return &Interval{p.Price / base, (p.Price - Z*p.SE) / base, (p.Price + Z*p.SE) / base}
}
func typicalOf(f Fit, n map[string]int, positive bool) *float64 {
	kinds := []string{}
	for _, k := range f.Kinds {
		if k != "player:unknown" && (!positive || f.Prices[k].Price > 0) {
			kinds = append(kinds, k)
		}
	}
	slices.SortStableFunc(kinds, func(a, b string) int {
		if f.Prices[a].Price < f.Prices[b].Price {
			return -1
		}
		if f.Prices[a].Price > f.Prices[b].Price {
			return 1
		}
		return 0
	})
	total := 0
	for _, k := range kinds {
		total += n[k]
	}
	seen := 0
	for _, k := range kinds {
		seen += n[k]
		if float64(seen) >= float64(total)/2 {
			return ptr(f.Prices[k].Price)
		}
	}
	return nil
}
