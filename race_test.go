package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	raceapp "github.com/secureprospective/TheWarRoom/internal/app/race"
	"github.com/secureprospective/TheWarRoom/internal/app/season/seasontest"
	"github.com/secureprospective/TheWarRoom/internal/engine/race"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
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
	board, err := a.TargetRace(0, "", "")
	if err != nil || len(transport.requests) != 0 || board.Standings.Status != "failed" {
		t.Fatal("reading fetched or invented standings", err, transport.requests)
	}
	if _, err := a.TargetRace(3, "", ""); err == nil {
		t.Fatal("accepted a week with no held results")
	}
	recordSeasonArchive(t, a, "weeklyResults", "YTD", transport.bodies["weeklyResults"], raceHeldAt)
	transport.failure = "weeklyResults"
	if err := a.TargetRaceRefresh(false); err != nil {
		t.Fatal(err)
	}
	a.core.SeasonMu.Lock()
	weeks := a.race.weeks
	a.core.SeasonMu.Unlock()
	if len(weeks.Value) != 17 || weeks.RefreshError == "" {
		t.Fatal("archived weekly results not served as stale", weeks.RefreshError)
	}
	requests := len(transport.requests)
	if err := a.TargetRaceRefresh(false); err != nil || len(transport.requests) == requests {
		t.Fatal("a stale feed must refresh again", err)
	}
	transport.failure = ""
	requests = len(transport.requests)
	_ = a.TargetRaceRefresh(true)
	if err := a.TargetRaceRefresh(false); err != nil || len(transport.requests) != requests+6 {
		t.Fatal("fresh feeds refetched", transport.requests[requests:])
	}
}

func TestRaceWireMatchesGo(t *testing.T) {
	t.Parallel()
	in := realRace(t, "2026")
	in.held.power = map[string]PowerRankingsResult{ViewSeason: testPower()}
	board, err := buildRace(in)
	if err != nil {
		t.Fatal(err)
	}
	board.Standings.Rows = board.Standings.Rows[:5]
	for i := range board.Board.Reports {
		r := &board.Board.Reports[i]
		r.Rows = r.Rows[:min(len(r.Rows), 3)]
	}
	body, err := json.MarshalIndent(struct {
		Race RaceBoard `json:"race"`
	}{board}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join("frontend", "src", "app", "race", "wire.testdata.json")
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
		t.Fatal("race wire stale; TWR_WRITE_WIRE=1 go test -run TestRaceWireMatchesGo .")
	}
}

func TestRaceBindingsGolden(t *testing.T) {
	t.Parallel()
	a := targetTestApp(t)
	board, err := a.TargetRace(0, "", "")
	if err != nil {
		t.Fatal(err)
	}
	checkGolden(t, "race", "unfetched", board)
	in := realRace(t, "2026")
	in.held.power = map[string]PowerRankingsResult{ViewSeason: testPower()}
	board, err = buildRace(in)
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
	if _, err := a.TargetRace(40, "", ""); err != nil {
		errs["week"] = err.Error()
	}
	if _, err := a.TargetRace(0, "unknown", ""); err != nil {
		errs["team"] = err.Error()
	}
	refresh, _, _, _ := seasonTestApp(t)
	if err := refresh.TargetRaceRefresh(true); err != nil {
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
func TestRaceWireTwins(t *testing.T) {
	t.Parallel()
	var b raceapp.Board
	n := 0
	fillWire(reflect.ValueOf(&b).Elem(), &n)
	if string(sourcesWire(t, b)) != string(sourcesWire(t, wireRace(b))) {
		t.Fatal("race wire twin changed")
	}
}

func TestRaceSnapshotDetachedAndConstructionUnlocked(t *testing.T) {
	t.Parallel()
	a := targetTestApp(t)
	fixture := realRace(t, "2026")
	a.race = fixture.held
	a.race.power = map[string]PowerRankingsResult{ViewSeason: testPower()}
	in := a.raceReadInput()
	in.config, in.lookup, in.hasLookup, in.zone = fixture.config, fixture.lookup, true, time.UTC
	a.core.SeasonMu.Lock()
	delete(a.race.power, ViewSeason)
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
	go func() { refreshed <- a.TargetRaceRefresh(true) }()
	select {
	case <-published:
	case <-time.After(5 * time.Second):
		t.Fatal("refresh did not publish weekly results")
	}
	read := make(chan struct{}, 1)
	readDone := make(chan error, 1)
	go func() {
		for i := 0; i < 100; i++ {
			_, err := a.TargetRace(0, "", "")
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
