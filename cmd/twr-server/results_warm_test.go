package main

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leagueschedule"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/weeklyresults"
	"github.com/secureprospective/TheWarRoom/internal/leagueload"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/migrationtest"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// seedWarmLeague imports a synthetic 8-team league with a 2-week regular season (week 1 recorded
// with lineups for all 8 franchises, week 2 open post-cutover) and pins matching football evidence
// so NativeStandings, NativeOdds, NativeBracket and NativeResults all compute real results.
func seedWarmLeague(t *testing.T, files stateFiles) (string, string, int) {
	t.Helper()
	ctx := t.Context()
	export := migrationtest.Synthetic("Warm", migrationtest.Specs()["8 teams"])
	var err error
	export.Schedule, err = leagueschedule.Parse([]byte(`{"schedule":{"weeklySchedule":[` +
		`{"week":"1","matchup":[` +
		`{"franchise":[{"id":"0001","isHome":"1","score":"110.50"},` +
		`{"id":"0002","isHome":"0","score":"98.00"}]},` +
		`{"franchise":[{"id":"0003","isHome":"1","score":"104.25"},` +
		`{"id":"0004","isHome":"0","score":"91.00"}]},` +
		`{"franchise":[{"id":"0005","isHome":"1","score":"112.00"},` +
		`{"id":"0006","isHome":"0","score":"95.50"}]},` +
		`{"franchise":[{"id":"0007","isHome":"1","score":"101.75"},` +
		`{"id":"0008","isHome":"0","score":"89.25"}]}]},` +
		`{"week":"2","matchup":[` +
		`{"franchise":[{"id":"0001","isHome":"1","score":""},{"id":"0003","isHome":"0","score":""}]},` +
		`{"franchise":[{"id":"0002","isHome":"1","score":""},{"id":"0004","isHome":"0","score":""}]},` +
		`{"franchise":[{"id":"0005","isHome":"1","score":""},{"id":"0007","isHome":"0","score":""}]},` +
		`{"franchise":[{"id":"0006","isHome":"1","score":""},{"id":"0008","isHome":"0","score":""}]}]}]}}`))
	if err != nil {
		t.Fatal(err)
	}
	export.Results, err = weeklyresults.Parse([]byte(`{"allWeeklyResults":{"weeklyResults":[` +
		`{"week":"1","matchup":[` +
		`{"regularSeason":"1","franchise":[{"id":"0001","isHome":"1","score":"110.50","result":"W"},` +
		`{"id":"0002","isHome":"0","score":"98.00","result":"L"}]},` +
		`{"regularSeason":"1","franchise":[{"id":"0003","isHome":"1","score":"104.25","result":"W"},` +
		`{"id":"0004","isHome":"0","score":"91.00","result":"L"}]},` +
		`{"regularSeason":"1","franchise":[{"id":"0005","isHome":"1","score":"112.00","result":"W"},` +
		`{"id":"0006","isHome":"0","score":"95.50","result":"L"}]},` +
		`{"regularSeason":"1","franchise":[{"id":"0007","isHome":"1","score":"101.75","result":"W"},` +
		`{"id":"0008","isHome":"0","score":"89.25","result":"L"}]}]},` +
		`{"week":"2","matchup":[` +
		`{"regularSeason":"1","franchise":[{"id":"0001","isHome":"1","score":""},` +
		`{"id":"0003","isHome":"0","score":""}]},` +
		`{"regularSeason":"1","franchise":[{"id":"0002","isHome":"1","score":""},` +
		`{"id":"0004","isHome":"0","score":""}]},` +
		`{"regularSeason":"1","franchise":[{"id":"0005","isHome":"1","score":""},` +
		`{"id":"0007","isHome":"0","score":""}]},` +
		`{"regularSeason":"1","franchise":[{"id":"0006","isHome":"1","score":""},` +
		`{"id":"0008","isHome":"0","score":""}]}]}]}}`))
	if err != nil {
		t.Fatal(err)
	}
	league := leaguemigration.AppLeagueID(export.LeagueID, export.Season)
	store := leagueOn(t, files, league)
	svc := &leaguemigration.Service{
		Now: time.Now,
		Source: func(string) (leaguemigration.Source, error) {
			return &migrationtest.Source{Export: export}, nil
		},
		Ledger: leagueservice.MigrationLedger{Store: store},
	}
	if _, err := svc.Import(ctx, export.LeagueID, "Seed warm league", "creator"); err != nil {
		t.Fatalf("import warm league: %v", err)
	}
	records, err := store.HeldCommissionerRecords(ctx, league)
	if err != nil {
		t.Fatal(err)
	}
	rev := int64(len(records))
	for _, rule := range []struct{ field, value string }{
		{"schedule.regularWeeks", "2"},
		{"playoffs.perConference", "4"},
		{"playoffs.divisionWinners", "2"},
		{"playoffs.wildCards", "2"},
		{"playoffs.byeSeed", "0"},
		{"playoffs.seeding", `"division_winners_first"`},
		{"playoffs.reseed", "false"},
		{"playoffs.homeBonus", "0"},
		{"playoffs.startWeek", "3"},
		{"playoffs.finalWeek", "4"},
	} {
		raw := commissioner.RawCommand{
			League: league, Command: commissioner.RulesSet,
			Args: commissioner.RawArgs{
				Field: rule.field, Value: json.RawMessage(rule.value),
			},
			Expected:  rev,
			Reason:    "Configure two-week season and playoffs for warm test",
			Request:   "warm-rule-" + rule.field,
			Effective: "now", Visibility: "league",
		}
		if _, err := store.ExecuteCommissioner(ctx, raw, "creator", nil); err != nil {
			t.Fatalf("set %s: %v", rule.field, err)
		}
		rev++
	}
	savePreset := commissioner.RawCommand{
		League: league, Command: commissioner.BracketPresetSave,
		Args: commissioner.RawArgs{Bracket: &commissioner.RawBracketChange{
			Season: export.Season,
			Name:   "four_team",
			Shape: []commissioner.BracketNodeSpec{
				{ID: "semi_1", Week: 3, HomeSeed: 1, AwaySeed: 4},
				{ID: "semi_2", Week: 3, HomeSeed: 2, AwaySeed: 3},
				{ID: "final", Week: 4, HomeFrom: "winner of semi_1", AwayFrom: "winner of semi_2"},
			},
		}},
		Expected: rev, Reason: "Save four-team bracket shape", Request: "warm-bracket-save",
		Effective: "now", Visibility: "league",
	}
	if _, err := store.ExecuteCommissioner(ctx, savePreset, "creator", nil); err != nil {
		t.Fatalf("save bracket preset: %v", err)
	}
	rev++
	buildBracket := commissioner.RawCommand{
		League: league, Command: commissioner.BracketBuild,
		Args: commissioner.RawArgs{Bracket: &commissioner.RawBracketChange{
			Season: export.Season,
			Preset: "four_team",
		}},
		Expected:  rev,
		Reason:    "Use four-team bracket shape",
		Request:   "warm-bracket-build",
		Effective: "now", Visibility: "league",
	}
	if _, err := store.ExecuteCommissioner(ctx, buildBracket, "creator", nil); err != nil {
		t.Fatalf("build bracket: %v", err)
	}
	list := make([]players.RawPlayer, 0, len(export.Players))
	for id := range export.Players {
		list = append(list, players.RawPlayer{
			ID: id, Name: "Player " + id, Position: "WR", Team: "DAL", DraftYear: "2020",
		})
	}
	at := time.Date(export.Season, 9, 1, 0, 0, 0, 0, time.UTC)
	weeks := []leagueweek.Week{
		{Number: 1, Games: []leagueweek.Game{{Teams: [2]string{"DAL", "NYG"}, Kickoff: at.Add(24 * time.Hour)}}},
		{Number: 2, Games: []leagueweek.Game{{Teams: [2]string{"DAL", "NYG"}, Kickoff: at.Add(192 * time.Hour)}}},
		{Number: 3, Games: []leagueweek.Game{{Teams: [2]string{"DAL", "NYG"}, Kickoff: at.Add(360 * time.Hour)}}},
		{Number: 4, Games: []leagueweek.Game{{Teams: [2]string{"DAL", "NYG"}, Kickoff: at.Add(528 * time.Hour)}}},
	}
	body, digest, err := leagueload.BuildFootball(list, weeks, export.Season, at)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.ImportFootball(ctx, body, digest); err != nil {
		t.Fatal(err)
	}
	if err := recordDirectory(ctx, store, signInStore(files)); err != nil {
		t.Fatal(err)
	}
	return league, digest, export.Season
}

// The startup warm-up reads standings, odds, bracket and weekly results for each pinned league
// without rebuilding on subsequent reads, and its loop stops promptly when the server stops.
func TestResultsWarmReadsPinnedLeaguesAndStops(t *testing.T) {
	ctx := t.Context()
	h := newTwoFileHost(t)
	league, digest, season := seedWarmLeague(t, h.files)
	signIns := signInStore(h.files)
	pins := newLeaguePins(h.files.leagues)
	if err := pins.selectPins(ctx, "", nil); err != nil {
		t.Fatal(err)
	}
	if err := pins.setPin(ctx, signIns, league, digest); err != nil {
		t.Fatal(err)
	}
	warmResults(ctx, pins, nil)
	pin := pins.pin(ctx, league)
	before := pin.Store.WarmCacheStats()
	if before.StarterMisses == 0 {
		t.Fatalf("expected warm-up to populate starter cache, got %+v", before)
	}
	if before.ScoreMisses == 0 {
		t.Fatalf("expected warm-up to populate score cache, got %+v", before)
	}
	if before.OddsMisses != 1 {
		t.Fatalf("expected warm-up to populate odds cache once, got %+v", before)
	}
	if _, err := pin.Store.NativeResults(ctx, league, season, 1); err != nil {
		t.Fatalf("results after warm-up: %v", err)
	}
	standings, err := pin.Store.NativeStandings(ctx, league, season)
	if err != nil {
		t.Fatalf("standings after warm-up: %v", err)
	}
	if standings.Status != "Ranked" {
		t.Fatalf("standings status = %q, want Ranked", standings.Status)
	}
	odds, err := pin.Store.NativeOdds(ctx, league, season, leagueservice.HostedOddsValues())
	if err != nil {
		t.Fatalf("odds after warm-up: %v", err)
	}
	if odds.Runs != commissioner.OddsRuns {
		t.Fatalf("odds runs = %d, want %d (words=%q)", odds.Runs, commissioner.OddsRuns, odds.OddsWords)
	}
	if _, err := pin.Store.NativeBracket(ctx, league, season); err != nil {
		t.Fatalf("bracket after warm-up: %v", err)
	}
	after := pin.Store.WarmCacheStats()
	if after.StarterMisses != before.StarterMisses || after.StarterHits <= before.StarterHits {
		t.Fatalf("results rebuilt after warm-up: before=%+v after=%+v", before, after)
	}
	if after.ScoreMisses != before.ScoreMisses || after.ScoreHits <= before.ScoreHits {
		t.Fatalf("scores rebuilt after warm-up: before=%+v after=%+v", before, after)
	}
	if after.OddsMisses != before.OddsMisses || after.OddsHits != 1 {
		t.Fatalf("odds rebuilt after warm-up: before=%+v after=%+v", before, after)
	}
	ready := make(chan struct{})
	close(ready)
	stop := startResultsWarm(ctx, pins, ready)
	stopped := make(chan struct{})
	go func() { stop(); close(stopped) }()
	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("warm-up loop did not stop")
	}
}

// Startup warm-up waits for live checkpoint restoration before its first pass so caches are
// keyed under the restored LiveOverlay, and stops promptly if cancelled while waiting.
func TestResultsWarmWaitsForLiveRestore(t *testing.T) {
	ctx := t.Context()
	h := newTwoFileHost(t)
	league, digest, season := seedWarmLeague(t, h.files)
	signIns := signInStore(h.files)
	pins := newLeaguePins(h.files.leagues)
	if err := pins.selectPins(ctx, "", nil); err != nil {
		t.Fatal(err)
	}
	if err := pins.setPin(ctx, signIns, league, digest); err != nil {
		t.Fatal(err)
	}

	ready := make(chan struct{})
	warmed := make(chan struct{}, 1)
	stop := startResultsWarmWithLog(ctx, pins, ready, func(format string, args ...any) {
		if strings.Contains(fmt.Sprintf(format, args...), "server startup warm-up:") {
			select {
			case warmed <- struct{}{}:
			default:
			}
		}
	})
	defer stop()

	select {
	case <-warmed:
		t.Fatal("warm-up ran before live restore completed")
	case <-time.After(60 * time.Millisecond):
	}
	pin := pins.pin(ctx, league)
	if stats := pin.Store.WarmCacheStats(); stats != (history.WarmCacheStats{}) {
		t.Fatalf("cache touched before live restore: %+v", stats)
	}

	// Simulate live checkpoint restoration populating pins.live before signalling ready.
	gen := history.LiveGeneration{
		ID:        "restored-week-2",
		Season:    season,
		Week:      2,
		Authority: source.ESPNLive,
		At:        time.Now().UTC(),
		Sources:   map[string]time.Time{"https://example.invalid/board": time.Now().UTC()},
		Lines: map[string]source.PlayerWeek{
			"p1": {Season: season, Week: 2, Position: "QB", Line: fantasyscore.StatLine{
				Quantities: map[leaguerules.StatID]leaguerules.Decimal{"pass_yd": 100000},
			}},
		},
		Identities: map[string]history.LiveIdentity{"p1": {GSIS: "00-0000001"}},
	}
	if err := pins.live.Publish(gen); err != nil {
		t.Fatalf("publish restored live generation: %v", err)
	}
	close(ready)

	select {
	case <-warmed:
	case <-time.After(5 * time.Second):
		t.Fatal("warm-up did not run after live restore signalled ready")
	}
	before := pin.Store.WarmCacheStats()
	if _, err := pin.Store.NativeStandings(ctx, league, season); err != nil {
		t.Fatal(err)
	}
	if _, err := pin.Store.NativeOdds(ctx, league, season, leagueservice.HostedOddsValues()); err != nil {
		t.Fatal(err)
	}
	after := pin.Store.WarmCacheStats()
	if after.ScoreMisses != before.ScoreMisses || after.OddsMisses != before.OddsMisses || after.OddsHits != 1 {
		t.Fatalf("reads after restored warm-up missed cache: before=%+v after=%+v", before, after)
	}

	// Cancelling while still waiting on ready exits promptly.
	blocked := make(chan struct{})
	stopBlocked := startResultsWarm(ctx, pins, blocked)
	done := make(chan struct{})
	go func() { stopBlocked(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("warm-up loop did not stop while waiting on live restore")
	}
}

// A warm-up error for one league is logged once with context and error, and never prevents the
// server from warming and serving other leagues.
func TestStartupWarmLogsFailureAndKeepsServing(t *testing.T) {
	ctx := t.Context()
	h := newTwoFileHost(t)
	digest, _ := importFootball(t, h.files, "WarmErr")
	signIns := signInStore(h.files)
	healthy := hostedLeague(t, h.files)
	brokenLeague := "native.broken"
	broken := leagueOn(t, h.files, brokenLeague)
	if err := recordDirectory(ctx, healthy, signIns); err != nil {
		t.Fatal(err)
	}
	if err := recordDirectory(ctx, broken, signIns); err != nil {
		t.Fatal(err)
	}
	pins := newLeaguePins(h.files.leagues)
	if err := pins.selectPins(ctx, digest, nil); err != nil {
		t.Fatal(err)
	}
	brokenPools, ok := h.files.leagues.pools(brokenLeague)
	if !ok {
		t.Fatalf("missing pools for %s", brokenLeague)
	}
	if err := brokenPools.Close(); err != nil {
		t.Fatal(err)
	}

	var (
		mu   sync.Mutex
		logs []string
	)
	logf := func(format string, args ...any) {
		mu.Lock()
		logs = append(logs, fmt.Sprintf(format, args...))
		mu.Unlock()
	}
	warmResults(ctx, pins, logf)

	mu.Lock()
	captured := append([]string{}, logs...)
	mu.Unlock()
	var resultsErrors, standingsErrors, oddsErrors, bracketErrors int
	for _, line := range captured {
		if strings.Contains(line, brokenLeague+" results") {
			resultsErrors++
		}
		if strings.Contains(line, brokenLeague+" standings") {
			standingsErrors++
		}
		if strings.Contains(line, brokenLeague+" odds") {
			oddsErrors++
		}
		if strings.Contains(line, brokenLeague+" bracket") {
			bracketErrors++
		}
	}
	if standingsErrors != 1 || oddsErrors != 1 || bracketErrors != 1 || resultsErrors != 1 {
		t.Fatalf("expected 1 error log each for standings/odds/bracket/results, got %d/%d/%d/%d: %v",
			standingsErrors, oddsErrors, bracketErrors, resultsErrors, captured)
	}

	pin := pins.pin(ctx, fixtureLeague)
	season := pin.Football.Season()
	if _, err := pin.Store.NativeResults(ctx, fixtureLeague, season, 1); err != nil {
		t.Fatalf("healthy league results failed after broken warm-up: %v", err)
	}
	if _, err := pin.Store.NativeStandings(ctx, fixtureLeague, season); err != nil {
		t.Fatalf("healthy league standings failed after broken warm-up: %v", err)
	}
	if _, err := pin.Store.NativeOdds(
		ctx, fixtureLeague, season, leagueservice.HostedOddsValues(),
	); err != nil {
		t.Fatalf("healthy league odds failed after broken warm-up: %v", err)
	}
	if _, err := pin.Store.NativeBracket(ctx, fixtureLeague, season); err != nil {
		t.Fatalf("healthy league bracket failed after broken warm-up: %v", err)
	}
}

// Warm-up warms standings, odds and bracket across every pinned league before the 18-week results
// pass, so a second league never waits behind the first league's full weekly results scan.
func TestResultsWarmHeadlineBeforeAllWeeks(t *testing.T) {
	ctx := t.Context()
	h := newTwoFileHost(t)
	digest, _ := importFootball(t, h.files, "WarmOrder")
	signIns := signInStore(h.files)
	first := leagueOn(t, h.files, "native.alpha")
	second := leagueOn(t, h.files, "native.beta")
	if err := recordDirectory(ctx, first, signIns); err != nil {
		t.Fatal(err)
	}
	if err := recordDirectory(ctx, second, signIns); err != nil {
		t.Fatal(err)
	}
	pins := newLeaguePins(h.files.leagues)
	if err := pins.selectPins(ctx, digest, nil); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"native.alpha", "native.beta"} {
		pools, ok := h.files.leagues.pools(id)
		if !ok {
			t.Fatalf("missing pools for %s", id)
		}
		if err := pools.Close(); err != nil {
			t.Fatal(err)
		}
	}
	var logs []string
	warmResults(ctx, pins, func(format string, args ...any) {
		logs = append(logs, fmt.Sprintf(format, args...))
	})
	betaStandings := slices.IndexFunc(logs, func(s string) bool {
		return strings.Contains(s, "native.beta standings")
	})
	alphaResults := slices.IndexFunc(logs, func(s string) bool {
		return strings.Contains(s, "native.alpha results")
	})
	if betaStandings < 0 || alphaResults < 0 || betaStandings > alphaResults {
		t.Fatalf("expected all leagues' standings before 18-week results: betaStandings=%d alphaResults=%d logs=%v",
			betaStandings, alphaResults, logs)
	}
}
