package composition

import (
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	archiveapp "github.com/secureprospective/TheWarRoom/internal/app/archive"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livescoring"
	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
)

func pulseLive(t *testing.T) leaguefeed.Lineups {
	t.Helper()
	raw, err := livescoring.Parse([]byte(seasonFixture(t, "livescoring", "liveScoring-w5.json")))
	if err != nil {
		t.Fatal(err)
	}
	live, err := livescoring.ToLineups(raw)
	if err != nil {
		t.Fatal(err)
	}
	return live
}

func TestPulseRaceLiveSchedule(t *testing.T) {
	t.Parallel()
	in := realRace(t, "2026")
	live := pulseLive(t)
	in.team = live.Matchups[0].Home
	for i := range live.Franchises {
		switch live.Franchises[i].Franchise {
		case live.Matchups[0].Home:
			live.Franchises[i].Score = 246.15
		case live.Matchups[0].Away:
			live.Franchises[i].Score = 221.25
		}
	}
	in.live = held(live)
	v, err := newRaceView(in)
	if err != nil {
		t.Fatal(err)
	}
	r := v.scheduleByTeam()
	for _, row := range r.Rows {
		if row.Cells[0] == "Week 5" {
			if row.Cells[3] != "In progress · 246.15 to 221.25" {
				t.Fatal(row.Cells)
			}
		}
	}
	// Before the live week's first kickoff its 0.00 scores are not a game in progress.
	in.liveUnplayed = true
	if v, err = newRaceView(in); err != nil {
		t.Fatal(err)
	}
	if got := v.scheduleByTeam().Rows[4].Cells; got[0] != "Week 5" || got[3] != "Not played yet" {
		t.Fatalf("live week before kickoff: %v", got)
	}
}

func TestPulsePluralAndProjectedBracket(t *testing.T) {
	t.Parallel()
	if countText(1, "NFL game") != "1 NFL game" || pluralNoun(1, "game") != "game" ||
		pluralNoun(1.5, "game") != "games" {
		t.Fatal("incorrect singular/plural")
	}
	a := targetTestApp(t)
	at := raceHeldAt
	a.core.ClockNow = func() time.Time { return at }
	a.core.Week = leagueweek.Week{Number: 5, Games: []leagueweek.Game{{Kickoff: at.Add(-time.Hour)}}}
	status, err := a.controlAPI().TargetShellStatus()
	if err != nil || !strings.Contains(status.Gameday, "1 NFL game in progress") {
		t.Fatal("status strip plural", status, err)
	}
	v, err := newRaceView(realRace(t, "2026"))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, row := range v.bubble().Rows {
		if row.Cells[3] == "1 game back" {
			found = true
		}
		if strings.Contains(row.Cells[3], "1 games") {
			t.Fatal(row.Cells)
		}
	}
	if !found {
		t.Fatal("fixture did not exercise one game back")
	}
	fields, _ := v.fields()
	for _, f := range fields {
		for _, seed := range f.field.Seeds {
			got := v.projectedBracketSide(f.bracket, leaguefeed.BracketSide{Seed: seed.Number}, nil)
			want := fmt.Sprintf("Seed %d · %s (projected)", seed.Number, v.name(seed.Franchise))
			if got != want {
				t.Fatalf("%q != %q", got, want)
			}
		}
	}
}

func TestPulseSeriesUsesCurrentOpponent(t *testing.T) {
	t.Parallel()
	a := targetTestApp(t)
	a.core.Week.Number = 5
	a.core.SeasonLineups = held(pulseLive(t))
	match := a.core.SeasonLineups.Value.Matchups[0]
	board, err := archiveapp.New(archiveHost{a.core}).TargetArchive(0, match.Home, "")
	if err != nil || board.Other != match.Away {
		t.Fatal("series did not open on current opponent", board.Other, err)
	}
	if got := a.archiveOpponent(match.Home); got != match.Away {
		t.Fatal("home default", got, match.Away)
	}
	if got := a.archiveOpponent(match.Away); got != match.Home {
		t.Fatal("away default", got, match.Home)
	}
	a.raceHeld.PublishWeeks(raceFeed(realRace(t, "2026").held.weeks))
	a.core.SeasonLineups.Value.Week = 4
	if got := a.archiveOpponent(match.Home); got != match.Away {
		t.Fatal("weekly schedule default", got, match.Away)
	}
}

func TestPulseClockKeepsWeekAfterLastKickoff(t *testing.T) {
	t.Parallel()
	a, _, _ := weekTestApp(t, "17")
	at := time.Date(2026, 10, 13, 1, 0, 0, 0, time.UTC)
	a.core.ClockNow = func() time.Time { return at }
	a.core.Week = leagueweek.Week{Number: 5, Games: []leagueweek.Game{{
		Teams: [2]string{"BUF", "LAR"}, Kickoff: at.Add(-time.Hour),
	}}}
	a.core.WeekFetchedAt = at.Add(-time.Minute)
	reading, err := a.platformAPI().TargetClock()
	if err != nil || reading.Value.Week == nil || *reading.Value.Week != 5 {
		t.Fatal("last game in progress lost week", reading, err)
	}
	if len(reading.Value.Deadlines) == 0 || reading.Value.Deadlines[0].At == nil ||
		!reading.Value.Deadlines[0].At.Before(at) {
		t.Fatal("lineup lock must be passed", reading.Value.Deadlines)
	}
	a.core.Week.Games[0].Final = true
	if r, err := a.platformAPI().TargetClock(); err != nil || r.Value.Week == nil || *r.Value.Week != 5 {
		t.Fatal("do not lose week until next schedule is held", r, err)
	}
	a.core.Week = leagueweek.Week{Number: 6, Games: []leagueweek.Game{{Kickoff: at.Add(48 * time.Hour)}}}
	if r, err := a.platformAPI().TargetClock(); err != nil || r.Value.Week == nil || *r.Value.Week != 6 {
		t.Fatal("next held schedule not adopted", r, err)
	}
}

func TestPulseStartupDatesWindowsWithoutRefresh(t *testing.T) {
	t.Parallel()
	a, transport, _ := weekTestApp(t, "17")
	a.core.ClockNow = func() time.Time { return time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC) }
	body := weekFixture(t, "nflSchedule-w5.json")
	for _, week := range []int{9, 13, 14} {
		scheduleBody := strings.Replace(body, `"week":"5"`, fmt.Sprintf(`"week":"%d"`, week), 1)
		if scheduleBody == body {
			t.Fatal("fixture week not replaced")
		}
		recordSeasonArchiveForm(t, a, "nflSchedule", strconv.Itoa(week), scheduleBody, raceHeldAt, true)
	}
	pools, err := db.Open(a.core.Ctx, filepath.Join(t.TempDir(), "startup-whatif.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := pools.Close(); err != nil {
			t.Error(err)
		}
	})
	a.core.WhatifPools = pools
	a.attachSeat(a.core.History)
	if _, err := a.initStoreFloor(a.core.Ctx, a.core.History); err != nil {
		t.Fatal(err)
	}
	reading, err := a.platformAPI().TargetClock()
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []leagueclock.WindowKind{leagueclock.TradeDeadline, leagueclock.PlayoffBidsOnly,
		leagueclock.MultiYearBidsClose} {
		if !slices.ContainsFunc(reading.Value.Windows, func(w leagueclock.Window) bool {
			return w.Kind == kind && w.At != nil
		}) {
			t.Fatal("undated on first paint", kind, reading.Value.Windows)
		}
	}
	if len(transport.requests) != 0 {
		t.Fatal("startup seeding contacted MFL", transport.requests)
	}
}

// Before the week's first kickoff a held live matchup is not in progress, and a starter whose NFL
// team has no game that week reads bye (or no NFL team), never 0.00.
