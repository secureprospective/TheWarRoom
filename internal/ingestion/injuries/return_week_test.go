package injuries

import (
	"strconv"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

func TestReturnWeek(t *testing.T) {
	t.Parallel()
	weeks := returnWeekSchedule(t)
	tests := []struct {
		name   string
		exp    string
		season int
		week   int
		ok     bool
		reason string
	}{
		{name: "valid mid-season", exp: "Oct 15, 2026", season: 2026, week: 6, ok: true},
		{name: "date in bye gap", exp: "Oct 11, 2026", season: 2026, week: 6, ok: true},
		{name: "before week one", exp: "Aug 28, 2026", season: 2026, week: 1, ok: true},
		{name: "after season end", exp: "Feb 15, 2027", reason: "after this season"},
		{name: "empty", exp: "", reason: "return date not given"},
		{name: "garbage", exp: "Week 6", reason: "return date not given"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			season, week, ok, reason := ReturnWeek(test.exp, weeks)
			if season != test.season || week != test.week || ok != test.ok || reason != test.reason {
				t.Fatalf("ReturnWeek(%q) = (%d, %d, %v, %q), want (%d, %d, %v, %q)",
					test.exp, season, week, ok, reason, test.season, test.week, test.ok, test.reason)
			}
		})
	}
}

func TestResolveReturnWeeksUsesOnlyHeldSchedule(t *testing.T) {
	id, err := playerid.New("0099")
	if err != nil {
		t.Fatal(err)
	}
	rows := []leaguefeed.Injury{{ID: id, Status: "Out", ExpReturn: "Oct 15, 2026"}}
	dates := []string{"2026-09-10", "2026-09-17", "2026-09-24", "2026-10-01", "2026-10-08", "2026-10-15"}
	weeks := make([]nflschedule.RawWeek, len(dates))
	for i, date := range dates {
		weeks[i] = returnWeekFixture(t, i+1, date)
	}
	resolved := ResolveReturnWeeks(rows, weeks)
	if len(resolved) != 1 || resolved[0].ReturnSeason != 2026 || resolved[0].ReturnWeek != 6 || resolved[0].ReturnReason != "" {
		t.Fatalf("complete schedule resolution = %+v", resolved)
	}
	if rows[0].ReturnWeek != 0 || rows[0].ReturnSeason != 0 {
		t.Fatalf("resolution changed source facts: %+v", rows[0])
	}
	partial := append(append([]nflschedule.RawWeek(nil), weeks[:4]...), weeks[5:]...)
	resolved = ResolveReturnWeeks(rows, partial)
	if resolved[0].ReturnWeek != 0 || resolved[0].ReturnSeason != 0 || resolved[0].ReturnReason != "schedule not loaded" {
		t.Fatalf("partial schedule resolution = %+v", resolved[0])
	}
}

func TestWeeksHeldThrough(t *testing.T) {
	for _, test := range []struct {
		name  string
		weeks []nflschedule.RawWeek
		last  int
		want  bool
	}{
		{name: "all earlier weeks held", weeks: []nflschedule.RawWeek{{Week: "1"}, {Week: "2"}, {Week: "3"}}, last: 3, want: true},
		{name: "missing earlier week", weeks: []nflschedule.RawWeek{{Week: "1"}, {Week: "3"}}, last: 3},
		{name: "invalid week ignored", weeks: []nflschedule.RawWeek{{Week: "1"}, {Week: "2"}, {Week: "week 3"}}, last: 3},
		{name: "week zero", weeks: []nflschedule.RawWeek{{Week: "1"}}, last: 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := WeeksHeldThrough(test.weeks, test.last); got != test.want {
				t.Fatalf("WeeksHeldThrough(%v, %d) = %t, want %t", test.weeks, test.last, got, test.want)
			}
		})
	}
}

func returnWeekSchedule(t *testing.T) []nflschedule.RawWeek {
	t.Helper()
	starts := []string{
		"2026-09-10", "2026-09-17", "2026-09-24", "2026-10-01",
		"2026-10-15", "2026-10-22", "2026-10-29", "2026-11-05",
		"2026-11-12", "2026-11-19", "2026-11-26", "2026-12-03",
		"2026-12-10", "2026-12-17", "2026-12-24", "2026-12-31", "2027-01-07",
	}
	weeks := make([]nflschedule.RawWeek, 0, len(starts))
	for i, start := range starts {
		number := i + 1
		if number >= 5 {
			number++
		}
		weeks = append(weeks, returnWeekFixture(t, number, start))
	}
	return weeks
}

func returnWeekFixture(t *testing.T, week int, start string) nflschedule.RawWeek {
	t.Helper()
	date, err := time.Parse("2006-01-02", start)
	if err != nil {
		t.Fatal(err)
	}
	return nflschedule.RawWeek{
		Week: strconv.Itoa(week),
		Matchups: []nflschedule.RawMatchup{{
			Teams: [2]string{"AAA", "BBB"}, Kickoff: strconv.FormatInt(date.Unix(), 10),
		}},
	}
}
