package commissioner

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// A Legacy league seeded before part 2 (the preset exactly as main dda866b8 recorded it, written by
// that commit's own Prepared()) upgrades on read to MFL parity: its rules equal a new Legacy
// league's, with no commissioner step and nothing waiting for a ruling.
func TestPrePartTwoLegacySeedUpgradesToANewLegacyLeague(t *testing.T) {
	body, err := os.ReadFile("testdata/legacy-seed-dda866b8.json")
	if err != nil {
		t.Fatal(err)
	}
	var old []SeedValue
	if err := json.Unmarshal(body, &old); err != nil {
		t.Fatal(err)
	}
	upgraded := legacyLeague(t)
	fresh := legacyLeague(t)
	if len(old) != len(fresh[0].Command.Args.Seed)-17 {
		t.Fatalf("the held old seed should lack the six scoring kinds, two auto-move fields, division conferences, six startup draft settings, sides and player pools: %d vs %d", len(old), len(fresh[0].Command.Args.Seed))
	}
	for _, s := range old {
		if strings.Contains(string(s.Value), "blocked_fg_td") || strings.Contains(string(s.Value), "other_return_td") {
			t.Fatal("the held old seed already carries the kick-return touchdown rows")
		}
	}
	upgraded[0].Command.Args.Seed = old
	got, err := RulesInForce(upgraded, 2026, noChangeTime())
	if err != nil {
		t.Fatal(err)
	}
	want, err := RulesInForce(fresh, 2026, noChangeTime())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		for _, f := range []string{"scoring.catalogue", "scoring.ranges", "scoring.extras", "scoring.scales", "scoring.totals", "scoring.lineUnit"} {
			a, _ := ruleValue(upgraded, f)
			b, _ := ruleValue(fresh, f)
			if a != b {
				t.Errorf("%s differs:\n upgraded %s\n new      %s", f, a, b)
			}
		}
		t.Fatal("an upgraded pre-part-2 Legacy league's rules differ from a new Legacy league's")
	}
	if _, ok := got.Scoring.Catalogue.Value["blocked_punt_td"]; !ok || len(got.Scoring.Ranges.Value) == 0 {
		t.Fatal("the upgrade did not add the kick-return rows and MFL's ranges")
	}
	sc := got.Scoring
	for _, state := range []leaguerules.State{sc.Catalogue.State, sc.Extras.State, sc.Ranges.State, sc.Scales.State, sc.Totals.State, sc.LineUnit.State} {
		if state != leaguerules.IsKnown {
			t.Fatal("an upgraded league's scoring waits for a ruling")
		}
	}
}

// A seed recorded before the scoring kinds whose catalogue is not the recorded Legacy one (a
// commissioner-edited or imported league) gets only the kinds recorded empty: it scores exactly as
// it did, keeps its own catalogue, and nothing waits for a ruling.
func TestPreKindsNonLegacySeedGetsEmptyKindsOnly(t *testing.T) {
	records := legacyLeague(t)
	var old []SeedValue
	for _, s := range records[0].Command.Args.Seed {
		if !strings.HasPrefix(s.Field, "scoring.") || s.Field == "scoring.catalogue" || s.Field == "scoring.roundStatPoints" {
			old = append(old, s)
		}
	}
	if len(old) != len(records[0].Command.Args.Seed)-6 {
		t.Fatalf("expected six newer scoring settings in the preset, kept %d of %d", len(old), len(records[0].Command.Args.Seed))
	}
	records[0].Command.Args.Seed = old
	rules, err := RulesInForce(records, 2026, noChangeTime())
	if err != nil {
		t.Fatal(err)
	}
	s := rules.Scoring
	for _, state := range []leaguerules.State{s.Extras.State, s.Ranges.State, s.Scales.State, s.Totals.State, s.LineUnit.State, s.Rules.State} {
		if state != leaguerules.IsKnown {
			t.Fatalf("an upgraded seed left a scoring kind unset: %+v", s)
		}
	}
	if len(s.Extras.Value)+len(s.Ranges.Value)+len(s.Scales.Value)+len(s.Totals.Value)+len(s.LineUnit.Value.Weights)+len(s.Rules.Value) != 0 || s.LineUnit.Value.Used {
		t.Fatalf("an upgraded seed must record the kinds empty: %+v", s)
	}
	before := leaguerules.LegacyNFL().Scoring
	before.Ranges = leaguerules.Known(map[leaguerules.StatID]leaguerules.StatRange{})
	line := fantasyscore.StatLine{Quantities: map[leaguerules.StatID]leaguerules.Decimal{
		"rushing_yards": -510000, "rushing_attempt": 30000, "reception": 100000, "receiving_yards": 1230000,
		"receiving_td": 10000, "fumbles_total": 10000, "kickoff_return_yards": 350000}}
	got, _, err := fantasyscore.Score(s, leaguerules.RB, line)
	want, _, werr := fantasyscore.Score(before, leaguerules.RB, line)
	if err != nil || werr != nil || got.State != leaguerules.IsKnown || got.Value != want.Value {
		t.Fatalf("upgraded league scored %+v (%v), pre-upgrade Legacy %+v (%v)", got, err, want, werr)
	}
	if got.Value != -51000+4500+100000+123000+60000-20000+8800 {
		t.Fatalf("old Legacy rules changed: %d", got.Value)
	}
}

// A seed that already records the kinds is left as recorded.
func TestCurrentLegacySeedIsNotUpgraded(t *testing.T) {
	rules, err := RulesInForce(legacyLeague(t), 2026, noChangeTime())
	if err != nil {
		t.Fatal(err)
	}
	if len(rules.Scoring.Ranges.Value) == 0 {
		t.Fatal("a current Legacy seed lost MFL's ranges")
	}
}
