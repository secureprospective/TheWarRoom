package commissioner

import (
	"strings"
	"testing"
	"time"
)

func TestLineupRosterAtWeekLock(t *testing.T) {
	b := rolloverLedger(t)
	player := b.records[1].Command.Args.Native.Players[2]
	b.add(RawCommand{League: rolloverLeague, Command: RosterCut,
		Args: RawArgs{Waiver: &RawWaiverArgs{Player: player.ID, Franchise: player.Franchise},
			WaiverOutcome: &WaiverOutcome{Player: &player, Season: 2026}},
		Effective: seasonMoment(8).Format(time.RFC3339Nano)}, seasonMoment(10))
	bid := b.add(RawCommand{League: rolloverLeague, Command: BidPlace,
		Args:      RawArgs{Bid: &RawBidArgs{Player: player.ID, Franchise: "franchise.01", Salary: 400, Years: 2}, BidOutcome: &BidOutcome{}},
		Effective: "now"}, seasonMoment(9))
	b.add(RawCommand{League: rolloverLeague, Command: BidsProcess,
		Args:      RawArgs{Bid: &RawBidArgs{Player: player.ID}, BidOutcome: &BidOutcome{Winner: "franchise.01", WinningBid: bid, Season: 2026}},
		Effective: "now"}, seasonMoment(11))

	for _, tc := range []struct {
		name, franchise string
		locks           map[string]time.Time
		held, want      bool
	}{
		{"cut after lock", "franchise.02", map[string]time.Time{"later": seasonMoment(7), "first": seasonMoment(6)}, true, true},
		{"added after lock", "franchise.01", map[string]time.Time{"first": seasonMoment(6)}, true, false},
		{"current week before kickoff uses current roster", "franchise.01", map[string]time.Time{"first": seasonMoment(13)}, true, true},
		{"current week cut remains absent", "franchise.02", map[string]time.Time{"first": seasonMoment(13)}, true, false},
		{"no schedule uses current roster", "franchise.01", nil, false, true},
		{"empty schedule uses current roster", "franchise.01", nil, true, true},
		{"cut at lock is absent", "franchise.02", map[string]time.Time{"first": seasonMoment(8)}, true, false},
		{"addition at lock is present", "franchise.01", map[string]time.Time{"first": seasonMoment(11)}, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := RawCommand{League: rolloverLeague, Command: LineupCorrect, Args: RawArgs{Lineup: &RawLineupChange{
				Season: 2026, Week: 1, Franchise: tc.franchise, Starters: []string{player.ID},
			}}}
			err := CheckLineupAppend(raw, b.records, seasonMoment(12), LineupFacts{ScheduleHeld: tc.held, Locks: tc.locks})
			if tc.want && err != nil {
				t.Fatalf("expected acceptance, got %v", err)
			}
			if !tc.want && (err == nil || !strings.Contains(err.Error(), "not on this franchise's active roster")) {
				t.Fatalf("expected roster refusal, got %v", err)
			}
		})
	}
}
