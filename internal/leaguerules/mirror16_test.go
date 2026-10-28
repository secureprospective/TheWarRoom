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

func TestMirror16ForTeamsAsksForTheCountDependentValues(t *testing.T) {
	if !reflect.DeepEqual(Mirror16ForTeams(16), Mirror16()) {
		t.Error("at 16 the Mirror changed")
	}
	want, got := Mirror16(), Mirror16ForTeams(20)
	if got.League.Teams.State != IsKnown || got.League.Teams.Value != 20 {
		t.Errorf("teams: %+v, want 20", got.League.Teams)
	}
	if got.League.Pools != want.League.Pools {
		t.Errorf("pools: %+v, want a pool per side", got.League.Pools)
	}
	for name, state := range map[string]State{
		"league.divisions": got.League.Divisions.State, "league.alignment": got.League.Alignment.State,
		"league.divisionConferences": got.League.DivisionConferences.State,
		"playoffs.perConference":     got.Playoffs.PerConference.State,
		"playoffs.graph":             got.Playoffs.Graph.State, "schedule.pattern": got.Schedule.Pattern.State,
	} {
		if state != NeedsRuling {
			t.Errorf("%s: %v, want Needs a ruling", name, state)
		}
	}
	if !reflect.DeepEqual(got.Lineup, want.Lineup) || !reflect.DeepEqual(got.Roster, want.Roster) ||
		!reflect.DeepEqual(got.Scoring, want.Scoring) {
		t.Error("lineup, roster or scoring changed with the team count")
	}
}
