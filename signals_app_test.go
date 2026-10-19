package main

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/college"
	"github.com/secureprospective/TheWarRoom/internal/measures"
)

func TestCoverageCountsRosteredPlayersByPosition(t *testing.T) {
	t.Parallel()
	got := coverage(map[string]string{"1": "QB", "2": "QB", "3": "S"}, map[string]bool{"1": true, "3": true, "9": true})
	if len(got) != 2 || got[0] != (PositionShare{"QB", 2, 1}) || got[1] != (PositionShare{"S", 1, 1}) {
		t.Errorf("coverage = %+v", got)
	}
}

// The last and current seasons are scored under today's rules; older ones only exist under
// their own.
func TestScoringYear(t *testing.T) {
	t.Parallel()
	for y, want := range map[int]int{2021: 2021, 2024: 2024, 2025: 2026, 2026: 2026} {
		if got := scoringYear(y, 2026); got != want {
			t.Errorf("scoringYear(%d) = %d, want %d", y, got, want)
		}
	}
}

func TestMapsNewMeasureSpotsAMappingTheSeasonLacks(t *testing.T) {
	t.Parallel()
	reg, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	b := measures.Batch{Source: college.Source, Facts: []measures.Fact{
		{Field: "player_season.defensive.SACKS"}, {Field: "team_season.defensive.QB HUR"}, {Field: "unmapped"}}}
	if !mapsNewMeasure(reg, b, map[string]bool{"prior.college_sacks": true}) {
		t.Error("the team's hurries are new, but no new measure was seen")
	}
	if mapsNewMeasure(reg, b, map[string]bool{"prior.college_sacks": true, "prior.college_team_qb_hurries": true}) {
		t.Error("every mapped measure is held, but a new one was seen")
	}
}
