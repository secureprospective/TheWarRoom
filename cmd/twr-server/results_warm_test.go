package main

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
)

// The startup warm-up reads standings, odds, bracket and weekly results for each pinned league
// without rebuilding on subsequent reads, and its loop stops promptly when the server stops.
func TestResultsWarmReadsPinnedLeaguesAndStops(t *testing.T) {
	ctx := t.Context()
	h := newTwoFileHost(t)
	digest, _ := importFootball(t, h.files, "Warm")
	signIns := signInStore(h.files)
	if err := recordDirectory(ctx, hostedLeague(t, h.files), signIns); err != nil {
		t.Fatal(err)
	}
	pins := newLeaguePins(h.files.leagues)
	if err := pins.selectPins(ctx, "", nil); err != nil {
		t.Fatal(err)
	}
	if err := pins.setPin(ctx, signIns, fixtureLeague, digest); err != nil {
		t.Fatal(err)
	}
	warmResults(ctx, pins, nil)
	pin := pins.pin(ctx, fixtureLeague)
	season := pin.Football.Season()
	before := pin.Store.WarmCacheStats()
	if before.StarterMisses == 0 {
		t.Fatalf("expected warm-up to populate starter cache, got %+v", before)
	}
	if _, err := pin.Store.NativeResults(ctx, fixtureLeague, season, 1); err != nil {
		t.Fatalf("results after warm-up: %v", err)
	}
	if _, err := pin.Store.NativeStandings(ctx, fixtureLeague, season); err != nil {
		t.Fatalf("standings after warm-up: %v", err)
	}
	if _, err := pin.Store.NativeOdds(
		ctx, fixtureLeague, season, leagueservice.HostedOddsValues(),
	); err != nil {
		t.Fatalf("odds after warm-up: %v", err)
	}
	if _, err := pin.Store.NativeBracket(ctx, fixtureLeague, season); err != nil {
		t.Fatalf("bracket after warm-up: %v", err)
	}
	after := pin.Store.WarmCacheStats()
	if after.StarterMisses != before.StarterMisses || after.StarterHits <= before.StarterHits {
		t.Fatalf("results rebuilt after warm-up: before=%+v after=%+v", before, after)
	}
	if after.ScoreMisses != before.ScoreMisses {
		t.Fatalf("scores rebuilt after warm-up: before=%+v after=%+v", before, after)
	}
	if after.OddsMisses != before.OddsMisses {
		t.Fatalf("odds rebuilt after warm-up: before=%+v after=%+v", before, after)
	}
	stop := startResultsWarm(ctx, pins)
	stopped := make(chan struct{})
	go func() { stop(); close(stopped) }()
	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("warm-up loop did not stop")
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
	var resultsErrors, standingsErrors int
	for _, line := range captured {
		if strings.Contains(line, brokenLeague+" results") {
			resultsErrors++
		}
		if strings.Contains(line, brokenLeague+" standings") {
			standingsErrors++
		}
	}
	if standingsErrors != 1 {
		t.Fatalf("expected 1 standings warm-up error log for %s, got %d: %v",
			brokenLeague, standingsErrors, captured)
	}
	if resultsErrors != 1 {
		t.Fatalf("expected 1 results warm-up error log for %s (not per-week spam), got %d: %v",
			brokenLeague, resultsErrors, captured)
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
