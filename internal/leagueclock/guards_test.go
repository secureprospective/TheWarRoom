package leagueclock

import (
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func TestUnruledBoundaryKeepsTheRestOfTheCalendar(t *testing.T) {
	rules := leaguerules.LegacyNFL().Calendar
	at := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	got, err := Calendar(at, 2026, rules, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Windows)+2 != len(rules.Windows.Value) {
		t.Fatalf("windows read: %d of %d", len(got.Windows), len(rules.Windows.Value))
	}
	if !got.Decide(at, "lineup", "one").Open {
		t.Fatal("a window with unambiguous dates must still open its action")
	}
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
