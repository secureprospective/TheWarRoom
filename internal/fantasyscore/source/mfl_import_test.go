package source_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func heldMFLScoring(t *testing.T, year string) leaguerules.ScoringRules {
	t.Helper()
	leagueBody, err := os.ReadFile("../../ingestion/league/testdata/league-" + year + ".json")
	if err != nil {
		t.Fatal(err)
	}
	rulesBody, err := os.ReadFile("../../ingestion/league/testdata/rules-" + year + ".json")
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := league.Parse(leagueBody, rulesBody)
	if err != nil {
		t.Fatal(err)
	}
	var blocks []leaguerules.MFLRuleBlock
	for _, set := range cfg.ScoringRules {
		block := leaguerules.MFLRuleBlock{Positions: set.Positions}
		for _, r := range set.Rules {
			block.Rules = append(block.Rules, leaguerules.MFLRule{Event: r.Event, Points: r.Points, Range: r.Range})
		}
		blocks = append(blocks, block)
	}
	scoring, err := leaguerules.FromMFLScoring(blocks, "MFL import "+year)
	if err != nil {
		t.Fatal(err)
	}
	return scoring
}

// An imported Legacy rule table scores every held rostered player-week exactly as the hand-built
// Legacy preset does, so a league carried over rule for rule loses nothing and changes nothing.
func TestImportedMFLScoringMatchesLegacy(t *testing.T) {
	weeks := heldWeeks(t)
	for _, year := range []string{"2025", "2026"} {
		imported := heldMFLScoring(t, year)
		body, err := leaguerules.EncodeStored(imported.Rules)
		if err != nil || len(body) == 0 {
			t.Fatalf("%s: the imported rules do not store: %v", year, err)
		}
		if err := json.Unmarshal(body, new(any)); err != nil {
			t.Fatal(err)
		}
		scored := 0
		for _, w := range weeks {
			for _, position := range []leaguerules.Position{leaguerules.QB, leaguerules.RB, leaguerules.WR,
				leaguerules.TE, leaguerules.PK, leaguerules.DT, leaguerules.DE, leaguerules.LB, leaguerules.CB,
				leaguerules.S} {
				want, _, err := fantasyscore.Score(leaguerules.LegacyNFL().Scoring, position, w.Line)
				if err != nil || want.State != leaguerules.IsKnown {
					t.Fatalf("legacy %s %s: %+v %v", w.GSIS, position, want, err)
				}
				got, items, err := fantasyscore.Score(imported, position, w.Line)
				if err != nil || got.State != leaguerules.IsKnown {
					t.Fatalf("imported %s %s: %+v %v", w.GSIS, position, got, err)
				}
				if got.Value != want.Value {
					t.Fatalf("%s %s week %d as %s: imported %d, Legacy %d\n%+v", year, w.GSIS, w.Week, position,
						got.Value, want.Value, items)
				}
				scored++
			}
		}
		t.Logf("%s: %d player-week positions score the same", year, scored)
	}
}

// Every MFL event the held dictionary names is a stat in the app's dictionary.
func TestEveryMFLEventIsAStat(t *testing.T) {
	body, err := os.ReadFile("../../ingestion/league/testdata/allRules.json")
	if err != nil {
		t.Fatal(err)
	}
	codes, err := league.ParseRuleDictionary(body)
	if err != nil {
		t.Fatal(err)
	}
	var missing []string
	for code, name := range league.RuleNames(codes) {
		if _, ok := leaguerules.StatByMFL(code); !ok {
			missing = append(missing, code+" "+name)
		}
	}
	if len(codes) != 153 || len(missing) != 0 {
		t.Fatalf("%d MFL events; not scorable: %v", len(codes), missing)
	}
}

// The dictionary says a stat has a live feed exactly when a feed records it, or every stat it is
// counted from.
func TestStatFeedsAreTrue(t *testing.T) {
	recorded := map[leaguerules.StatID]bool{}
	for _, w := range heldWeeks(t) {
		for id := range w.Line.Quantities {
			recorded[id] = true
		}
		for id := range w.Line.Events {
			recorded[id] = true
		}
	}
	sleeper, err := source.RawSleeperStats{}.Convert()
	if err != nil {
		t.Fatal(err)
	}
	for id := range sleeper.Quantities {
		recorded[id] = true
	}
	for id := range leaguerules.Default32Scoring().LineUnit.Value.Weights {
		recorded[id] = true
	}
	for _, id := range []leaguerules.StatID{"third_down_conversion", "fourth_down_conversion",
		"third_down_passing_conversion", "fourth_down_passing_conversion", "red_zone_carry", "red_zone_target",
		"red_zone_pass_attempt", "rushing_tfl"} {
		recorded[id] = true // play-by-play edges, recorded where the held games carry them
	}
	// A true-position rate row is priced from its base stat's record.
	for _, id := range []leaguerules.StatID{"tackle_line", "assist_line", "tackle_secondary",
		"pass_defensed_secondary", "interception_caught_secondary"} {
		recorded[id] = true
	}
	for id := range recorded {
		if def, ok := leaguerules.StatByID(id); !ok || !def.Feed {
			t.Errorf("%s is recorded but not a fed stat in the dictionary", id)
		}
	}
	for _, def := range leaguerules.StatDictionary() {
		from := append(append([]leaguerules.StatID{}, def.Parts...), def.Plays...)
		fed := recorded[def.ID]
		if len(from) != 0 {
			fed = true
			for _, part := range from {
				fed = fed && recorded[part]
			}
		}
		if fed != def.Feed {
			t.Errorf("%s: dictionary feed %v, recorded %v", def.ID, def.Feed, fed)
		}
	}
}
