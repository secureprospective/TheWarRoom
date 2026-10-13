package main

import (
	"fmt"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

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

func TestPulsePlayerKickoffAndStableSlots(t *testing.T) {
	in := realNow(t, 5)
	v, err := newNowView(in)
	if err != nil {
		t.Fatal(err)
	}
	v.live = true
	me, them, _, ok := v.myGame()
	if !ok {
		t.Fatal("fixture has no matchup")
	}
	me.Scored = false
	var checked bool
	for _, p := range me.Players {
		g, found := v.games[v.team(p)]
		if !found {
			continue
		}
		p.Points = 0
		v.now.at = g.kickoff.Add(-time.Second)
		want := "yet to play · " + g.kickoff.In(v.in.zone).Format("Mon 3:04 PM")
		if got := v.playerPointsText(me, p); got != want {
			t.Fatalf("before kickoff: %q, want %q", got, want)
		}
		me.Players = []leaguefeed.WeekPlayer{p}
		p.Starter = true
		me.Players[0] = p
		if rows := v.classicRows(me, them); !strings.Contains(fmt.Sprint(rows), want) {
			t.Fatal("matchup bypassed the kickoff rule", rows)
		}
		if row := v.boxRow(me, p, ""); row.Cells[3] != want {
			t.Fatal("box score bypassed the kickoff rule", row)
		}
		v.now.at = g.kickoff
		if got := v.playerPointsText(me, p); got != "0.00" {
			t.Fatal("at kickoff", got)
		}
		checked = true
		break
	}
	if !checked {
		t.Fatal("no player with a held NFL kickoff")
	}
}

func TestPulseSlotsIgnoreFeedShuffle(t *testing.T) {
	v, err := newNowView(realNow(t, 5))
	if err != nil {
		t.Fatal(err)
	}
	me, them, _, _ := v.myGame()
	lastName, lastPos := "", ""
	for _, p := range v.starters(me) {
		name, pos := v.player(p.ID)
		if pos == lastPos && name < lastName {
			t.Fatal("fallback order is not alphabetical", lastName, name)
		}
		lastName, lastPos = name, pos
	}
	before := v.classicRows(me, them)
	me.Players = slices.Clone(me.Players)
	them.Players = slices.Clone(them.Players)
	slices.Reverse(me.Players)
	slices.Reverse(them.Players)
	after := v.classicRows(me, them)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("a shuffled feed changed the slot rows")
	}
}

func TestPulseRaceLiveSchedule(t *testing.T) {
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
			return
		}
	}
	t.Fatal("week 5 missing")
}

func TestPulsePluralAndProjectedBracket(t *testing.T) {
	if countText(1, "NFL game") != "1 NFL game" || pluralNoun(1, "game") != "game" ||
		pluralNoun(1.5, "game") != "games" {
		t.Fatal("incorrect singular/plural")
	}
	a := targetTestApp(t)
	at := raceHeldAt
	a.clockNow = func() time.Time { return at }
	a.week = leagueweek.Week{Number: 5, Games: []leagueweek.Game{{Kickoff: at.Add(-time.Hour)}}}
	status, err := a.TargetShellStatus()
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

func TestPulseReportsDefaultLastScored(t *testing.T) {
	in := nowInput{race: realRace(t, "2026"), current: 5, at: raceHeldAt}
	for i := range in.race.held.weeks.value {
		w := &in.race.held.weeks.value[i]
		if w.Week == 4 {
			for j := range w.Games {
				g := &w.Games[j]
				g.Home.Scored, g.Away.Scored = true, true
				g.Home.Result, g.Away.Result = "W", "L"
				g.Home.Points, g.Away.Points = 24615, 22125
			}
		}
	}
	board := nowBoard(t, in)
	if board.Week != 5 || board.ReportWeek != 4 || len(board.ReportWeeks) != 5 {
		t.Fatal(board.Week, board.ReportWeek, board.ReportWeeks)
	}
	for i, w := range board.ReportWeeks {
		if w.Value != strconv.Itoa(i+1) {
			t.Fatal("missing report week", w)
		}
	}
	for _, id := range []string{"M-003", "M-007", "M-021", "M-022"} {
		r := nowReport(t, board, id)
		if r.Status != "ok" || !strings.Contains(fmt.Sprint(r.Summary), "Week 4") {
			t.Fatal(id, r.Status, r.Summary)
		}
	}
	in.reportWeek = 5
	chosen := nowBoard(t, in)
	if r := nowReport(t, chosen, "M-003"); r.Status != "empty" || chosen.ReportWeek != 5 {
		t.Fatal("explicit unscored week must be selectable", r)
	}
	in.reportWeek = 3
	if b := nowBoard(t, in); b.ReportWeek != 3 || nowReport(t, b, "M-003").Status != "ok" {
		t.Fatal("explicit scored week not selected")
	}
}

func TestPulseSeriesUsesCurrentOpponent(t *testing.T) {
	a := targetTestApp(t)
	a.week.Number = 5
	a.seasonLineups = held(pulseLive(t))
	match := a.seasonLineups.value.Matchups[0]
	board, err := a.TargetArchive(0, match.Home, "")
	if err != nil || board.Other != match.Away {
		t.Fatal("series did not open on current opponent", board.Other, err)
	}
	if got := a.archiveOpponent(match.Home); got != match.Away {
		t.Fatal("home default", got, match.Away)
	}
	if got := a.archiveOpponent(match.Away); got != match.Home {
		t.Fatal("away default", got, match.Home)
	}
	a.race.weeks = realRace(t, "2026").held.weeks
	a.seasonLineups.value.Week = 4
	if got := a.archiveOpponent(match.Home); got != match.Away {
		t.Fatal("weekly schedule default", got, match.Away)
	}
}

func TestPulseArchiveNamesTotalsAndFirstChampion(t *testing.T) {
	cache := reducedArchive(t)
	board, err := buildArchive(cache, false, 2026, "", "")
	if err != nil {
		t.Fatal(err)
	}
	odd := archiveReport(t, board, "M-089")
	if strings.Contains(strings.ToLower(odd.Note+odd.How), "ring") || len(odd.Rows) == 0 {
		t.Fatal(odd.Note)
	}
	if !strings.Contains(odd.Rows[0].Cells[1], " · ") || !strings.Contains(odd.Rows[0].Cells[2], ": ") {
		t.Fatal("oddities lack count or franchise grouping", odd.Rows[0])
	}
	scoring := archiveReport(t, board, "M-073")
	for _, row := range scoring.Summary {
		if !strings.Contains(row.Cells[0], "all scored weeks") || (row.Cells[1] != "0.00" && !strings.Contains(row.Cells[1], ",")) {
			t.Fatal("unlabelled or unformatted era total", row.Cells)
		}
	}
	points := 51484.67
	if archivePoints(&points) != "51,484.67" || archiveRecordValue(archiveRecordEntry{value: points}) != "51,484.67" {
		t.Fatal("archive points formatter")
	}
	champ := archiveReport(t, board, "M-114")
	if len(champ.Rows) == 0 || champ.Rows[0].Cells[0] != "2013" || champ.Rows[0].Cells[1] != "not in MFL's archive" {
		t.Fatal("2013 silently omitted", champ.Rows)
	}
	if trophyTitle("2015 Legacy Superbowl III", "2015 Legacy Champions") !=
		"2015 Legacy Superbowl III · 2015 Legacy Champions" {
		t.Fatal("league's written title changed")
	}
}

func TestPulseClockKeepsWeekAfterLastKickoff(t *testing.T) {
	a, _, _ := weekTestApp(t, "17")
	at := time.Date(2026, 10, 13, 1, 0, 0, 0, time.UTC)
	a.clockNow = func() time.Time { return at }
	a.week = leagueweek.Week{Number: 5, Games: []leagueweek.Game{{
		Teams: [2]string{"BUF", "LAR"}, Kickoff: at.Add(-time.Hour),
	}}}
	a.weekFetchedAt = at.Add(-time.Minute)
	reading, err := a.TargetClock()
	if err != nil || reading.Value.Week == nil || *reading.Value.Week != 5 {
		t.Fatal("last game in progress lost week", reading, err)
	}
	if len(reading.Value.Deadlines) == 0 || reading.Value.Deadlines[0].At == nil ||
		!reading.Value.Deadlines[0].At.Before(at) {
		t.Fatal("lineup lock must be passed", reading.Value.Deadlines)
	}
	a.week.Games[0].Final = true
	if r, err := a.TargetClock(); err != nil || r.Value.Week == nil || *r.Value.Week != 5 {
		t.Fatal("do not lose week until next schedule is held", r, err)
	}
	a.week = leagueweek.Week{Number: 6, Games: []leagueweek.Game{{Kickoff: at.Add(48 * time.Hour)}}}
	if r, err := a.TargetClock(); err != nil || r.Value.Week == nil || *r.Value.Week != 6 {
		t.Fatal("next held schedule not adopted", r, err)
	}
}

func TestPulseStartupDatesWindowsWithoutRefresh(t *testing.T) {
	a, transport, _ := weekTestApp(t, "17")
	a.clockNow = func() time.Time { return time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC) }
	body := weekFixture(t, "nflSchedule-w5.json")
	for _, week := range []int{9, 13, 14} {
		scheduleBody := strings.Replace(body, `"week":"5"`, fmt.Sprintf(`"week":"%d"`, week), 1)
		if scheduleBody == body {
			t.Fatal("fixture week not replaced")
		}
		recordSeasonArchiveForm(t, a, "nflSchedule", strconv.Itoa(week), scheduleBody, raceHeldAt, true)
	}
	pools, err := db.Open(a.ctx, filepath.Join(t.TempDir(), "startup-whatif.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := pools.Close(); err != nil {
			t.Error(err)
		}
	})
	a.whatifPools = pools
	if _, err := a.initStoreFloor(a.ctx, a.history); err != nil {
		t.Fatal(err)
	}
	reading, err := a.TargetClock()
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
