package history

import (
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/researchfeeds"
)

func TestDepthMembershipIsDatedAndDoesNotRetainDepartedPlayers(t *testing.T) {
	cutoff := time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)
	rows := []researchfeeds.RawDepth{
		{DT: "2026-09-08T12:00:00Z", Team: "BUF", Name: "Departed"},
		{DT: "2026-09-09T12:00:00Z", Team: "NYJ", Name: "Current"},
		{DT: "2026-09-11T12:00:00Z", Team: "MIA", Name: "Future"},
	}
	got := depthAt(rows, cutoff)
	if len(got) != 1 || got[0].Name != "Current" {
		t.Fatalf("wrong historical affiliation: %+v", got)
	}
	if got := depthAt(rows, cutoff.AddDate(0, 0, 10)); len(got) != 0 {
		t.Fatalf("stale roster admitted: %+v", got)
	}
	if got := depthAt(rows, cutoff.AddDate(0, 0, -10)); len(got) != 0 {
		t.Fatalf("future roster admitted: %+v", got)
	}
}

func TestRosterByesRequireHistoricalTeamAndPositiveSchedule(t *testing.T) {
	gsis := "00-0030001"
	depth := []researchfeeds.RawDepth{{DT: "2026-10-07T12:00:00Z", Team: "NE", GSIS: &gsis, Position: "LB"}, {DT: "2026-10-09T12:00:00Z", Team: "BUF", GSIS: &gsis, Position: "LB"}}
	snaps := []source.RawSnap{{Proof: source.RawAbsenceProof{Week: "5", Team: "NYJ"}, Date: "20261008"}}
	for _, tc := range []struct {
		name string
		byes []nflschedule.RawBye
		want int
	}{
		{"dated team", []nflschedule.RawBye{{Team: "NEP", Week: "5"}}, 1},
		{"future team", []nflschedule.RawBye{{Team: "BUF", Week: "5"}}, 0},
		{"no schedule", nil, 0},
		{"different week", []nflschedule.RawBye{{Team: "NEP", Week: "6"}}, 0},
		{"team playing", []nflschedule.RawBye{{Team: "NYJ", Week: "5"}}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := rosterByes(2026, depth, snaps, tc.byes)
			if len(got) != tc.want {
				t.Fatalf("unsupported bye: %+v", got)
			}
		})
	}
	depth[0].GSIS = nil
	if got := rosterByes(2026, depth, snaps, []nflschedule.RawBye{{Team: "NEP", Week: "5"}}); len(got) != 0 {
		t.Fatal("unknown identity zeroed", got)
	}
}
