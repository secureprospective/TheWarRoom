package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/core"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

func calendarTestApp(t *testing.T, at time.Time) (*App, func()) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "history.db")
	reg, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	pools, err := db.Open(t.Context(), path, history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	app := &App{core: &core.State{Ctx: t.Context(), Started: make(chan struct{}), ClockNow: func() time.Time { return at },
		CommissionerSession: testCommissionerSession{"creator"}}}
	close(app.core.Started)
	initialize := func() {
		app.core.History = history.New(pools, reg)
		app.core.History.SetClock(func() time.Time { return at })
		if err := app.core.History.Initialize(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
	initialize()
	t.Cleanup(func() { _ = pools.Close() })
	return app, func() {
		if err := pools.Close(); err != nil {
			t.Fatal(err)
		}
		var err error
		pools, err = db.Open(t.Context(), path, history.SchemaVersion())
		if err != nil {
			t.Fatal(err)
		}
		initialize()
	}
}

func seedCalendarTest(t *testing.T, app *App) commissioner.RawCommand {
	t.Helper()
	raw := commissioner.RawCommand{League: "native.calendar", Command: commissioner.Create,
		Args:   commissioner.RawArgs{Name: "Calendar league", Preset: commissioner.PresetEmpty, Season: 2026, FranchiseCount: 2},
		Reason: "Start our league", Request: "create", Effective: "now", Visibility: "league"}
	executeCommissionerTest(t, app, raw)
	raw.Command = commissioner.RulesSet
	for i, setting := range []struct{ field, value string }{
		{"calendar.timezone", `"America/New_York"`},
		{"calendar.boundary", `"start_of_day"`},
		{"calendar.windows", `{
			"rfa":{"Start":{"Month":5,"Day":1},"End":{"Month":5,"Day":3},
			"Opens":["rfa_tender"]},
			"ufa":{"Start":{"Month":5,"Day":3},"End":{"Month":5,"Day":10},"Opens":["ufa_bid"]}
		}`},
	} {
		raw.Args = commissioner.RawArgs{Field: setting.field, Value: json.RawMessage(setting.value)}
		raw.Expected, raw.Request = int64(i+1), setting.field
		executeCommissionerTest(t, app, raw)
	}
	return raw
}

func calendarOwnerRead(t *testing.T, app *App) commissioner.Calendar {
	t.Helper()
	reading, err := app.TargetLeagueCalendar(RawLeagueCalendarScope{League: "native.calendar", Season: 2026,
		LocalZone: "America/Los_Angeles", Franchise: "franchise.01", Action: "ufa_bid"})
	if err != nil {
		t.Fatal(err)
	}
	return reading
}

func TestCalendarCommandOwnerRevertRestart(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 5, 3, 4, 0, 0, 0, time.UTC)
	app, restart := calendarTestApp(t, at)
	raw := seedCalendarTest(t, app)
	before := calendarOwnerRead(t, app)
	if !before.Decision.Open || before.Clock.Windows[1].Start.Format(time.RFC3339) != "2026-05-02T21:00:00-07:00" {
		t.Fatal(before)
	}
	raw.Args = commissioner.RawArgs{Field: "calendar.boundary", Value: json.RawMessage(`"end_of_day"`)}
	raw.Expected, raw.Request = 4, "side"
	changed := executeCommissionerTest(t, app, raw)
	for range 2 {
		reading := calendarOwnerRead(t, app)
		if reading.Decision.Open || reading.Decision.Window != "ufa" ||
			reading.Decision.Next.Format(time.RFC3339) != "2026-05-03T21:00:00-07:00" {
			t.Fatal(reading)
		}
		restart()
	}
	raw.Command, raw.Args = commissioner.Revert, commissioner.RawArgs{Entry: changed.Entry}
	raw.Expected, raw.Request = 5, "restore-side"
	executeCommissionerTest(t, app, raw)
	restart()
	if !calendarOwnerRead(t, app).Decision.Open {
		t.Fatal("compensation did not restore the action")
	}
	writeCalendarWire(t, calendarOwnerRead(t, app))
}

func TestCalendarExceptionCommandOwnerExpiryRevertRestart(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	app, restart := calendarTestApp(t, at)
	raw := seedCalendarTest(t, app)
	if calendarOwnerRead(t, app).Decision.Open {
		t.Fatal("the window is not open yet")
	}
	raw.Command, raw.Expected, raw.Request = commissioner.CalendarException, 4, "exception"
	raw.Args = commissioner.RawArgs{Exception: &commissioner.RawWindowException{
		Action: "ufa_bid", Franchise: "franchise.01", Until: at.Add(time.Hour)}}
	changed := executeCommissionerTest(t, app, raw)
	if again := executeCommissionerTest(t, app, raw); again.Entry != changed.Entry {
		t.Fatal("retry appended twice", again)
	}
	restart()
	if !calendarOwnerRead(t, app).Decision.Open {
		t.Fatal("exception lost on restart")
	}
	other, err := app.TargetLeagueCalendar(RawLeagueCalendarScope{League: raw.League, Season: 2026,
		LocalZone: "UTC", Franchise: "franchise.02", Action: "ufa_bid"})
	if err != nil || other.Decision.Open {
		t.Fatal(other, err)
	}
	app.core.ClockNow = func() time.Time { return at.Add(time.Hour) }
	if calendarOwnerRead(t, app).Decision.Open {
		t.Fatal("expired exception opened the action")
	}
	app.core.ClockNow = func() time.Time { return at }
	raw.Command, raw.Args = commissioner.Revert, commissioner.RawArgs{Entry: changed.Entry}
	raw.Expected, raw.Request = 5, "restore-exception"
	executeCommissionerTest(t, app, raw)
	restart()
	if calendarOwnerRead(t, app).Decision.Open {
		t.Fatal("compensated exception is still active")
	}
}

func TestCalendarExceptionSafety(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	app, _ := calendarTestApp(t, at)
	raw := seedCalendarTest(t, app)
	raw.Command, raw.Expected, raw.Request = commissioner.CalendarException, 4, "exception"
	raw.Args = commissioner.RawArgs{Exception: &commissioner.RawWindowException{
		Action: "ufa_bid", Franchise: "franchise.01", Until: at.Add(time.Hour)}}
	for _, change := range []func(*commissioner.RawCommand){
		func(r *commissioner.RawCommand) { r.Reason = " " },
		func(r *commissioner.RawCommand) { r.Visibility = "private" },
		func(r *commissioner.RawCommand) { r.Args.Exception.Until = at },
		func(r *commissioner.RawCommand) { r.Args.Exception.Franchise = "foreign" },
		func(r *commissioner.RawCommand) { r.Args.Exception.Action = "unknown" },
	} {
		trial := raw
		exception := *raw.Args.Exception
		trial.Args.Exception = &exception
		change(&trial)
		body, err := json.Marshal(trial)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := app.TargetCommissionerExecute(string(body)); err == nil {
			t.Fatal("accepted unsafe exception", string(body))
		}
	}
	if calendarOwnerRead(t, app).Revision != 4 {
		t.Fatal("refusal wrote an event")
	}
	app.core.CommissionerSession = testCommissionerSession{"outsider"}
	body, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.TargetCommissionerExecute(string(body)); err == nil {
		t.Fatal("owner bypassed commissioner permission")
	}
}

func writeCalendarWire(t *testing.T, reading commissioner.Calendar) {
	t.Helper()
	body, err := json.MarshalIndent(reading, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join("internal", "leagueclock", "testdata", "calendar-wire.json")
	if os.Getenv("TWR_WRITE_WIRE") == "1" {
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, append(body, '\n'), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	held, err := os.ReadFile(path)
	if err != nil || strings.TrimSpace(string(held)) != string(body) {
		t.Fatal("calendar wire changed; regenerate with TWR_WRITE_WIRE=1", err)
	}
}

func TestCalendarTimezoneCommandRevertRestart(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 3, 8, 5, 0, 0, 0, time.UTC)
	app, restart := calendarTestApp(t, at)
	raw := seedCalendarTest(t, app)
	raw.Args = commissioner.RawArgs{Field: "calendar.windows", Value: json.RawMessage(`{
		"ufa":{"Start":{"Month":3,"Day":8},"End":{"Month":3,"Day":8},"Opens":["ufa_bid"]}}`)}
	raw.Expected, raw.Request = 4, "dst-window"
	executeCommissionerTest(t, app, raw)
	before := calendarOwnerRead(t, app)
	if !before.Decision.Open || *before.Clock.Windows[0].Seconds != 23*3600 {
		t.Fatal(before)
	}
	raw.Args = commissioner.RawArgs{Field: "calendar.timezone", Value: json.RawMessage(`"America/Los_Angeles"`)}
	raw.Expected, raw.Request = 5, "zone"
	change := executeCommissionerTest(t, app, raw)
	restart()
	if calendarOwnerRead(t, app).Decision.Open {
		t.Fatal("the west-coast midnight has not happened yet")
	}
	raw.Command, raw.Args = commissioner.Revert, commissioner.RawArgs{Entry: change.Entry}
	raw.Expected, raw.Request = 6, "restore-zone"
	executeCommissionerTest(t, app, raw)
	restart()
	if got := calendarOwnerRead(t, app); !got.Decision.Open || *got.Clock.Windows[0].Seconds != 23*3600 {
		t.Fatal(got)
	}
}

func TestCalendarDeadlineCommandRevertRestart(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 5, 3, 4, 0, 0, 0, time.UTC)
	app, restart := calendarTestApp(t, at)
	raw := seedCalendarTest(t, app)
	raw.Args = commissioner.RawArgs{Field: "calendar.tradeDeadline", Value: json.RawMessage(`"2026-05-03T04:00:10Z"`)}
	raw.Expected, raw.Request = 4, "deadline"
	change := executeCommissionerTest(t, app, raw)
	for range 2 {
		got := calendarOwnerRead(t, app)
		if got.Clock.TradeDeadline.Format(time.RFC3339) != "2026-05-02T21:00:10-07:00" ||
			*got.Clock.TradeSeconds != 10 {
			t.Fatal(got)
		}
		restart()
	}
	raw.Command, raw.Args = commissioner.Revert, commissioner.RawArgs{Entry: change.Entry}
	raw.Expected, raw.Request = 5, "restore-deadline"
	executeCommissionerTest(t, app, raw)
	restart()
	if got := calendarOwnerRead(t, app); got.Clock.TradeDeadline != nil || got.Clock.TradeSeconds != nil {
		t.Fatal("a missing chosen moment must not be derived", got)
	}
}
