package fantasyscore

import (
	"fmt"
	"maps"
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func kindsReason(rules leaguerules.ScoringRules) string {
	for _, row := range []struct {
		state    leaguerules.State
		citation string
		label    string
	}{{rules.Extras.State, rules.Extras.Citation, "Choose the extra points by position"},
		{rules.Ranges.State, rules.Ranges.Citation, "Choose the scoring ranges"},
		{rules.Scales.State, rules.Scales.Citation, "Choose the points by play distance"},
		{rules.Totals.State, rules.Totals.Citation, "Choose the stats counted as a total of others"}} {
		if row.state != leaguerules.IsKnown {
			return row.label + ": " + row.citation
		}
	}
	return ""
}

// withTotals adds each total stat whose parts were all recorded; a missing part leaves it absent.
func withTotals(totals map[leaguerules.StatID][]leaguerules.StatID, line StatLine) (StatLine, error) {
	if len(totals) == 0 {
		return line, nil
	}
	result := StatLine{Quantities: maps.Clone(line.Quantities), Events: line.Events, Labels: line.Labels}
	if result.Quantities == nil {
		result.Quantities = map[leaguerules.StatID]leaguerules.Decimal{}
	}
	for _, id := range slices.Sorted(maps.Keys(totals)) {
		if _, recorded := line.Quantities[id]; recorded {
			return StatLine{}, fmt.Errorf("%s is counted from its parts; do not record it", id)
		}
		if _, recorded := line.Events[id]; recorded {
			return StatLine{}, fmt.Errorf("%s is counted from its parts; do not record it", id)
		}
		sum, complete := leaguerules.Decimal(0), len(totals[id]) != 0
		for _, part := range totals[id] {
			quantity, recorded := line.Quantities[part]
			if !recorded {
				complete = false
				break
			}
			var err error
			if sum, err = add(sum, quantity); err != nil {
				return StatLine{}, err
			}
		}
		if complete {
			result.Quantities[id] = sum
		}
	}
	return result, nil
}

// scaledQuantity sums each play's priced distance for the week; the stat's rate then prices the sum,
// so rounding happens once for the week, not per play.
func scaledQuantity(scale leaguerules.DistanceScale, events []leaguerules.Decimal) (leaguerules.Decimal, error) {
	total := leaguerules.Decimal(0)
	for _, distance := range events {
		if distance < scale.From {
			continue
		}
		above, err := multiply(max(0, distance-scale.Pivot), scale.Above)
		if err != nil {
			return 0, err
		}
		below, err := multiply(max(0, scale.Pivot-distance), scale.Below)
		if err != nil {
			return 0, err
		}
		play := scale.Base
		for _, part := range []leaguerules.Decimal{above, below} {
			if play, err = add(play, part); err != nil {
				return 0, err
			}
		}
		if scale.Capped {
			play = min(play, scale.Cap)
		}
		if total, err = add(total, play); err != nil {
			return 0, err
		}
	}
	return total, nil
}

// extraItems prices each position extra that applies as its own line, rounded on its own as MFL
// rounds each overlapping rule.
func extraItems(rules leaguerules.ScoringRules, base LineItem, position leaguerules.Position) ([]LineItem, error) {
	if base.Unused || base.OutOfRange || base.Points.State != leaguerules.IsKnown {
		return nil, nil
	}
	quantity := base.Quantity
	if stat := rules.Catalogue.Value[base.Rule]; stat.Once.Value && quantity > 0 {
		quantity = 10000
	}
	var result []LineItem
	for _, extra := range rules.Extras.Value[base.Rule] {
		if !slices.Contains(extra.Positions, position) {
			continue
		}
		points, err := multiply(quantity, extra.Points)
		if err != nil {
			return nil, err
		}
		result = append(result, LineItem{Stat: base.Stat, Rule: base.Rule, Quantity: base.Quantity, Extra: true,
			Label: base.Label + " (" + string(position) + " extra)", Points: leaguerules.Known(points)})
	}
	return result, nil
}
