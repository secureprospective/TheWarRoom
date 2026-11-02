package history

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
)

// heldSeasonLeague is the 32-team bracket league scored from real held weekly files instead of supplied
// evidence: every franchise's starter has a gsis link and a line in weeks 1-3.
func heldSeasonLeague(t *testing.T) *bracketFixture {
	t.Helper()
	f := bracketTest(t)
	f.s.evidence = nil
	links := make([]PlayerIDLink, 0, 32)
	for n := 1; n <= 32; n++ {
		links = append(links, PlayerIDLink{IDType: "gsis", IDValue: heldSeasonGSIS(n), PlayerID: fmt.Sprintf("10%02d", n)})
	}
	if _, err := f.s.LinkPlayerIDs(t.Context(), "dynastyprocess", links); err != nil {
		t.Fatal(err)
	}
	putHeldSeasonWeeks(t, f.s, 0, t0())
	return f
}

func heldSeasonGSIS(n int) string { return fmt.Sprintf("00-%07d", 9000000+n) }

// putHeldSeasonWeeks archives one revision of the 2026 weekly file; bump changes every player's yards.
func putHeldSeasonWeeks(t *testing.T, s *Store, bump int, at time.Time) {
	t.Helper()
	body, err := os.ReadFile("../../fantasyscore/source/testdata/weeks.csv")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(bytes.NewReader(body)).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	col := map[string]int{}
	for i, name := range rows[0] {
		col[name] = i
	}
	out := [][]string{rows[0]}
	for n := 1; n <= 32; n++ {
		for week := 1; week <= 3; week++ {
			row := append([]string(nil), rows[1]...)
			row[col["player_id"]] = heldSeasonGSIS(n)
			row[col["season"]] = "2026"
			row[col["week"]] = fmt.Sprint(week)
			row[col["game_id"]] = fmt.Sprintf("2026_%02d_PIT_NYJ", week)
			row[col["passing_yards"]] = fmt.Sprint(100 + 7*n + 3*week + bump)
			out = append(out, row)
		}
	}
	var buf strings.Builder
	if err := csv.NewWriter(&buf).WriteAll(out); err != nil {
		t.Fatal(err)
	}
	feed, ok := s.reg.Feed("stats_player_week")
	if !ok {
		t.Fatal("missing weekly feed")
	}
	if err := s.Record(t.Context(), archiveFetch(feed.URLFor(2026), buf.String(), at)); err != nil {
		t.Fatal(err)
	}
}

// heldSeasonReadings is every week's results and the standings, as the screens read them.
func heldSeasonReadings(t *testing.T, s *Store) string {
	t.Helper()
	var out []any
	for week := 1; week <= 3; week++ {
		results, err := s.NativeResults(t.Context(), bracketLeague, 2026, week)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, results)
	}
	standings, err := s.NativeStandings(t.Context(), bracketLeague, 2026)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(append(out, standings))
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}

func setHeldSeasonCache(s *Store, on bool) {
	s.seasons.mu.Lock()
	s.seasons.off = !on
	s.seasons.mu.Unlock()
}

func heldSeasonBuilds(s *Store) int {
	s.seasons.mu.Lock()
	defer s.seasons.mu.Unlock()
	return s.seasons.builds
}

func scoredStarters(t *testing.T, s *Store) int {
	t.Helper()
	results, err := s.NativeResults(t.Context(), bracketLeague, 2026, 1)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, score := range results.Scores {
		encoded, err := json.Marshal(score)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(encoded), `"provisional":`) || score.Points != nil {
			n++
		}
	}
	return n
}

// Cached week results are byte-identical to results read without the cache, before and after the archive
// and the identity links change, and a changed input is never served from the old build.
func TestHeldSeasonCacheMatchesUncachedReadings(t *testing.T) {
	t.Parallel()
	f := heldSeasonLeague(t)
	if got := scoredStarters(t, f.s); got != 32 {
		t.Fatalf("%d of 32 franchises scored from held files, want all", got)
	}
	compare := func(stage string) string {
		t.Helper()
		setHeldSeasonCache(f.s, false)
		uncached := heldSeasonReadings(t, f.s)
		setHeldSeasonCache(f.s, true)
		cold := heldSeasonReadings(t, f.s)
		warm := heldSeasonReadings(t, f.s)
		if cold != uncached || warm != uncached {
			t.Fatalf("%s: cached readings differ from uncached:\nuncached %s\ncold     %s\nwarm     %s", stage, uncached, cold, warm)
		}
		return uncached
	}
	first := compare("first revision")
	builds := heldSeasonBuilds(f.s)
	heldSeasonReadings(t, f.s)
	if got := heldSeasonBuilds(f.s); got != builds {
		t.Fatalf("an unchanged season was built %d more times", got-builds)
	}
	putHeldSeasonWeeks(t, f.s, 50, t0().Add(time.Hour))
	second := compare("new weekly revision")
	if second == first {
		t.Fatal("a new weekly revision did not change the readings")
	}
	if _, err := f.s.LinkPlayerIDs(t.Context(), "dynastyprocess", []PlayerIDLink{
		{IDType: "gsis", IDValue: heldSeasonGSIS(1), PlayerID: "1002"}}); err != nil {
		t.Fatal(err)
	}
	before := heldSeasonBuilds(f.s)
	compare("relinked identity")
	if heldSeasonBuilds(f.s) == before {
		t.Fatal("a changed identity link did not rebuild the held season")
	}
}

// Concurrent readings of one revision share one build.
func TestHeldSeasonCacheBuildsOncePerRevision(t *testing.T) {
	t.Parallel()
	f := heldSeasonLeague(t)
	records, err := f.s.nativeRecords(t.Context(), bracketLeague)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 16)
	for range 16 {
		wg.Go(func() {
			src := f.s.nativeSources(bracketLeague, 2026)
			if _, err := commissioner.NativeStandingsReading(t.Context(), bracketLeague, records, src, 2026, src.now); err != nil {
				errs <- err
			}
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	if got := heldSeasonBuilds(f.s); got != 1 {
		t.Fatalf("16 concurrent readings built the season %d times, want 1", got)
	}
	if got := len(f.s.seasons.entries); got != 1 {
		t.Fatalf("%d seasons held, want the one read", got)
	}
}

func TestHeldSeasonCacheIgnoresUnrelatedArchive(t *testing.T) {
	t.Parallel()
	f := scoreCacheLeague(t, true)
	before := heldSeasonReadings(t, f.s)
	builds := heldSeasonBuilds(f.s)
	if err := f.s.Record(t.Context(), archiveFetch("https://example.test/unrelated-directory", "directory update", t0().Add(time.Hour))); err != nil {
		t.Fatal(err)
	}
	after := heldSeasonReadings(t, f.s)
	if after != before {
		t.Fatal("unrelated archive changed readings")
	}
	if heldSeasonBuilds(f.s) != builds {
		t.Fatal("unrelated archive rebuilt the season")
	}
	f.s.SetSeasonScoreCache(false)
	setHeldSeasonCache(f.s, false)
	if got := heldSeasonReadings(t, f.s); got != after {
		t.Fatal("narrowed cache differs from uncached")
	}
}

func TestHeldSeasonKeyIncludesEveryScoringFeed(t *testing.T) {
	t.Parallel()
	f := heldSeasonLeague(t)
	urls := []string{source.PlaysURL(2026), source.DepthHistoryURL(2026),
		"https://example.test/2026/export?TYPE=nflByeWeeks", "https://example.test/espn", "https://example.test/sleeper"}
	for _, name := range []string{"stats_player_week", "snap_counts", "injuries"} {
		feed, ok := f.s.reg.Feed(name)
		if !ok {
			t.Fatalf("missing feed %s", name)
		}
		urls = append(urls, feed.URLFor(2026))
	}
	for i, url := range urls {
		before, _, err := f.s.heldSeasonKey(t.Context(), 2026)
		if err != nil {
			t.Fatal(err)
		}
		if err := f.s.Record(t.Context(), archiveFetch(url, fmt.Sprint(i), t0().Add(time.Duration(i+1)*time.Hour))); err != nil {
			t.Fatal(err)
		}
		after, _, err := f.s.heldSeasonKey(t.Context(), 2026)
		if err != nil {
			t.Fatal(err)
		}
		if after == before {
			t.Fatalf("scoring dependency omitted: %s", url)
		}
	}
}
