package fantasyscore

import (
	"fmt"
	"maps"
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// UnitLine is one NFL team's offensive-line components for one team-game. Played false is a bye or
// no game, which scores 0; an absent component is missing evidence, not zero.
type UnitLine struct {
	Played     bool
	Quantities map[leaguerules.StatID]leaguerules.Decimal
}

// ScoreUnit prices a team-game under the league's offensive line rules: each weighted term rounded
// to the cent, the no-sack bonus, then the clamp, shown as its own line so the breakdown adds up.
func ScoreUnit(rules leaguerules.ScoringRules, line UnitLine) (Points, []LineItem, error) {
	if rules.LineUnit.State != leaguerules.IsKnown {
		return leaguerules.Needs[leaguerules.Decimal]("Team offensive line scoring: " + rules.LineUnit.Citation), nil, nil
	}
	unit := rules.LineUnit.Value
	if !unit.Used {
		return Points{}, nil, fmt.Errorf("this league does not score an offensive line")
	}
	if !line.Played {
		return leaguerules.Known[leaguerules.Decimal](0), nil, nil
	}
	items, total, missing, err := unitTerms(unit, line)
	if err != nil {
		return Points{}, nil, err
	}
	if len(missing) != 0 {
		return leaguerules.Needs[leaguerules.Decimal](joinReasons(missing)), items, nil
	}
	if sacks := line.Quantities[unit.Sacks]; unit.NoSack != 0 && sacks == 0 {
		if total, err = add(total, unit.NoSack); err != nil {
			return Points{}, nil, err
		}
		items = append(items, LineItem{Stat: "line_no_sack", Rule: "line_no_sack", Quantity: 10000,
			Label: "No sacks allowed", Points: leaguerules.Known(unit.NoSack)})
	}
	if unit.Clamped && (total > unit.Clamp || total < -unit.Clamp) {
		clamped := max(-unit.Clamp, min(unit.Clamp, total))
		items = append(items, LineItem{Stat: "line_clamp", Rule: "line_clamp",
			Label: "Held to the game limit", Points: leaguerules.Known(clamped - total)})
		total = clamped
	}
	return leaguerules.Known(total), items, nil
}

// unitTerms rounds each weighted component to the cent; unweighted components are shown as unused.
func unitTerms(unit leaguerules.LineUnit, line UnitLine) ([]LineItem, leaguerules.Decimal, []string, error) {
	var items []LineItem
	var missing []string
	total := leaguerules.Decimal(0)
	for _, id := range slices.Sorted(maps.Keys(unit.Weights)) {
		quantity, recorded := line.Quantities[id]
		if !recorded {
			missing = append(missing, string(id)+" not recorded yet")
			continue
		}
		points, err := multiply(quantity, unit.Weights[id])
		if err == nil {
			points, err = roundCent(points)
		}
		if err == nil {
			total, err = add(total, points)
		}
		if err != nil {
			return nil, 0, nil, fmt.Errorf("count %s: %w", id, err)
		}
		items = append(items, LineItem{Stat: id, Rule: id, Quantity: quantity, Label: unitLabel(unit, id), Points: leaguerules.Known(points)})
	}
	for _, id := range slices.Sorted(maps.Keys(line.Quantities)) {
		if _, weighted := unit.Weights[id]; !weighted {
			items = append(items, LineItem{Stat: id, Quantity: line.Quantities[id], Label: unitLabel(unit, id), Unused: true})
		}
	}
	return items, total, missing, nil
}

func unitLabel(unit leaguerules.LineUnit, id leaguerules.StatID) string {
	if label := unit.Labels[id]; label != "" {
		return label
	}
	return string(id)
}
