package lab

import (
	"fmt"
	"slices"
	"time"
)

type PhaseCount struct {
	N         int     `json:"n"`
	Months    [12]int `json:"months"`
	PerSeason float64 `json:"perSeason"`
	Span      [2]int  `json:"span"`
}
type PhaseTotals struct {
	Phases  map[string]PhaseCount `json:"phases"`
	Order   []string              `json:"order"`
	Seasons int                   `json:"seasons"`
	Total   int                   `json:"total"`
}

func TradesByPhase(d *Data, s Scope) (PhaseTotals, error) {
	zone, err := time.LoadLocation("America/New_York")
	if err != nil {
		return PhaseTotals{}, fmt.Errorf("load Eastern calendar: %w", err)
	}
	out := PhaseTotals{Phases: make(map[string]PhaseCount), Order: []string{}}
	years := make(map[int]bool)
	for _, t := range d.Trades {
		if t.OneSided || t.Year < s.From || t.Year > s.To || t.Phase == "" || !moves(t, s.Assets) {
			continue
		}
		p, exists := out.Phases[t.Phase]
		if !exists {
			out.Order = append(out.Order, t.Phase)
		}
		p.N++
		month := int(t.Date.In(zone).Month()) - 1
		p.Months[month]++
		out.Phases[t.Phase] = p
		years[t.Year] = true
		out.Total++
	}
	out.Seasons = len(years)
	for k, p := range out.Phases {
		p.PerSeason = float64(p.N) / float64(max(1, out.Seasons))
		held := []int{}
		for i, n := range p.Months {
			if float64(n) >= 0.1*float64(p.N) {
				held = append(held, i)
			}
		}
		if len(held) > 0 {
			p.Span = [2]int{held[0], held[len(held)-1]}
		}
		out.Phases[k] = p
	}
	return out, nil
}
func moves(t Trade, assets []string) bool {
	if assets == nil {
		return true
	}
	for _, l := range t.Legs {
		if l.XClass != nil && slices.Contains(assets, AssetOf(*l.XClass)) {
			return true
		}
	}
	return false
}

type ClockValue struct {
	Value float64 `json:"value"`
	Lo    float64 `json:"lo"`
	Hi    float64 `json:"hi"`
	Clear bool    `json:"clear"`
}
type ClockHalf struct {
	From   int         `json:"from"`
	To     int         `json:"to"`
	Trades int         `json:"trades"`
	Clock  *ClockValue `json:"clock"`
}
type Clock struct {
	Halves []ClockHalf `json:"halves"`
}

func PickClock(d *Data, s Scope) *Clock {
	trades := ValuedTrades(d, InScope(s))
	if len(trades) < MinWhole {
		return nil
	}
	seasons, mid := seasonSplit(d, trades)
	out := &Clock{Halves: []ClockHalf{}}
	for _, ab := range [][2]int{{seasons[0], mid}, {mid + 1, seasons[len(seasons)-1]}} {
		if ab[0] > ab[1] {
			continue
		}
		sub := []ValuedTrade{}
		for _, v := range trades {
			year := *d.Trades[v.ID].Season
			if year >= ab[0] && year <= ab[1] {
				sub = append(sub, v)
			}
		}
		h := ClockHalf{From: ab[0], To: ab[1], Trades: len(sub)}
		if len(sub) >= MinPart {
			if p, ok := FitPrices(sub).Prices["pick1:later"]; ok {
				h.Clock = &ClockValue{p.Price, p.Price - Z*p.SE, p.Price + Z*p.SE, p.Price+Z*p.SE < 1 && p.Price > 0}
			}
		}
		out.Halves = append(out.Halves, h)
	}
	return out
}
