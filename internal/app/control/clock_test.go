package control

import (
	"reflect"
	"testing"
	"time"

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
