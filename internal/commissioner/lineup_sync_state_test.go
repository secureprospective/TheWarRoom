package commissioner

import (
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

func syncLineupRecord(entry, request string, week int) Record {
	r := lineupRecord(entry, LineupCorrect, week, "1001")
	r.Command.Request = request
	return r
}

func TestLineupSyncPendingOnlyWhileASyncSourceHasNotReachedTheWeek(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		records []Record
		want    bool
	}{
		{"no sync source at all", []Record{lineupRecord("a", LineupSet, 8, "1001")}, false},
		{"sync on, week 8 not read", []Record{syncLineupRecord("a", "lineup.sync.3.franchise.02.7", 7)}, true},
		{"sync on and week 8 read", []Record{syncLineupRecord("a", "lineup.sync.3.franchise.02.7", 7),
			syncLineupRecord("b", "lineup.sync.4.franchise.02.8", 8)}, false},
		{"the only sync lineup was reverted", []Record{syncLineupRecord("a", "lineup.sync.3.franchise.02.7", 7),
			revertRecord("r", "a")}, false},
		{"owner lineups are not a sync source", []Record{lineupRecord("a", LineupSet, 7, "1001")}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := LineupSyncPending(tc.records, 2026, 8); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

// A missed lineup is still a missed lineup, but it is not flagged to the commissioner while the
// league's lineup sync has not read the week being played.
func TestMissedLineupIsNotFlaggedWhileTheSyncHasNotReachedTheWeek(t *testing.T) {
	t.Parallel()
	seat := NativeDirectorySeat{PeopleSeat: PeopleSeat{Franchise: "franchise.02", Name: "Beta"}}
	rules := leaguerules.LineupRules{MissedFlag: leaguerules.Known(true)}
	src := scoreSource{lineup: WeekLineup{}}
	flagged, err := franchiseWeekScore(t.Context(), nil, src, rules, seat, 2026, 8)
	if err != nil {
		t.Fatal(err)
	}
	if !flagged.Lineup.Flagged || flagged.Lineup.Status != nativecheck.LineupMissed {
		t.Fatalf("a league with no sync source should still flag: %+v", flagged.Lineup)
	}
	pending := []Record{syncLineupRecord("a", "lineup.sync.3.franchise.02.7", 7)}
	got, err := franchiseWeekScore(t.Context(), pending, src, rules, seat, 2026, 8)
	if err != nil {
		t.Fatal(err)
	}
	if got.Lineup.Flagged {
		t.Fatalf("flagged while the sync had not read week 8: %+v", got.Lineup)
	}
	if got.Lineup.Status != nativecheck.LineupMissed {
		t.Fatalf("the lineup is still missing: %+v", got.Lineup)
	}
	if !strings.Contains(got.Words, "has not read this week's lineups yet") {
		t.Fatalf("words %q", got.Words)
	}
	read := []Record{pending[0], syncLineupRecord("b", "lineup.sync.4.franchise.02.8", 8)}
	after, err := franchiseWeekScore(t.Context(), read, src, rules, seat, 2026, 8)
	if err != nil {
		t.Fatal(err)
	}
	if !after.Lineup.Flagged {
		t.Fatalf("not flagged after the sync read week 8: %+v", after.Lineup)
	}
}
