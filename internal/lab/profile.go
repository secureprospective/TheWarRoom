package lab

import (
	"math"
	"slices"
)

type Stretch struct {
	Phase   string         `json:"phase"`
	Trades  int            `json:"trades"`
	Fit     *Fit           `json:"fit"`
	N       map[string]int `json:"n,omitempty"`
	Typical *float64       `json:"typical,omitempty"`
}

func StretchPrices(d *Data, s Scope, phases []string) []Stretch {
	out := []Stretch{}
	of := classifier(s.ByAge)
	for _, ph := range phases {
		scope := s
		scope.Phases = []string{ph}
		trades := ValuedTrades(d, InScope(scope))
		sx := Stretch{Phase: ph, Trades: len(trades)}
		if len(trades) >= MinPart {
			fit := FitPrices(regroup(trades, of))
			sx.Fit = &fit
			sx.N = legCounts(trades, of)
			sx.Typical = typicalOf(FitPrices(regroup(trades, GroupOf)), legCounts(trades, GroupOf), true)
		}
		out = append(out, sx)
	}
	return out
}

type PaidTime struct {
	Value float64 `json:"value"`
	Lo    float64 `json:"lo"`
	Hi    float64 `json:"hi"`
	SE    float64 `json:"se"`
}
type ProfileTime struct {
	Phase    string    `json:"phase"`
	N        int       `json:"n"`
	Trades   int       `json:"trades"`
	Paid     *PaidTime `json:"paid"`
	Cheapest bool      `json:"cheapest,omitempty"`
	Dearest  bool      `json:"dearest,omitempty"`
}
type Against struct {
	Kind        string  `json:"kind"`
	N           int     `json:"n"`
	Ratio       float64 `json:"ratio"`
	Lo          float64 `json:"lo"`
	Hi          float64 `json:"hi"`
	ByDelivered bool    `json:"byDelivered"`
	Side        string  `json:"side"`
}
type Return struct {
	Kind  string  `json:"kind"`
	Wins  float64 `json:"wins"`
	Share float64 `json:"share"`
}
type Standing struct {
	Contender float64 `json:"contender"`
	Middle    float64 `json:"middle"`
	Rebuilder float64 `json:"rebuilder"`
}
type Buyers struct {
	Mine Standing `json:"mine"`
	All  Standing `json:"all"`
	N    int      `json:"n"`
}
type Profile struct {
	Kind      string        `json:"kind"`
	N         int           `json:"n"`
	Moved     int           `json:"moved"`
	Trades    int           `json:"trades"`
	Paid      *Interval     `json:"paid"`
	Delivered *Delivered    `json:"delivered"`
	When      []ProfileTime `json:"when"`
	Against   []Against     `json:"against"`
	Back      []Return      `json:"back"`
	Buyers    Buyers        `json:"buyers"`
}

func profileWhen(kind string, stretches []Stretch) []ProfileTime {
	out := []ProfileTime{}
	times := []MarketTime{}
	for _, sx := range stretches {
		w := ProfileTime{Phase: sx.Phase, N: sx.N[kind], Trades: sx.Trades}
		if sx.Fit != nil && sx.Typical != nil && *sx.Typical != 0 {
			if p, ok := sx.Fit.Prices[kind]; ok && w.N >= 12 && p.SE < 0.6*math.Max(p.Price, 0.2) {
				r := relative(p, *sx.Typical)
				w.Paid = &PaidTime{r.Value, r.Lo, r.Hi, p.SE / *sx.Typical}
				times = append(times, MarketTime{Window: sx.Phase, Cost: ptr(r.Value), SE: ptr(w.Paid.SE)})
			}
		}
		out = append(out, w)
	}
	cheap, dear := bestTimes(times)
	for i := range out {
		out[i].Cheapest = cheap != nil && *cheap == out[i].Phase
		out[i].Dearest = dear != nil && *dear == out[i].Phase
	}
	return out
}
func comparison(kind, k string, price, q Price, fit Fit, n int, d, dq *Delivered) *Against {
	if price.Price <= 0 || q.Price <= 0 {
		return nil
	}
	ratio := price.Price / q.Price
	a, b := price.SE/price.Price, q.SE/q.Price
	v := a*a + b*b - 2*fit.Cov(kind, k)/(price.Price*q.Price)
	by := d != nil && dq != nil && d.Ratio > 0 && dq.Ratio > 0
	if by {
		ratio *= dq.Ratio / d.Ratio
		a, b = d.SE/d.Ratio, dq.SE/dq.Ratio
		v += a*a + b*b
	}
	sd := math.Sqrt(math.Max(0, v))
	lo, hi := ratio*math.Exp(-Z*sd), ratio*math.Exp(Z*sd)
	side := "even"
	if hi < 1 {
		side = "cheaper"
	} else if lo > 1 {
		side = "dearer"
	}
	return &Against{k, n, ratio, lo, hi, by, side}
}
func profileAgainst(d *Data, s Scope, kind string, fit Fit, n map[string]int, delivered *Delivered) []Against {
	out := []Against{}
	of := classifier(s.ByAge)
	price := fit.Prices[kind]
	for _, k := range fit.Kinds {
		if k == kind || k == "player:unknown" || n[k] < 25 {
			continue
		}
		dq := DeliveredRatio(recordsFor(d, InScope(s), of, k))
		a := comparison(kind, k, price, fit.Prices[k], fit, n[k], delivered, dq)
		if a != nil {
			out = append(out, *a)
		}
	}
	slices.SortStableFunc(out, func(a, b Against) int {
		if a.Ratio < b.Ratio {
			return -1
		}
		if a.Ratio > b.Ratio {
			return 1
		}
		return 0
	})
	return out
}
func standing(rank int) int {
	if rank <= 10 {
		return 0
	}
	if rank >= 23 {
		return 2
	}
	return 1
}
func share(n [3]int) Standing {
	total := n[0] + n[1] + n[2]
	if total == 0 {
		return Standing{}
	}
	return Standing{float64(n[0]) / float64(total), float64(n[1]) / float64(total), float64(n[2]) / float64(total)}
}
func profileReturns(d *Data, trades []ValuedTrade, kind string, of func(string) string) ([]Return, Buyers, int) {
	back := make(map[string]float64)
	order := []string{}
	total := 0.0
	moved := 0
	var buyers, everyone [3]int
	for _, v := range trades {
		legs := d.Trades[v.ID].Legs
		var sides [2]bool
		mine := false
		for _, l := range legs {
			if l.XClass == nil {
				continue
			}
			everyone[standing(l.ToRank)]++
			if of(*l.XClass) == kind {
				mine = true
				sides[l.Side] = true
				buyers[standing(l.ToRank)]++
			}
		}
		if !mine {
			continue
		}
		moved++
		for _, l := range legs {
			if l.XClass == nil || sides[l.Side] || l.XValue == nil {
				continue
			}
			k := GroupOf(*l.XClass)
			if _, ok := back[k]; !ok {
				order = append(order, k)
			}
			back[k] += *l.XValue
			total += *l.XValue
		}
	}
	out := []Return{}
	for _, k := range order {
		r := Return{Kind: k, Wins: back[k]}
		if total != 0 {
			r.Share = r.Wins / total
		}
		out = append(out, r)
	}
	slices.SortStableFunc(out, func(a, b Return) int {
		if a.Wins > b.Wins {
			return -1
		}
		if a.Wins < b.Wins {
			return 1
		}
		return 0
	})
	return out, Buyers{share(buyers), share(everyone), buyers[0] + buyers[1] + buyers[2]}, moved
}
func AssetProfile(d *Data, s Scope, kind string, stretches []Stretch) *Profile {
	trades := ValuedTrades(d, InScope(s))
	if len(trades) < MinWhole {
		return nil
	}
	of := classifier(s.ByAge)
	fit := FitPrices(regroup(trades, of))
	price, ok := fit.Prices[kind]
	if !ok {
		return &Profile{Kind: kind}
	}
	n := legCounts(trades, of)
	typical := typicalOf(FitPrices(regroup(trades, GroupOf)), legCounts(trades, GroupOf), true)
	delivered := DeliveredRatio(recordsFor(d, InScope(s), of, kind))
	p := &Profile{Kind: kind, N: n[kind], Trades: len(trades), Delivered: delivered, When: profileWhen(kind, stretches), Against: profileAgainst(d, s, kind, fit, n, delivered)}
	if typical != nil && *typical != 0 {
		p.Paid = relative(price, *typical)
	}
	p.Back, p.Buyers, p.Moved = profileReturns(d, trades, kind, of)
	return p
}
