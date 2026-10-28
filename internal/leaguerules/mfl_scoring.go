package leaguerules

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
)

// MFLRuleBlock is one MFL positionRules block as held: its pipe-joined positions and its rules.
type MFLRuleBlock struct {
	Positions string
	Rules     []MFLRule
}

// MFLRule is one raw MFL scoring rule: Event "PY", Points "*.05", Range "-50-999".
type MFLRule struct {
	Event, Points, Range string
}

// FromMFLScoring carries an MFL league's scoring into commissioner settings rule for rule: every rule
// becomes a range rule on its stat, with its positions, points and range as MFL holds them. Nothing
// is dropped or rounded; an event code the dictionary does not know yet is kept under its own code.
// The only refusal is a value MFL itself could not have meant: unreadable points or range.
func FromMFLScoring(blocks []MFLRuleBlock, citation string) (ScoringRules, error) {
	rules := map[StatID][]RangeRule{}
	totals := map[StatID][]StatID{}
	for _, block := range blocks {
		positions, err := mflPositions(block.Positions)
		if err != nil {
			return ScoringRules{}, err
		}
		for _, raw := range block.Rules {
			rule, err := mflRule(raw)
			if err != nil {
				return ScoringRules{}, fmt.Errorf("MFL rule %s for %s: %w", raw.Event, block.Positions, err)
			}
			rule.Positions = positions
			id := StatID("mfl_" + strings.TrimSpace(raw.Event))
			if def, ok := StatByMFL(strings.TrimSpace(raw.Event)); ok {
				id = def.ID
				if len(def.Parts) != 0 {
					totals[id] = slices.Clone(def.Parts)
				}
			}
			rules[id] = append(rules[id], rule)
		}
	}
	var r ScoringRules
	emptyScoringKinds(&r, citation)
	r.Catalogue = cited(map[StatID]Stat{}, citation)
	r.RoundStatPoints = cited(true, citation)
	r.Totals = cited(totals, citation)
	r.Rules = cited(rules, citation)
	return r, nil
}

func mflPositions(raw string) ([]Position, error) {
	var out []Position
	for _, p := range strings.Split(raw, "|") {
		position := Position(strings.TrimSpace(p))
		if validateMenu("Position", string(position)) != nil {
			return nil, fmt.Errorf("MFL rule block names an unknown position %q", p)
		}
		out = append(out, position)
	}
	return out, nil
}

func mflRule(raw MFLRule) (RangeRule, error) {
	var rule RangeRule
	points := strings.TrimSpace(raw.Points)
	var err error
	switch {
	case strings.HasPrefix(points, "*"):
		rule.Points, err = mflDecimal(points[1:])
	case strings.Contains(points, "/"):
		award, per, _ := strings.Cut(points, "/")
		if rule.Points, err = mflDecimal(award); err == nil {
			rule.Per, err = mflDecimal(per)
		}
		if err == nil && rule.Per <= 0 {
			err = fmt.Errorf("points per a unit need a unit above zero")
		}
	default:
		rule.Flat = true
		rule.Points, err = mflDecimal(points)
	}
	if err != nil {
		return RangeRule{}, err
	}
	low, high, ok := cutRange(strings.TrimSpace(raw.Range))
	if !ok {
		return RangeRule{}, fmt.Errorf("read the range %q", raw.Range)
	}
	if rule.Range.Min, err = mflDecimal(low); err != nil {
		return RangeRule{}, err
	}
	if rule.Range.Max, err = mflDecimal(high); err != nil {
		return RangeRule{}, err
	}
	return rule, nil
}

// cutRange splits "low-high" where either end may be negative, such as "-50-999".
func cutRange(raw string) (string, string, bool) {
	for i := 1; i < len(raw); i++ {
		if raw[i] == '-' && raw[i-1] != '-' {
			return raw[:i], raw[i+1:], raw[i+1:] != ""
		}
	}
	return "", "", false
}

// mflDecimal reads an MFL number exactly in ten-thousandths, the engine's unit; a finer number
// cannot be counted exactly and is refused rather than rounded.
func mflDecimal(raw string) (Decimal, error) {
	raw = strings.TrimSpace(raw)
	negative := strings.HasPrefix(raw, "-")
	whole, frac, _ := strings.Cut(strings.TrimPrefix(raw, "-"), ".")
	if whole == "" && frac == "" || len(frac) > 4 {
		return 0, fmt.Errorf("read %q as a number to four decimal places", raw)
	}
	frac += strings.Repeat("0", 4-len(frac))
	if whole == "" {
		whole = "0"
	}
	n, err := strconv.ParseInt(whole+frac, 10, 64)
	if err != nil || n > math.MaxInt64/2 {
		return 0, fmt.Errorf("read %q as a number to four decimal places", raw)
	}
	if negative {
		n = -n
	}
	return Decimal(n), nil
}
