package leaguerules

import (
	"reflect"
	"testing"
)

// Every 16-Team Mirror value is cited and valid as a commissioner setting; it is per-side pools on two
// conferences of eight, with no O-Line anywhere, and its startup draft fills the whole roster.
func TestMirror16PresetIsCitedAndValid(t *testing.T) {
	preset := Mirror16()
	assertCitations(t, reflect.ValueOf(preset))
	for _, c := range SeedChanges(preset, "test-league", 2026, testTime()) {
		if err := c.Validate(); err != nil {
			t.Fatalf("preset %s: %v", c.Field, err)
		}
	}
	if !PoolsPerSide(preset) || OLineOn(preset) || preset.Scoring.LineUnit.Value.Used {
		t.Fatal("the mirror is per-side pools with O-Line off")
	}
	if _, ok := preset.Lineup.Positions.Value[OL]; ok {
		t.Fatal("the lineup still starts an O-Line card")
	}
	sides := map[string]int{}
	for _, side := range CountedSides(preset) {
		sides[side]++
	}
	if len(sides) != 2 || sides["AFC"] != 8 || sides["NFC"] != 8 {
		t.Fatalf("sides: %v", sides)
	}
	if preset.Draft.StartupRounds.Value != preset.Roster.Limits.Value.Max.Value {
		t.Fatalf("startup rounds %d", preset.Draft.StartupRounds.Value)
	}
}
