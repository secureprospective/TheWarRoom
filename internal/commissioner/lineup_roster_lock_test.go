package commissioner

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
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

func TestLineupMembershipIgnoresEligibilityExceptions(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		players  []NativePlayer
		starters []string
		want     string
	}{
		{"active", []NativePlayer{{ID: "0001", Franchise: "home", Status: domain.RosterActive}}, []string{"0001"}, ""},
		{"IR", []NativePlayer{{ID: "0001", Franchise: "home", Status: domain.RosterIR}}, []string{"0001"}, "Held name"},
		{"taxi", []NativePlayer{{ID: "0001", Franchise: "home", Status: domain.RosterTaxi}}, []string{"0001"}, "Held name"},
		{"other franchise", []NativePlayer{{ID: "0001", Franchise: "away", Status: domain.RosterActive}}, []string{"0001"}, "Held name"},
		{"absent", nil, []string{"0001"}, "Held name"},
		{"same ID on another side", []NativePlayer{
			{ID: "0001", Franchise: "home", Status: domain.RosterIR},
			{ID: "0001", Franchise: "away", Status: domain.RosterActive},
		}, []string{"0001"}, "Held name"},
		{"active copy unaffected by other side", []NativePlayer{
			{ID: "0001", Franchise: "home", Status: domain.RosterActive},
			{ID: "0001", Franchise: "away", Status: domain.RosterIR},
		}, []string{"0001"}, ""},
		{"first refusal follows starters", nil, []string{"0002", "0001"}, "player 0002"},
		{"empty starters", nil, nil, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base := NativeBase{Now: seasonMoment(12), Assets: NativeAssets{Seeded: true,
				Snapshot: &RawNativeSnapshot{Players: tc.players}}}
			for _, rule := range []string{"roster.limits", "roster.positionBounds", "cap.enforcement", "ir.eligible", "taxi.maximumYears"} {
				base.Records = append(base.Records, Record{Command: RawCommand{Command: RuleException,
					Args:   RawArgs{Correction: &RawCorrection{Rule: rule, Scope: &ExceptionScope{Franchise: "home", Player: "0001"}}},
					Reason: "Eligibility only"}, Receipt: Receipt{Entry: rule, RecordedAt: seasonMoment(1)}})
			}
			before := fmt.Sprintf("%+v", base.Assets.Snapshot.Players)
			change := RawLineupChange{Franchise: "home", Starters: tc.starters}
			env := LineupFacts{Facts: nativecheck.Facts{"0001": {Name: "Held name"}}}
			err := checkLineupRoster(base, change, env)
			want := tc.want
			if want != "" {
				want += " is not on this franchise's active roster, so cannot start"
			}
			got := ""
			if err != nil {
				got = err.Error()
			}
			if got != want || fmt.Sprintf("%+v", base.Assets.Snapshot.Players) != before {
				t.Fatalf("got %q, want %q, or membership input changed", got, want)
			}
		})
	}
}

func BenchmarkLineupRosterMembership(b *testing.B) {
	for _, size := range []int{100, 1000, 3000} {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			base := NativeBase{Assets: NativeAssets{Seeded: true, Snapshot: &RawNativeSnapshot{}}}
			for i := range size {
				base.Records = append(base.Records, Record{Receipt: Receipt{Entry: fmt.Sprint(i)}})
				base.Assets.Snapshot.Players = append(base.Assets.Snapshot.Players,
					NativePlayer{ID: fmt.Sprint(i), Franchise: "home", Status: domain.RosterActive})
			}
			change := RawLineupChange{Franchise: "home", Starters: []string{"0"}}
			b.ResetTimer()
			for b.Loop() {
				if err := checkLineupRoster(base, change, LineupFacts{}); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
