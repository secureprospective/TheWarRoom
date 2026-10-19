package commissioner

import (
	"reflect"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
)

func TestNativeWeekTransitionsAndBye(t *testing.T) {
	at := time.Date(2026, 9, 10, 20, 0, 0, 0, time.UTC)
	first, second := at.Add(time.Hour), at.AddDate(0, 0, 7)
	reading := FranchiseSchedule{Games: []FranchiseGame{{Week: 1, Venue: "Bye"},
		{Week: 2, Venue: "Home", Opponent: "b"}}}
	weeks := []leagueweek.Week{
		{Number: 2, Games: []leagueweek.Game{{Kickoff: second, Final: true}}},
		{Number: 1, Games: []leagueweek.Game{{Kickoff: first, Final: true}}},
	}
	for _, test := range []struct {
		at      time.Time
		current int
		next    int
	}{
		{at, 1, 2}, {first, 2, 0}, {second, 0, 0},
	} {
		got, err := NativeWeekReading(reading, test.at, weeks)
		if err != nil || got.NextWeek != test.next {
			t.Fatal(got, err)
		}
		if test.current == 0 {
			if got.Current != nil || got.Message != "The regular-season schedule is finished." {
				t.Fatal(got)
			}
		} else if got.Current == nil || got.Current.Week != test.current {
			t.Fatal("future final flag advanced the week prematurely", got)
		}
	}
	weeks[1].Games[0].Final = false
	got, err := NativeWeekReading(reading, second, weeks)
	if err != nil || got.Current == nil || got.Current.Week != 1 || got.Current.Venue != "Bye" {
		t.Fatal("unfinished bye week skipped", got, err)
	}
}

func TestScheduleSeasonBoundaryKeepsEndedPublication(t *testing.T) {
	b := newLedger(t, PresetEmpty)
	b.set("calendar.rollover", rolloverAt, "now", seasonMoment(2))
	old := []ScheduleWeek{{Week: 1, Games: []ScheduleGame{{ID: "old", Home: "franchise.01", Away: "franchise.02"}}}}
	b.add(RawCommand{Command: ScheduleGenerate}, seasonMoment(3))
	b.records[len(b.records)-1].GeneratedSchedule = old
	b.add(RawCommand{Command: SchedulePublish}, seasonMoment(3))
	rollover := time.Date(2027, 3, 1, 5, 0, 0, 0, time.UTC)
	got, err := ScheduleSeasonReading("native.test", b.records, 2027, rollover)
	if err != nil || len(got.Weeks) != 0 {
		t.Fatal("old schedule carried into the next season", got, err)
	}
	b.add(RawCommand{Command: ScheduleGenerate}, rollover.Add(time.Hour))
	b.records[len(b.records)-1].GeneratedSchedule = []ScheduleWeek{{Week: 1,
		Games: []ScheduleGame{{ID: "new", Home: "franchise.01", Away: "franchise.03"}}}}
	b.add(RawCommand{Command: SchedulePublish}, rollover.Add(time.Hour))
	later := rollover.Add(2 * time.Hour)
	got, err = ScheduleSeasonReading("native.test", b.records, 2026, later)
	if err != nil || !reflect.DeepEqual(got.Weeks, old) {
		t.Fatal("new publication rewrote the ended season", got, err)
	}
	got, err = ScheduleSeasonReading("native.test", b.records, 2027, later)
	if err != nil || got.Weeks[0].Games[0].Away != "franchise.03" {
		t.Fatal("new season lost its publication", got, err)
	}
	if _, err := ScheduleSeasonReading("native.test", b.records, 2028, later); err == nil {
		t.Fatal("invented a future season")
	}
}

func TestFranchiseGameRejectsIdleAndReadsBothSides(t *testing.T) {
	week := ScheduleWeek{Week: 3, Games: []ScheduleGame{{ID: "g", Home: "a", Away: "b"}}}
	for _, test := range []struct{ team, opponent, venue string }{
		{"a", "b", "Home"}, {"b", "a", "Away"},
	} {
		got, err := franchiseScheduleGame(week, test.team)
		if err != nil || got.Game != "g" || got.Opponent != test.opponent || got.Venue != test.venue {
			t.Fatal(got, err)
		}
	}
	if _, err := franchiseScheduleGame(week, "c"); err == nil {
		t.Fatal("unaccounted-for idle franchise accepted")
	}
	week.Bye = "c"
	got, err := franchiseScheduleGame(week, "c")
	if err != nil || got.Venue != "Bye" || got.Opponent != "" {
		t.Fatal(got, err)
	}
}
