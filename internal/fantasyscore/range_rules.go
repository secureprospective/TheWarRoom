package fantasyscore

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"strconv"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// withPlayCounts records each stat the range rules score that is counted from other stats' plays,
// such as completions of 40 yards or more, when every stat it counts from was recorded play by play.
func withPlayCounts(rules leaguerules.ScoringRules, line StatLine) StatLine {
	if rules.Rules.State != leaguerules.IsKnown || len(rules.Rules.Value) == 0 {
		return line
	}
	result := StatLine{Quantities: maps.Clone(line.Quantities), Events: line.Events, Labels: line.Labels}
	if result.Quantities == nil {
		result.Quantities = map[leaguerules.StatID]leaguerules.Decimal{}
	}
	for _, id := range slices.Sorted(maps.Keys(rules.Rules.Value)) {
		def, known := leaguerules.StatByID(id)
		_, recorded := line.Quantities[id]
		if !known || len(def.Plays) == 0 || recorded || line.Events[id] != nil {
			continue
		}
		count, complete := leaguerules.Decimal(0), true
		for _, from := range def.Plays {
			plays, held := line.Events[from]
			if !held {
				complete = false
				break
			}
			for _, distance := range plays {
				if distance >= def.AtLeast {
					count += 10000
				}
			}
		}
		if complete {
			result.Quantities[id] = count
		}
	}
	return result
}

// rangeRuleItems prices a recorded stat under each range rule that names the position: one line per
// rule, so each is rounded on its own as MFL rounds each rule.
func rangeRuleItems(rules leaguerules.ScoringRules, id leaguerules.StatID, position leaguerules.Position,
	line StatLine) ([]LineItem, error) {
	if rules.Rules.State != leaguerules.IsKnown {
		return nil, nil
	}
	def, _ := leaguerules.StatByID(id)
	label := cmp.Or(def.Label, line.Labels[id], string(id))
	quantity, err := recordedQuantity(id, line)
	if err != nil {
		return nil, err
	}
	plays, byPlay := line.Events[id]
	var items []LineItem
	for _, rule := range rules.Rules.Value[id] {
		if len(rule.Positions) != 0 && !slices.Contains(rule.Positions, position) {
			continue
		}
		item := LineItem{Stat: id, Rule: id, Quantity: quantity, Label: label + " (" + rangeWords(rule.Range) + ")"}
		var points leaguerules.Decimal
		switch {
		case def.Lengths && byPlay:
			if points, err = playAwards(rule, plays); err != nil {
				return nil, err
			}
		case quantity < rule.Range.Min || quantity > rule.Range.Max:
			item.OutOfRange = true
		default:
			if points, err = ruleAward(rule, quantity); err != nil {
				return nil, err
			}
		}
		item.Points = leaguerules.Known(points)
		items = append(items, item)
	}
	return items, nil
}

// playAwards sums a rule's award over each play whose length falls in its range.
func playAwards(rule leaguerules.RangeRule, plays []leaguerules.Decimal) (leaguerules.Decimal, error) {
	points := leaguerules.Decimal(0)
	for _, distance := range plays {
		if distance < rule.Range.Min || distance > rule.Range.Max {
			continue
		}
		award, err := ruleAward(rule, distance)
		if err != nil {
			return 0, err
		}
		if points, err = add(points, award); err != nil {
			return 0, err
		}
	}
	return points, nil
}

// ruleAward is one rule's points for a value: the flat award, the rate for each one, or the award for
// each whole Per of the value.
func ruleAward(rule leaguerules.RangeRule, value leaguerules.Decimal) (leaguerules.Decimal, error) {
	switch {
	case rule.Flat:
		return rule.Points, nil
	case rule.Per == 0:
		return multiply(value, rule.Points)
	default:
		points, err := multiply(value/rule.Per*10000, rule.Points)
		if err != nil {
			return 0, fmt.Errorf("range rule: %w", err)
		}
		return points, nil
	}
}

func rangeWords(r leaguerules.StatRange) string {
	return decimalWords(r.Min) + " to " + decimalWords(r.Max)
}

func decimalWords(d leaguerules.Decimal) string {
	return strconv.FormatFloat(float64(d)/10000, 'f', -1, 64)
}
