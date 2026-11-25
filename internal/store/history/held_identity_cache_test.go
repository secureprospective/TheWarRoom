package history

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/measures"
)

func gzipBody(t *testing.T, body string) string {
	t.Helper()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write([]byte(body)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func TestHeldIdentityKeyInvalidatesOnOutsideIDWrite(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	f := scoreCacheLeague(t, true)
	// Archive play-by-play so weekly player scores are complete and count in standings.
	plays := linePlayHeader + "2026_01_PIT_NYJ,1,2026,1,REG,,,,NA,NA,NA,0,,,,0,0,0,0,0,PIT,pass,0,0,NA,0,1,10,0,0,0,0,NA,NA,1,0,NA\n"
	if err := f.s.Record(ctx, archiveFetch(source.PlaysURL(2026), gzipBody(t, plays), t0())); err != nil {
		t.Fatal(err)
	}

	resultsBefore := readResults(t, f)
	standingsBefore, err := f.s.NativeStandings(ctx, bracketLeague, 2026)
	if err != nil {
		t.Fatal(err)
	}
	if standingsBefore.Decided == 0 {
		t.Fatalf("expected decided games in standings before write: %s", standingsBefore.Words)
	}
	keyBefore, _, err := f.s.heldSeasonKey(ctx, 2026)
	if err != nil {
		t.Fatal(err)
	}
	buildsAfterWarm := f.s.seasons.identity.builds
	if buildsAfterWarm != 1 {
		t.Fatalf("identity builds after warm reads = %d, want 1", buildsAfterWarm)
	}

	// Steady-state heavy reads reuse the cached identity digest without rescanning player_ids.
	if got := readResults(t, f); !reflect.DeepEqual(got.Scores, resultsBefore.Scores) {
		t.Fatal("steady-state results read drifted")
	}
	if _, err := f.s.NativeStandings(ctx, bracketLeague, 2026); err != nil {
		t.Fatal(err)
	}
	if f.s.seasons.identity.builds != buildsAfterWarm {
		t.Fatalf("steady-state reads rebuilt identity cache: %d -> %d", buildsAfterWarm, f.s.seasons.identity.builds)
	}

	// Reassign player 1001's gsis id to player 1032: 1001 loses its held stat line and 1032's MIN(gsis)
	// switches to the lower-scoring player, flipping both results and standings.
	if _, err := f.s.LinkPlayerIDs(ctx, "dynastyprocess", []PlayerIDLink{
		{IDType: "gsis", IDValue: heldSeasonGSIS(1), PlayerID: "1032"},
	}); err != nil {
		t.Fatal(err)
	}

	keyAfter, _, err := f.s.heldSeasonKey(ctx, 2026)
	if err != nil {
		t.Fatal(err)
	}
	if keyAfter.identity == keyBefore.identity {
		t.Fatal("heldSeasonKey kept the old identity digest after a gsis write")
	}
	resultsAfter := readResults(t, f)
	if reflect.DeepEqual(resultsAfter.Scores, resultsBefore.Scores) {
		t.Fatal("results.read after a gsis write still served the old scores")
	}
	standingsAfter, err := f.s.NativeStandings(ctx, bracketLeague, 2026)
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(standingsAfter, standingsBefore) {
		t.Fatal("standings.read after a gsis write still served the old standings")
	}

	// A pfr-only link write must also move the identity digest on the very next call.
	if _, err := f.s.LinkPlayerIDs(ctx, "dynastyprocess", []PlayerIDLink{
		{IDType: "pfr", IDValue: "TestPfr00", PlayerID: "1001"},
	}); err != nil {
		t.Fatal(err)
	}
	keyAfterPFR, _, err := f.s.heldSeasonKey(ctx, 2026)
	if err != nil {
		t.Fatal(err)
	}
	if keyAfterPFR.identity == keyAfter.identity {
		t.Fatal("heldSeasonKey kept the old identity digest after a pfr write")
	}
}

func splitScoreCacheLeague(t *testing.T) (*bracketFixture, *Store) {
	t.Helper()
	ctx := t.Context()
	dir := t.TempDir()
	reg, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}

	refPools, err := db.Open(ctx, filepath.Join(dir, "reference.db"), ReferenceSchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = refPools.Close() })
	if err := PrepareReferenceFile(ctx, refPools); err != nil {
		t.Fatal(err)
	}
	refStore := New(refPools, reg)
	refStore.SetClock(t0)

	leaguePools, err := db.OpenWithReference(ctx, filepath.Join(dir, "league.db"), LeagueSchemaVersion(), refPools)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = leaguePools.Close() })
	if err := CreateLeagueFile(ctx, leaguePools, LeagueFile{League: bracketLeague}); err != nil {
		t.Fatal(err)
	}
	leagueStore := New(leaguePools, reg)
	leagueStore.SetClock(t0)
	if err := leagueStore.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	leagueStore.SetNativeLineups(&fixedLineups{})
	leagueStore.SetSeasonScoreCache(true)

	seedSplitScoreCacheLeague(t, leagueStore, refStore)
	return &bracketFixture{s: leagueStore, pools: leaguePools}, refStore
}

func seedSplitScoreCacheLeague(t *testing.T, leagueStore, refStore *Store) {
	t.Helper()
	runScoreCommand(t, leagueStore, importedLegacy(0), "creator")
	runScoreCommand(t, leagueStore, bracketSeedCommand(t), "creator")
	runScoreCommand(t, leagueStore, ruleCommand("schedule.regularWeeks", "1", "now", "Choose the season length", 2), "creator")
	runScoreCommand(t, leagueStore, ruleCommand("lineup.invalidPenalty", `"none"`, "now", "Score one-player lineups as they are", 3), "creator")
	raw := commissioner.RawCommand{League: bracketLeague, Command: commissioner.ScheduleGenerate,
		Args: commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{Weeks: 3}}, Expected: 4, Request: "generate",
		Effective: "now", Visibility: "league"}
	runScoreCommand(t, leagueStore, raw, "creator")
	raw.Command, raw.Expected, raw.Request = commissioner.SchedulePublish, 5, "publish"
	raw.Args = commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{}}
	runScoreCommand(t, leagueStore, raw, "creator")

	links := make([]PlayerIDLink, 0, 32)
	for n := 1; n <= 32; n++ {
		links = append(links, PlayerIDLink{IDType: "gsis", IDValue: heldSeasonGSIS(n), PlayerID: fmt.Sprintf("10%02d", n)})
	}
	if _, err := refStore.LinkPlayerIDs(t.Context(), "dynastyprocess", links); err != nil {
		t.Fatal(err)
	}
	putHeldSeasonWeeks(t, refStore, 0, t0())
}

func TestHeldIdentityKeyInvalidatesAcrossSeparateReferenceStore(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	f, refStore := splitScoreCacheLeague(t)
	leagueStore := f.s
	before := readResults(t, f)
	keyBefore, _, err := leagueStore.heldSeasonKey(ctx, 2026)
	if err != nil {
		t.Fatal(err)
	}
	if leagueStore.seasons.identity.builds != 1 {
		t.Fatalf("identity builds = %d, want 1", leagueStore.seasons.identity.builds)
	}

	// A league write commits only to league.db, so reference.db's data_version stays put and the
	// identity cache does not rebuild.
	changeScore(t, f, before.Games[0].Home, "split-league-write")
	assertHostedResultsBurstChanged(t, f, before)
	if leagueStore.seasons.identity.builds != 1 {
		t.Fatalf("league write rebuilt reference identity cache: builds=%d, want 1", leagueStore.seasons.identity.builds)
	}

	afterLeagueWrite := readResults(t, f)

	// An outside-id write from the separate reference store (as keepLiveIdentities does) must
	// invalidate leagueStore on the very next read.
	if _, err := refStore.LinkPlayerIDs(ctx, "dynastyprocess", []PlayerIDLink{
		{IDType: "gsis", IDValue: heldSeasonGSIS(1), PlayerID: "1032"},
	}); err != nil {
		t.Fatal(err)
	}
	keyAfter, _, err := leagueStore.heldSeasonKey(ctx, 2026)
	if err != nil {
		t.Fatal(err)
	}
	if keyAfter.identity == keyBefore.identity {
		t.Fatal("league store kept stale identity digest after separate reference store wrote gsis")
	}
	if leagueStore.seasons.identity.builds != 2 {
		t.Fatalf("identity builds after reference write = %d, want 2", leagueStore.seasons.identity.builds)
	}
	afterIDWrite := readResults(t, f)
	if reflect.DeepEqual(afterIDWrite.Scores, afterLeagueWrite.Scores) {
		t.Fatal("league store results.read served old scores after separate reference store wrote gsis")
	}
}

func assertHostedResultsBurstChanged(t *testing.T, f *bracketFixture, before commissioner.NativeResults) {
	t.Helper()
	hosted := commissioner.WithCredentialAdmission(t.Context(), nil)
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			after, err := f.s.NativeResults(hosted, bracketLeague, 2026, 1)
			if err != nil {
				t.Error(err)
				return
			}
			if after.Revision == before.Revision || reflect.DeepEqual(after.Scores, before.Scores) {
				t.Error("hosted results after a score write kept the old scores or revision")
			}
		})
	}
	wg.Wait()
}
