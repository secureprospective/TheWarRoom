package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	nowapp "github.com/secureprospective/TheWarRoom/internal/app/now"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"

	"github.com/secureprospective/TheWarRoom/internal/app/now/nowtest"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/weeklyresults"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/marketfeeds"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
)

func realNow(t *testing.T, week int) nowInput {
	t.Helper()
	d := nowtest.Load(t)
	return nowInput{race: raceInput{held: raceHeld{weeks: held(d.Race.Weeks), brackets: held(d.Race.Brackets), standings: held(d.Race.Standings)}, config: d.Race.Config, lookup: d.Race.Lookup, hasLookup: true, team: "0001", mine: "0001", zone: time.UTC}, week: week, current: 5, at: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), held: nowHeld{schedules: map[int]heldSeasonFeed[[]nflschedule.RawWeek]{5: held([]nflschedule.RawWeek{d.Schedule})}, byes: held(d.Byes)}, projections: held([]marketfeeds.Feed{d.Projections})}
}
func nowBoard(t *testing.T, in nowInput) NowBoard {
	t.Helper()
	board, err := buildNow(in)
	if err != nil {
		t.Fatal(err)
	}
	return board
}

func nowReport(t *testing.T, board NowBoard, id string) Report {
	t.Helper()
	for _, r := range append(board.Board.Reports, board.Matchup) {
		if r.ID == id {
			return r
		}
	}
	t.Fatalf("no report %s", id)
	return Report{}
}

func archivedCopies(copies ...struct {
	src  string
	body string
	at   time.Time
}) archivedBodies {
	return func(_ context.Context, part string, accept func(string, []byte, time.Time) bool) (bool, error) {
		sorted := slices.Clone(copies)
		slices.SortFunc(sorted, func(a, b struct {
			src  string
			body string
			at   time.Time
		}) int {
			return b.at.Compare(a.at)
		})
		for _, c := range sorted {
			if strings.Contains(c.src, part) && accept(c.src, []byte(c.body), c.at) {
				return true, nil
			}
		}
		return false, nil
	}
}

func TestNowStatChangesCompareTheFirstHeldCopy(t *testing.T) {
	t.Parallel()
	body := seasonGzFixture(t, "weeklyresults", "weeklyResults-2025.json.gz")
	ytd := "/2025/export?TYPE=weeklyResults&L=" + "14432" + "&W=YTD"
	day := func(d int) time.Time { return time.Date(2026, 1, d, 12, 0, 0, 0, time.UTC) }
	type copyOf = struct {
		src  string
		body string
		at   time.Time
	}
	bodies := archivedCopies(copyOf{ytd, body, day(2)}, copyOf{ytd, body, day(3)},
		copyOf{"/2025/export?TYPE=weeklyResults&L=14432&W=5", body, day(4)},
		copyOf{"/2024/export?TYPE=weeklyResults&L=14432&W=YTD", body, day(5)})
	firsts, err := readScoreFirsts(context.Background(), scoreFirsts{}, 2025, bodies)
	if err != nil {
		t.Fatal(err)
	}
	if firsts.copies != 2 || !firsts.through.Equal(day(3)) || len(firsts.weeks) != 17 || !firsts.weeks[5].at.Equal(day(2)) {
		t.Fatal("the oldest copy of each scored week is the first", firsts.copies, firsts.through, len(firsts.weeks))
	}
	again, err := readScoreFirsts(context.Background(), firsts, 2025, bodies)
	if err != nil || again.copies != 2 {
		t.Fatal("a second read reads only newer copies", again.copies, err)
	}
	in := realNow(t, 5)
	in.held.firsts = firsts
	if r := nowReport(t, nowBoard(t, in), "M-088"); r.Status != "empty" ||
		!strings.Contains(r.Note, "No score has changed since the first held copy.") {
		t.Fatal(r.Status, r.Note)
	}
	weeks := in.race.held.weeks.value
	for i, w := range weeks {
		if w.Week != 3 {
			continue
		}
		weeks[i].Games = slices.Clone(w.Games)
		g := &weeks[i].Games[0]
		g.Home.Players = slices.Clone(g.Home.Players)
		for j, p := range g.Home.Players {
			if p.Starter {
				g.Home.Players[j].Points += 650
				break
			}
		}
		g.Home.Result, g.Away.Result = g.Away.Result, g.Home.Result
	}
	r := nowReport(t, nowBoard(t, in), "M-088")
	if r.Status != "ok" || len(r.Rows) != 1 || r.Rows[0].Cells[4] != "+6.50" ||
		r.Rows[0].Group != "Week 3 · first held Jan 2, 2026, 12:00 PM" {
		t.Fatal(r.Status, r.Rows)
	}
	flips := 0
	for _, row := range r.Summary {
		if row.Cells[0] == "Week 3 result changed" {
			flips++
		}
	}
	if flips != 2 {
		t.Fatal("both sides of a flipped result are named", r.Summary)
	}
}

func TestNowBindingReadsHeldAndRefreshReadsTheWeek(t *testing.T) {
	t.Parallel()
	a, transport, _, _ := seasonTestApp(t)
	transport.bodies["nflSchedule"] = seasonFixture(t, "nflschedule", "nflSchedule-w5.json")
	transport.bodies["nflByeWeeks"] = seasonFixture(t, "nflschedule", "nflByeWeeks-2025.json")
	a.weekMu.Lock()
	a.week.Number = 5
	a.weekMu.Unlock()
	board, err := a.TargetNow(0, 0, "")
	if err != nil || len(transport.requests) != 0 || board.Week != 0 || board.Matchup.Status != "failed" {
		t.Fatal("reading fetched or invented a week", err, transport.requests, board.Week)
	}
	if _, err := a.TargetNow(3, 0, ""); err == nil {
		t.Fatal("accepted a week no held schedule lists")
	}
	if err := a.TargetNowRefresh(0, false); err != nil {
		t.Fatal(err)
	}
	asked := map[string]string{}
	for i, r := range transport.requests {
		asked[r] = transport.weeks[i]
	}
	if asked["nflSchedule"] != "5" || asked["weeklyResults"] != "YTD" {
		t.Fatal("refresh must read the season's results and the current week's NFL schedule", asked)
	}
	if _, ok := asked["nflByeWeeks"]; !ok {
		t.Fatal("refresh must read the bye weeks", asked)
	}
	// The test app's rulebook holds no league, so the held feeds are read with the real 2026 league.
	read := func(week int) NowBoard {
		a.seasonMu.Lock()
		in := nowInput{race: realRace(t, "2026"), held: a.now, current: 5, week: week, at: raceHeldAt}
		in.race.held = a.race
		a.seasonMu.Unlock()
		return nowBoard(t, in)
	}
	board = read(0)
	nfl := nowReport(t, board, "M-015")
	if board.Week != 5 || nfl.Status != "ok" || len(nfl.Rows) != 15 {
		t.Fatal(board.Week, nfl.Status, nfl.Note, len(nfl.Rows))
	}
	requests := len(transport.requests)
	if err := a.TargetNowRefresh(5, false); err != nil || slices.Contains(transport.requests[requests:], "nflSchedule") {
		t.Fatal("a fresh week refetched", err, transport.requests[requests:])
	}
	if err := a.TargetNowRefresh(6, false); err != nil {
		t.Fatal(err)
	}
	if r := nowReport(t, read(6), "M-015"); r.Status != "failed" || !strings.Contains(r.Note, "refresh from MFL") {
		t.Fatal("a schedule for the wrong week must not stand in", r.Status, r.Note)
	}
	week7 := strings.Replace(seasonFixture(t, "nflschedule", "nflSchedule-w5.json"), `"week":"5"`, `"week":"7"`, 1)
	recordSeasonArchiveForm(t, a, "nflSchedule", "7", week7, raceHeldAt, true)
	recordSeasonArchiveForm(t, a, "nflSchedule", "8", week7, raceHeldAt.Add(time.Hour), true)
	transport.failure = "nflSchedule"
	if err := a.TargetNowRefresh(7, false); err != nil {
		t.Fatal(err)
	}
	if r := nowReport(t, read(7), "M-015"); r.Status != "stale" || len(r.Rows) != 15 {
		t.Fatal("an offline week must fall back to its own archived copy", r.Status, r.Note)
	}
	if err := a.TargetNowRefresh(99, false); err == nil {
		t.Fatal("accepted a week the league's schedule does not list")
	}
}

// Just after launch the league's weeks may not be held yet; the current NFL week's schedule is still
// fetched, because it does not depend on them. Another unlisted week is still refused.

func TestNowRefreshFetchesTheCurrentWeekBeforeLeagueWeeksAreHeld(t *testing.T) {
	t.Parallel()
	a, transport, _, _ := seasonTestApp(t)
	transport.bodies["nflSchedule"] = seasonFixture(t, "nflschedule", "nflSchedule-w5.json")
	transport.bodies["nflByeWeeks"] = seasonFixture(t, "nflschedule", "nflByeWeeks-2025.json")
	transport.failure = "weeklyResults"
	a.weekMu.Lock()
	a.week.Number = 5
	a.weekMu.Unlock()
	if err := a.TargetNowRefresh(0, false); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(transport.requests, "nflSchedule") {
		t.Fatal("the current week's NFL schedule was skipped", transport.requests)
	}
	if err := a.TargetNowRefresh(9, false); err == nil {
		t.Fatal("accepted an unlisted week that is not the current one")
	}
}

func TestNowWireMatchesGo(t *testing.T) {
	t.Parallel()
	board := nowBoard(t, realNow(t, 5))
	board.Matchup.Rows = board.Matchup.Rows[:3]
	for i := range board.Board.Reports {
		r := &board.Board.Reports[i]
		r.Rows = r.Rows[:min(len(r.Rows), 3)]
	}
	body, err := json.MarshalIndent(struct {
		Now NowBoard `json:"now"`
	}{board}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	file := "frontend/src/app/now/wire.testdata.json"
	body = append(body, '\n')
	if os.Getenv("TWR_WRITE_WIRE") == "1" {
		if err := os.WriteFile(file, body, 0o644); err != nil { //nolint:gosec // a committed test fixture
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(want) != string(body) {
		t.Fatal("now wire stale; TWR_WRITE_WIRE=1 go test -run TestNowWireMatchesGo .")
	}
}

func TestNowBindingsGolden(t *testing.T) {
	a := targetTestApp(t)
	board, err := a.TargetNow(0, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	checkGolden(t, "now", "TargetNow", board)
	populated := nowBoard(t, realNow(t, 5))
	populated.Matchup.Rows = populated.Matchup.Rows[:min(3, len(populated.Matchup.Rows))]
	for i := range populated.Board.Reports {
		r := &populated.Board.Reports[i]
		r.Rows = r.Rows[:min(3, len(r.Rows))]
	}
	checkGolden(t, "now", "held", populated)
	stopped := &App{started: make(chan struct{}), startupErr: fmt.Errorf("fixture startup failed")}
	close(stopped.started)
	checkGolden(t, "now", "TargetNowRefresh", stopped.TargetNowRefresh(0, false).Error())
	if _, err := a.TargetLiveGame(false, "invalid!", ""); err == nil {
		t.Fatal("invalid follow accepted")
	} else {
		checkGolden(t, "now", "TargetLiveGame-invalid", err.Error())
	}
	a.liveGame.adapter = livegame.Adapter{Fetch: func(context.Context, string, string) ([]byte, error) {
		t.Fatal("closed binding fetched")
		return nil, nil
	}}
	view, err := a.TargetLiveGame(false, "", "")
	if err != nil {
		t.Fatal(err)
	}
	checkGolden(t, "now", "TargetLiveGame", view)
	checkGolden(t, "now", "live-held", liveWire(t))
}
func TestNowParsesOnlyValidWeeklyResults(t *testing.T) {
	t.Parallel()
	bodies := archivedCopies(struct {
		src  string
		body string
		at   time.Time
	}{"/2025/export?TYPE=weeklyResults&L=14432&W=YTD", `{"allWeeklyResults":{"weeklyResults":[]}}`, raceHeldAt})
	firsts, err := readScoreFirsts(context.Background(), scoreFirsts{}, 2025, bodies)
	if err != nil || firsts.copies != 0 || len(firsts.weeks) != 0 {
		t.Fatal("a body that does not parse is passed over", firsts, err)
	}
	if _, err := weeklyresults.Parse([]byte(`{}`)); err == nil {
		t.Fatal("weeklyresults accepted an empty export")
	}
}

type nowInput struct {
	race        raceInput
	held        nowHeld
	live        heldSeasonFeed[leaguefeed.Lineups]
	projections heldSeasonFeed[[]marketfeeds.Feed]
	at          time.Time
	reportWeek  int
	current     int
	week        int
}

func (in nowInput) snapshot() nowapp.Snapshot {
	schedules := make(map[int]nowapp.Feed[[]nflschedule.RawWeek], len(in.held.schedules))
	for k, v := range in.held.schedules {
		schedules[k] = nowFeed(v)
	}
	firsts := make(map[int]nowapp.FirstScore, len(in.held.firsts.weeks))
	for k, v := range in.held.firsts.weeks {
		firsts[k] = nowapp.FirstScore{At: v.at, Week: v.week}
	}
	return nowapp.Snapshot{Race: in.race.snapshot(), Held: nowapp.Held{Schedules: schedules, Byes: nowFeed(in.held.byes), Firsts: nowapp.FirstScores{Weeks: firsts, Copies: in.held.firsts.copies}}, Live: nowFeed(in.live), Projections: nowFeed(in.projections), At: in.at, ReportWeek: in.reportWeek, Current: in.current, Week: in.week}
}
func buildNow(in nowInput) (NowBoard, error) {
	b, e := nowapp.Build(in.snapshot())
	return wireNow(b), e
}

func TestNowWireTwins(t *testing.T) {
	t.Parallel()
	var board nowapp.Board
	n := 0
	fillWire(reflect.ValueOf(&board).Elem(), &n)
	if string(sourcesWire(t, board)) != string(sourcesWire(t, wireNow(board))) {
		t.Fatal("Now converter lost a field")
	}
	var live nowapp.LiveGameView
	// Game payloads are shared transport types; populate each root-owned envelope twin.
	fillWire(reflect.ValueOf(&live).Elem().FieldByName("Box"), &n)
	var player nowapp.SpinePlayer
	fillWire(reflect.ValueOf(&player).Elem(), &n)
	live.Spine = []nowapp.SpineEntry{{Player: player, Game: "BUF at LAR", Play: livegame.Play{ID: "play", At: raceHeldAt}}}
	live.WeekSpine = live.Spine
	live.Source, live.AsOf, live.Failure, live.DetailAsOf, live.DetailFailure = "source", "at", "failure", "detail at", "detail failure"
	live.Unmatched, live.WeekHeld, live.WeekGames, live.Active = 3, 4, 5, true
	for _, v := range []nowapp.LiveGameView{live, {}, {Spine: []nowapp.SpineEntry{}, WeekSpine: []nowapp.SpineEntry{}}} {
		if string(sourcesWire(t, v)) != string(sourcesWire(t, wireLive(v))) {
			t.Fatal("Live game converter lost a field or slice shape")
		}
	}
}

func TestNowSnapshotDetachedAndConstructionUnlocked(t *testing.T) {
	t.Parallel()
	a := targetTestApp(t)
	fixture := realNow(t, 5)
	if _, err := a.rulebook.Sync(a.ctx, fixture.race.config); err != nil {
		t.Fatal(err)
	}
	a.race, a.now, a.seasonLineups = fixture.race.held, fixture.held, fixture.live
	a.now.firsts = scoreFirsts{weeks: map[int]firstScores{5: {at: raceHeldAt, week: fixture.race.held.weeks.value[4]}}, copies: 1}
	in := a.nowReadInput(5, 0, "")
	a.seasonMu.Lock()
	delete(a.now.schedules, 5)
	delete(a.now.firsts.weeks, 5)
	a.now.byes.refreshError = "replaced"
	a.seasonMu.Unlock()
	if len(in.Held.Schedules) != 1 || len(in.Held.Firsts.Weeks) != 1 || in.Held.Byes.RefreshError != "" {
		t.Fatal("snapshot retained mutable root state")
	}
	a.seasonMu.Lock()
	defer a.seasonMu.Unlock()
	done := make(chan error, 1)
	go func() { _, err := nowapp.Build(in); done <- err }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("construction waited on season lock")
	}
}

func TestNowConcurrentRefreshAndRead(t *testing.T) {
	t.Parallel()
	a, transport, _, _ := seasonTestApp(t)
	if _, err := a.rulebook.Sync(a.ctx, realRace(t, "2026").config); err != nil {
		t.Fatal(err)
	}
	a.week.Number = 5
	transport.bodies["nflSchedule"] = seasonFixture(t, "nflschedule", "nflSchedule-w5.json")
	transport.bodies["nflByeWeeks"] = seasonFixture(t, "nflschedule", "nflByeWeeks-2025.json")
	published, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	a.seasonPause = func(ctx context.Context, _ time.Duration) error {
		once.Do(func() {
			close(published)
			select {
			case <-release:
			case <-ctx.Done():
			}
		})
		return ctx.Err()
	}
	defer close(release)
	refreshed := make(chan error, 1)
	go func() { refreshed <- a.TargetNowRefresh(5, true) }()
	select {
	case <-published:
	case <-time.After(5 * time.Second):
		t.Fatal("refresh did not publish")
	}
	firstRead, readDone := make(chan struct{}, 1), make(chan error, 1)
	go func() {
		for i := 0; i < 12; i++ {
			if _, err := a.TargetNow(0, 0, ""); err != nil {
				readDone <- err
				return
			}
			if i == 0 {
				firstRead <- struct{}{}
			}
		}
		readDone <- nil
	}()
	select {
	case <-firstRead:
	case err := <-readDone:
		t.Fatalf("reading during refresh: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("read blocked behind refresh")
	}
	release <- struct{}{}
	if err := <-refreshed; err != nil {
		t.Fatal(err)
	}
	if err := <-readDone; err != nil {
		t.Fatal(err)
	}
	in := a.nowReadInput(5, 0, "")
	if in.Held.Schedules[5].FetchedAt.IsZero() || in.Held.Byes.FetchedAt.IsZero() || in.Race.Held.Weeks.FetchedAt.IsZero() {
		t.Fatal("refresh failed to publish feeds")
	}
}
