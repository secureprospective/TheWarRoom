package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// lineupApp is the seeded Legacy league with its owner of franchise.02 seated, the NFL's week 1 held with
// the fixture players' team kicking off on Sunday, and a clock the test moves.
type lineupApp struct {
	*legalityFixture
	at       time.Time
	kickoff  time.Time
	requests int
}

func lineupAppTest(t *testing.T) *lineupApp {
	t.Helper()
	f := &lineupApp{legalityFixture: legalityApp(t), at: time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC),
		kickoff: time.Date(2026, 10, 18, 17, 0, 0, 0, time.UTC)}
	f.wire(t)
	body := fmt.Sprintf(`{"nflSchedule":{"week":"1","matchup":[
		{"kickoff":"%d","gameSecondsRemaining":"3600","team":[{"id":"AAA"},{"id":"BBB"}]},
		{"kickoff":"%d","gameSecondsRemaining":"3600","team":[{"id":"CCC"},{"id":"DDD"}]}]}}`,
		f.kickoff.Unix(), f.kickoff.Add(3*time.Hour).Unix())
	recordSeasonArchiveForm(t, f.app, "nflSchedule", "1", body, f.at, true)
	f.execute(t, commissioner.RawCommand{Command: "commish.seat.replace", Visibility: "league",
		Args: commissioner.RawArgs{People: &commissioner.RawPeople{Franchise: "franchise.02", Principal: "creator"}}})
	f.execute(t, commissioner.RawCommand{Command: commissioner.ScheduleGenerate,
		Args: commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{Weeks: 13}}})
	f.execute(t, commissioner.RawCommand{Command: commissioner.SchedulePublish,
		Args: commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{}}})
	return f
}

// wire pins the app's clock, zone and lineup facts; a restarted app needs it again.
func (f *lineupApp) wire(t *testing.T) {
	t.Helper()
	clock := func() time.Time { return f.at }
	f.app.clockNow, f.app.localZone = clock, westCoast(t)
	f.app.history.SetClock(clock)
	f.app.history.SetLineupEnvironment(appLineupEnv{app: f.app})
}

func (f *lineupApp) version(t *testing.T) int64 {
	t.Helper()
	houses, err := f.app.TargetNativeHouseRules(legalityLeague)
	if err != nil {
		t.Fatal(err)
	}
	return houses.Revision
}

// execute runs a commissioner command at the league's current version.
func (f *lineupApp) execute(t *testing.T, raw commissioner.RawCommand) commissioner.Receipt {
	t.Helper()
	f.requests++
	raw.League, raw.Expected, raw.Request = legalityLeague, f.version(t), fmt.Sprintf("lineup-app.%d", f.requests)
	if raw.Reason == "" {
		raw.Reason = "Our league's ruling"
	}
	if raw.Effective == "" {
		raw.Effective = "now"
	}
	if raw.Visibility == "" {
		raw.Visibility = "league"
	}
	return executeCommissionerTest(t, f.app, raw)
}

func (f *lineupApp) openLineupWindow(t *testing.T) {
	t.Helper()
	f.execute(t, commissioner.RawCommand{Command: commissioner.WindowSet, Visibility: "league",
		Args: commissioner.RawArgs{Window: &commissioner.RawWindowArgs{ID: "lineups", Start: date(1, 1), End: date(12, 31),
			Opens: []leaguerules.Action{"lineup"}}}})
}

func (f *lineupApp) setBody(t *testing.T, franchise string, week int, starters []string) string {
	t.Helper()
	f.requests++
	body, err := json.Marshal(RawLineupSet{League: legalityLeague, Franchise: franchise, Season: 2026, Week: week,
		Starters: starters, Expected: f.version(t), Request: fmt.Sprintf("lineup-app.%d", f.requests)})
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func fixtureStarters(from, to int) []string {
	out := []string{}
	for i := from; i < to; i++ {
		out = append(out, legalityID(1000, i))
	}
	return out
}

func TestNativeLineupSetThroughTheGateReadingAndRestart(t *testing.T) {
	f := lineupAppTest(t)
	legal := fixtureStarters(0, 21)
	f.execute(t, commissioner.RawCommand{Command: commissioner.RulesSet,
		Args: commissioner.RawArgs{Field: "calendar.boundary", Value: json.RawMessage(`"start_of_day"`)}})
	// The Legacy preset keeps lineups open all year; this league removes that window to shut them.
	f.execute(t, commissioner.RawCommand{Command: commissioner.WindowRemove,
		Args: commissioner.RawArgs{Window: &commissioner.RawWindowArgs{ID: "lineups"}}})
	version := f.version(t)
	_, err := f.app.TargetNativeLineupSet(f.setBody(t, "franchise.02", 1, legal))
	if err == nil || err.Error() != "No league window opens this action now." || f.version(t) != version {
		t.Fatalf("a lineup went through a shut window, or the sentence is wrong: %v", err)
	}
	generic, err := json.Marshal(commissioner.RawCommand{League: legalityLeague, Command: commissioner.LineupSet,
		Visibility: "private", Effective: "now", Request: "generic-door", Expected: f.version(t),
		Args: commissioner.RawArgs{Lineup: &commissioner.RawLineupChange{Season: 2026, Franchise: "franchise.02",
			Week: 1, Starters: legal}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.app.TargetCommissionerExecute(string(generic)); err == nil || f.version(t) != version {
		t.Fatalf("a lineup went through the generic commissioner door, past the calendar: %v", err)
	}
	f.openLineupWindow(t)
	short := fixtureStarters(0, 20)
	if _, err := f.app.TargetNativeLineupSet(f.setBody(t, "franchise.02", 1, short)); err == nil ||
		!strings.HasPrefix(err.Error(), "Invalid lineup: ") || !strings.Contains(err.Error(), "Total: 20 starting, needs exactly 21") {
		t.Fatalf("an illegal lineup was set: %v", err)
	}
	receipt, err := f.app.TargetNativeLineupSet(f.setBody(t, "franchise.02", 1, legal))
	if err != nil || receipt.Revision != f.version(t) {
		t.Fatalf("a legal lineup was not recorded: %+v %v", receipt, err)
	}
	for range 2 {
		reading, err := f.app.TargetNativeLineup(legalityLeague, "franchise.02", 1, "")
		if err != nil || !reading.Set || len(reading.Starters) != 21 || reading.Verdict.Words != "Valid lineup" ||
			reading.Starters[0].Words != "Locks at kickoff, Oct 18, 2026 at 10:00 AM PDT" ||
			reading.Starters[0].LocksAt == nil || reading.Zone != "America/Los_Angeles" {
			t.Fatalf("the lineup is not read back with its locks in the owner's time: %+v %v", reading, err)
		}
		f.reopen(t)
		f.wire(t)
	}
	f.at = f.kickoff
	swapped := append([]string{}, legal...)
	swapped[3] = legalityID(1000, 21) // a wide receiver for a wide receiver: legal, but the first is locked
	_, err = f.app.TargetNativeLineupSet(f.setBody(t, "franchise.02", 1, swapped))
	want := "Player, 1021 locked at kickoff (Oct 18, 2026 at 10:00 AM PDT); " +
		"Player, 1003 locked at kickoff (Oct 18, 2026 at 10:00 AM PDT); a lineup can't add or drop a player once that player's game has started"
	if err == nil || err.Error() != want {
		t.Fatalf("a locked player moved, or the refusal is wrong:\n got %v\nwant %s", err, want)
	}
	if _, err := f.app.TargetNativeLineupSet(f.setBody(t, "franchise.01", 1, legal)); err == nil ||
		err.Error() != "You may set a lineup only for your own franchise" {
		t.Fatalf("a lineup was set for a franchise the owner does not hold: %v", err)
	}
}

func TestNativeLineupCorrectionPreviewsAndRecordsThroughTheCommand(t *testing.T) {
	f := lineupAppTest(t)
	f.at = f.kickoff.Add(time.Hour)
	wrs := []string{}
	for i := 0; i < 21; i++ {
		wrs = append(wrs, legalityID(4000, i))
	}
	correction := commissioner.RawCommand{League: legalityLeague, Command: commissioner.LineupCorrect, Visibility: "league",
		Args: commissioner.RawArgs{Lineup: &commissioner.RawLineupChange{Season: 2026, Franchise: "franchise.01", Week: 1,
			Starters: wrs}}, Reason: "The owner's lineup was lost in the outage", Effective: "now", Request: "correct-1",
		Expected: f.version(t)}
	body, err := json.Marshal(correction)
	if err != nil {
		t.Fatal(err)
	}
	version := f.version(t)
	preview, err := f.app.TargetNativeScorePreview(string(body))
	wantWords := "We can't tell yet whether this changes Held team 1 v Held team 32: " +
		"Held team 1's score is not complete and Held team 32 has no lineup set."
	if err != nil || preview.Flip.Words != wantWords || len(preview.Warnings) < 1 || preview.Warnings[0] != wantWords ||
		preview.Lineup == nil || f.version(t) != version {
		t.Fatalf("the preview must name the one game it cannot call, and record nothing: %+v %v", preview, err)
	}
	receipt := executeCommissionerTest(t, f.app, correction)
	reading, err := f.app.TargetNativeLineup(legalityLeague, "franchise.01", 1, "")
	if err != nil || !reading.Set || !reading.Corrected || reading.Reason != correction.Reason {
		t.Fatalf("the correction is not read back: %+v %v", reading, err)
	}
	results, err := f.app.TargetNativeResults(legalityLeague, 2026, 1)
	if err != nil {
		t.Fatal(err)
	}
	if score := results.Scores[0]; score.Franchise != "franchise.01" || len(score.Starters) != 21 || score.Points != nil {
		t.Fatalf("results must read the recorded lineup, still incomplete without NFL statistics: %+v", score)
	}
	f.reopen(t)
	f.wire(t)
	f.execute(t, commissioner.RawCommand{Command: commissioner.Revert, Visibility: "league",
		Args: commissioner.RawArgs{Entry: receipt.Entry}})
	reading, err = f.app.TargetNativeLineup(legalityLeague, "franchise.01", 1, "")
	if err != nil || reading.Set {
		t.Fatalf("compensation did not remove the correction: %+v %v", reading, err)
	}
}

// TestLineupSetIsRegisteredWithTheGate holds lineup.set to the registry the gate guard test walks: an
// action left out of it would escape that guard.
func TestLineupSetIsRegisteredWithTheGate(t *testing.T) {
	for _, action := range nativeOwnerActions() {
		if action.Name == "lineup.set" && action.Opens == "lineup" && action.Run != nil {
			return
		}
	}
	t.Fatal("lineup.set is not in nativeOwnerActions, so the gate guard test would never run it")
}
