package leagueclock

import (
	"slices"
	"strings"
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
	if len(empty.Deadlines) != 0 || empty.Deadlines == nil || len(empty.Windows) != 9 {
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

func TestWindowsDateFromTheRulebookAndTheCommissionerWins(t *testing.T) {
	at := clockAt(t)
	kick := func(days int) time.Time { return at.Add(time.Duration(days) * 24 * time.Hour) }
	got := Clock(at, Inputs{Rules: WindowInputs{
		LastRegular:  13,
		FirstKickoff: map[int]time.Time{9: kick(-3), 13: kick(20), 14: kick(27)},
		Moved:        map[WindowKind]time.Time{CutDay: kick(40)},
	}})
	byKind := map[WindowKind]Window{}
	for _, w := range got.Windows {
		byKind[w.Kind] = w
	}
	if w := byKind[TradeDeadline]; w.Status != WindowDated || !w.At.Equal(kick(-3)) || w.Moved || !strings.Contains(w.Reason, "§14") {
		t.Fatal("trade deadline from Week 9's first kickoff", w)
	}
	if w := byKind[PlayoffBidsOnly]; !w.At.Equal(kick(20)) || !strings.Contains(w.Reason, "after Week 12") {
		t.Fatal("§6 narrowing at Week 13's first kickoff", w)
	}
	if w := byKind[MultiYearBidsClose]; !w.At.Equal(kick(27)) || !strings.Contains(w.Reason, "end of Week 13") {
		t.Fatal("§6 close at Week 14's first kickoff", w)
	}
	if w := byKind[CutDay]; !w.Moved || !w.At.Equal(kick(40)) || w.Reason != "Set by the commissioner" {
		t.Fatal("the commissioner's date wins", w)
	}
	if w := byKind[Draft]; w.Status != WindowUnknown || w.At != nil {
		t.Fatal("no rule and no move stays unknown", w)
	}
	ids := []string{}
	for _, d := range got.Deadlines {
		ids = append(ids, d.ID)
	}
	if want := []string{"window:playoff_bids_only", "window:multiyear_bids_close", "window:cut_day"}; !slices.Equal(ids, want) {
		t.Fatal("only upcoming dated windows are deadlines, soonest first", ids)
	}
	unheld := Clock(at, Inputs{Rules: WindowInputs{LastRegular: 13}})
	if w := unheld.Windows[0]; w.Status != WindowUnknown || !strings.Contains(w.Reason, "Week 9 schedule not held yet") {
		t.Fatal(w)
	}
}
