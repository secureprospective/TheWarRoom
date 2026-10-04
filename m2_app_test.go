package main

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/leaguestandings"
	"github.com/secureprospective/TheWarRoom/internal/m2service"
	"github.com/secureprospective/TheWarRoom/internal/powerrankings"
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
		map[string]int{"0002": 2}, nil)
	if !rows[0].DeltaOK || rows[0].RankDelta != 1 {
		t.Errorf("0002 went from 2nd to 1st: %+v", rows[0])
	}
	if rows[1].DeltaOK {
		t.Errorf("0001 was not on the previous board, so it has no move: %+v", rows[1])
	}
}

// This season's weight follows the weeks played unless the slider sets it; the franchise ranks on
// the roster and its age, and each view counts the roster its own way by default.
func TestBoardWeightsAndDefaultsPerView(t *testing.T) {
	if w := boardWeights(ViewSeason, 0.3, true, 4); w.Roster != 0.5 || w.Age != 0 {
		t.Errorf("auto after 4 weeks: %+v, want roster 0.5", w)
	}
	if w := boardWeights(ViewSeason, 0.3, false, 4); w.Roster != 0.3 {
		t.Errorf("the slider's weight must hold when auto is off: %+v", w)
	}
	if w := boardWeights(ViewFranchise, 0.3, true, 4); w.Roster != 1 || w.Age != powerrankings.FranchiseAgeWeight {
		t.Errorf("franchise: %+v", w)
	}
	if defaultAgg(ViewSeason) != m2service.AggLineup || defaultAgg(ViewFranchise) != m2service.AggSum {
		t.Error("view defaults")
	}
}
