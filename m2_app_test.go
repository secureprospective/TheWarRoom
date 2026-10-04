package main

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/leaguestandings"
	"github.com/secureprospective/TheWarRoom/internal/m2service"
)

// A franchise plays one head-to-head game a week.
func TestWeeksScoredReadsHeadToHeadGames(t *testing.T) {
	standings := []leaguestandings.RawStanding{{H2HW: "2"}, {H2HW: "1", H2HL: "1"}, {H2HL: "2", H2HT: " "}}
	if got := weeksScored(standings); got != 2 {
		t.Errorf("weeks = %d, want 2", got)
	}
	if got := weeksScored([]leaguestandings.RawStanding{{}, {H2HW: "0"}}); got != 0 {
		t.Errorf("an unplayed season has %d weeks, want 0", got)
	}
}

func TestPowerRowsMoveAgainstThePreviousRun(t *testing.T) {
	rows := powerRows([]m2service.Row{{Rank: 1, FranchiseID: "0002"}, {Rank: 2, FranchiseID: "0001"}},
		map[string]int{"0002": 2})
	if !rows[0].DeltaOK || rows[0].RankDelta != 1 {
		t.Errorf("0002 went from 2nd to 1st: %+v", rows[0])
	}
	if rows[1].DeltaOK {
		t.Errorf("0001 was not on the previous board, so it has no move: %+v", rows[1])
	}
}
