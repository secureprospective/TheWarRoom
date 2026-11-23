package commissioner

import (
	"encoding/json"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/engine/competition"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func TestOddsInputSeedIgnoresLedgerRevisionButTracksSimulationInputs(t *testing.T) {
	standings := NativeStandings{Season: 2026, Revision: 12, Through: 8,
		teams:    []competition.RankTeam{{ID: "east", Conference: "A"}},
		criteria: []leaguerules.Criterion{leaguerules.WinPercentage},
		counted: seasonGames{through: 8, games: []competition.RankGame{{Home: "east", Away: "west", HomePoints: 100, AwayPoints: 90}},
			allPlay: map[string]competition.Tally{"east": {W: 1}}, points: map[string]leaguerules.Decimal{"east": 100}}}
	field := competition.FieldInput{Order: []string{"east"}, Teams: map[string]competition.FieldTeam{"east": {Conference: "A"}}}
	expected := map[string]float64{"east": 100}
	seed := oddsInputSeed(standings, expected, field)

	// A lineup-only ledger write changes the public reading revision, not any simulation input.
	standings.Revision++
	if got := oddsInputSeed(standings, expected, field); got != seed {
		t.Fatalf("lineup-only revision changed simulation seed: %d != %d", got, seed)
	}
	standings.counted.games[0].HomePoints++
	if got := oddsInputSeed(standings, expected, field); got == seed {
		t.Fatal("changed result retained simulation seed")
	}
}

func TestNativeOddsSeedIsSerializedAsDecimalString(t *testing.T) {
	body, err := json.Marshal(NativeOdds{Seed: ^uint64(0)})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `"seed":"18446744073709551615"`) {
		t.Fatalf("seed must cross JSON as an exact decimal string: %s", body)
	}
}

func TestOddsCacheReturnsIdenticalSimulationForSameInputs(t *testing.T) {
	standings := NativeStandings{Season: 2026, Through: 1,
		teams: []competition.RankTeam{{ID: "east", Division: "east", Conference: "A"},
			{ID: "west", Division: "west", Conference: "A"}},
		criteria: []leaguerules.Criterion{leaguerules.WinPercentage},
		counted: seasonGames{games: []competition.RankGame{{Home: "east", Away: "west", HomePoints: 100, AwayPoints: 90}},
			points:  map[string]leaguerules.Decimal{"east": 100, "west": 90},
			allPlay: map[string]competition.Tally{"east": {W: 1}, "west": {L: 1}},
			open:    []OpenGame{{Week: 2, Home: "east", Away: "west"}}}}
	field := competition.FieldInput{Order: []string{"east", "west"},
		Teams: map[string]competition.FieldTeam{"east": {Division: "east", Conference: "A"},
			"west": {Division: "west", Conference: "A"}}, Conferences: []string{"A"},
		Rules: competition.PlayoffRules{PerConference: 2, DivisionWinners: 1, WildCards: 1}}
	expected := map[string]float64{"east": 100, "west": 90}
	seed := oddsInputSeed(standings, expected, field)
	before, words := playOutChunks(standings, expected, field, seed)
	if words != "" {
		t.Fatal(words)
	}
	var cache OddsCache
	key := oddsInputKey(standings, expected, field)
	cached, words := cache.result(key, func() (map[string]int, string) {
		return playOutChunks(standings, expected, field, seed)
	})
	if words != "" || !reflect.DeepEqual(cached, before) {
		t.Fatalf("cached simulation differs: got %v, want %v (%s)", cached, before, words)
	}
}

func TestOddsCacheReturnsCopiesAndDoesNotCacheFailures(t *testing.T) {
	var cache OddsCache
	key := [32]byte{1}
	calls := 0
	run := func() (map[string]int, string) {
		calls++
		return map[string]int{"east": 4}, ""
	}
	first, words := cache.result(key, run)
	if words != "" || first["east"] != 4 {
		t.Fatalf("first result = %v, %q", first, words)
	}
	first["east"] = 9
	second, words := cache.result(key, run)
	if words != "" || second["east"] != 4 || calls != 1 {
		t.Fatalf("cached result = %v, %q after %d runs", second, words, calls)
	}
	failed := [32]byte{2}
	_, words = cache.result(failed, func() (map[string]int, string) { return nil, "not ready" })
	if words != "not ready" {
		t.Fatalf("failure words = %q", words)
	}
	_, _ = cache.result(failed, func() (map[string]int, string) { calls++; return map[string]int{"east": 1}, "" })
	if calls != 2 {
		t.Fatalf("failed result was cached: calls=%d", calls)
	}
}

func TestOddsCacheDeduplicatesInFlight(t *testing.T) {
	var cache OddsCache
	key := [32]byte{7}
	started := make(chan struct{})
	release := make(chan struct{})
	calls := 0
	var (
		wg         sync.WaitGroup
		got1, got2 map[string]int
		w1, w2     string
	)
	wg.Go(func() {
		got1, w1 = cache.result(key, func() (map[string]int, string) {
			calls++
			close(started)
			<-release
			return map[string]int{"east": 1200, "west": 800}, ""
		})
	})
	<-started
	wg.Go(func() {
		got2, w2 = cache.result(key, func() (map[string]int, string) {
			calls++
			return map[string]int{"east": 1}, ""
		})
	})
	time.Sleep(15 * time.Millisecond)
	close(release)
	wg.Wait()

	if calls != 1 {
		t.Fatalf("concurrent result ran simulation %d times, want 1", calls)
	}
	if w1 != "" || w2 != "" || got1["east"] != 1200 || got2["east"] != 1200 {
		t.Fatalf("got1=%v (%q) got2=%v (%q)", got1, w1, got2, w2)
	}
	got1["east"] = 0
	if got2["east"] != 1200 {
		t.Fatal("in-flight callers share map memory")
	}
	hits, misses := cache.Stats()
	if hits != 1 || misses != 1 {
		t.Fatalf("stats = hits:%d misses:%d, want 1/1", hits, misses)
	}
}
