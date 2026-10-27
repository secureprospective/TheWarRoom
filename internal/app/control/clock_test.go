package control

import (
	"reflect"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
)

func TestWeekSnapshotReadings(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	kickoff := now.Add(time.Hour)
	games := []leagueweek.Game{{Kickoff: kickoff}, {Kickoff: now.Add(-time.Hour), Final: true}, {Kickoff: now.Add(-time.Hour)}}
	for _, tc := range []struct {
		name          string
		in            WeekSnapshot
		note, gameday string
		state         string
		hasLock       bool
	}{
		{name: "empty", note: "NFL schedule not fetched yet", gameday: "Gameday unavailable", state: domain.FreshLive},
		{name: "failed", in: WeekSnapshot{RefreshError: "offline"}, note: "NFL schedule refresh failed: offline", gameday: "Gameday unavailable", state: domain.FreshStale},
		{name: "held", in: WeekSnapshot{Week: leagueweek.Week{Number: 5, Games: games}, FetchedAt: now}, note: "NFL schedule: week 5, fetched 2026-10-08T12:00:00Z", gameday: "Week 5 · 1 NFL game in progress · 1 final", state: domain.FreshLive, hasLock: true},
		{name: "stale", in: WeekSnapshot{Week: leagueweek.Week{Number: 5, Games: games}, FetchedAt: now, RefreshError: "offline"}, note: "NFL schedule refresh failed: offline; showing week 5 fetched 2026-10-08T12:00:00Z", gameday: "Gameday unavailable", state: domain.FreshStale, hasLock: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			lock, fresh := WeekClock(tc.in, now)
			if (lock != nil) != tc.hasLock || fresh.Note != tc.note || fresh.State != tc.state || Gameday(tc.in, now) != tc.gameday {
				t.Fatalf("lock=%+v freshness=%+v gameday=%s", lock, fresh, Gameday(tc.in, now))
			}
			if lock != nil && (!lock.At.Equal(kickoff) || lock.Week != 5) {
				t.Fatal(lock)
			}
		})
	}
	lock, _ := WeekClock(WeekSnapshot{Week: leagueweek.Week{Number: 5, Games: games}}, kickoff.Add(time.Hour))
	if lock == nil || !lock.At.Equal(kickoff) {
		t.Fatal("last lock fallback lost", lock)
	}
}

func TestWindowSnapshotReadsOnlyMatchingValidSchedules(t *testing.T) {
	t.Parallel()
	raw := nflschedule.RawWeek{Week: "9", Matchups: []nflschedule.RawMatchup{{Teams: [2]string{"BUF", "NYJ"}, Kickoff: "1791460800"}}}
	bad := raw
	bad.Week = "13"
	in := WindowSnapshot{LastRegular: 13, Schedules: map[int][]nflschedule.RawWeek{9: {raw}, 13: {raw}, 14: {bad}}}
	got := WindowInputs(in, time.Unix(0, 0))
	if got.LastRegular != 13 || len(got.FirstKickoff) != 1 || got.FirstKickoff[9].Unix() != 1791460800 {
		t.Fatal(got)
	}
	if !reflect.DeepEqual(in.Schedules[9][0], raw) {
		t.Fatal("service changed caller input")
	}
}

func TestCaptureWindowsKeepsOnlyRuleWeeksAndDetachesMatchups(t *testing.T) {
	t.Parallel()
	schedules := map[int]report.HeldFeed[[]nflschedule.RawWeek]{1: {Value: []nflschedule.RawWeek{{Week: "1"}}}, 14: {Value: []nflschedule.RawWeek{{Week: "14", Matchups: []nflschedule.RawMatchup{{Kickoff: "1800000000"}}}}}}
	got := CaptureWindows(13, schedules)
	if got.LastRegular != 13 || len(got.Schedules[14]) != 1 {
		t.Fatal(got)
	}
	if _, ok := got.Schedules[1]; ok {
		t.Fatal("unconfigured week copied")
	}
	if rows, ok := got.Schedules[0]; !ok || rows != nil {
		t.Fatal("missing rule week must stay represented as nil")
	}
	got.Schedules[14][0].Week = "changed"
	got.Schedules[14][0].Matchups[0].Kickoff = "changed"
	if schedules[14].Value[0].Week != "14" || schedules[14].Value[0].Matchups[0].Kickoff != "1800000000" {
		t.Fatal("window capture aliases published schedules")
	}
}

func TestLastRegularWeekAndClockFallback(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		raw  string
		want int
	}{{"", 0}, {"bad", 0}, {"1", 0}, {"2", 2}, {"18", 18}, {"19", 0}, {"13", 13}} {
		if got := LastRegularWeek(tc.raw); got != tc.want {
			t.Errorf("%q: %d", tc.raw, got)
		}
	}
	at := time.Unix(1800000000, 0)
	reads := 0
	if got := TimeNow(func() time.Time { reads++; return at }); got != at || reads != 1 {
		t.Fatal(got, reads)
	}
	before := time.Now()
	wall := TimeNow(nil)
	after := time.Now()
	if wall.Before(before) || wall.After(after) {
		t.Fatal("wall clock fallback outside observation span")
	}
}
