package leaguerules

import (
	"reflect"
	"testing"
)

// Each way MFL writes points and ranges is carried exactly; only an unreadable value is refused.
func TestFromMFLScoring(t *testing.T) {
	all := []Position{QB, RB, WR, TE, PK, DT, DE, LB, CB, S}
	cases := []struct {
		rule MFLRule
		id   StatID
		want RangeRule
		err  bool
	}{
		{MFLRule{"#P", "*5", "0-20"}, "passing_td", RangeRule{Points: 50000, Range: StatRange{Max: 200000}}, false},
		{MFLRule{"PY", "*.05", "-50-999"}, "passing_yards", RangeRule{Points: 500, Range: StatRange{Min: -500000, Max: 9990000}}, false},
		{MFLRule{"IN", "*-2", "0-10"}, "interception_thrown", RangeRule{Points: -20000, Range: StatRange{Max: 100000}}, false},
		{MFLRule{"PY", "1/25", "0-999"}, "passing_yards", RangeRule{Points: 10000, Per: 250000, Range: StatRange{Max: 9990000}}, false},
		{MFLRule{"FG", "3", "0-39"}, "fg_made_distance", RangeRule{Points: 30000, Flat: true, Range: StatRange{Max: 390000}}, false},
		{MFLRule{"MG", "-3", "0-29"}, "fg_missed_distance", RangeRule{Points: -30000, Flat: true, Range: StatRange{Max: 290000}}, false},
		{MFLRule{"RATE", "*0.0125", "0-158.3"}, "passer_rating", RangeRule{Points: 125, Range: StatRange{Max: 1583000}}, false},
		{MFLRule{"TPA", "-1", "-5--1"}, "team_points_allowed", RangeRule{Points: -10000, Flat: true, Range: StatRange{Min: -50000, Max: -10000}}, false},
		{MFLRule{"ZZ", "*1", "0-9"}, "mfl_ZZ", RangeRule{Points: 10000, Range: StatRange{Max: 90000}}, false},
		{MFLRule{"PY", "*.00001", "0-9"}, "", RangeRule{}, true},
		{MFLRule{"PY", "1/0", "0-9"}, "", RangeRule{}, true},
		{MFLRule{"PY", "*1", "9"}, "", RangeRule{}, true},
	}
	for _, c := range cases {
		t.Run(c.rule.Event+" "+c.rule.Points+" "+c.rule.Range, func(t *testing.T) {
			got, err := FromMFLScoring([]MFLRuleBlock{{Positions: "QB|RB|WR|TE|PK|DT|DE|LB|CB|S",
				Rules: []MFLRule{c.rule}}}, "test import")
			if (err != nil) != c.err {
				t.Fatalf("error %v, want error %v", err, c.err)
			}
			if c.err {
				return
			}
			c.want.Positions = all
			if rules := got.Rules.Value[c.id]; len(rules) != 1 || !reflect.DeepEqual(rules[0], c.want) {
				t.Fatalf("got %+v, want %s %+v", got.Rules.Value, c.id, c.want)
			}
			if err := validateScoringKinds(got); err != nil {
				t.Fatal(err)
			}
		})
	}
	got, err := FromMFLScoring([]MFLRuleBlock{{Positions: "CB|S", Rules: []MFLRule{{"EM", "*-1", "0-10"}}}}, "test")
	if err != nil || !reflect.DeepEqual(got.Totals.Value["pat_missed_or_blocked"], []StatID{"pat_missed", "pat_blocked"}) ||
		!reflect.DeepEqual(got.Rules.Value["pat_missed_or_blocked"][0].Positions, []Position{CB, S}) {
		t.Fatalf("a total event counts from its parts for its block's positions: %+v %v", got, err)
	}
	if _, err := FromMFLScoring([]MFLRuleBlock{{Positions: "QB|DST", Rules: nil}}, "test"); err == nil {
		t.Fatal("a position the engine has no players for is named, not ignored")
	}
}

// A stat no feed records yet is offered, and its name says so.
func TestStatChoicesNameMissingFeeds(t *testing.T) {
	field, _ := Field("scoring.rules")
	choices := field.Schema.Key.Choices
	if len(choices) != len(StatDictionary()) || len(choices) < 153 {
		t.Fatalf("%d stat choices", len(choices))
	}
	words := map[string]string{}
	for _, c := range choices {
		words[c.Value] = c.Label
	}
	if words["passer_rating"] != "Passer rating (no live feed yet)" || words["passing_td"] != "Passing touchdown" {
		t.Fatalf("stat words: %q %q", words["passer_rating"], words["passing_td"])
	}
}

func TestRangeRuleValidation(t *testing.T) {
	for _, c := range []struct {
		rule RangeRule
		ok   bool
	}{
		{RangeRule{Points: -50000, Range: StatRange{Min: -10, Max: 10}}, true},
		{RangeRule{Points: 10000, Per: 100000, Range: StatRange{Max: 10}}, true},
		{RangeRule{Range: StatRange{Min: 10, Max: 0}}, false},
		{RangeRule{Per: -1}, false},
		{RangeRule{Flat: true, Per: 10000}, false},
	} {
		err := validateRangeRules(Known(map[StatID][]RangeRule{"x": {c.rule}}))
		if (err == nil) != c.ok {
			t.Errorf("%+v: %v", c.rule, err)
		}
	}
}
