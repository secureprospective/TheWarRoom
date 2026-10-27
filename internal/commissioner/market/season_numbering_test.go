package market

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

func TestClocksHandleUnseededLeagueWithoutRecordedSeason(t *testing.T) {
	b := newLedger(t, commissioner.PresetLegacy)
	b.records[0].Command.Args.Season = 0
	b.records[0].Command.Reason = "start"
	// The clocks run every second for every league: an unseeded one has nothing due and is not an error.
	if due, err := DueWaiverWindows("native.test", b.records, seasonMoment(5)); err != nil || len(due) != 0 {
		t.Fatalf("the waiver clock failed on a league with nothing to process: %v %v", due, err)
	}
	if due, err := DueBids("native.test", b.records, seasonMoment(5), nativecheck.Facts{}); err != nil || len(due) != 0 {
		t.Fatalf("the bid clock failed on a league with nothing to process: %v %v", due, err)
	}
}
