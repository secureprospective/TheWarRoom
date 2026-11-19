package leagueservice

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
)

type weekTestHost struct {
	viewTestHost
	weeks []leagueweek.Week
}

func (h weekTestHost) NativeScheduleDates(context.Context, int, time.Time) ([]leagueweek.Week, error) {
	return h.weeks, nil
}

func TestNativeWeekExplainsHowCommissionerPublishesMissingSchedule(t *testing.T) {
	f := newDueFixture(t)
	base := viewService(t, f, "creator")
	host := weekTestHost{viewTestHost: base.host.(viewTestHost)}
	service := NewHosted(host, base.admission)
	raw := RawNativeWeekScope{RawNativeScheduleScope: RawNativeScheduleScope{
		League: dueLeague, Season: 2026, LocalZone: "UTC",
	}, At: f.clock.Now()}

	got, err := service.TargetNativeWeek(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.Current != nil || !strings.Contains(got.Message, "commissioner") ||
		!strings.Contains(got.Message, "Generate schedule") ||
		!strings.Contains(got.Message, "Publish schedule") ||
		!strings.Contains(got.Message, "Control Room > Season") {
		t.Fatalf("missing schedule should say which commissioner action is needed: %+v", got)
	}
}

func TestNativeWeekReadsCurrentBeforeMissingFutureDates(t *testing.T) {
	f := newDueFixture(t)
	at := time.Date(2026, time.November, 18, 12, 0, 0, 0, time.UTC)
	f.clock.instant.Store(at.UnixNano())
	fixtures := make([]commissioner.ScheduleWeek, 17)
	for number := range fixtures {
		week := number + 1
		fixtures[number] = commissioner.ScheduleWeek{Week: week,
			Games: []commissioner.ScheduleGame{{ID: fmt.Sprintf("game.%d", week), Home: "franchise.01", Away: "franchise.02"}}}
	}
	f.execute(t, "creator", commissioner.ScheduleGenerate,
		commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{Weeks: len(fixtures), Fixtures: fixtures}})
	f.execute(t, "creator", commissioner.SchedulePublish, commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{}})
	kickoff := time.Date(2026, time.November, 19, 18, 0, 0, 0, time.UTC)
	weeks := make([]leagueweek.Week, 12)
	for number := 1; number <= len(weeks); number++ {
		weekKickoff := at.Add(-time.Duration(20-number) * 24 * time.Hour)
		final := true
		if number == 12 {
			weekKickoff, final = kickoff, false
		}
		weeks[number-1] = leagueweek.Week{Number: number,
			Games: []leagueweek.Game{{Kickoff: weekKickoff, Final: final}}}
	}
	base := viewService(t, f, "creator")
	read := func(held []leagueweek.Week) commissioner.NativeWeek {
		host := weekTestHost{viewTestHost: base.host.(viewTestHost), weeks: held}
		service := NewHosted(host, base.admission)
		raw := RawNativeWeekScope{RawNativeScheduleScope: RawNativeScheduleScope{
			League: dueLeague, Season: 2026, LocalZone: "UTC",
		}, At: at}
		got, err := service.TargetNativeWeek(raw)
		if err != nil {
			t.Fatal(err)
		}
		return got
	}

	got := read(weeks)
	if got.Current == nil || got.Current.Week != 12 || got.NextWeek != 13 {
		t.Fatalf("current week should be selected before missing future dates: %+v", got)
	}
	unknown := read(weeks[:11])
	if unknown.Current != nil || unknown.Message != "The current NFL week cannot be identified from this server's held schedule data. Ask the server operator to update the NFL schedule; changing the league matchup schedule will not fix Pulse." {
		t.Fatalf("a missing current week must remain unknown rather than guessed: %+v", unknown)
	}
}

func TestNativeWeekLeagueOnlyAndOwner(t *testing.T) {
	f := newDueFixture(t)
	fixtures := []commissioner.ScheduleWeek{}
	for n := 1; n <= 11; n++ {
		fixtures = append(fixtures, commissioner.ScheduleWeek{Week: n, Games: []commissioner.ScheduleGame{{ID: fmt.Sprintf("game.%d", n), Home: "franchise.01", Away: "franchise.02"}}})
	}
	f.execute(t, "creator", commissioner.ScheduleGenerate, commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{Weeks: 11, Fixtures: fixtures}})
	f.execute(t, "creator", commissioner.SchedulePublish, commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{}})
	weeks := []leagueweek.Week{}
	for n := 1; n <= 11; n++ {
		weeks = append(weeks, leagueweek.Week{Number: n, Games: []leagueweek.Game{{Kickoff: f.clock.Now().Add(-time.Hour), Final: n < 10}}})
	}
	for _, tc := range []struct {
		name, principal, franchise string
		admitted                   bool
	}{
		{"teamless commissioner", "creator", "", true},
		{"owner league view", "owner.one", "", true},
		{"no credential", "outsider", "", false},
		{"owner unchanged", "owner.one", "franchise.01", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base := viewService(t, f, tc.principal)
			host := weekTestHost{viewTestHost: base.host.(viewTestHost), weeks: weeks}
			gate := base.admission
			if !tc.admitted {
				gate = nil
			}
			service := NewHosted(host, gate)
			raw := RawNativeWeekScope{RawNativeScheduleScope: RawNativeScheduleScope{League: dueLeague, Franchise: tc.franchise, Season: 2026, LocalZone: "UTC"}, At: f.clock.Now()}
			got, err := service.TargetNativeWeek(raw)
			if !tc.admitted {
				if err == nil || !reflect.DeepEqual(got, commissioner.NativeWeek{}) {
					t.Fatalf("unadmitted reading: %+v %v", got, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.Current == nil || got.Current.Week != 10 || got.NextWeek != 11 {
				t.Fatalf("wrong week: %+v", got)
			}
			if tc.franchise == "" {
				for _, game := range append(got.Schedule.Games, *got.Current) {
					if game.Game != "" || game.Opponent != "" || game.Venue != "" || game.Kickoff == nil {
						t.Fatalf("not league-only timing: %+v", game)
					}
				}
				return
			}
			assertOwnerWeekBytes(t, service, raw, weeks, got)
		})
	}
}

func assertOwnerWeekBytes(t *testing.T, service *Service, raw RawNativeWeekScope,
	weeks []leagueweek.Week, got commissioner.NativeWeek) {
	t.Helper()
	records, err := service.host.CommissionerRecords(t.Context(), dueLeague)
	if err != nil {
		t.Fatal(err)
	}
	kickoff := weeks[0].Games[0].Kickoff
	schedule := commissioner.FranchiseSchedule{League: dueLeague, Franchise: raw.Franchise, Season: 2026,
		Revision: commissioner.ViewRevision(service.viewContext(t.Context()), records, int64(len(records))),
		Games:    []commissioner.FranchiseGame{}}
	for n := 1; n <= 11; n++ {
		schedule.Games = append(schedule.Games, commissioner.FranchiseGame{Week: n,
			Game: fmt.Sprintf("game.%d", n), Opponent: "franchise.02", Venue: "Home", Kickoff: &kickoff})
	}
	current := schedule.Games[9]
	want := commissioner.NativeWeek{Schedule: schedule, Current: &current, NextWeek: 11}
	gotJSON, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	wantJSON, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotJSON) != string(wantJSON) {
		t.Fatalf("owner bytes changed: %s != %s", gotJSON, wantJSON)
	}
}
