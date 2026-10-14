package fantasyscore

import (
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func TestLegacyRatesAndTruePositions(t *testing.T) {
	rates := map[leaguerules.StatID]leaguerules.Decimal{
		"passing_td": 50000, "passing_yards": 500, "interception_thrown": -20000,
		"passing_two_point": 20000, "rushing_td": 60000, "rushing_yards": 1000,
		"rushing_attempt": 1500, "rushing_two_point": 20000, "receiving_td": 60000,
		"receiving_yards": 1000, "reception": 10000, "receiving_two_point": 20000,
		"fg_made_0_39": 30000, "fg_made_40_49": 40000, "fg_made_50_59": 50000,
		"fg_made_60_69": 60000, "fg_missed_0_29": -30000, "fg_missed_30_99": -10000,
		"punt_return_td": 60000, "punt_return_yards": 250, "kickoff_return_td": 100000,
		"kickoff_return_yards": 250, "fumble_lost": -20000, "own_fumble_recovery": 20000,
		"opponent_fumble_recovery": 30000, "opponent_fumble_recovery_yards": 250,
		"forced_fumble": 40000, "interception_return_td": 60000, "interception_caught": 50000,
		"interception_return_yards": 250, "pass_defensed": 25000, "qb_hit": 10000,
		"blocked_fg": 70000, "blocked_punt": 70000, "blocked_extra_point": 70000,
		"sack": 45000, "tackle_for_loss": 25000, "safety": 100000,
	}
	for id, rate := range rates {
		t.Run(string(id), func(t *testing.T) {
			line := StatLine{Quantities: map[leaguerules.StatID]leaguerules.Decimal{id: 20000}}
			assertScore(t, leaguerules.LegacyNFL().Scoring, leaguerules.QB, line, 2*rate)
		})
	}
	for _, pos := range []leaguerules.Position{leaguerules.QB, leaguerules.RB, leaguerules.WR,
		leaguerules.TE, leaguerules.PK, leaguerules.LB, leaguerules.CB, leaguerules.S, leaguerules.DT, leaguerules.DE} {
		line := StatLine{Quantities: map[leaguerules.StatID]leaguerules.Decimal{
			"tackle": 20000, "assist": 10000,
			"tackle_for_loss": 10000, "sack": 5000}}
		want := leaguerules.Decimal(87500)
		if pos == leaguerules.DT || pos == leaguerules.DE {
			want = 112500
		}
		assertScore(t, leaguerules.LegacyNFL().Scoring, pos, line, want)
	}
}

func TestCommissionerPositionsChooseRatesWithoutLegacyHardcoding(t *testing.T) {
	rules := leaguerules.LegacyNFL().Scoring
	base := rules.Catalogue.Value["tackle"]
	base.Positions = leaguerules.Known([]leaguerules.Position{leaguerules.DT})
	rules.Catalogue.Value["tackle"] = base
	alternate := rules.Catalogue.Value["tackle_line"]
	alternate.Positions = leaguerules.Known([]leaguerules.Position{leaguerules.LB})
	rules.Catalogue.Value["tackle_line"] = alternate
	line := StatLine{Quantities: map[leaguerules.StatID]leaguerules.Decimal{"tackle": 10000}}
	assertScore(t, rules, leaguerules.DT, line, 15000)
	assertScore(t, rules, leaguerules.LB, line, 25000)
	base.Positions = leaguerules.Known([]leaguerules.Position{leaguerules.LB})
	rules.Catalogue.Value["tackle"] = base
	points, _, err := Score(rules, leaguerules.LB, line)
	if err != nil || points.State != leaguerules.NeedsRuling || points.Value != 0 ||
		!strings.Contains(points.Citation, "Choose one") {
		t.Fatalf("overlapping rates were silently selected or stacked: %+v %v", points, err)
	}
	alternate.Positions = leaguerules.Needs[[]leaguerules.Position]("position ruling")
	rules.Catalogue.Value["tackle_line"] = alternate
	points, _, err = Score(rules, leaguerules.DT, line)
	if err != nil || points.State != leaguerules.NeedsRuling || !strings.Contains(points.Citation, "position ruling") {
		t.Fatalf("unset rate positions were guessed: %+v %v", points, err)
	}
}

func TestLongPlayBoundariesStackAndRetainDistances(t *testing.T) {
	line := StatLine{Events: map[leaguerules.StatID][]leaguerules.Decimal{
		"long_pass":      {390000, 400000, 650000},
		"long_rush":      {190000, 200000, 390000, 400000, 650000},
		"long_reception": {190000, 200000, 390000, 400000, 650000}}}
	rules := leaguerules.LegacyNFL().Scoring
	assertScore(t, rules, leaguerules.WR, line, 140000)
	stat := rules.Catalogue.Value["long_reception"]
	stat.Bonuses = leaguerules.Known([]leaguerules.Bonus{{At: leaguerules.Known(leaguerules.Decimal(650000)),
		Points: leaguerules.Known(leaguerules.Decimal(30000))}})
	rules.Catalogue.Value["long_reception"] = stat
	assertScore(t, rules, leaguerules.WR, line, 110000)
}

func TestUnusedAndRulingsAreNotZeroRules(t *testing.T) {
	rules := leaguerules.LegacyNFL().Scoring
	rules.Catalogue.Value["pat_made"] = leaguerules.UnusedStat("Extra points made", "Kicking")
	line := StatLine{Quantities: map[leaguerules.StatID]leaguerules.Decimal{"pat_made": 20000, "unknown": 10000}}
	points, items, err := Score(rules, leaguerules.PK, line)
	if err != nil || points.Value != 0 || len(items) != 2 {
		t.Fatalf("points %+v items %+v error %v", points, items, err)
	}
	for _, item := range items {
		if !item.Unused || item.Points.State == leaguerules.IsKnown {
			t.Fatalf("unused stat became a zero rule: %+v", item)
		}
	}
	stat := rules.Catalogue.Value["pat_made"]
	stat.Used = leaguerules.Known(true)
	stat.Points = leaguerules.Known(leaguerules.Decimal(10000))
	rules.Catalogue.Value["pat_made"] = stat
	assertScore(t, rules, leaguerules.PK, line, 20000)
}

func TestEveryUnresolvedSettingReturnsReasonNotPartialTotal(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*leaguerules.Stat)
	}{
		{"used", func(s *leaguerules.Stat) { s.Used = leaguerules.Needs[bool]("test ruling") }},
		{"label", func(s *leaguerules.Stat) { s.Label = leaguerules.Needs[string]("test ruling") }},
		{"group", func(s *leaguerules.Stat) { s.Group = leaguerules.Needs[string]("test ruling") }},
		{"points", func(s *leaguerules.Stat) { s.Points = leaguerules.Needs[leaguerules.Decimal]("test ruling") }},
		{"positions", func(s *leaguerules.Stat) { s.Positions = leaguerules.Needs[[]leaguerules.Position]("test ruling") }},
		{"bonuses", func(s *leaguerules.Stat) { s.Bonuses = leaguerules.Needs[[]leaguerules.Bonus]("test ruling") }},
		{"threshold", func(s *leaguerules.Stat) {
			s.Bonuses.Value[0].At = leaguerules.Needs[leaguerules.Decimal]("test ruling")
		}},
		{"bonus value", func(s *leaguerules.Stat) {
			s.Bonuses.Value[0].Points = leaguerules.Needs[leaguerules.Decimal]("test ruling")
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rules := leaguerules.LegacyNFL().Scoring
			stat := rules.Catalogue.Value["long_rush"]
			tc.mutate(&stat)
			rules.Catalogue.Value["long_rush"] = stat
			line := StatLine{Quantities: map[leaguerules.StatID]leaguerules.Decimal{"rushing_td": 10000},
				Events: map[leaguerules.StatID][]leaguerules.Decimal{"long_rush": {400000}}}
			points, _, err := Score(rules, leaguerules.RB, line)
			if err != nil || points.State != leaguerules.NeedsRuling || points.Value != 0 ||
				!strings.Contains(points.Citation, "test ruling") {
				t.Fatalf("missing reason or leaked partial number: %+v %v", points, err)
			}
		})
	}
	points, _, err := Score(leaguerules.ScoringRules{}, leaguerules.QB, StatLine{})
	if err != nil || points.State != leaguerules.NeedsRuling || points.Citation == "" {
		t.Fatalf("unset catalogue became a number: %+v %v", points, err)
	}
}

func TestExactArithmeticRejectsOverflowAndLostPrecision(t *testing.T) {
	for _, line := range []StatLine{
		{Quantities: map[leaguerules.StatID]leaguerules.Decimal{"passing_td": math.MaxInt64}},
		{Quantities: map[leaguerules.StatID]leaguerules.Decimal{"passing_yards": 1}},
		{Quantities: map[leaguerules.StatID]leaguerules.Decimal{"long_rush": 10000}},
		{Quantities: map[leaguerules.StatID]leaguerules.Decimal{"reception": 10000},
			Events: map[leaguerules.StatID][]leaguerules.Decimal{"reception": {10000}}},
		{Quantities: map[leaguerules.StatID]leaguerules.Decimal{"": 10000}},
		{Quantities: map[leaguerules.StatID]leaguerules.Decimal{"tackle_line": 10000}},
	} {
		if _, _, err := Score(leaguerules.LegacyNFL().Scoring, leaguerules.QB, line); err == nil {
			t.Fatalf("accepted malformed/overflow line %+v", line)
		}
	}
	for _, pair := range [][2]leaguerules.Decimal{{math.MaxInt64, 1}, {math.MinInt64, -1}} {
		if _, err := add(pair[0], pair[1]); err == nil {
			t.Fatal("addition wrapped")
		}
	}
	assertScore(t, leaguerules.LegacyNFL().Scoring, leaguerules.RB,
		StatLine{Quantities: map[leaguerules.StatID]leaguerules.Decimal{"rushing_yards": -10000}}, -1000)
	if _, _, err := Score(leaguerules.LegacyNFL().Scoring, "DL", StatLine{}); err == nil {
		t.Fatal("ambiguous position accepted")
	}
}

func TestScoreIsDeterministicAndDoesNotMutateInput(t *testing.T) {
	rules := leaguerules.LegacyNFL().Scoring
	line := StatLine{Quantities: map[leaguerules.StatID]leaguerules.Decimal{"sack": 5000, "rushing_attempt": 30000}}
	p, items, err := Score(rules, leaguerules.DE, line)
	p2, items2, err2 := Score(rules, leaguerules.DE, line)
	if err != nil || err2 != nil || p != p2 || !reflect.DeepEqual(items, items2) ||
		line.Quantities["sack"] != 5000 || items[0].Label != "Rushing attempt" || items[0].Quantity != 30000 {
		t.Fatalf("non-deterministic or mislabeled: %+v %+v %v", p, items, err)
	}
}

func assertScore(t *testing.T, rules leaguerules.ScoringRules, pos leaguerules.Position,
	line StatLine, want leaguerules.Decimal) {
	t.Helper()
	points, _, err := Score(rules, pos, line)
	if err != nil || points.State != leaguerules.IsKnown || points.Value != want {
		t.Fatalf("points %+v error %v, want %d", points, err, want)
	}
}
