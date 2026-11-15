package leagueclock

import (
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func testDate(month, day int) leaguerules.Value[leaguerules.MonthDay] {
	return leaguerules.Known(leaguerules.MonthDay{
		Month: leaguerules.Known(month), Day: leaguerules.Known(day)})
}

func testWindows(windows map[string]leaguerules.Window) leaguerules.CalendarRules {
	return leaguerules.CalendarRules{Timezone: leaguerules.Known("America/New_York"),
		Windows: leaguerules.Known(windows), Boundary: leaguerules.Known(leaguerules.StartOfDay)}
}

func TestSharedDateHasOneSide(t *testing.T) {
	rules := testWindows(map[string]leaguerules.Window{
		"rfa": {Start: testDate(5, 1), End: testDate(5, 3),
			Opens: leaguerules.Known([]leaguerules.Action{"rfa_tender"})},
		"ufa": {Start: testDate(5, 3), End: testDate(5, 10),
			Opens: leaguerules.Known([]leaguerules.Action{"ufa_bid"})},
	})
	at := time.Date(2026, 5, 3, 4, 0, 0, 0, time.UTC)
	for _, side := range []leaguerules.BoundarySide{leaguerules.StartOfDay, leaguerules.EndOfDay} {
		rules.Boundary = leaguerules.Known(side)
		got, err := Calendar(at, 2026, rules, nil)
		if err != nil {
			t.Fatal(err)
		}
		if got.Decide(at, "rfa_tender", "one").Open != (side == leaguerules.EndOfDay) ||
			got.Decide(at, "ufa_bid", "one").Open != (side == leaguerules.StartOfDay) {
			t.Fatal(side, got)
		}
	}
	rules.Boundary = leaguerules.Needs[leaguerules.BoundarySide]("choose a side")
	got, err := Calendar(at, 2026, rules, nil)
	if err != nil {
		t.Fatal(err)
	}
	// An unset side is not a guess and not an empty calendar: the window that shares the date is
	// left out, the rest of the calendar reads, and neither shared action opens.
	if len(got.Windows) != 1 || got.Windows[0].ID != "rfa" {
		t.Fatal(got.Windows)
	}
	if got.Decide(at, "rfa_tender", "one").Open || got.Decide(at, "ufa_bid", "one").Open {
		t.Fatal("unset side must not guess")
	}
}

func TestExceptionIsScopedAndExpires(t *testing.T) {
	at := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	c := ActionCalendar{Exceptions: []WindowException{{Action: "buyout", Franchise: "one",
		Start: at, Until: at.Add(time.Hour), Reason: "Correct a missed deadline"}}}
	for _, tc := range []struct {
		action, franchise string
		at                time.Time
		open              bool
	}{
		{"buyout", "one", at, true}, {"buyout", "two", at, false},
		{"trade", "one", at, false}, {"buyout", "one", at.Add(time.Hour), false},
		{"buyout", "one", at.Add(-time.Second), false},
	} {
		if got := c.Decide(tc.at, leaguerules.Action(tc.action), tc.franchise); got.Open != tc.open {
			t.Fatal(tc, got)
		}
	}
}

func TestExactDeadline(t *testing.T) {
	at := time.Date(2026, 10, 29, 18, 30, 0, 0, time.UTC)
	rules := testWindows(map[string]leaguerules.Window{"trade": {
		Start: testDate(3, 1), Opens: leaguerules.Known([]leaguerules.Action{"trade"}),
	}})
	rules.TradeDeadline = leaguerules.Known(at.Add(time.Second))
	c, err := Calendar(at, 2026, rules, nil)
	if err != nil {
		t.Fatal(err)
	}
	if *c.TradeSeconds != 1 || !c.Decide(at, "trade", "one").Open ||
		c.Decide(at.Add(time.Second), "trade", "one").Open {
		t.Fatal(c)
	}
}
