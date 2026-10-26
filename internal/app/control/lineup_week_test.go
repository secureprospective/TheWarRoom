package control

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
)

func TestLineupWeekFrom(t *testing.T) {
	t.Parallel()
	now := time.Unix(1800000000, 0)
	raw := func(week int, final bool) nflschedule.RawWeek {
		kickoff := now.Add(time.Hour)
		remaining := "3600"
		if final {
			kickoff = now.Add(-5 * time.Hour)
			remaining = "0"
		}
		return nflschedule.RawWeek{Week: strconv.Itoa(week), Matchups: []nflschedule.RawMatchup{{Teams: [2]string{"NE", "NYJ"}, Kickoff: strconv.FormatInt(kickoff.Unix(), 10), GameSecondsRemaining: remaining}}}
	}
	for _, tc := range []struct {
		name    string
		rows    []nflschedule.RawWeek
		want    int
		calls   []int
		failure string
	}{
		{"default", []nflschedule.RawWeek{raw(2, false)}, 2, []int{0}, ""},
		{"advance", []nflschedule.RawWeek{raw(2, true), raw(3, false)}, 3, []int{0, 3}, ""},
		{"final season", []nflschedule.RawWeek{raw(18, true)}, 0, []int{0}, ""},
		{"beyond season", []nflschedule.RawWeek{raw(19, false)}, 0, []int{0}, "week refresh: requested 0, received 19"},
		{"wrong week", []nflschedule.RawWeek{raw(2, true), raw(4, false)}, 0, []int{0, 3}, "week refresh: requested 3, received 4"},
		{"invalid", []nflschedule.RawWeek{{Week: "bad"}}, 0, []int{0}, "week refresh: convert:"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var calls []int
			got, err := LineupWeekFrom(now, func(w int) (nflschedule.RawWeek, error) {
				calls = append(calls, w)
				if len(calls) > len(tc.rows) {
					t.Fatal("unexpected read")
				}
				return tc.rows[len(calls)-1], nil
			})
			if got.Number != tc.want || !reflect.DeepEqual(calls, tc.calls) {
				t.Fatalf("week %d calls %v", got.Number, calls)
			}
			if tc.failure == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.HasPrefix(err.Error(), tc.failure) {
				t.Fatalf("error %v", err)
			}
		})
	}
	sentinel := errors.New("read failed")
	if _, err := LineupWeekFrom(now, func(int) (nflschedule.RawWeek, error) { return nflschedule.RawWeek{}, sentinel }); !errors.Is(err, sentinel) || err.Error() != "week refresh: read failed" {
		t.Fatalf("wrapped error: %v", err)
	}
}
