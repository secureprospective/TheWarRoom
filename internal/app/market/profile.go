package market

import (
	"fmt"
	"slices"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/lab"
)

// ValidateAsset checks an asset profile request: a valid scope and a known kind, an age split
// of a position only when the scope splits by age.
func ValidateAsset(scope Scope, kind string) error {
	if err := scope.Validate(); err != nil {
		return fmt.Errorf("asset profile: %w", err)
	}
	if slices.Contains(Assets(), kind) {
		return nil
	}
	parts := strings.Split(kind, ":")
	if scope.ByAge && len(parts) == 2 && slices.Contains(Assets()[:8], parts[0]) && slices.Contains([]string{"young", "prime", "older"}, parts[1]) {
		return nil
	}
	return fmt.Errorf("asset profile: invalid kind %q for this age setting", kind)
}

// Profile is one asset's market profile: why it is cheap or dear, its stretches, its price
// against every other asset, what came back for it and who buys it.
func (l Lab) Profile(raw Scope, kind string) report.Reading[Surface] {
	d := l.Data
	s := LabScope(raw)
	if slices.Contains(Assets()[:8], kind) {
		s.ByAge = false
	}
	stretches := lab.StretchPrices(d, s, Phases())
	p := lab.AssetProfile(d, s, kind, stretches)
	value := Surface{Summary: Name(kind), Sections: []Section{}}
	count := len(lab.ValuedTrades(d, lab.InScope(s)))
	reason := ""
	switch {
	case p == nil:
		reason = "Fewer than 80 valued trades; widen the seasons or stretches"
	case p.N == 0:
		reason = "This asset was not traded in the chosen scope"
	default:
		value.Sections = profileSections(p, profileDeliveryCount(d, s, kind))
	}
	r := l.surfaceReading(raw, value, count, reason)
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
func profileSections(p *lab.Profile, deliveryCount int) []Section {
	why := Section{ID: "L-09", Title: "Why cheap or dear", Note: "Paid forecast wins and delivered wins are kept separate.", Rows: []Line{{Label: "Paid", Count: p.N, Range: RangeOf(p.Paid), Detail: "Relative to the typical asset's forecast-win price"}}}
	if p.Kind == lab.Anchor {
		why.Rows[0].Range = nil
		why.Rows[0].Detail = "Every other price is measured against this one; it cannot be cheap or dear"
	}
	delivered := Line{Label: "Delivered", Count: deliveryCount, Detail: "Fewer than 8 played-out records or less than 0.5 forecast wins; no delivery estimate"}
	if p.Delivered != nil {
		d := p.Delivered
		delivered.Count = d.N
		delivered.Range = RangeOf(&lab.Interval{Value: d.Ratio, Lo: d.Ratio - lab.Z*d.SE, Hi: d.Ratio + lab.Z*d.SE})
		delivered.Detail = "Wins delivered relative to the forecast at trade time"
	}
	why.Rows = append(why.Rows, delivered)
	when := Section{ID: "L-10", Title: "Cheapest / dearest stretches", Note: "Only beyond-chance extremes are marked. All stretches of the chosen seasons are compared.", Rows: []Line{}}
	for _, w := range p.When {
		line := Line{Label: PhaseName(w.Phase), Count: w.N, Detail: fmt.Sprintf("%s in this stretch; too few or too uncertain to price", report.Count(w.Trades, "valued trade"))}
		if w.Paid != nil {
			line.Range = RangeOf(&lab.Interval{Value: w.Paid.Value, Lo: w.Paid.Lo, Hi: w.Paid.Hi})
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
	against := Section{ID: "L-11", Title: "Against every other asset", Note: "100% means the same cost. Forecast comparison is labelled when delivery is unavailable. Other assets with fewer than 25 legs cannot be compared.", Rows: []Line{}}
	for _, x := range p.Against {
		basis := "forecast wins"
		if x.ByDelivered {
			basis = "delivered wins"
		}
		against.Rows = append(against.Rows, Line{Key: x.Kind, Label: Name(x.Kind), Count: x.N, Range: RangeOf(&lab.Interval{Value: x.Ratio, Lo: x.Lo, Hi: x.Hi}), Detail: x.Side + " · " + basis})
	}
	back := Section{ID: "L-12", Title: "What came back", Note: fmt.Sprintf("Opposite-side forecast wins across %s moving this asset. These are not realised returns.", report.Count(p.Moved, "trade")), Rows: []Line{}}
	for _, x := range p.Back {
		back.Rows = append(back.Rows, Line{Key: x.Kind, Label: Name(x.Kind), Detail: fmt.Sprintf("%s of returned forecast wins · %.3f wins", Percent(x.Share), x.Wins)})
	}
	buyers := Section{ID: "L-13", Title: "Who buys it", Note: "Shares by standing at the time; the whole league is the baseline, not a claim of statistical significance.", Rows: []Line{}}
	mine := []float64{p.Buyers.Mine.Contender, p.Buyers.Mine.Middle, p.Buyers.Mine.Rebuilder}
	all := []float64{p.Buyers.All.Contender, p.Buyers.All.Middle, p.Buyers.All.Rebuilder}
	for i, label := range []string{"Contenders · ranks 1–10", "Middle · ranks 11–22", "Rebuilders · ranks 23–32"} {
		buyers.Rows = append(buyers.Rows, Line{Label: label, Count: p.Buyers.N, Detail: Percent(mine[i]) + " of this asset; " + Percent(all[i]) + " of all traded assets"})
	}
	return []Section{why, when, against, back, buyers}
}
