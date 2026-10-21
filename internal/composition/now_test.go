package composition

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	marketapp "github.com/secureprospective/TheWarRoom/internal/app/market"
	raceapp "github.com/secureprospective/TheWarRoom/internal/app/race"
	"github.com/secureprospective/TheWarRoom/internal/app/report"
	researchapp "github.com/secureprospective/TheWarRoom/internal/app/research"

	"github.com/secureprospective/TheWarRoom/internal/app/core"
	nowapp "github.com/secureprospective/TheWarRoom/internal/app/now"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leaguestandings"
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
	return nowInput{race: raceInput{held: raceHeld{weeks: held(d.Race.Weeks), brackets: held(d.Race.Brackets), standings: held(d.Race.Standings)}, config: d.Race.Config, lookup: d.Race.Lookup, hasLookup: true, team: "0001", mine: "0001", zone: time.UTC}, week: week, current: 5, at: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), held: nowTestHeld{schedules: map[int]heldSeasonFeed[[]nflschedule.RawWeek]{5: held([]nflschedule.RawWeek{d.Schedule})}, byes: held(d.Byes)}, projections: held([]marketfeeds.Feed{d.Projections})}
}
func nowBoard(t *testing.T, in nowInput) nowapp.Board {
	t.Helper()
	board, err := buildNow(in)
	if err != nil {
		t.Fatal(err)
	}
	return board
}

func nowReport(t *testing.T, board nowapp.Board, id string) report.Report {
	t.Helper()
	for _, r := range append(board.Board.Reports, board.Matchup) {
		if r.ID == id {
			return r
		}
	}
	t.Fatalf("no report %s", id)
	return report.Report{}
}

func archivedCopies(copies ...struct {
	src  string
	body string
	at   time.Time
}) researchapp.ArchivedBodies {
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
	firsts, err := nowapp.ReadFirsts(context.Background(), nowapp.FirstScores{}, 2025, nowapp.Bodies(bodies))
	if err != nil {
		t.Fatal(err)
	}
	if firsts.Copies != 2 || !firsts.Through.Equal(day(3)) || len(firsts.Weeks) != 17 || !firsts.Weeks[5].At.Equal(day(2)) {
		t.Fatal("the oldest copy of each scored week is the first", firsts.Copies, firsts.Through, len(firsts.Weeks))
	}
	again, err := nowapp.ReadFirsts(context.Background(), firsts, 2025, nowapp.Bodies(bodies))
	if err != nil || again.Copies != 2 {
		t.Fatal("a second read reads only newer copies", again.Copies, err)
	}
	in := realNow(t, 5)
	in.held.firsts = firsts
	if r := nowReport(t, nowBoard(t, in), "M-088"); r.Status != "empty" ||
		!strings.Contains(r.Note, "No score has changed since the first held copy.") {
		t.Fatal(r.Status, r.Note)
	}
	weeks := in.race.held.weeks.Value
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
	a.core.WeekMu.Lock()
	a.core.Week.Number = 5
	a.core.WeekMu.Unlock()
	board, err := a.nowAPI().TargetNow(0, 0, "")
	if err != nil || len(transport.requests) != 0 || board.Week != 0 || board.Matchup.Status != "failed" {
		t.Fatal("reading fetched or invented a week", err, transport.requests, board.Week)
	}
	if _, err := a.nowAPI().TargetNow(3, 0, ""); err == nil {
		t.Fatal("accepted a week no held schedule lists")
	}
	if err := a.nowAPI().TargetNowRefresh(0, false); err != nil {
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
	read := func(week int) nowapp.Board {
		a.core.SeasonMu.Lock()
		in := nowInput{race: realRace(t, "2026"), held: nowTestHeld{schedules: a.now.schedules, byes: heldFeed(a.nowHeld.SnapshotByes()), firsts: a.nowHeld.SnapshotFirsts()}, current: 5, week: week, at: raceHeldAt}
		in.race.held = raceHeld{weeks: heldSeasonFeed[[]leaguefeed.WeekResult](a.raceHeld.Snapshot().Weeks), brackets: heldSeasonFeed[[]leaguefeed.Bracket](a.raceHeld.Snapshot().Brackets), standings: heldSeasonFeed[[]leaguestandings.RawStanding](a.raceHeld.Snapshot().Standings)}
		a.core.SeasonMu.Unlock()
		return nowBoard(t, in)
	}
	board = read(0)
	nfl := nowReport(t, board, "M-015")
	if board.Week != 5 || nfl.Status != "ok" || len(nfl.Rows) != 15 {
		t.Fatal(board.Week, nfl.Status, nfl.Note, len(nfl.Rows))
	}
	requests := len(transport.requests)
	if err := a.nowAPI().TargetNowRefresh(5, false); err != nil || slices.Contains(transport.requests[requests:], "nflSchedule") {
		t.Fatal("a fresh week refetched", err, transport.requests[requests:])
	}
	if err := a.nowAPI().TargetNowRefresh(6, false); err != nil {
		t.Fatal(err)
	}
	if r := nowReport(t, read(6), "M-015"); r.Status != "failed" || !strings.Contains(r.Note, "refresh from MFL") {
		t.Fatal("a schedule for the wrong week must not stand in", r.Status, r.Note)
	}
	week7 := strings.Replace(seasonFixture(t, "nflschedule", "nflSchedule-w5.json"), `"week":"5"`, `"week":"7"`, 1)
	recordSeasonArchiveForm(t, a, "nflSchedule", "7", week7, raceHeldAt, true)
	recordSeasonArchiveForm(t, a, "nflSchedule", "8", week7, raceHeldAt.Add(time.Hour), true)
	transport.failure = "nflSchedule"
	if err := a.nowAPI().TargetNowRefresh(7, false); err != nil {
		t.Fatal(err)
	}
	if r := nowReport(t, read(7), "M-015"); r.Status != "stale" || len(r.Rows) != 15 {
		t.Fatal("an offline week must fall back to its own archived copy", r.Status, r.Note)
	}
	if err := a.nowAPI().TargetNowRefresh(99, false); err == nil {
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
	a.core.WeekMu.Lock()
	a.core.Week.Number = 5
	a.core.WeekMu.Unlock()
	if err := a.nowAPI().TargetNowRefresh(0, false); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(transport.requests, "nflSchedule") {
		t.Fatal("the current week's NFL schedule was skipped", transport.requests)
	}
	if err := a.nowAPI().TargetNowRefresh(9, false); err == nil {
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
		Now nowapp.Board `json:"now"`
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
	board, err := a.nowAPI().TargetNow(0, 0, "")
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
	stopped := &App{marketHeld: &marketapp.Held{}, raceHeld: &raceapp.Held{}, core: &core.State{Started: make(chan struct{}), StartupErr: fmt.Errorf("fixture startup failed")}}
	close(stopped.core.Started)
	checkGolden(t, "now", "TargetNowRefresh", stopped.nowAPI().TargetNowRefresh(0, false).Error())
	if _, err := a.nowAPI().TargetLiveGame(false, "invalid!", ""); err == nil {
		t.Fatal("invalid follow accepted")
	} else {
		checkGolden(t, "now", "TargetLiveGame-invalid", err.Error())
	}
	a.nowHeld.PublishLive(nowapp.LiveGame{Week: a.marketWeek(), Adapter: livegame.Adapter{Fetch: func(context.Context, string, string) ([]byte, error) {
		t.Fatal("closed binding fetched")
		return nil, nil
	}}})
	view, err := a.nowAPI().TargetLiveGame(false, "", "")
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
	firsts, err := nowapp.ReadFirsts(context.Background(), nowapp.FirstScores{}, 2025, nowapp.Bodies(bodies))
	if err != nil || firsts.Copies != 0 || len(firsts.Weeks) != 0 {
		t.Fatal("a body that does not parse is passed over", firsts, err)
	}
	if _, err := weeklyresults.Parse([]byte(`{}`)); err == nil {
		t.Fatal("weeklyresults accepted an empty export")
	}
}

// nowTestHeld is the fixture's Now feeds: the schedules App holds and the holder's byes and first scores.
type nowTestHeld struct {
	schedules map[int]heldSeasonFeed[[]nflschedule.RawWeek]
	byes      heldSeasonFeed[[]nflschedule.RawBye]
	firsts    nowapp.FirstScores
}

type nowInput struct {
	race        raceInput
	held        nowTestHeld
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
	return nowapp.Snapshot{Race: in.race.snapshot(), Held: nowapp.Feeds{Schedules: schedules, Byes: nowFeed(in.held.byes), Firsts: in.held.firsts}, Live: nowFeed(in.live), Projections: nowFeed(in.projections), At: in.at, ReportWeek: in.reportWeek, Current: in.current, Week: in.week}
}
func buildNow(in nowInput) (nowapp.Board, error) {
	return nowapp.Build(in.snapshot())
}

func TestNowSnapshotDetachedAndConstructionUnlocked(t *testing.T) {
	t.Parallel()
	a := targetTestApp(t)
	fixture := realNow(t, 5)
	if _, err := a.core.Rulebook.Sync(a.core.Ctx, fixture.race.config); err != nil {
		t.Fatal(err)
	}
	publishRaceFixture(a, fixture.race.held)
	a.now, a.core.SeasonLineups = nowSchedules{schedules: fixture.held.schedules}, fixture.live
	a.nowHeld.PublishByes(nowFeed(fixture.held.byes))
	a.nowHeld.PublishFirsts(nowapp.FirstScores{Weeks: map[int]nowapp.FirstScore{5: {At: raceHeldAt, Week: fixture.race.held.weeks.Value[4]}}, Copies: 1})
	in := nowHost{State: a.core, app: a}.NowInput(5, 0, "")
	a.core.SeasonMu.Lock()
	delete(a.now.schedules, 5)
	firsts := a.nowHeld.SnapshotFirsts()
	delete(firsts.Weeks, 5)
	a.nowHeld.PublishFirsts(firsts)
	byes := a.nowHeld.SnapshotByes()
	byes.RefreshError = "replaced"
	a.nowHeld.PublishByes(byes)
	a.core.SeasonMu.Unlock()
	if len(in.Held.Schedules) != 1 || len(in.Held.Firsts.Weeks) != 1 || in.Held.Byes.RefreshError != "" {
		t.Fatal("snapshot retained mutable root state")
	}
	a.core.SeasonMu.Lock()
	defer a.core.SeasonMu.Unlock()
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
	if _, err := a.core.Rulebook.Sync(a.core.Ctx, realRace(t, "2026").config); err != nil {
		t.Fatal(err)
	}
	a.core.Week.Number = 5
	transport.bodies["nflSchedule"] = seasonFixture(t, "nflschedule", "nflSchedule-w5.json")
	transport.bodies["nflByeWeeks"] = seasonFixture(t, "nflschedule", "nflByeWeeks-2025.json")
	published, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	a.core.SeasonPause = func(ctx context.Context, _ time.Duration) error {
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
	go func() { refreshed <- a.nowAPI().TargetNowRefresh(5, true) }()
	select {
	case <-published:
	case <-time.After(5 * time.Second):
		t.Fatal("refresh did not publish")
	}
	firstRead, readDone := make(chan struct{}, 1), make(chan error, 1)
	go func() {
		for i := 0; i < 12; i++ {
			if _, err := a.nowAPI().TargetNow(0, 0, ""); err != nil {
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
	in := nowHost{State: a.core, app: a}.NowInput(5, 0, "")
	if in.Held.Schedules[5].FetchedAt.IsZero() || in.Held.Byes.FetchedAt.IsZero() || in.Race.Held.Weeks.FetchedAt.IsZero() {
		t.Fatal("refresh failed to publish feeds")
	}
}
