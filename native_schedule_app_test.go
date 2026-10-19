package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func nativeOwnerScope() RawNativeScheduleScope {
	return RawNativeScheduleScope{League: "native.calendar", Franchise: "franchise.01",
		Season: 2026, LocalZone: "America/Los_Angeles"}
}

func heldNativeWeek(t *testing.T, app *App, week int, kickoff time.Time, final bool) {
	t.Helper()
	seconds := "3600"
	if final {
		seconds = "0"
	}
	body := fmt.Sprintf(`{"nflSchedule":{"week":"%d","matchup":[
		{"kickoff":"%d","gameSecondsRemaining":"%s","team":[{"id":"A"},{"id":"B"}]},
		{"kickoff":"%d","gameSecondsRemaining":"%s","team":[{"id":"C"},{"id":"D"}]}]}}`,
		week, kickoff.Add(time.Hour).Unix(), seconds, kickoff.Unix(), seconds)
	recordSeasonArchiveForm(t, app, "nflSchedule", fmt.Sprint(week), body, app.timeNow(), true)
}

func nativeOwnerRead(t *testing.T, app *App, at time.Time) (commissioner.NativeWeek, commissioner.FranchiseSchedule) {
	t.Helper()
	week, err := app.TargetNativeWeek(RawNativeWeekScope{RawNativeScheduleScope: nativeOwnerScope(), At: at})
	if err != nil {
		t.Fatal(err)
	}
	season, err := app.TargetNativeFranchiseSchedule(nativeOwnerScope())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(week.Schedule, season) {
		t.Fatal("owner readings disagree", week, season)
	}
	return week, season
}

func TestNativeScheduleOwnerCommandsRevertRestart(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 9, 10, 18, 0, 0, 0, time.UTC)
	app, restart := calendarTestApp(t, at)
	raw := seedScheduleTest(t, app)
	raw.Args.Schedule.Weeks = 2
	raw.Args.Schedule.Pattern = ""
	executeCommissionerTest(t, app, raw)
	first := time.Date(2026, 9, 11, 0, 20, 0, 0, time.UTC)
	heldNativeWeek(t, app, 1, first, false)
	heldNativeWeek(t, app, 2, first.AddDate(0, 0, 7), false)
	week, season := nativeOwnerRead(t, app, at)
	if week.Message != "The schedule is not published yet." || len(season.Games) != 0 || week.Current != nil {
		t.Fatal("unpublished generation leaked", week, season)
	}
	raw.Command, raw.Expected, raw.Request = commissioner.SchedulePublish, 7, "publish-owner"
	raw.Args = commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{}}
	executeCommissionerTest(t, app, raw)
	before, baseline := nativeOwnerRead(t, app, at)
	if before.Current == nil || before.Current.Opponent != "franchise.02" || before.NextWeek != 2 ||
		before.Current.Kickoff == nil || before.Current.Kickoff.Format(time.RFC3339) != "2026-09-10T17:20:00-07:00" ||
		len(baseline.Games) != 2 || baseline.Games[1].Week != 2 {
		t.Fatal("owner opponent, first kickoff or next week is wrong", before, baseline)
	}
	game := scheduleRead(t, app).Weeks[0].Games[0]
	raw.Command, raw.Expected, raw.Request = commissioner.ScheduleEdit, 8, "owner-draft-edit"
	raw.Args = commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{
		Week: 1, Games: []commissioner.ScheduleGame{{ID: game.ID, Home: game.Away, Away: game.Home}},
	}}
	executeCommissionerTest(t, app, raw)
	for range 2 {
		week, season = nativeOwnerRead(t, app, at)
		if !reflect.DeepEqual(week.Current, before.Current) || !reflect.DeepEqual(season.Games, baseline.Games) {
			t.Fatal("unpublished edit leaked to an owner", week, season)
		}
		restart()
	}
	raw.Args.Schedule.Published = true
	raw.Expected, raw.Request, raw.Reason = 9, "owner-published-edit", "Correct home field"
	changed := executeCommissionerTest(t, app, raw)
	for range 2 {
		week, season = nativeOwnerRead(t, app, at)
		if week.Current == nil || week.Current.Venue == before.Current.Venue ||
			season.Games[0].Venue == baseline.Games[0].Venue ||
			week.Current.Game != before.Current.Game || !week.Current.Kickoff.Equal(*before.Current.Kickoff) {
			t.Fatal("published edit was not read consistently", week, season)
		}
		restart()
	}
	raw.Command, raw.Expected, raw.Request = commissioner.Revert, 10, "owner-revert"
	raw.Args = commissioner.RawArgs{Entry: changed.Entry}
	executeCommissionerTest(t, app, raw)
	for range 2 {
		week, season = nativeOwnerRead(t, app, at)
		if !reflect.DeepEqual(week.Current, before.Current) || !reflect.DeepEqual(season.Games, baseline.Games) {
			t.Fatal("compensation did not restore both owner readings", week, season)
		}
		restart()
	}
	writeNativeScheduleWire(t, week)
}

func writeNativeScheduleWire(t *testing.T, reading commissioner.NativeWeek) {
	t.Helper()
	body, err := json.MarshalIndent(reading, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	body = append(body, '\n')
	path := filepath.Join("internal", "commissioner", "testdata", "native-schedule-wire.json")
	if os.Getenv("TWR_WRITE_WIRE") == "1" {
		if err := os.WriteFile(path, body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil || string(want) != string(body) {
		t.Fatal("native schedule wire differs; regenerate with TWR_WRITE_WIRE=1", err)
	}
}

func TestNativeScheduleMissingDatesAndScope(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 9, 10, 18, 0, 0, 0, time.UTC)
	app, _ := calendarTestApp(t, at)
	raw := seedScheduleTest(t, app)
	raw.Args.Schedule.Weeks = 2
	executeCommissionerTest(t, app, raw)
	raw.Command, raw.Expected, raw.Request = commissioner.SchedulePublish, 7, "publish-owner"
	raw.Args = commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{}}
	executeCommissionerTest(t, app, raw)
	week, season := nativeOwnerRead(t, app, at)
	if week.Current != nil || week.Message != "The NFL week dates are not held yet." || season.Games[0].Kickoff != nil {
		t.Fatal("invented week dates", week, season)
	}
	for _, change := range []func(*RawNativeScheduleScope){
		func(r *RawNativeScheduleScope) { r.League = "mfl.1" },
		func(r *RawNativeScheduleScope) { r.Franchise = "unknown" },
		func(r *RawNativeScheduleScope) { r.Season = 2025 },
		func(r *RawNativeScheduleScope) { r.LocalZone = "Local" },
		func(r *RawNativeScheduleScope) { r.LocalZone = "Not/AZone" },
		func(r *RawNativeScheduleScope) { r.Season = 0 },
	} {
		scope := nativeOwnerScope()
		change(&scope)
		if _, err := app.TargetNativeFranchiseSchedule(scope); err == nil {
			t.Fatal("accepted invalid native owner scope", scope)
		}
	}
	if _, err := app.TargetNativeWeek(RawNativeWeekScope{RawNativeScheduleScope: nativeOwnerScope()}); err == nil {
		t.Fatal("accepted zero reading time")
	}
}

func TestNativeScheduleRegenerationChangesOwnerOpponentRevertRestart(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 9, 10, 18, 0, 0, 0, time.UTC)
	app, restart := calendarTestApp(t, at)
	raw := commissioner.RawCommand{League: "native.calendar", Command: commissioner.Create,
		Args:   commissioner.RawArgs{Name: "Four teams", Preset: commissioner.PresetEmpty, FranchiseCount: 4, Season: 2026},
		Reason: "Start our league", Request: "create", Effective: "now", Visibility: "league"}
	executeCommissionerTest(t, app, raw)
	raw.Command, raw.Expected, raw.Request = commissioner.RulesSet, 1, "align"
	raw.Args = commissioner.RawArgs{Field: "league.alignment", Value: json.RawMessage(
		`{"franchise.01":"one","franchise.02":"one","franchise.03":"two","franchise.04":"two"}`)}
	executeCommissionerTest(t, app, raw)
	raw.Command, raw.Expected, raw.Request = commissioner.ScheduleGenerate, 2, "generate"
	raw.Args = commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{Weeks: 1, Pattern: "division_weighted"}}
	executeCommissionerTest(t, app, raw)
	raw.Command, raw.Expected, raw.Request = commissioner.SchedulePublish, 3, "publish"
	raw.Args = commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{}}
	executeCommissionerTest(t, app, raw)
	heldNativeWeek(t, app, 1, at.Add(time.Hour), false)
	before, baseline := nativeOwnerRead(t, app, at)
	if before.Current == nil || before.Current.Opponent != "franchise.02" {
		t.Fatal(before)
	}
	game := scheduleRead(t, app).Weeks[0].Games[0]
	raw.Command, raw.Expected, raw.Request = commissioner.ScheduleEdit, 4, "double-booking"
	raw.Args = commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{
		Week: 1, Games: []commissioner.ScheduleGame{{ID: game.ID, Home: "franchise.01", Away: "franchise.03"}}, Published: true,
	}}
	body, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.TargetCommissionerExecute(string(body)); err == nil {
		t.Fatal("accepted duplicate participation and an idle franchise")
	}
	ledger, err := app.TargetCommissionerLedger(raw.League)
	if err != nil || ledger.Revision != 4 {
		t.Fatal("refused edit appended an event", ledger, err)
	}
	raw.Command, raw.Request = commissioner.RulesSet, "realign"
	raw.Args = commissioner.RawArgs{Field: "league.alignment", Value: json.RawMessage(
		`{"franchise.01":"one","franchise.03":"one","franchise.02":"two","franchise.04":"two"}`)}
	executeCommissionerTest(t, app, raw)
	raw.Command, raw.Expected, raw.Request = commissioner.ScheduleGenerate, 5, "regenerate"
	raw.Args = commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{Weeks: 1, Pattern: "division_weighted"}}
	executeCommissionerTest(t, app, raw)
	week, season := nativeOwnerRead(t, app, at)
	if !reflect.DeepEqual(week.Current, before.Current) || !reflect.DeepEqual(season.Games, baseline.Games) {
		t.Fatal("regeneration exposed a draft", week, season)
	}
	raw.Command, raw.Expected, raw.Request = commissioner.SchedulePublish, 6, "republish"
	raw.Args = commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{}}
	published := executeCommissionerTest(t, app, raw)
	for range 2 {
		week, season = nativeOwnerRead(t, app, at)
		if week.Current == nil || week.Current.Opponent != "franchise.03" || season.Games[0].Opponent != "franchise.03" ||
			!week.Current.Kickoff.Equal(*before.Current.Kickoff) {
			t.Fatal("new publication did not change both opponents, or moved the NFL kickoff", week, season)
		}
		restart()
	}
	raw.Command, raw.Expected, raw.Request = commissioner.Revert, 7, "restore-publication"
	raw.Args = commissioner.RawArgs{Entry: published.Entry}
	executeCommissionerTest(t, app, raw)
	for range 2 {
		week, season = nativeOwnerRead(t, app, at)
		if !reflect.DeepEqual(week.Current, before.Current) || !reflect.DeepEqual(season.Games, baseline.Games) {
			t.Fatal("publication reversal did not restore both opponents", week, season)
		}
		restart()
	}
}
