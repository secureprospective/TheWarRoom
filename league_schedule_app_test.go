package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func scheduleRead(t *testing.T, app *App) commissioner.Schedule {
	t.Helper()
	reading, err := app.TargetLeagueSchedule(RawLeagueScheduleScope{League: "native.calendar"})
	if err != nil {
		t.Fatal(err)
	}
	return reading
}

func seedScheduleTest(t *testing.T, app *App) commissioner.RawCommand {
	t.Helper()
	raw := seedCalendarTest(t, app)
	for i, setting := range []struct{ field, value string }{
		{"league.alignment", `{"franchise.01":"one","franchise.02":"one"}`},
		{"schedule.pattern", `"division_weighted"`},
	} {
		raw.Args = commissioner.RawArgs{Field: setting.field, Value: json.RawMessage(setting.value)}
		raw.Expected, raw.Request = int64(i+4), setting.field
		executeCommissionerTest(t, app, raw)
	}
	raw.Command, raw.Args = commissioner.ScheduleGenerate,
		commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{Weeks: 13, Pattern: leaguerules.DivisionWeighted}}
	raw.Expected, raw.Request = 6, "generate"
	return raw
}

func TestSchedulePublicationOwnerRevertRestart(t *testing.T) {
	app, restart := calendarTestApp(t, time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC))
	raw := seedScheduleTest(t, app)
	generated := executeCommissionerTest(t, app, raw)
	if retry := executeCommissionerTest(t, app, raw); retry.Entry != generated.Entry {
		t.Fatal("retry appended a second draft")
	}
	for range 2 {
		if len(scheduleRead(t, app).Weeks) != 0 {
			t.Fatal("owners read an unpublished draft")
		}
		restart()
	}
	raw.Command, raw.Expected, raw.Request = commissioner.SchedulePublish, 7, "publish"
	raw.Args = commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{}}
	published := executeCommissionerTest(t, app, raw)
	baseline := scheduleRead(t, app).Weeks
	if len(baseline) != 13 || baseline[0].Games[0].Home == baseline[0].Games[0].Away {
		t.Fatal(baseline)
	}
	restart()
	if !reflect.DeepEqual(scheduleRead(t, app).Weeks, baseline) {
		t.Fatal("publication lost on restart")
	}
	raw.Command, raw.Args = commissioner.Revert, commissioner.RawArgs{Entry: published.Entry}
	raw.Expected, raw.Request = 8, "unpublish"
	executeCommissionerTest(t, app, raw)
	if len(scheduleRead(t, app).Weeks) != 0 {
		t.Fatal("compensation left schedule published")
	}
	restart()
	if len(scheduleRead(t, app).Weeks) != 0 {
		t.Fatal("replay left schedule published")
	}
}

func TestScheduleEditOwnerRevertRestart(t *testing.T) {
	app, restart := calendarTestApp(t, time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC))
	raw := seedScheduleTest(t, app)
	executeCommissionerTest(t, app, raw)
	raw.Command, raw.Expected, raw.Request = commissioner.SchedulePublish, 7, "publish"
	raw.Args = commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{}}
	executeCommissionerTest(t, app, raw)
	before := scheduleRead(t, app).Weeks
	changed := commissioner.ScheduleWeek{Week: 1, Games: []commissioner.ScheduleGame{
		{ID: before[0].Games[0].ID, Home: before[0].Games[0].Away, Away: before[0].Games[0].Home},
	}}
	raw.Command, raw.Expected, raw.Request = commissioner.ScheduleEdit, 8, "edit-draft"
	raw.Args = commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{Week: 1, Game: changed.Games[0].ID,
		Home: changed.Games[0].Home, Away: changed.Games[0].Away}}
	executeCommissionerTest(t, app, raw)
	if !reflect.DeepEqual(scheduleRead(t, app).Weeks, before) {
		t.Fatal("draft edit leaked to owners")
	}
	raw.Args.Schedule.Published = true
	raw.Expected, raw.Request, raw.Reason = 9, "edit-published", ""
	body, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := app.TargetCommissionerExecute(string(body))
	if err == nil {
		t.Fatal("published edit accepted without reason", outcome, err)
	}
	raw.Reason = "Correct home field"
	receipt := executeCommissionerTest(t, app, raw)
	ledger, err := app.TargetCommissionerLedger(raw.League)
	if err != nil {
		t.Fatal(err)
	}
	if ledger.Entries[0].Gravity != "Recorded reason required" {
		t.Fatal("published edit lost G3", ledger)
	}
	for range 2 {
		if !reflect.DeepEqual(scheduleRead(t, app).Weeks[0], changed) {
			t.Fatal("owner did not read edited week")
		}
		restart()
	}
	raw.Command, raw.Args = commissioner.Revert, commissioner.RawArgs{Entry: receipt.Entry}
	raw.Expected, raw.Request = 10, "undo-edit"
	executeCommissionerTest(t, app, raw)
	for range 2 {
		if !reflect.DeepEqual(scheduleRead(t, app).Weeks, before) {
			t.Fatal("compensation did not restore week")
		}
		restart()
	}
	writeScheduleWire(t, scheduleRead(t, app))
}

func writeScheduleWire(t *testing.T, reading commissioner.Schedule) {
	t.Helper()
	body, err := json.MarshalIndent(reading, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	body = append(body, '\n')
	path := filepath.Join("internal", "commissioner", "testdata", "schedule-wire.json")
	if os.Getenv("TWR_WRITE_WIRE") == "1" {
		if err := os.WriteFile(path, body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != string(want) {
		t.Fatal("schedule wire differs; regenerate with TWR_WRITE_WIRE=1")
	}
}

func TestScheduleCommandSafety(t *testing.T) {
	app, _ := calendarTestApp(t, time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC))
	raw := seedScheduleTest(t, app)
	refuse := func(command commissioner.RawCommand) {
		t.Helper()
		body, err := json.Marshal(command)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := app.TargetCommissionerExecute(string(body)); err == nil {
			t.Fatal("accepted invalid schedule change", command)
		}
	}
	publish := raw
	publish.Command, publish.Request = commissioner.SchedulePublish, "empty-publication"
	publish.Args = commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{}}
	refuse(publish)
	stale := raw
	stale.Expected = 5
	refuse(stale)
	app.commissionerSession = testCommissionerSession{"outsider"}
	refuse(raw)
	app.commissionerSession = testCommissionerSession{"creator"}
	executeCommissionerTest(t, app, raw)
	raw.Command, raw.Expected = commissioner.ScheduleEdit, 7
	for i, patch := range []commissioner.RawScheduleArgs{
		{Week: 14, Game: "week.1.game.1", Home: "franchise.01", Away: "franchise.02"},
		{Week: 1, Game: "missing", Home: "franchise.01", Away: "franchise.02"},
		{Week: 1, Game: "week.1.game.1", Home: "franchise.01", Away: "franchise.01"},
		{Week: 1, Game: "week.1.game.1", Home: "franchise.01", Away: "foreign"},
		{Week: 1, Game: "week.1.game.1", Home: "franchise.01"},
	} {
		raw.Request = string(rune('a' + i))
		raw.Args = commissioner.RawArgs{Schedule: &patch}
		refuse(raw)
	}
	ledger, err := app.TargetCommissionerLedger(raw.League)
	if err != nil || ledger.Revision != 7 {
		t.Fatal("refusal appended an event", ledger, err)
	}
}

func TestSmallerScheduleRequiresCommissionerPattern(t *testing.T) {
	app, _ := calendarTestApp(t, time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC))
	raw := seedScheduleTest(t, app)
	raw.Command, raw.Args = commissioner.Revert, commissioner.RawArgs{Entry: ""}
	ledger, err := app.TargetCommissionerLedger(raw.League)
	if err != nil {
		t.Fatal(err)
	}
	raw.Args.Entry = ledger.Entries[0].ID
	raw.Request = "unset-pattern"
	executeCommissionerTest(t, app, raw)
	raw.Command, raw.Args = commissioner.ScheduleGenerate,
		commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{Weeks: 13}}
	raw.Expected, raw.Request = 7, "generate-without-pattern"
	body, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.TargetCommissionerExecute(string(body)); err == nil ||
		!strings.Contains(err.Error(), "Choose how often teams meet in this smaller league.") {
		t.Fatal("wrong unset-pattern consequence", err)
	}
	raw.Args.Schedule.Pattern = leaguerules.DivisionWeighted
	raw.Request = "generate-with-explicit-pattern"
	executeCommissionerTest(t, app, raw)
	raw.Command, raw.Expected, raw.Request = commissioner.SchedulePublish, 8, "publish-explicit-pattern"
	raw.Args = commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{}}
	executeCommissionerTest(t, app, raw)
	if len(scheduleRead(t, app).Weeks) != 13 {
		t.Fatal("explicit pattern did not override the unset schedule rule")
	}
}

func TestScheduleGenerationArgumentsSurviveRuleReversal(t *testing.T) {
	app, restart := calendarTestApp(t, time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC))
	raw := seedScheduleTest(t, app)
	ledger, err := app.TargetCommissionerLedger(raw.League)
	if err != nil {
		t.Fatal(err)
	}
	alignmentEntry := ledger.Entries[1].ID
	raw.Args.Schedule.Weeks = 3
	executeCommissionerTest(t, app, raw)
	raw.Command, raw.Expected, raw.Request = commissioner.SchedulePublish, 7, "publish-three"
	raw.Args = commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{}}
	executeCommissionerTest(t, app, raw)
	before := scheduleRead(t, app).Weeks
	if len(before) != 3 {
		t.Fatal("generate ignored its weeks argument", before)
	}
	raw.Command, raw.Expected, raw.Request = commissioner.Revert, 8, "remove-generation-alignment"
	raw.Args = commissioner.RawArgs{Entry: alignmentEntry}
	executeCommissionerTest(t, app, raw)
	for range 2 {
		if !reflect.DeepEqual(scheduleRead(t, app).Weeks, before) {
			t.Fatal("reversing generation inputs changed a frozen publication")
		}
		restart()
	}
}
