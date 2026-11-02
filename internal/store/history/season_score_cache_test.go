package history

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// fixedLineups is the results fixture's lineups, held fixed for the test so the season score cache may
// share weeks scored from them.
type fixedLineups struct{ resultLineups }

func (fixedLineups) lineupsFollowLedger() {}

func scoreCacheLeague(t *testing.T, on bool) *bracketFixture {
	t.Helper()
	f := heldSeasonLeague(t)
	f.s.SetNativeLineups(&fixedLineups{})
	f.s.SetSeasonScoreCache(on)
	return f
}

// seasonReadings is every reading that scores the season, as the bytes a screen would receive.
func seasonReadings(t *testing.T, s *Store) map[string]string {
	t.Helper()
	ctx := t.Context()
	out := map[string]string{}
	put := func(name string, v any, err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		body, jerr := json.Marshal(v)
		if jerr != nil {
			t.Fatal(jerr)
		}
		out[name] = string(body)
	}
	standings, err := s.NativeStandings(ctx, bracketLeague, 2026)
	put("standings", standings, err)
	odds, err := s.NativeOdds(ctx, bracketLeague, 2026, commissioner.OddsValues{})
	put("odds", odds, err)
	bracket, err := s.NativeBracket(ctx, bracketLeague, 2026)
	put("bracket", bracket, err)
	for week := 1; week <= 17; week++ {
		results, err := s.NativeResults(ctx, bracketLeague, 2026, week)
		put("results", results, err)
		out["results"+string(rune('a'+week))] = out["results"]
	}
	final, err := commissioner.FinalStandingsReading(ctx, bracketLeague, mustRecords(t, s), s.nativeSources(bracketLeague, 2026),
		2026, time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC))
	put("final", final, err)
	draft, err := s.NativeDraft(ctx, commissioner.RawNativeDraftScope{League: bracketLeague, Franchise: franchiseID(1), Season: 2027, At: draftStart()}, time.UTC, nil)
	put("draft", draft, err)
	return out
}

func mustRecords(t *testing.T, s *Store) []commissioner.Record {
	t.Helper()
	records, err := s.nativeRecords(t.Context(), bracketLeague)
	if err != nil {
		t.Fatal(err)
	}
	return records
}

// The cache changes no reading: every season reading is byte-identical with and without it, read cold and warm.
func TestSeasonScoreCacheReadingsAreByteIdentical(t *testing.T) {
	t.Parallel()
	want := seasonReadings(t, scoreCacheLeague(t, false).s)
	f := scoreCacheLeague(t, true)
	for pass := range 2 {
		got := seasonReadings(t, f.s)
		for name, body := range want {
			if got[name] != body {
				t.Fatalf("pass %d: %s differs with the cache\nwant %s\ngot  %s", pass, name, body, got[name])
			}
		}
	}
	if hits, _ := f.s.seasonScoreCounts(); hits == 0 {
		t.Fatal("the second pass scored every week again")
	}
}

// Each week is scored once per revision: a second standings reading and an odds reading score nothing new.
func TestSeasonScoreCacheScoresEachWeekOnce(t *testing.T) {
	t.Parallel()
	f := scoreCacheLeague(t, true)
	if _, err := f.s.NativeStandings(t.Context(), bracketLeague, 2026); err != nil {
		t.Fatal(err)
	}
	_, misses := f.s.seasonScoreCounts()
	if _, err := f.s.NativeOdds(t.Context(), bracketLeague, 2026, commissioner.OddsValues{}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.NativeStandings(t.Context(), bracketLeague, 2026); err != nil {
		t.Fatal(err)
	}
	if _, again := f.s.seasonScoreCounts(); again != misses {
		t.Fatalf("odds and a second standings scored %d weeks again", again-misses)
	}
}

// A score handed out is the caller's own: changing it does not change the next reading.
func TestSeasonScoreCacheCallerCannotChangeAnother(t *testing.T) {
	t.Parallel()
	f := scoreCacheLeague(t, true)
	if _, err := f.s.NativeResults(t.Context(), bracketLeague, 2026, 1); err != nil {
		t.Fatal(err)
	}
	// Both readings below are served from the cache, so the change is made to a cached score's copy.
	first, err := f.s.NativeResults(t.Context(), bracketLeague, 2026, 1)
	if err != nil || len(first.Scores) == 0 || len(first.Scores[0].Starters) == 0 || first.Scores[0].Provisional == nil {
		t.Fatalf("results: %+v, %v", first, err)
	}
	want, _ := json.Marshal(first)
	*first.Scores[0].Provisional += 99
	first.Scores[0].Starters[0].Name = "changed"
	first.Scores[0].Changes = append(first.Scores[0].Changes, commissioner.ScoreChange{})
	first.Scores[0].Lineup.Words = "changed"
	hits, _ := f.s.seasonScoreCounts()
	second, err := f.s.NativeResults(t.Context(), bracketLeague, 2026, 1)
	if err != nil {
		t.Fatal(err)
	}
	if again, _ := f.s.seasonScoreCounts(); again == hits {
		t.Fatal("the second reading did not come from the cache")
	}
	if got, _ := json.Marshal(second); string(got) != string(want) {
		t.Fatalf("one caller changed the score another read\nwant %s\ngot  %s", want, got)
	}
}

// A live line is a new revision of its week: the cached week is not served, and the reading matches one
// taken without the cache.
func TestSeasonScoreCacheFollowsLiveStats(t *testing.T) {
	t.Parallel()
	on, off := scoreCacheLeague(t, true), scoreCacheLeague(t, false)
	before, err := on.s.NativeResults(t.Context(), bracketLeague, 2026, 1)
	if err != nil {
		t.Fatal(err)
	}
	player := before.Scores[0].Starters[0].Player
	_, misses := on.s.seasonScoreCounts()
	at := time.Date(2026, 9, 10, 20, 0, 0, 0, time.UTC)
	for _, f := range []*bracketFixture{on, off} {
		setLiveStats(t, f.s, 2026, 1, at, source.ESPNLive, nil, map[string]source.PlayerWeek{player: liveLine(990000)})
	}
	got, err := on.s.NativeResults(t.Context(), bracketLeague, 2026, 1)
	if err != nil {
		t.Fatal(err)
	}
	want, err := off.s.NativeResults(t.Context(), bracketLeague, 2026, 1)
	if err != nil {
		t.Fatal(err)
	}
	gb, _ := json.Marshal(got)
	wb, _ := json.Marshal(want)
	if string(gb) != string(wb) {
		t.Fatalf("cached week after a live line\nwant %s\ngot  %s", wb, gb)
	}
	if _, after := on.s.seasonScoreCounts(); after == misses {
		t.Fatal("the week was served from the cache after its live lines changed")
	}
}

// A trial ledger (a proposed change) is never cached.
func TestSeasonScoreCacheSkipsTrialLedgers(t *testing.T) {
	t.Parallel()
	f := scoreCacheLeague(t, true)
	records := mustRecords(t, f.s)
	trial := slices.Clone(records)
	trial = append(trial, commissioner.Record{Receipt: commissioner.Receipt{Entry: "proposed.x"}})
	if _, ok := f.s.durableLedgerDigest(t.Context(), bracketLeague, trial); ok {
		t.Fatal("a proposed entry was taken as durable")
	}
	if _, ok := f.s.durableLedgerDigest(t.Context(), bracketLeague, records); !ok {
		t.Fatal("the durable ledger was refused")
	}
}

func TestSeasonScoreCacheJoinsConcurrentWeeks(t *testing.T) {
	t.Parallel()
	f := scoreCacheLeague(t, true)
	records := mustRecords(t, f.s)
	var calls atomic.Int32
	var wg sync.WaitGroup
	for range 12 {
		wg.Go(func() {
			src := f.s.nativeSources(bracketLeague, 2026)
			_, err := src.MemoWeekScore(t.Context(), records, leaguerules.LineupRules{},
				commissioner.NativeDirectorySeat{}, 2026, 1, func() (commissioner.FranchiseWeekScore, error) {
					calls.Add(1)
					time.Sleep(50 * time.Millisecond)
					return commissioner.FranchiseWeekScore{Franchise: "one"}, nil
				})
			if err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatalf("scored one week %d times", calls.Load())
	}
}

func TestSeasonScoreCacheRevisionsAndArchive(t *testing.T) {
	t.Parallel()
	f := scoreCacheLeague(t, true)
	_ = heldSeasonReadings(t, f.s)
	_, misses := f.s.seasonScoreCounts()
	runScoreCommand(t, f.s, ruleCommand("lineup.invalidPenalty", `"zero_top_starter"`, "now", "Change the penalty", 6), "creator")
	after := heldSeasonReadings(t, f.s)
	if _, n := f.s.seasonScoreCounts(); n == misses {
		t.Fatal("new ledger revision reused old scores")
	}
	f.s.SetSeasonScoreCache(false)
	if want := heldSeasonReadings(t, f.s); want != after {
		t.Fatal("new revision differs from uncached")
	}
	f.s.SetSeasonScoreCache(true)
	before := heldSeasonReadings(t, f.s)
	putHeldSeasonWeeks(t, f.s, 50, t0().Add(time.Hour))
	after = heldSeasonReadings(t, f.s)
	if after == before {
		t.Fatal("archive update did not change scores")
	}
	f.s.SetSeasonScoreCache(false)
	if want := heldSeasonReadings(t, f.s); want != after {
		t.Fatal("new archive differs from uncached")
	}
}

func TestSeasonScoreCacheBoundsRevisions(t *testing.T) {
	t.Parallel()
	var c seasonScoreCache
	for i := range keepScoreRevisions * 3 {
		c.put(fmt.Sprint(i), weekScoreKey{}, commissioner.FranchiseWeekScore{})
	}
	if len(c.revisions) != keepScoreRevisions || len(c.order) != keepScoreRevisions {
		t.Fatalf("unbounded cache: %d revisions, %d order", len(c.revisions), len(c.order))
	}
	for i := range keepScoreRevisions * 2 {
		if _, ok := c.get(fmt.Sprint(i), weekScoreKey{}); ok {
			t.Fatal("old revision retained")
		}
	}
}

func TestSeasonScoreCacheWaitCanCancel(t *testing.T) {
	t.Parallel()
	f := scoreCacheLeague(t, true)
	records := mustRecords(t, f.s)
	src := f.s.nativeSources(bracketLeague, 2026)
	started, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	seat := commissioner.NativeDirectorySeat{}
	go func() {
		defer close(done)
		_, err := src.MemoWeekScore(t.Context(), records, leaguerules.LineupRules{}, seat, 2026, 1,
			func() (commissioner.FranchiseWeekScore, error) {
				close(started)
				<-release
				return commissioner.FranchiseWeekScore{}, nil
			})
		if err != nil {
			t.Error(err)
		}
	}()
	<-started
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := f.s.nativeSources(bracketLeague, 2026).MemoWeekScore(ctx, records, leaguerules.LineupRules{}, seat, 2026, 1,
		func() (commissioner.FranchiseWeekScore, error) {
			t.Error("cancelled waiter scored")
			return commissioner.FranchiseWeekScore{}, nil
		})
	close(release)
	<-done
	if err == nil {
		t.Fatal("cancelled waiter succeeded")
	}
}

type scoreFactsCounter struct {
	LineupEnvironment
	calls int
}

func (e *scoreFactsCounter) PlayerFacts() nativecheck.Facts {
	e.calls++
	return nativecheck.Facts{"1001": {Name: "Held name", Position: "LB"}}
}

func TestSeasonScoreCacheUsesOneFactsSnapshot(t *testing.T) {
	t.Parallel()
	f := scoreCacheLeague(t, true)
	env := &scoreFactsCounter{}
	f.s.SetLineupEnvironment(env)
	n := f.s.nativeSources(bracketLeague, 2026)
	records := mustRecords(t, f.s)
	_, ok := n.scoreRevision(t.Context(), records)
	if !ok {
		t.Fatal("fixture revision not cached")
	}
	for range 3 {
		got := f.s.withScoringPosition(source.NativeEvidence{}, "1001", n)
		if got.ListedPosition != "LB" {
			t.Fatal("lost the scoring position")
		}
	}
	if env.calls != 1 {
		t.Fatalf("built player facts %d times in one reading", env.calls)
	}
}

func TestSeasonScoreCacheDoesNotRequireADatabaseForTransfer(t *testing.T) {
	t.Parallel()
	s := &Store{lineups: ledgerLineups{}, now: t0}
	src := s.nativeSources(bracketLeague, 2026)
	records := []commissioner.Record{{Receipt: commissioner.Receipt{Entry: "transferred"}}}
	called := false
	got, err := src.MemoWeekScore(t.Context(), records, leaguerules.LineupRules{}, commissioner.NativeDirectorySeat{}, 2026, 1,
		func() (commissioner.FranchiseWeekScore, error) {
			called = true
			return commissioner.FranchiseWeekScore{Name: "transferred"}, nil
		})
	if err != nil || !called || got.Name != "transferred" {
		t.Fatalf("database-free scoring: %+v, %v", got, err)
	}
}
