package composition

import (
	"context"
	"sync"
	"testing"
	"time"

	raceapp "github.com/secureprospective/TheWarRoom/internal/app/race"
	"github.com/secureprospective/TheWarRoom/internal/app/season"
	"github.com/secureprospective/TheWarRoom/internal/app/season/seasontest"
	"github.com/secureprospective/TheWarRoom/internal/engine/race"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leaguestandings"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
)

var raceHeldAt = time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC) //nolint:gochecknoglobals // fixed fixture clock
func held[T any](v T) heldSeasonFeed[T]                         { return heldSeasonFeed[T]{Value: v, FetchedAt: raceHeldAt} }
func realRace(t *testing.T, year string) raceInput {
	d := seasontest.Load(t, year)
	return raceInput{held: raceHeld{weeks: held(d.Weeks), brackets: held(d.Brackets), standings: held(d.Standings)}, config: d.Config, lookup: d.Lookup, hasLookup: true, team: "0001", mine: "0001", zone: time.UTC}
}
func testPower() PowerRankingsResult { return wirePower(seasontest.Power()) }
func TestRaceBindingReadsHeldAndRefreshFallsBack(t *testing.T) {
	t.Parallel()
	a, transport, _, _ := seasonTestApp(t)
	a.core.SeasonChanged = func(context.Context) { t.Error("Race refresh emitted a season event; its caller owns notification") }
	board, err := raceapp.New(a.core, a.raceHeld).TargetRace(0, "", "")
	if err != nil || len(transport.requests) != 0 || board.Standings.Status != "failed" {
		t.Fatal("reading fetched or invented standings", err, transport.requests)
	}
	if _, err := raceapp.New(a.core, a.raceHeld).TargetRace(3, "", ""); err == nil {
		t.Fatal("accepted a week with no held results")
	}
	recordSeasonArchive(t, a, "weeklyResults", "YTD", transport.bodies["weeklyResults"], raceHeldAt)
	transport.failure = "weeklyResults"
	if err := raceapp.New(a.core, a.raceHeld).TargetRaceRefresh(false); err != nil {
		t.Fatal(err)
	}
	a.core.SeasonMu.Lock()
	weeks := a.raceHeld.Snapshot().Weeks
	a.core.SeasonMu.Unlock()
	if len(weeks.Value) != 17 || weeks.RefreshError == "" {
		t.Fatal("archived weekly results not served as stale", weeks.RefreshError)
	}
	requests := len(transport.requests)
	if err := raceapp.New(a.core, a.raceHeld).TargetRaceRefresh(false); err != nil || len(transport.requests) == requests {
		t.Fatal("a stale feed must refresh again", err)
	}
	transport.failure = ""
	requests = len(transport.requests)
	_ = raceapp.New(a.core, a.raceHeld).TargetRaceRefresh(true)
	if err := raceapp.New(a.core, a.raceHeld).TargetRaceRefresh(false); err != nil || len(transport.requests) != requests+6 {
		t.Fatal("fresh feeds refetched", transport.requests[requests:])
	}
}

func TestRaceBindingsGolden(t *testing.T) {
	t.Parallel()
	a := targetTestApp(t)
	board, err := raceapp.New(a.core, a.raceHeld).TargetRace(0, "", "")
	if err != nil {
		t.Fatal(err)
	}
	checkGolden(t, "race", "unfetched", board)
	in := realRace(t, "2026")
	in.held.power = map[string]PowerRankingsResult{ViewSeason: testPower()}
	populated := targetTestApp(t)
	if _, syncErr := populated.core.Rulebook.Sync(populated.core.Ctx, in.config); syncErr != nil {
		t.Fatal(syncErr)
	}
	populated.core.Lookup, populated.core.HasLookup = in.lookup, true
	populated.core.LocalZone = time.UTC
	publishRaceFixture(populated, in.held)
	board, err = raceapp.New(populated.core, populated.raceHeld).TargetRace(0, in.team, in.mine)
	if err != nil {
		t.Fatal(err)
	}
	board.Standings.Rows = board.Standings.Rows[:5]
	for i := range board.Board.Reports {
		r := &board.Board.Reports[i]
		r.Rows = r.Rows[:min(len(r.Rows), 3)]
	}
	checkGolden(t, "race", "held", board)
	errs := map[string]string{}
	if _, err := raceapp.New(a.core, a.raceHeld).TargetRace(40, "", ""); err != nil {
		errs["week"] = err.Error()
	}
	if _, err := raceapp.New(a.core, a.raceHeld).TargetRace(0, "unknown", ""); err != nil {
		errs["team"] = err.Error()
	}
	refresh, _, _, _ := seasonTestApp(t)
	if err := raceapp.New(refresh.core, refresh.raceHeld).TargetRaceRefresh(true); err != nil {
		errs["refresh"] = err.Error()
	}
	checkGolden(t, "race", "errors", errs)
}

type raceField struct {
	bracket leaguefeed.Bracket
	field   race.Field
}

func (v raceView) fields() ([]raceField, string) {
	fields, note := v.view().Fields()
	out := make([]raceField, len(fields))
	for i, f := range fields {
		out[i] = raceField{bracket: f.Bracket, field: f.Seeding}
	}
	return out, note
}
func (v raceView) projectedBracketSide(b leaguefeed.Bracket, s leaguefeed.BracketSide, names map[int]string) string {
	return v.view().ProjectedBracketSide(b, s, names)
}
func TestRaceSnapshotDetachedAndConstructionUnlocked(t *testing.T) {
	t.Parallel()
	a := targetTestApp(t)
	fixture := realRace(t, "2026")
	fixture.held.power = map[string]PowerRankingsResult{ViewSeason: testPower()}
	publishRaceFixture(a, fixture.held)
	in := a.raceReadInput()
	in.config, in.lookup, in.hasLookup, in.zone = fixture.config, fixture.lookup, true, time.UTC
	a.core.SeasonMu.Lock()
	held := a.raceHeld.Snapshot()
	delete(held.Power, ViewSeason)
	a.raceHeld.PublishStandings(held.Standings, held.Power, true)
	a.core.SeasonMu.Unlock()
	if _, ok := in.held.power[ViewSeason]; !ok {
		t.Fatal("snapshot retained the mutable power index")
	}
	snapshot := in.snapshot()
	a.core.SeasonMu.Lock()
	defer a.core.SeasonMu.Unlock()
	done := make(chan error, 1)
	go func() {
		_, err := raceapp.Build(snapshot)
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("pure construction waited on the root season lock")
	}
}

func TestRaceConcurrentRefreshAndRead(t *testing.T) {
	t.Parallel()
	a, _, _, _ := seasonTestApp(t)
	if _, err := a.core.Rulebook.Sync(a.core.Ctx, realRace(t, "2026").config); err != nil {
		t.Fatal(err)
	}
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
	go func() { refreshed <- raceapp.New(a.core, a.raceHeld).TargetRaceRefresh(true) }()
	select {
	case <-published:
	case <-time.After(5 * time.Second):
		t.Fatal("refresh did not publish weekly results")
	}
	read := make(chan struct{}, 1)
	readDone := make(chan error, 1)
	go func() {
		for i := 0; i < 100; i++ {
			_, err := raceapp.New(a.core, a.raceHeld).TargetRace(0, "", "")
			if err != nil {
				readDone <- err
				return
			}
			if i == 0 {
				read <- struct{}{}
			}
		}
		readDone <- nil
	}()
	select {
	case <-read:
	case err := <-readDone:
		t.Fatalf("reading failed during refresh: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("reading blocked behind refresh")
	}
	release <- struct{}{}
	if err := <-refreshed; err != nil {
		t.Fatal(err)
	}
	if err := <-readDone; err != nil {
		t.Fatal(err)
	}
	in := a.raceReadInput()
	if len(in.held.weeks.Value) == 0 || in.held.brackets.FetchedAt.IsZero() || in.held.standings.FetchedAt.IsZero() {
		t.Fatal("refresh did not publish all race feeds")
	}
}

// Shared fixture facade for Now/Pulse integration tests.
type raceHeld struct {
	weeks     heldSeasonFeed[[]leaguefeed.WeekResult]
	brackets  heldSeasonFeed[[]leaguefeed.Bracket]
	standings heldSeasonFeed[[]leaguestandings.RawStanding]
	power     map[string]PowerRankingsResult
}

func publishRaceFixture(a *App, h raceHeld) {
	a.raceHeld.PublishWeeks(raceFeed(h.weeks))
	a.raceHeld.PublishBrackets(raceFeed(h.brackets))
	a.raceHeld.PublishStandings(raceFeed(h.standings), racePower(h.power), true)
}

func (a *App) raceReadInput() raceInput {
	in := a.raceSnapshot(a.raceHeld.Snapshot)
	h := in.Held
	power := make(map[string]PowerRankingsResult, len(h.Power))
	for key, p := range h.Power {
		power[key] = wirePower(p)
	}
	return raceInput{held: raceHeld{weeks: heldSeasonFeed[[]leaguefeed.WeekResult](h.Weeks), brackets: heldSeasonFeed[[]leaguefeed.Bracket](h.Brackets), standings: heldSeasonFeed[[]leaguestandings.RawStanding](h.Standings), power: power}, live: heldSeasonFeed[leaguefeed.Lineups](in.Live), liveUnplayed: in.LiveUnplayed}
}

type raceInput struct {
	live heldSeasonFeed[leaguefeed.Lineups]
	// liveUnplayed: no NFL game of the live week has kicked off, so its 0.00 scores are not a game in progress.
	liveUnplayed bool
	held         raceHeld
	config       league.RawConfig
	lookup       normalize.Lookup
	week         int
	team         string
	mine         string
	zone         *time.Location
	hasLookup    bool
}

func (in raceInput) snapshot() season.Snapshot {
	power := racePower(in.held.power)
	liveWeeks := map[int]leaguefeed.WeekResult{}
	for _, w := range in.held.weeks.Value {
		if w.Week == in.live.Value.Week {
			liveWeeks[w.Week] = liveWeek(in.live.Value, w)
		}
	}
	return season.Snapshot{LiveWeeks: liveWeeks, Held: season.Held{Weeks: raceFeed(in.held.weeks), Brackets: raceFeed(in.held.brackets), Standings: raceFeed(in.held.standings), Power: power}, Live: raceFeed(in.live), LiveUnplayed: in.liveUnplayed, Config: in.config, Lookup: in.lookup, HasLookup: in.hasLookup, Week: in.week, Team: in.team, Mine: in.mine, Zone: in.zone}
}

// raceView is the shared Pulse integration-test facade.
type raceView struct {
	names   map[string]string
	service season.View
	in      raceInput
	table   race.Table
	weeks   []leaguefeed.WeekResult
	week    int
	team    string
}

func newRaceView(in raceInput) (raceView, error) {
	v, err := season.NewView(in.snapshot())
	return raceView{names: v.Names, service: v, in: in, table: v.Table, weeks: v.Weeks, week: v.Week, team: v.Team}, err
}
func (v raceView) view() season.View      { s := v.service; s.Week = v.week; s.Team = v.team; return s }
func (v raceView) name(id string) string  { return v.view().Name(id) }
func (v raceView) scheduleByTeam() Report { return wireReport(v.view().ScheduleByTeam()) }
func (v raceView) bubble() Report         { return wireReport(v.view().Bubble()) }

func TestRaceJointCaptureConsistency(t *testing.T) {
	a := targetTestApp(t)
	a.core.SeasonLineups.Value.Week = 1
	a.raceHeld.PublishWeeks(season.Feed[[]leaguefeed.WeekResult]{Value: []leaguefeed.WeekResult{{Week: 1}}})
	in := a.raceSnapshot(func() season.Held {
		if a.core.SeasonMu.TryLock() {
			a.core.SeasonMu.Unlock()
			t.Fatal("joint capture lost SeasonMu")
		}
		return a.raceHeld.Snapshot()
	})
	if in.Held.Weeks.Value[0].Week != in.Live.Value.Week {
		t.Fatal("joint capture split")
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 1; i <= 500; i++ {
			a.core.SeasonMu.Lock()
			a.raceHeld.PublishWeeks(season.Feed[[]leaguefeed.WeekResult]{Value: []leaguefeed.WeekResult{{Week: i}}})
			a.core.SeasonLineups.Value.Week = i
			a.core.SeasonMu.Unlock()
		}
	}()
	for range 500 {
		snapshot := a.raceSnapshot(a.raceHeld.Snapshot)
		if snapshot.Held.Weeks.Value[0].Week != snapshot.Live.Value.Week {
			t.Error("mixed publication generations")
			break
		}
	}
	<-done
}
