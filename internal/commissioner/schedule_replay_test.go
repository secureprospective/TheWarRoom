package commissioner

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

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
	patch := RawScheduleArgs{Week: 1, Games: []ScheduleGame{{ID: "first", Home: "b", Away: "a"}}}
	if err := checkSchedulePatch(patch, []ScheduleWeek{week}, records); err != nil {
		t.Fatal(err)
	}
	changed := patchScheduleGames([]ScheduleWeek{week}, patch)[0]
	if changed.Games[0].ID != "first" || changed.Games[0].Home != "b" ||
		changed.Games[1] != week.Games[1] || week.Games[0].Home != "a" {
		t.Fatal("patch changed an identity, another game, or its input", changed, week)
	}
	patch.Games[0].Home = "c"
	if err := checkSchedulePatch(patch, []ScheduleWeek{week}, records); err == nil {
		t.Fatal("accepted a team playing twice and another team left idle")
	}
	patch.Games[0].ID = "missing"
	if err := checkSchedulePatch(patch, []ScheduleWeek{week}, records); err == nil {
		t.Fatal("accepted an unknown game")
	}
	// Re-pairing two games is one edit: one game at a time would pass through a team playing twice.
	swap := RawScheduleArgs{Week: 1, Games: []ScheduleGame{{ID: "first", Home: "a", Away: "c"},
		{ID: "second", Home: "b", Away: "d"}}}
	if err := checkSchedulePatch(swap, []ScheduleWeek{week}, records); err != nil {
		t.Fatal("refused re-pairing two games together", err)
	}
}

func TestScheduleCommandArgumentShape(t *testing.T) {
	for _, test := range []struct {
		command string
		args    RawScheduleArgs
		valid   bool
	}{
		{ScheduleGenerate, RawScheduleArgs{Weeks: 3, Pattern: leaguerules.DivisionWeighted}, true},
		{ScheduleGenerate, RawScheduleArgs{Weeks: 3}, true},
		{ScheduleGenerate, RawScheduleArgs{Weeks: 0, Pattern: leaguerules.DivisionWeighted}, false},
		{ScheduleGenerate, RawScheduleArgs{Weeks: 53, Pattern: leaguerules.DivisionWeighted}, false},
		{ScheduleGenerate, RawScheduleArgs{Weeks: 3, Pattern: "unknown"}, false},
		{ScheduleGenerate, RawScheduleArgs{Weeks: 3, Pattern: leaguerules.DivisionWeighted, Week: 1}, false},
		{ScheduleEdit, RawScheduleArgs{Week: 1, Games: []ScheduleGame{{ID: "first", Home: "a", Away: "b"}}}, true},
		{ScheduleEdit, RawScheduleArgs{Week: 1}, false},
		{ScheduleEdit, RawScheduleArgs{Week: 1, Games: []ScheduleGame{{ID: "first", Home: "a", Away: "b"},
			{ID: "first", Home: "c", Away: "d"}}}, false},
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

func TestScheduleGenerationUsesInForcePatternWhenOmitted(t *testing.T) {
	for _, count := range []int{2, 32} {
		create, err := (RawCommand{League: "native.test", Command: Create,
			Args:   RawArgs{Name: "Ours", Preset: PresetLegacy, FranchiseCount: count},
			Reason: "Start", Request: "create", Effective: "now", Visibility: "league"}).Prepared()
		if err != nil {
			t.Fatal(err)
		}
		alignment := make(map[string]string)
		for i, franchise := range create.Args.Franchises {
			alignment[franchise] = fmt.Sprint(i / 4)
		}
		body, err := json.Marshal(alignment)
		if err != nil {
			t.Fatal(err)
		}
		records := []Record{
			{Command: create, Receipt: Receipt{Entry: "create", RecordedAt: time.Unix(1, 0)}},
			{Command: RawCommand{League: "native.test", Command: RulesSet, Effective: "now",
				Args: RawArgs{Field: "league.alignment", Value: body}},
				Receipt: Receipt{Entry: "align", RecordedAt: time.Unix(2, 0)}},
		}
		raw := RawCommand{Command: ScheduleGenerate, Args: RawArgs{Schedule: &RawScheduleArgs{Weeks: 13}}}
		weeks, err := FreezeSchedule(raw, records)
		if count == 32 {
			if err != nil || len(weeks) != 13 {
				t.Fatal("omitted pattern did not use the in-force Legacy rule", weeks, err)
			}
		} else if err == nil || err.Error() != "Choose how often teams meet in this smaller league." {
			t.Fatal("unset rule did not demand a commissioner choice", weeks, err)
		}
	}
}
