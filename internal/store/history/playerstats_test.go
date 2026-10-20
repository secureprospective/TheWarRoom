package history

import (
	"bytes"
	"context"
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/measures"
)

func statStore(t *testing.T, path string) (*Store, *db.Pools) {
	t.Helper()
	reg, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	pools, err := db.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := pools.Close(); err != nil {
			t.Error(err)
		}
	})
	s := New(pools, reg)
	s.SetClock(t0)
	if err := s.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	return s, pools
}

func statBody(t *testing.T, yards string, include bool) string {
	t.Helper()
	body, err := os.ReadFile("../../fantasyscore/source/testdata/weeks.csv")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(bytes.NewReader(body)).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	rows = rows[:3]
	for i, column := range rows[0] {
		if column == "passing_yards" {
			rows[1][i] = yards
		}
	}
	if !include {
		rows = [][]string{rows[0], rows[2]}
	}
	var out strings.Builder
	w := csv.NewWriter(&out)
	if err := w.WriteAll(rows); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func putStatBody(t *testing.T, s *Store, body, suffix string, at time.Time) string {
	t.Helper()
	feed, ok := s.reg.Feed("stats_player_week")
	if !ok {
		t.Fatal("missing weekly feed")
	}
	f := archiveFetch(feed.URLFor(2025)+suffix, body, at)
	if err := s.Record(context.Background(), f); err != nil {
		t.Fatal(err)
	}
	return f.SHA256
}

func readStatPort(t *testing.T, s *Store) []source.HeldWeek {
	t.Helper()
	var port source.StatPort = s
	weeks, err := port.PlayerWeekStats(context.Background(), source.RawStatScope{
		GSIS: "00-0023459", Season: 2025, Week: 1})
	if err != nil {
		t.Fatal(err)
	}
	return weeks
}

func TestPlayerWeekStatsArchiveRevisions(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "history.db")
	s, pools := statStore(t, path)
	if weeks := readStatPort(t, s); len(weeks) != 0 {
		t.Fatalf("invented statistics before the first held file: %+v", weeks)
	}
	old := putStatBody(t, s, statBody(t, "244", true), "", t0())
	latestBody := statBody(t, "245", true)
	latest := putStatBody(t, s, latestBody, "", t0().Add(time.Hour))
	putStatBody(t, s, latestBody, "", t0().Add(2*time.Hour))
	putStatBody(t, s, statBody(t, "999", true), ".backup", t0().Add(3*time.Hour))
	for pass := range 2 {
		weeks := readStatPort(t, s)
		if len(weeks) != 2 || weeks[0].SHA256 != latest || weeks[1].SHA256 != old {
			t.Fatalf("pass %d: revisions not distinct and newest first: %+v", pass, weeks)
		}
		for i, yards := range []leaguerules.Decimal{2450000, 2440000} {
			w := weeks[i]
			if w.Week == nil || w.Week.Position != "QB" || w.Week.Line.Quantities["passing_yards"] != yards {
				t.Fatalf("pass %d: revision %d lost exact stats or feed position: %+v", pass, i, w)
			}
			if len(w.Missing) != 1 || !strings.Contains(w.Missing[0], "play-by-play") {
				t.Fatalf("incomplete NFL evidence presented as complete: %+v", w)
			}
			if _, ok := w.Week.Line.Events["long_pass"]; ok {
				t.Fatal("missing play distances were fabricated as zero")
			}
		}
		if !weeks[0].FetchedAt.Equal(t0().Add(2 * time.Hour)) {
			t.Fatalf("identical refresh did not retain latest held time: %+v", weeks[0])
		}
		if pass == 0 {
			if err := pools.Close(); err != nil {
				t.Fatal(err)
			}
			s, _ = statStore(t, path)
		}
	}
}

func TestPlayerWeekStatsMissingDoesNotReuseOldLine(t *testing.T) {
	t.Parallel()
	s, _ := statStore(t, filepath.Join(t.TempDir(), "history.db"))
	putStatBody(t, s, statBody(t, "244", true), "", t0())
	latest := putStatBody(t, s, statBody(t, "0", false), "", t0().Add(time.Hour))
	weeks := readStatPort(t, s)
	if len(weeks) != 2 || weeks[0].SHA256 != latest || weeks[0].Week != nil || len(weeks[0].Missing) != 2 {
		t.Fatalf("missing latest player was replaced by an older line or zero: %+v", weeks)
	}
}

func TestPlayerWeekStatsRejectsMalformedHeldFile(t *testing.T) {
	t.Parallel()
	s, _ := statStore(t, filepath.Join(t.TempDir(), "history.db"))
	putStatBody(t, s, statBody(t, "244", true), "", t0())
	putStatBody(t, s, statBody(t, "not a number", true), "", t0().Add(time.Hour))
	weeks, err := s.PlayerWeekStats(context.Background(), source.RawStatScope{
		GSIS: "00-0023459", Season: 2025, Week: 1})
	if err == nil || len(weeks) != 0 || !strings.Contains(err.Error(), "passing_yards") {
		t.Fatalf("bad newest file silently became old or zero statistics: %+v, %v", weeks, err)
	}
}

func TestPlayerWeekStatsScope(t *testing.T) {
	t.Parallel()
	s, _ := statStore(t, filepath.Join(t.TempDir(), "history.db"))
	for _, scope := range []source.RawStatScope{
		{}, {GSIS: "00-0023459", Season: 2020, Week: 1},
		{GSIS: "00-0023459", Season: 2025, Week: 0},
		{GSIS: "00-0023459", Season: 2025, Week: 23},
		{GSIS: "00-002345x", Season: 2025, Week: 1},
		{GSIS: "99-0023459", Season: 2025, Week: 1},
	} {
		if _, err := s.PlayerWeekStats(context.Background(), scope); err == nil {
			t.Fatalf("accepted invalid or unloaded scope: %+v", scope)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.PlayerWeekStats(ctx, source.RawStatScope{
		GSIS: "00-0023459", Season: 2025, Week: 1}); err == nil {
		t.Fatal("ignored cancelled reading")
	}
}

func TestWeeklyFileDecodedOnceByHash(t *testing.T) {
	t.Parallel()
	s, _ := statStore(t, filepath.Join(t.TempDir(), "history.db"))
	body := statBody(t, "244", true)
	sha := putStatBody(t, s, body, "", t0())
	first := readStatPort(t, s)
	first[0].Week.Line.Quantities["passing_yards"] = 999
	weeks, err := s.decodedWeeks(sha, []byte("not CSV"))
	if err != nil || len(weeks) != 2 {
		t.Fatalf("cached body was parsed again: %d, %v", len(weeks), err)
	}
	again := readStatPort(t, s)
	if again[0].Week.Line.Quantities["passing_yards"] != 2440000 {
		t.Fatal("caller mutated the cached statistics")
	}
	putStatBody(t, s, statBody(t, "245", true), "", t0().Add(time.Hour))
	if got := readStatPort(t, s); len(got) != 2 || got[0].Week.Line.Quantities["passing_yards"] != 2450000 {
		t.Fatalf("new hash did not invalidate statistics: %+v", got)
	}
}

func TestExactArchiveLookupExcludesSimilarURLs(t *testing.T) {
	t.Parallel()
	s, _ := statStore(t, filepath.Join(t.TempDir(), "history.db"))
	putStatBody(t, s, statBody(t, "244", true), "", t0())
	putStatBody(t, s, statBody(t, "999", true), ".backup", t0().Add(time.Hour))
	feed, _ := s.reg.Feed("stats_player_week")
	calls := 0
	found, err := s.ExactArchivedBodies(t.Context(), feed.URLFor(2025), func(url string, _ []byte, _ time.Time) bool {
		calls++
		if url != feed.URLFor(2025) {
			t.Errorf("substring URL admitted: %s", url)
		}
		return false
	})
	if err != nil || found || calls != 1 {
		t.Fatalf("exact archive: %d calls, found %v, %v", calls, found, err)
	}
}
