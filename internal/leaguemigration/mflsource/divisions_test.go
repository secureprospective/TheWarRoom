package mflsource

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
)

func TestDivisionsLandByTheirOwnNames(t *testing.T) {
	config := league.RawConfig{
		Conferences: []league.Conference{{ID: "00", Name: "AFC"}},
		Divisions: []league.Division{{ID: "00", Name: " AFC East ", Conference: "00"},
			{ID: "01", Name: "AFC North", Conference: "00"}},
	}
	names := divisionNames(config)
	if names["00"] != "AFC East" || names["01"] != "AFC North" {
		t.Fatalf("divisions keep MFL ids instead of their names: %v", names)
	}
	if got := conferences(config); got["AFC East"] != "AFC" || got["AFC North"] != "AFC" || len(got) != 2 {
		t.Fatalf("conferences are not keyed by the division's name: %v", got)
	}
}

func TestDivisionsKeepIdsWhenNamesCannotTellThemApart(t *testing.T) {
	for label, divisions := range map[string][]league.Division{
		"shared name": {{ID: "00", Name: "East"}, {ID: "01", Name: "East"}},
		"blank name":  {{ID: "00", Name: "East"}, {ID: "01", Name: " "}},
	} {
		names := divisionNames(league.RawConfig{Divisions: divisions})
		if names["00"] != "00" || names["01"] != "01" {
			t.Fatalf("%s: divisions would merge or go unnamed: %v", label, names)
		}
	}
}
