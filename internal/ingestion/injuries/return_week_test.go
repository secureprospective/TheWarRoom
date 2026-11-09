package injuries

import (
	"strconv"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
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
