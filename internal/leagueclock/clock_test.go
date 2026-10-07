package leagueclock

import (
	"testing"
	"time"
)

const instant = "2026-10-06T12:00:00Z"

func clockAt(t *testing.T) time.Time {
	t.Helper()
	at, err := time.Parse(time.RFC3339, instant)
	if err != nil {
		t.Fatal(err)
	}
	return at
}

func planned(id string, when time.Time) Event {
	return Event{ID: id, Kind: id, Status: "PLANNED", At: when}
}

func TestUrgencyBoundaries(t *testing.T) {
	at := clockAt(t)
	for _, tc := range []struct {
		name             string
		delta            time.Duration
		urgency          Urgency
		pinned, promoted bool
	}{
		{"passed", -time.Second, U3, true, true},
		{"now", 0, U3, true, true},
		{"under hour", time.Hour - time.Nanosecond, U3, true, true},
		{"hour", time.Hour, U2, true, true},
		{"48h", 48 * time.Hour, U2, true, true},
		{"over 48h", 48*time.Hour + time.Nanosecond, U1, false, true},
		{"7d", 7 * 24 * time.Hour, U1, false, true},
		{"over 7d", 7*24*time.Hour + time.Nanosecond, U1, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := Clock(at, Inputs{Events: []Event{planned("x", at.Add(tc.delta))}}).Deadlines[0]
			if got.Urgency != tc.urgency || got.Pinned != tc.pinned || got.Promoted != tc.promoted {
				t.Fatal(got)
			}
		})
	}
}

func TestEmptyReadingKeepsWindowsUnknown(t *testing.T) {
	at := clockAt(t)
	empty := Clock(at, Inputs{})
	if len(empty.Deadlines) != 0 || empty.Deadlines == nil || len(empty.Windows) != 7 {
		t.Fatal(empty)
	}
	for _, w := range empty.Windows {
		if w.Status != WindowUnknown || w.Reason == "" {
			t.Fatal(w)
		}
	}
}

func TestDeadlinesSoonestFirstUndatedLast(t *testing.T) {
	at := clockAt(t)
	got := Clock(at, Inputs{Events: []Event{
		planned("later", at.Add(5*24*time.Hour)),
		planned("none", time.Time{}),
		planned("sooner", at.Add(3*24*time.Hour)),
		planned("b", at),
		planned("a", at),
		{ID: "fired", Status: "FIRED", At: at},
		{ID: "cancelled", Status: "CANCELLED", At: at},
	}})
	want := []string{"a", "b", "sooner", "later", "none"}
	if len(got.Deadlines) != len(want) {
		t.Fatal(got.Deadlines)
	}
	for i, id := range want {
		if got.Deadlines[i].ID != id {
			t.Fatalf("position %d: got %s, want %s", i, got.Deadlines[i].ID, id)
		}
	}
	if last := got.Deadlines[4]; last.Urgency != U0 || last.At != nil {
		t.Fatal(last)
	}
}

func TestTimesAreUTC(t *testing.T) {
	at := clockAt(t)
	local := at.In(time.FixedZone("offset", -5*3600))
	d := Clock(local, Inputs{Events: []Event{planned("x", local)}}).Deadlines[0]
	if d.At.Location() != time.UTC || !d.At.Equal(at) {
		t.Fatal(d)
	}
}
