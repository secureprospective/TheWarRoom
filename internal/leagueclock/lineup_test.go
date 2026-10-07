package leagueclock

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestLineupDeadline(t *testing.T) {
	at := clockAt(t)
	for _, tc := range []struct {
		delta   time.Duration
		urgency Urgency
	}{
		{30 * time.Minute, U3},
		{time.Hour, U2},
		{48 * time.Hour, U2},
	} {
		lock := at.Add(tc.delta)
		in := Inputs{
			Lineup: &LineupLock{Week: 5, At: lock},
			Events: []Event{planned("later", lock.Add(time.Hour)), planned("earlier", lock.Add(-time.Hour))},
		}
		got := Clock(at, in)
		if got.Week == nil || *got.Week != 5 || len(got.Deadlines) != 3 {
			t.Fatal(got)
		}
		d := got.Deadlines[1]
		if got.Deadlines[0].ID != "earlier" || got.Deadlines[2].ID != "later" ||
			d.ID != "lineup-w5" || d.Label != "LINEUP_LOCK" || !d.At.Equal(lock) ||
			d.Urgency != tc.urgency || !d.Pinned || !d.Promoted {
			t.Fatal(got)
		}
		in.Lineup.Week = 6
		if *got.Week != 5 {
			t.Fatal("reading aliases mutable inputs")
		}
	}
}

func TestWeekJSON(t *testing.T) {
	for _, tc := range []struct {
		lineup *LineupLock
		want   string
	}{
		{nil, ""},
		{&LineupLock{Week: 5}, `"week":5`},
	} {
		body, err := json.Marshal(Clock(clockAt(t), Inputs{Lineup: tc.lineup}))
		if err != nil {
			t.Fatal(err)
		}
		if tc.want == "" {
			if strings.Contains(string(body), `"week"`) {
				t.Fatal(string(body))
			}
		} else if !strings.Contains(string(body), tc.want) {
			t.Fatal(string(body))
		}
	}
}
