// Package fantasyscore counts recorded statistics under commissioner-owned rules without I/O.
package fantasyscore

import (
	"fmt"
	"math"
	"math/big"
	"slices"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// StatLine distinguishes absent evidence from recorded zero. Events retain exact play distances.
type StatLine struct {
	Quantities map[leaguerules.StatID]leaguerules.Decimal
	Events     map[leaguerules.StatID][]leaguerules.Decimal
	Labels     map[leaguerules.StatID]string
}

type Points = leaguerules.Value[leaguerules.Decimal]

// LineItem shows how one recorded stat was counted. Unused items carry no points, not a zero rule.
type LineItem struct {
	Stat     leaguerules.StatID
	Quantity leaguerules.Decimal
	Points   Points
	Label    string
	Unused   bool
	// Rule is the catalogue row that priced the stat; Extra marks a position's added rate on it.
	Rule       leaguerules.StatID
	Extra      bool
	OutOfRange bool
}

// Score totals a true position's stat line. Any unset rule a recorded stat needs makes the total
// NeedsRuling with every reason; malformed lines and unrepresentable arithmetic are errors.
func Score(rules leaguerules.ScoringRules, position leaguerules.Position, line StatLine) (Points, []LineItem, error) {
	if !validPosition(position) {
		return Points{}, nil, fmt.Errorf("choose a recognised true position")
	}
	if rules.Catalogue.State != leaguerules.IsKnown {
		return leaguerules.Needs[leaguerules.Decimal]("Scoring values: " + rules.Catalogue.Citation), nil, nil
	}
	if rules.RoundStatPoints.State != leaguerules.IsKnown {
		return leaguerules.Needs[leaguerules.Decimal]("Choose whether to round each stat’s points to the cent"), nil, nil
	}
	if reason := kindsReason(rules); reason != "" {
		return leaguerules.Needs[leaguerules.Decimal](reason), nil, nil
	}
	line, err := withTotals(rules.Totals.Value, line)
	if err != nil {
		return Points{}, nil, err
	}
	return countLine(rules, position, line)
}

// countLine prices every recorded stat and its position extras, each rounded on its own.
func countLine(rules leaguerules.ScoringRules, position leaguerules.Position, line StatLine) (Points, []LineItem, error) {
	ids := statIDs(line)
	items := make([]LineItem, 0, len(ids))
	for _, id := range ids {
		item, err := countStat(id, rules, position, line)
		if err != nil {
			return Points{}, nil, fmt.Errorf("count %s: %w", id, err)
		}
		extras, err := extraItems(rules, item, position)
		if err != nil {
			return Points{}, nil, fmt.Errorf("count %s: %w", id, err)
		}
		for _, item := range append([]LineItem{item}, extras...) {
			if rules.RoundStatPoints.Value && item.Points.State == leaguerules.IsKnown {
				if item.Points.Value, err = roundCent(item.Points.Value); err != nil {
					return Points{}, nil, err
				}
			}
			items = append(items, item)
		}
	}
	return totalItems(items)
}

func totalItems(items []LineItem) (Points, []LineItem, error) {
	total := leaguerules.Decimal(0)
	var reasons []string
	for _, item := range items {
		if item.Unused {
			continue
		}
		if item.Points.State != leaguerules.IsKnown {
			reasons = append(reasons, item.Points.Citation)
			continue
		}
		var err error
		if total, err = add(total, item.Points.Value); err != nil {
			return Points{}, nil, fmt.Errorf("total fantasy points: %w", err)
		}
	}
	if len(reasons) != 0 {
		return leaguerules.Needs[leaguerules.Decimal](joinReasons(reasons)), items, nil
	}
	return leaguerules.Known(total), items, nil
}

func statIDs(line StatLine) []leaguerules.StatID {
	seen := make(map[leaguerules.StatID]bool)
	for id := range line.Quantities {
		seen[id] = true
	}
	for id := range line.Events {
		seen[id] = true
	}
	ids := make([]leaguerules.StatID, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids
}

func countStat(id leaguerules.StatID, rules leaguerules.ScoringRules,
	position leaguerules.Position, line StatLine) (LineItem, error) {
	catalogue := rules.Catalogue.Value
	quantity, err := recordedQuantity(id, line)
	if err != nil {
		return LineItem{}, err
	}
	events, hasEvents := line.Events[id]
	label := line.Labels[id]
	if label == "" {
		label = strings.ReplaceAll(string(id), "_", " ")
	}
	item := LineItem{Stat: id, Quantity: quantity, Label: label}
	ruleID, reason := positionStat(id, position, catalogue)
	if reason != "" {
		item.Points = leaguerules.Needs[leaguerules.Decimal](item.Label + ": " + reason)
		return item, nil
	}
	stat, found := catalogue[ruleID]
	if !found {
		item.Unused = true
		return item, nil
	}
	if stat.Label.State == leaguerules.IsKnown {
		item.Label = stat.Label.Value
	}
	if reason := unresolved(stat); reason != "" {
		item.Points = leaguerules.Needs[leaguerules.Decimal](item.Label + ": " + reason)
		return item, nil
	}
	if !stat.Used.Value || (len(stat.Positions.Value) != 0 && !slices.Contains(stat.Positions.Value, position)) {
		item.Unused = true
		return item, nil
	}
	item.Rule = ruleID
	return priceStat(rules, stat, item, quantity, events, hasEvents)
}

// priceStat applies a stat's distance scale and range, then its rate, once and play bonuses.
func priceStat(rules leaguerules.ScoringRules, stat leaguerules.Stat, item LineItem, quantity leaguerules.Decimal,
	events []leaguerules.Decimal, hasEvents bool) (LineItem, error) {
	if scale, scaled := rules.Scales.Value[item.Rule]; scaled {
		if !hasEvents {
			return LineItem{}, fmt.Errorf("play distances required for a distance scale")
		}
		var err error
		if quantity, err = scaledQuantity(scale, events); err != nil {
			return LineItem{}, err
		}
		item.Quantity, events = quantity, nil
	}
	if bounds, ranged := rules.Ranges.Value[item.Rule]; ranged && (quantity < bounds.Min || quantity > bounds.Max) {
		item.Points, item.OutOfRange = leaguerules.Known[leaguerules.Decimal](0), true
		return item, nil
	}
	points, err := contribution(stat, quantity, events, hasEvents)
	if err != nil {
		return LineItem{}, err
	}
	item.Points = leaguerules.Known(points)
	return item, nil
}

func positionStat(id leaguerules.StatID, position leaguerules.Position,
	catalogue map[leaguerules.StatID]leaguerules.Stat) (leaguerules.StatID, string) {
	if id != "tackle" && id != "assist" && id != "pass_defensed" && id != "interception_caught" {
		return id, ""
	}
	var selected leaguerules.StatID
	for _, candidate := range []leaguerules.StatID{id, id + "_line", id + "_secondary"} {
		stat, found := catalogue[candidate]
		if !found || (stat.Used.State == leaguerules.IsKnown && !stat.Used.Value) {
			continue
		}
		if stat.Positions.State != leaguerules.IsKnown {
			return id, "Choose the scoring positions: " + stat.Positions.Citation
		}
		if len(stat.Positions.Value) != 0 && !slices.Contains(stat.Positions.Value, position) {
			continue
		}
		if selected != "" {
			return id, "Choose one " + string(id) + " rate for this position"
		}
		selected = candidate
	}
	if selected == "" {
		return id, ""
	}
	return selected, ""
}

func recordedQuantity(id leaguerules.StatID, line StatLine) (leaguerules.Decimal, error) {
	quantity, hasQuantity := line.Quantities[id]
	events, hasEvents := line.Events[id]
	if id == "tackle_line" || id == "assist_line" || strings.HasSuffix(string(id), "_secondary") {
		return 0, fmt.Errorf("record tackles and assists once; true position selects their rate")
	}
	if id == "" || (hasQuantity && hasEvents) {
		return 0, fmt.Errorf("stat must have one quantity or one event list")
	}
	if hasEvents {
		if int64(len(events)) > math.MaxInt64/10000 {
			return 0, fmt.Errorf("too many recorded plays")
		}
		quantity = leaguerules.Decimal(len(events)) * 10000
	}
	return quantity, nil
}

func unresolved(stat leaguerules.Stat) string {
	if stat.Used.State != leaguerules.IsKnown {
		return "Choose whether this stat is used: " + stat.Used.Citation
	}
	if !stat.Used.Value {
		return ""
	}
	if stat.Once.State != leaguerules.IsKnown {
		return "Choose whether this stat scores once: " + stat.Once.Citation
	}
	if stat.Label.State != leaguerules.IsKnown {
		return "Choose a stat name: " + stat.Label.Citation
	}
	if stat.Group.State != leaguerules.IsKnown {
		return "Choose a stat group: " + stat.Group.Citation
	}
	if stat.Points.State != leaguerules.IsKnown {
		return "Choose a scoring value: " + stat.Points.Citation
	}
	if stat.Positions.State != leaguerules.IsKnown {
		return "Choose the scoring positions: " + stat.Positions.Citation
	}
	if stat.Bonuses.State != leaguerules.IsKnown {
		return "Choose the play bonuses: " + stat.Bonuses.Citation
	}
	for _, bonus := range stat.Bonuses.Value {
		if bonus.At.State != leaguerules.IsKnown || bonus.Points.State != leaguerules.IsKnown {
			return "Choose a bonus distance and value: " + bonus.At.Citation + "; " + bonus.Points.Citation
		}
	}
	return ""
}

func countBonuses(points leaguerules.Decimal, events []leaguerules.Decimal,
	bonuses []leaguerules.Bonus) (leaguerules.Decimal, error) {
	for _, distance := range events {
		for _, bonus := range bonuses {
			if distance < bonus.At.Value {
				continue
			}
			var err error
			points, err = add(points, bonus.Points.Value)
			if err != nil {
				return 0, err
			}
		}
	}
	return points, nil
}

func multiply(quantity, rate leaguerules.Decimal) (leaguerules.Decimal, error) {
	var product, remainder big.Int
	product.Mul(big.NewInt(int64(quantity)), big.NewInt(int64(rate)))
	product.QuoRem(&product, big.NewInt(10000), &remainder)
	if remainder.Sign() != 0 {
		return 0, fmt.Errorf("points cannot be represented in ten-thousandths")
	}
	if !product.IsInt64() {
		return 0, fmt.Errorf("fantasy points exceed the supported range")
	}
	return leaguerules.Decimal(product.Int64()), nil
}

func add(a, b leaguerules.Decimal) (leaguerules.Decimal, error) {
	if (b > 0 && a > math.MaxInt64-b) || (b < 0 && a < math.MinInt64-b) {
		return 0, fmt.Errorf("fantasy points exceed the supported range")
	}
	return a + b, nil
}

func validPosition(p leaguerules.Position) bool {
	return slices.Contains([]leaguerules.Position{leaguerules.QB, leaguerules.RB, leaguerules.WR,
		leaguerules.TE, leaguerules.PK, leaguerules.DT, leaguerules.DE, leaguerules.LB,
		leaguerules.CB, leaguerules.S}, p)
}

func joinReasons(reasons []string) string {
	result := ""
	for _, reason := range reasons {
		if result != "" {
			result += "; "
		}
		result += reason
	}
	return result
}

func roundCent(points leaguerules.Decimal) (leaguerules.Decimal, error) {
	whole, remainder := points/100, points%100
	if remainder >= 50 {
		whole++
	}
	if remainder <= -50 {
		whole--
	}
	return multiply(whole, 1000000)
}

func contribution(stat leaguerules.Stat, quantity leaguerules.Decimal,
	events []leaguerules.Decimal, hasEvents bool) (leaguerules.Decimal, error) {
	if stat.Once.Value && quantity > 0 {
		quantity = 10000
	}
	points, err := multiply(quantity, stat.Points.Value)
	if err != nil {
		return 0, err
	}
	if len(stat.Bonuses.Value) != 0 && !hasEvents {
		return 0, fmt.Errorf("play distances required for bonuses")
	}
	return countBonuses(points, events, stat.Bonuses.Value)
}
