package leaguerules

import (
	"reflect"
	"testing"
)

// Every 32-Team Default value is cited and valid as a commissioner setting.
func TestDefault32PresetIsCitedAndValid(t *testing.T) {
	preset := Default32()
	assertCitations(t, reflect.ValueOf(preset))
	for _, c := range SeedChanges(preset, "test-league", 2026, testTime()) {
		if err := c.Validate(); err != nil {
			t.Fatalf("preset %s: %v", c.Field, err)
		}
	}
}

// Default32ForTeams keeps every value that does not depend on the team count and asks for those that do.
func TestDefault32ForTeamsAsksForTheCountDependentValues(t *testing.T) {
	want := Default32()
	got := Default32ForTeams(20)
	if got.League.Teams.State != IsKnown || got.League.Teams.Value != 20 {
		t.Errorf("teams: %+v, want 20", got.League.Teams)
	}
	for _, c := range []struct {
		name  string
		state State
	}{
		{"league.divisions", got.League.Divisions.State},
		{"league.alignment", got.League.Alignment.State},
		{"league.divisionConferences", got.League.DivisionConferences.State},
		{"playoffs.perConference", got.Playoffs.PerConference.State},
		{"playoffs.divisionWinners", got.Playoffs.DivisionWinners.State},
		{"playoffs.wildCards", got.Playoffs.WildCards.State},
		{"playoffs.byeSeed", got.Playoffs.ByeSeed.State},
		{"playoffs.finalWeek", got.Playoffs.FinalWeek.State},
		{"playoffs.graph", got.Playoffs.Graph.State},
		{"schedule.pattern", got.Schedule.Pattern.State},
	} {
		if c.state != NeedsRuling {
			t.Errorf("%s: %v, want Needs a ruling", c.name, c.state)
		}
	}
	if !reflect.DeepEqual(got.Lineup, want.Lineup) {
		t.Error("lineup changed with the team count")
	}
	if !reflect.DeepEqual(got.Roster, want.Roster) {
		t.Error("roster changed with the team count")
	}
	if !reflect.DeepEqual(got.Scoring, want.Scoring) {
		t.Error("scoring changed with the team count")
	}
}
