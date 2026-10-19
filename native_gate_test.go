package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func setDeadline(t *testing.T, app *App, raw commissioner.RawCommand, request string, expected int64,
	instant string) commissioner.Receipt {
	t.Helper()
	raw.Command, raw.Request, raw.Expected = commissioner.RulesSet, request, expected
	raw.Args = commissioner.RawArgs{Field: "calendar.tradeDeadline", Value: json.RawMessage(`"` + instant + `"`)}
	return executeCommissionerTest(t, app, raw)
}

func revertEntry(t *testing.T, app *App, raw commissioner.RawCommand, entry, request string, expected int64) {
	t.Helper()
	raw.Command, raw.Request, raw.Expected = commissioner.Revert, request, expected
	raw.Args = commissioner.RawArgs{Entry: entry}
	executeCommissionerTest(t, app, raw)
}

// TestEveryRegisteredNativeActionAsksTheGate runs each registered native owner action with its window
// shut and then open: a shut gate must stop it, an open one must run it for that franchise only.
func TestEveryRegisteredNativeActionAsksTheGate(t *testing.T) {
	t.Parallel()
	shut := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	app, _ := calendarTestApp(t, shut)
	app.localZone = westCoast(t)
	raw := seedCalendarTest(t, app)
	probe := newGateProbe("rfa_tender")
	registry := probe.registry
	expected := switchContractRulesOn(t, app, raw, 4)
	for i, entry := range registry {
		expected++
		executeCommissionerTest(t, app, windowCommandFrom(raw, commissioner.WindowSet,
			fmt.Sprintf("guard.%d", i), expected-1, commissioner.RawWindowArgs{ID: fmt.Sprintf("guard%d", i),
				Start: date(6, 1), End: date(6, 5), Opens: []leaguerules.Action{entry.Opens}}))
	}
	if entry := registry[len(registry)-1]; entry.Name != "probe" {
		t.Fatal("the probe must be registered with the production actions")
	}
	setDeadline(t, app, raw, "deadline", expected, "2026-12-01T00:00:00Z")
	for _, entry := range registry {
		ran := 0
		entry := entry
		run := entry.Run
		entry.Run = func(ctx context.Context, a *App, p commissioner.Permit, in json.RawMessage) (commissioner.Receipt, error) {
			ran++
			return run(ctx, a, p, in)
		}
		one := []NativeOwnerAction{entry}
		// The guard is about the gate, not the action's own work: an action handed no request fails inside
		// its body, and that must not be mistaken for the gate's no.
		if _, err := app.runNativeAction(one, raw.League, "franchise.01", entry.Name, nil); !errors.Is(err, errActionClosed) || ran != 0 {
			t.Fatalf("%s ran with its window shut (ran %d, %v)", entry.Name, ran, err)
		}
		app.clockNow = func() time.Time { return time.Date(2026, 6, 2, 12, 0, 0, 0, time.UTC) }
		// A real action may refuse for want of its own details; it must still have been let through the gate.
		if _, err := app.runNativeAction(one, raw.League, "franchise.01", entry.Name, nil); errors.Is(err, errActionClosed) || ran != 1 {
			t.Fatalf("%s did not run in its window (ran %d, %v)", entry.Name, ran, err)
		}
		app.clockNow = func() time.Time { return shut }
	}
	if _, err := app.runNativeAction(registry, raw.League, "franchise.01", "unregistered", nil); err == nil {
		t.Fatal("an action outside the registry ran")
	}
}

func TestTradeDeadlineClosesTheGateAtTheChosenInstant(t *testing.T) {
	t.Parallel()
	recorded := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	app, restart := calendarTestApp(t, recorded)
	app.localZone = westCoast(t)
	raw := seedCalendarTest(t, app)
	executeCommissionerTest(t, app, windowCommandFrom(raw, commissioner.WindowSet, "season", 4,
		commissioner.RawWindowArgs{ID: "season", Start: date(3, 1), Opens: []leaguerules.Action{"trade"}}))
	probe := newGateProbe("trade")
	check := func(now string, open bool, sentence string) {
		t.Helper()
		instant, err := time.Parse(time.RFC3339, now)
		if err != nil {
			t.Fatal(err)
		}
		app.clockNow = func() time.Time { return instant }
		ran, why := probe.try(t, app, raw.League, "franchise.01")
		if ran != open || why != sentence {
			t.Fatalf("at %s: open=%v %q, want open=%v %q", now, ran, why, open, sentence)
		}
	}
	check("2026-10-30T12:00:00Z", false, "The trade deadline needs a ruling.")
	first := "2026-10-31T16:00:00Z"
	setDeadline(t, app, raw, "deadline-1", 5, first)
	passed := func(label string) string { return "The trade deadline passed on " + label + "." }
	for range 2 {
		check("2026-10-30T12:00:00Z", true, "")
		check("2026-10-31T15:59:59Z", true, "")
		check(first, false, passed("Oct 31, 2026 at 9:00 AM PDT"))
		restart()
	}
	second := setDeadline(t, app, raw, "deadline-2", 6, "2026-10-29T16:00:00Z")
	for range 2 {
		check("2026-10-29T15:59:59Z", true, "")
		check("2026-10-30T12:00:00Z", false, passed("Oct 29, 2026 at 9:00 AM PDT"))
		restart()
	}
	revertEntry(t, app, raw, second.Entry, "restore-deadline", 7)
	for range 2 {
		check("2026-10-30T12:00:00Z", true, "")
		check(first, false, passed("Oct 31, 2026 at 9:00 AM PDT"))
		restart()
	}
}

func TestWindowExceptionOpensOneFranchiseUntilItExpires(t *testing.T) {
	t.Parallel()
	recorded := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	app, restart := calendarTestApp(t, recorded)
	app.localZone = westCoast(t)
	raw := seedCalendarTest(t, app)
	probe := newGateProbe("ufa_bid")
	until := recorded.Add(time.Hour)
	raw.Command, raw.Request, raw.Expected = commissioner.CalendarException, "exception", 4
	raw.Args = commissioner.RawArgs{Exception: &commissioner.RawWindowException{
		Action: "ufa_bid", Franchise: "franchise.01", Until: until}}
	exception := executeCommissionerTest(t, app, raw)
	at := func(now time.Time) { app.clockNow = func() time.Time { return now } }
	decide := func(now time.Time, one, two bool) {
		t.Helper()
		at(now)
		for franchise, want := range map[string]bool{"franchise.01": one, "franchise.02": two} {
			if ran, why := probe.try(t, app, raw.League, franchise); ran != want {
				t.Fatalf("at %s %s: ran=%v (%s), want %v", now.Format(time.RFC3339), franchise, ran, why, want)
			}
		}
	}
	for range 2 {
		decide(recorded.Add(time.Minute), true, false)
		decide(until.Add(-time.Second), true, false)
		decide(until, false, false)
		restart()
	}
	at(recorded.Add(time.Minute))
	other := newGateProbe("rfa_tender")
	if ran, _ := other.try(t, app, raw.League, "franchise.01"); ran {
		t.Fatal("an exception for one action opened another")
	}
	revertEntry(t, app, raw, exception.Entry, "restore-exception", 5)
	for range 2 {
		decide(recorded.Add(time.Minute), false, false)
		restart()
	}
	if !strings.Contains(fmt.Sprint(probe.ran), "franchise.01") || len(probe.ran) != 4 {
		t.Fatalf("the exception ran %v", probe.ran)
	}
}
