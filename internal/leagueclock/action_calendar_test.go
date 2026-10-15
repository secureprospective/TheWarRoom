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

func TestCivilMidnightAcrossDaylightSaving(t *testing.T) {
	for _, tc := range []struct {
		month, day, hours int
		start, close      string
	}{
		{3, 8, 23, "2026-03-08T05:00:00Z", "2026-03-09T04:00:00Z"},
		{11, 1, 25, "2026-11-01T04:00:00Z", "2026-11-02T05:00:00Z"},
	} {
		rules := testWindows(map[string]leaguerules.Window{"test": {
			Start: testDate(tc.month, tc.day), End: testDate(tc.month, tc.day),
			Opens: leaguerules.Known([]leaguerules.Action{"buyout"}),
		}})
		at, err := time.Parse(time.RFC3339, tc.start)
		if err != nil {
			t.Fatal(err)
		}
		got, err := Calendar(at, 2026, rules, nil)
		if err != nil {
			t.Fatal(err)
		}
		w := got.Windows[0]
		if w.Start.Format(time.RFC3339) != tc.start || w.Close.Format(time.RFC3339) != tc.close ||
			*w.Seconds != int64(tc.hours*3600) || !w.Live {
			t.Fatal(w)
		}
		if got.Decide(*w.Close, "buyout", "one").Open {
			t.Fatal("closing midnight must exclude the action")
		}
	}
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
	if _, err := Calendar(at, 2026, rules, nil); err == nil {
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

func TestExactDeadlineAndBidCountdown(t *testing.T) {
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
	countdown := Countdown(at, at.Add(24*time.Hour), 20*time.Hour)
	if !countdown.SnipeMark.Equal(at.Add(20*time.Hour)) || !countdown.Close.Equal(at.Add(24*time.Hour)) {
		t.Fatal(countdown)
	}
	custom := Countdown(at, at.Add(24*time.Hour), 18*time.Hour)
	if !custom.SnipeMark.Equal(at.Add(18 * time.Hour)) {
		t.Fatal("threshold must remain a commissioner setting", custom)
	}
}

// Lineups have no closing window in the Legacy preset: each player locks at his own kickoff instead.
func TestLegacyLineupsStayOpenAllLeagueYear(t *testing.T) {
	rules := leaguerules.LegacyNFL().Calendar
	rules.Boundary = leaguerules.Known(leaguerules.StartOfDay)
	for _, at := range []string{"2026-03-01T05:00:00Z", "2026-09-13T17:00:00Z", "2026-12-27T18:00:00Z"} {
		instant, err := time.Parse(time.RFC3339, at)
		if err != nil {
			t.Fatal(err)
		}
		c, err := Calendar(instant, 2026, rules, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !c.Decide(instant, "lineup", "one").Open {
			t.Fatalf("a Legacy lineup must be open at %s", at)
		}
	}
}
