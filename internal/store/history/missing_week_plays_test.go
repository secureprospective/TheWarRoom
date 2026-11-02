package history

import (
	"bytes"
	"compress/gzip"
	"strconv"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// R11: a week whose play-by-play is not held stays pending (no line, never a zero) while every
// other week in the same file still completes.
func TestMissingWeekPlaysLeaveOnlyThatWeekPending(t *testing.T) {
	t.Parallel()
	week := func(n int, game string) source.PlayerWeek {
		return source.PlayerWeek{GSIS: "00-0000001", Position: "WR", Season: 2026, Week: n, SeasonType: "REG", GameID: game,
			Line: fantasyscore.StatLine{Quantities: map[leaguerules.StatID]leaguerules.Decimal{"receiving_yards": 400000},
				Events: map[leaguerules.StatID][]leaguerules.Decimal{}}}
	}
	cols := "game_id,play_id,season,week,season_type,passer_player_id,receiver_player_id,rusher_player_id," +
		"passing_yards,receiving_yards,rushing_yards,touchdown,td_player_id,td_team,return_team,interception," +
		"punt_attempt,kickoff_attempt,play_deleted,two_point_attempt"
	play := func(game, week string) string {
		return game + ",1,2026," + week + ",REG,NA,00-0000001,NA,NA,40,NA,0,NA,NA,NA,0,0,0,0,0"
	}
	for _, tc := range []struct {
		name    string
		missing int
		held    []string
	}{
		{"week two missing", 2, []string{play("2026_01_A_B", "1"), play("2026_03_A_C", "3")}},
		{"last week missing", 3, []string{play("2026_01_A_B", "1"), play("2026_02_D_A", "2")}},
		{"first week missing", 1, []string{play("2026_02_D_A", "2"), play("2026_03_A_C", "3")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var gz bytes.Buffer
			w := gzip.NewWriter(&gz)
			_, _ = w.Write([]byte(cols + "\n" + strings.Join(tc.held, "\n") + "\n"))
			_ = w.Close()
			weekly := []source.PlayerWeek{week(1, "2026_01_A_B"), week(2, "2026_02_D_A"), week(3, "2026_03_A_C")}
			// A teamless proof for the missing week must not turn its unread stat row into a zero.
			proofs := []source.RawAbsenceProof{{GSIS: "00-0000001", Season: "2026", Week: strconv.Itoa(tc.missing),
				Evidence: source.UnrosteredEvidence}}
			got, err := (&Store{}).completedWeeks(tc.name, weekly, gz.Bytes(), proofs)
			if err != nil {
				t.Fatalf("one missing week stopped the board: %v", err)
			}
			scored := map[int]bool{}
			for _, w := range got {
				if w.Week == tc.missing {
					t.Fatalf("missing week %d read %+v, want pending", tc.missing, w)
				}
				scored[w.Week] = len(w.Line.Events["long_reception"]) == 1
			}
			for n := 1; n <= 3; n++ {
				if n != tc.missing && !scored[n] {
					t.Fatalf("week %d did not complete: %+v", n, got)
				}
			}
		})
	}
}
