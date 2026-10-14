package commissioner

import (
	"reflect"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func TestScheduleReplayUsesRecordedWeeks(t *testing.T) {
	// A recorded result can outlive both the generation rules and the generator implementation.
	frozen := []ScheduleWeek{{Week: 1, Games: []ScheduleGame{
		{ID: "held.game", Home: "b", Away: "a"},
	}}}
	records := []Record{
		{Command: RawCommand{Command: ScheduleGenerate, Args: RawArgs{Schedule: &RawScheduleArgs{
			Weeks: 1, Pattern: leaguerules.DivisionWeighted,
		}}}, GeneratedSchedule: frozen, Receipt: Receipt{Entry: "generate"}},
		{Command: RawCommand{Command: SchedulePublish}, Receipt: Receipt{Entry: "publish"}},
	}
	reading, err := ScheduleReading("native.replay", records)
	if err != nil || !reflect.DeepEqual(reading.Weeks, frozen) {
		t.Fatal("replay did not use the recorded generation", reading, err)
	}
	reading.Weeks[0].Games[0].Home = "changed"
	if frozen[0].Games[0].Home != "b" {
		t.Fatal("owner reading mutated the recorded generation")
	}
	records[0].GeneratedSchedule = nil
	if _, err := ScheduleReading("native.replay", records); err == nil {
		t.Fatal("missing frozen result was silently recomputed")
	}
}

func TestSchedulePatchKeepsOtherGamesAndRejectsDoubleBooking(t *testing.T) {
	week := ScheduleWeek{Week: 1, Games: []ScheduleGame{
		{ID: "first", Home: "a", Away: "b"},
		{ID: "second", Home: "c", Away: "d"},
	}}
	records := []Record{{Command: RawCommand{Args: RawArgs{Franchises: []string{"a", "b", "c", "d"}}}}}
	patch := RawScheduleArgs{Week: 1, Game: "first", Home: "b", Away: "a"}
	if err := checkSchedulePatch(patch, []ScheduleWeek{week}, records); err != nil {
		t.Fatal(err)
	}
	changed := patchScheduleGame([]ScheduleWeek{week}, patch)[0]
	if changed.Games[0].ID != "first" || changed.Games[0].Home != "b" ||
		changed.Games[1] != week.Games[1] || week.Games[0].Home != "a" {
		t.Fatal("patch changed an identity, another game, or its input", changed, week)
	}
	patch.Home = "c"
	if err := checkSchedulePatch(patch, []ScheduleWeek{week}, records); err == nil {
		t.Fatal("accepted a team playing twice and another team left idle")
	}
	patch.Game = "missing"
	if err := checkSchedulePatch(patch, []ScheduleWeek{week}, records); err == nil {
		t.Fatal("accepted an unknown game")
	}
}

func TestScheduleCommandArgumentShape(t *testing.T) {
	for _, test := range []struct {
		command string
		args    RawScheduleArgs
		valid   bool
	}{
		{ScheduleGenerate, RawScheduleArgs{Weeks: 3, Pattern: leaguerules.DivisionWeighted}, true},
		{ScheduleGenerate, RawScheduleArgs{Weeks: 3}, false},
		{ScheduleGenerate, RawScheduleArgs{Weeks: 0, Pattern: leaguerules.DivisionWeighted}, false},
		{ScheduleGenerate, RawScheduleArgs{Weeks: 53, Pattern: leaguerules.DivisionWeighted}, false},
		{ScheduleGenerate, RawScheduleArgs{Weeks: 3, Pattern: "unknown"}, false},
		{ScheduleGenerate, RawScheduleArgs{Weeks: 3, Pattern: leaguerules.DivisionWeighted, Week: 1}, false},
		{ScheduleEdit, RawScheduleArgs{Week: 1, Game: "first", Home: "a", Away: "b"}, true},
		{ScheduleEdit, RawScheduleArgs{Week: 1, Home: "a", Away: "b"}, false},
		{SchedulePublish, RawScheduleArgs{}, true},
		{SchedulePublish, RawScheduleArgs{Weeks: 3}, false},
	} {
		err := (RawArgs{Schedule: &test.args}).Validate(test.command)
		if (err == nil) != test.valid {
			t.Fatal(test.command, test.args, err)
		}
	}
}

func TestScheduleFrozenPayloadValidation(t *testing.T) {
	record := Record{Command: RawCommand{Command: ScheduleGenerate,
		Args: RawArgs{Schedule: &RawScheduleArgs{Weeks: 1, Pattern: leaguerules.DivisionWeighted}}},
		GeneratedSchedule: []ScheduleWeek{{Week: 1, Games: []ScheduleGame{
			{ID: "first", Home: "a", Away: "b"}, {ID: "second", Home: "c", Away: "d"},
		}}},
	}
	if err := record.validateGeneratedSchedule(); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*Record){
		func(r *Record) { r.GeneratedSchedule = nil },
		func(r *Record) { r.Command.Command = SchedulePublish },
		func(r *Record) { r.GeneratedSchedule[0].Week = 2 },
		func(r *Record) { r.GeneratedSchedule[0].Games = nil },
		func(r *Record) { r.GeneratedSchedule[0].Games[0].ID = "" },
		func(r *Record) { r.GeneratedSchedule[0].Games[1].ID = "first" },
		func(r *Record) { r.GeneratedSchedule[0].Games[1].Home = "a" },
		func(r *Record) { r.GeneratedSchedule[0].Bye = "a" },
	} {
		invalid := record
		invalid.GeneratedSchedule = cloneSchedule(record.GeneratedSchedule)
		change(&invalid)
		if err := invalid.validateGeneratedSchedule(); err == nil {
			t.Fatal("accepted malformed frozen generation", invalid)
		}
	}
}
