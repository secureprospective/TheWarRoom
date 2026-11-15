package livestats_test

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livestats"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func TestLiveLinesNeverRecordAPartsTotal(t *testing.T) {
	lines, err := livestats.Lines(livegame.Game{Box: []livegame.Box{
		{Category: "kicking", Labels: []string{"XP", "FG"}, Rows: []livegame.BoxRow{{ID: "1", Stats: []string{"3/4", "1/2"}}}},
		{Category: "rushing", Labels: []string{"YDS"}, Rows: []livegame.BoxRow{{ID: "1", Stats: []string{"40"}}}},
		{Category: "receiving", Labels: []string{"YDS"}, Rows: []livegame.BoxRow{{ID: "1", Stats: []string{"12"}}}},
	}})
	if err != nil || len(lines) != 1 {
		t.Fatalf("%+v %v", lines, err)
	}
	for id := range lines[0].Stats.Quantities {
		if s, ok := leaguerules.StatByID(id); ok && len(s.Parts) > 0 {
			t.Fatalf("%s is a parts total and must not be recorded", id)
		}
	}
	if lines[0].Stats.Quantities["pat_missed"] != 10000 || lines[0].Stats.Quantities["pat_blocked"] != 0 {
		t.Fatalf("missed PAT parts: %+v", lines[0].Stats.Quantities)
	}
}
