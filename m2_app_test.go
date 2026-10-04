package main

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/m2service"
)

// Each all-play week is a game against every other franchise, so three franchises two weeks in
// have four all-play games each.
func TestWeeksScoredReadsAllPlayGames(t *testing.T) {
	rows := []m2service.Row{{AllPlayW: 3, AllPlayL: 1}, {AllPlayW: 2, AllPlayL: 2}, {AllPlayW: 1, AllPlayL: 3}}
	if got := weeksScored(rows); got != 2 {
		t.Errorf("weeks = %d, want 2", got)
	}
	if got := weeksScored([]m2service.Row{{}, {}}); got != 0 {
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
