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
