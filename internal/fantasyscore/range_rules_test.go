package fantasyscore

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// Every way MFL writes a rule scores as MFL counts it: a rate for each one, an award for each whole
// unit, a flat award inside a range, tiers that add up, negative and decimal values, positions, and
// ranges that apply to each play's length.
func TestRangeRules(t *testing.T) {
	type q = map[leaguerules.StatID]leaguerules.Decimal
	type e = map[leaguerules.StatID][]leaguerules.Decimal
	wide := leaguerules.StatRange{Min: -9990000, Max: 9990000}
	cases := []struct {
		name     string
		rules    map[leaguerules.StatID][]leaguerules.RangeRule
		position leaguerules.Position
		line     StatLine
		want     leaguerules.Decimal
	}{
		{"rate for each one", map[leaguerules.StatID][]leaguerules.RangeRule{"passing_yards": {{Points: 400, Range: wide}}},
			leaguerules.QB, StatLine{Quantities: q{"passing_yards": 2870000}}, 114800},
		{"award for each whole unit", map[leaguerules.StatID][]leaguerules.RangeRule{
			"passing_yards": {{Points: 10000, Per: 250000, Range: wide}}},
			leaguerules.QB, StatLine{Quantities: q{"passing_yards": 2870000}}, 110000},
		{"negative unit award", map[leaguerules.StatID][]leaguerules.RangeRule{
			"rushing_yards": {{Points: -10000, Per: 100000, Range: wide}}},
			leaguerules.RB, StatLine{Quantities: q{"rushing_yards": -250000}}, 20000},
		{"flat tiers add up", map[leaguerules.StatID][]leaguerules.RangeRule{"passing_yards": {
			{Points: 30000, Flat: true, Range: leaguerules.StatRange{Min: 3000000, Max: 9990000}},
			{Points: 20000, Flat: true, Range: leaguerules.StatRange{Min: 4000000, Max: 9990000}}}},
			leaguerules.QB, StatLine{Quantities: q{"passing_yards": 4100000}}, 50000},
		{"out of range scores nothing", map[leaguerules.StatID][]leaguerules.RangeRule{"passing_td": {
			{Points: 50000, Range: leaguerules.StatRange{Max: 20000}}}},
			leaguerules.QB, StatLine{Quantities: q{"passing_td": 30000}}, 0},
		{"zero is a value", map[leaguerules.StatID][]leaguerules.RangeRule{"passing_td": {{Points: 0, Range: wide}}},
			leaguerules.QB, StatLine{Quantities: q{"passing_td": 30000}}, 0},
		{"positions choose rules", map[leaguerules.StatID][]leaguerules.RangeRule{"tackle": {
			{Points: 15000, Range: wide}, {Positions: []leaguerules.Position{leaguerules.CB}, Points: 5000, Range: wide}}},
			leaguerules.CB, StatLine{Quantities: q{"tackle": 40000}}, 80000},
		{"other positions skip", map[leaguerules.StatID][]leaguerules.RangeRule{"tackle": {
			{Positions: []leaguerules.Position{leaguerules.CB}, Points: 5000, Range: wide}}},
			leaguerules.LB, StatLine{Quantities: q{"tackle": 40000}}, 0},
		{"length ranges apply per play", map[leaguerules.StatID][]leaguerules.RangeRule{"fg_made_distance": {
			{Points: 30000, Flat: true, Range: leaguerules.StatRange{Max: 390000}},
			{Points: 50000, Flat: true, Range: leaguerules.StatRange{Min: 400000, Max: 990000}}}},
			leaguerules.PK, StatLine{Events: e{"fg_made_distance": {250000, 450000, 520000}}}, 130000},
		{"length rate per yard", map[leaguerules.StatID][]leaguerules.RangeRule{"fg_made_distance": {
			{Points: 1000, Range: leaguerules.StatRange{Max: 990000}}}},
			leaguerules.PK, StatLine{Events: e{"fg_made_distance": {250000, 450000}}}, 70000},
		{"counted from play distances", map[leaguerules.StatID][]leaguerules.RangeRule{"long_pass_40": {
			{Points: 20000, Range: wide}}},
			leaguerules.QB, StatLine{Events: e{"long_pass": {390000, 400000, 650000}}}, 40000},
		{"stat no feed records yet", map[leaguerules.StatID][]leaguerules.RangeRule{"passer_rating": {
			{Points: 10000, Range: wide}}},
			leaguerules.QB, StatLine{Quantities: q{"passing_td": 10000}}, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var rules leaguerules.ScoringRules
			rules.Catalogue = leaguerules.Known(map[leaguerules.StatID]leaguerules.Stat{})
			rules.RoundStatPoints = leaguerules.Known(true)
			rules.Extras = leaguerules.Known(map[leaguerules.StatID][]leaguerules.PositionExtra{})
			rules.Ranges = leaguerules.Known(map[leaguerules.StatID]leaguerules.StatRange{})
			rules.Scales = leaguerules.Known(map[leaguerules.StatID]leaguerules.DistanceScale{})
			rules.Totals = leaguerules.Known(map[leaguerules.StatID][]leaguerules.StatID{})
			rules.Rules = leaguerules.Known(c.rules)
			assertScore(t, rules, c.position, c.line, c.want)
		})
	}
}
