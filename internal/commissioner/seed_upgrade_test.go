package commissioner

import (
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// A league seeded before the scoring rule kinds existed upgrades on read: the kinds are recorded
// empty, it scores exactly as the old Legacy rules did, and nothing waits for a ruling.
func TestOldLegacySeedUpgradesAndScoresIdentically(t *testing.T) {
	records := legacyLeague(t)
	var old []SeedValue
	for _, s := range records[0].Command.Args.Seed {
		if !strings.HasPrefix(s.Field, "scoring.") || s.Field == "scoring.catalogue" || s.Field == "scoring.roundStatPoints" {
			old = append(old, s)
		}
	}
	if len(old) != len(records[0].Command.Args.Seed)-5 {
		t.Fatalf("expected five newer scoring settings in the preset, kept %d of %d", len(old), len(records[0].Command.Args.Seed))
	}
	records[0].Command.Args.Seed = old
	rules, err := RulesInForce(records, 2026, noChangeTime())
	if err != nil {
		t.Fatal(err)
	}
	s := rules.Scoring
	for _, state := range []leaguerules.State{s.Extras.State, s.Ranges.State, s.Scales.State, s.Totals.State, s.LineUnit.State} {
		if state != leaguerules.IsKnown {
			t.Fatalf("an upgraded seed left a scoring kind unset: %+v", s)
		}
	}
	if len(s.Extras.Value)+len(s.Ranges.Value)+len(s.Scales.Value)+len(s.Totals.Value)+len(s.LineUnit.Value.Weights) != 0 || s.LineUnit.Value.Used {
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
