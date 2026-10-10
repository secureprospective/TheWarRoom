package main

import (
	"fmt"
	"slices"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/lab"
)

type RawMarketAsset struct {
	Scope RawMarketScope `json:"scope"`
	Kind  string         `json:"kind"`
}

func (r RawMarketAsset) Validate() error {
	if err := r.Scope.Validate(); err != nil {
		return fmt.Errorf("asset profile: %w", err)
	}
	if slices.Contains(marketAssets(), r.Kind) {
		return nil
	}
	parts := strings.Split(r.Kind, ":")
	if r.Scope.ByAge && len(parts) == 2 && slices.Contains(marketAssets()[:8], parts[0]) && slices.Contains([]string{"young", "prime", "older"}, parts[1]) {
		return nil
	}
	return fmt.Errorf("asset profile: invalid kind %q for this age setting", r.Kind)
}
func (a *App) TargetAssetProfile(raw RawMarketAsset) Reading[MarketSurface] {
	if err := raw.Validate(); err != nil {
		return failedSurface(err)
	}
	d, err := a.lab()
	if err != nil {
		return failedSurface(err)
	}
	s := marketScope(raw.Scope)
	if slices.Contains(marketAssets()[:8], raw.Kind) {
		s.ByAge = false
	}
	stretches := lab.StretchPrices(d, s, marketPhases())
	p := lab.AssetProfile(d, s, raw.Kind, stretches)
	value := MarketSurface{Summary: marketName(raw.Kind), Sections: []MarketSection{}}
	count := len(lab.ValuedTrades(d, lab.InScope(s)))
	reason := ""
	switch {
	case p == nil:
		reason = "Fewer than 80 valued trades; widen the seasons or stretches"
	case p.N == 0:
		reason = "This asset was not traded in the chosen scope"
	default:
		value.Sections = profileSections(p, profileDeliveryCount(d, s, raw.Kind))
	}
	r := surfaceReading(d, raw.Scope, value, count, reason)
	r.HowCounted += " Asset comparisons need 25 legs of the other kind. Stretch prices need 60 trades, 12 legs of this kind, and a bounded price error; only a stretch beating every other known stretch beyond chance is marked. Returns are the opposite side's forecast wins, not realised delivery. Buyers are franchise standing at trade time, not today's owner identity."
	return r
}
func profileDeliveryCount(d *lab.Data, s lab.Scope, kind string) int {
	count := 0
	keep := lab.InScope(s)
	for _, r := range d.Delivered {
		grouped := r.Class
		if !s.ByAge {
			grouped = lab.GroupOf(grouped)
		}
		if grouped == kind && keep(d.Trades[r.Trade]) {
			count++
		}
	}
	return count
}
func profileSections(p *lab.Profile, deliveryCount int) []MarketSection {
	why := MarketSection{ID: "L-09", Title: "Why cheap or dear", Note: "Paid forecast wins and delivered wins are kept separate.", Rows: []MarketLine{{Label: "Paid", Count: p.N, Range: marketRange(p.Paid), Detail: "Relative to the typical asset's forecast-win price"}}}
	delivered := MarketLine{Label: "Delivered", Count: deliveryCount, Detail: "Fewer than 8 played-out records or less than 0.5 forecast wins; no delivery estimate"}
	if p.Delivered != nil {
		d := p.Delivered
		delivered.Count = d.N
		delivered.Range = marketRange(&lab.Interval{Value: d.Ratio, Lo: d.Ratio - lab.Z*d.SE, Hi: d.Ratio + lab.Z*d.SE})
		delivered.Detail = "Wins delivered relative to the forecast at trade time"
	}
	why.Rows = append(why.Rows, delivered)
	when := MarketSection{ID: "L-10", Title: "Cheapest / dearest stretches", Note: "Only beyond-chance extremes are marked. All stretches of the chosen seasons are compared.", Rows: []MarketLine{}}
	for _, w := range p.When {
		line := MarketLine{Label: marketPhaseName(w.Phase), Count: w.N, Detail: fmt.Sprintf("%d valued trades in this stretch; too few or too uncertain to price", w.Trades)}
		if w.Paid != nil {
			line.Range = marketRange(&lab.Interval{Value: w.Paid.Value, Lo: w.Paid.Lo, Hi: w.Paid.Hi})
			line.Detail = "Against the typical asset in this stretch; no beyond-chance extreme"
		}
		if w.Cheapest {
			line.Mark = "cheapest"
			line.Detail = "Cheapest beyond chance"
		}
		if w.Dearest {
			line.Mark = "dearest"
			line.Detail = "Dearest beyond chance"
		}
		when.Rows = append(when.Rows, line)
	}
	against := MarketSection{ID: "L-11", Title: "Against every other asset", Note: "100% means the same cost. Forecast comparison is labelled when delivery is unavailable. Other assets with fewer than 25 legs cannot be compared.", Rows: []MarketLine{}}
	for _, x := range p.Against {
		basis := "forecast wins"
		if x.ByDelivered {
			basis = "delivered wins"
		}
		against.Rows = append(against.Rows, MarketLine{Key: x.Kind, Label: marketName(x.Kind), Count: x.N, Range: marketRange(&lab.Interval{Value: x.Ratio, Lo: x.Lo, Hi: x.Hi}), Detail: x.Side + " · " + basis})
	}
	back := MarketSection{ID: "L-12", Title: "What came back", Note: fmt.Sprintf("Opposite-side forecast wins across %d trades moving this asset. These are not realised returns.", p.Moved), Rows: []MarketLine{}}
	for _, x := range p.Back {
		back.Rows = append(back.Rows, MarketLine{Key: x.Kind, Label: marketName(x.Kind), Detail: fmt.Sprintf("%s of returned forecast wins · %.3f wins", percent(x.Share), x.Wins)})
	}
	buyers := MarketSection{ID: "L-13", Title: "Who buys it", Note: "Shares by standing at the time; the whole league is the baseline, not a claim of statistical significance.", Rows: []MarketLine{}}
	mine := []float64{p.Buyers.Mine.Contender, p.Buyers.Mine.Middle, p.Buyers.Mine.Rebuilder}
	all := []float64{p.Buyers.All.Contender, p.Buyers.All.Middle, p.Buyers.All.Rebuilder}
	for i, label := range []string{"Contenders · ranks 1–10", "Middle · ranks 11–22", "Rebuilders · ranks 23–32"} {
		buyers.Rows = append(buyers.Rows, MarketLine{Label: label, Count: p.Buyers.N, Detail: percent(mine[i]) + " of this asset; " + percent(all[i]) + " of all traded assets"})
	}
	return []MarketSection{why, when, against, back, buyers}
}
