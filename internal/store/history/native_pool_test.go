package history

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
)

func TestNativePoolSeasonSumUsesWeeklyScorer(t *testing.T) {
	t.Parallel()
	f := scoreCacheLeague(t, true)
	records := mustRecords(t, f.s)
	// Real weekly acquisition without plays is incomplete, never a fabricated zero.
	incomplete, err := f.s.scorePoolPlayers(t.Context(), &commissioner.NativeAssets{Pool: []commissioner.NativePoolPlayer{{Player: "1001"}}}, records, f.s.nativeSources(bracketLeague, 2026), []int{1})
	if err != nil || len(incomplete) != 0 {
		t.Fatalf("incomplete=%v err=%v", incomplete, err)
	}
	// Use the existing completed-evidence fixture; the native scorer still judges every week.
	f.install()
	f.bump["franchise.01"] = map[int]leaguerules.Decimal{1: 10000, 2: 20000, 3: 40000}
	for _, weeks := range [][]int{{1}, {1, 2}, {1, 2, 3}} {
		reading := commissioner.NativeAssets{League: bracketLeague, Pool: []commissioner.NativePoolPlayer{{Player: "1001"}, {Player: "9999"}}}
		totals, err := f.s.scorePoolPlayers(t.Context(), &reading, records, f.s.nativeSources(bracketLeague, 2026), weeks)
		if err != nil {
			t.Fatal(err)
		}
		var want leaguerules.Decimal
		for _, week := range weeks {
			score, err := f.s.nativeSources(bracketLeague, 2026).PlayerScore(t.Context(), records, "1001", week)
			if err != nil {
				t.Fatal(err)
			}
			if score.Points == nil {
				t.Fatalf("week %d missing complete sample: %+v", week, score)
			}
			want += *score.Points
		}
		if got, ok := totals["1001"]; !ok || got != want {
			t.Fatalf("weeks %v: got %v held=%v, want %v", weeks, got, ok, want)
		}
		if _, ok := totals["9999"]; ok {
			t.Fatal("invented total for a player with no statistics")
		}
	}
}

type poolSchedule struct {
	*lineupEnvFake
	seasons map[int][]leagueweek.Week
}

func (s poolSchedule) ScheduleDates(_ context.Context, season int) ([]leagueweek.Week, error) {
	return s.seasons[season], nil
}

func TestNativePoolFinishedSeasonSelection(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	game := func(n int, at time.Time, final bool) leagueweek.Week {
		return leagueweek.Week{Number: n, Games: []leagueweek.Game{{Kickoff: at, Final: final}}}
	}
	for _, tc := range []struct {
		name     string
		current  []leagueweek.Week
		season   int
		finished []int
	}{
		{"in season", []leagueweek.Week{game(1, now.Add(-time.Hour), true), game(2, now.Add(time.Hour), false)}, 2026, []int{1}},
		{"offseason", []leagueweek.Week{game(1, now.Add(time.Hour), false)}, 2025, []int{1, 2}},
		{"unfinished", []leagueweek.Week{game(1, now.Add(-time.Hour), false)}, 2026, []int{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &Store{now: func() time.Time { return now }, lineupEnv: poolSchedule{lineupEnvFake: &lineupEnvFake{}, seasons: map[int][]leagueweek.Week{
				2026: tc.current, 2025: {game(1, now.Add(-time.Hour), true), game(2, now.Add(-time.Hour), true)},
			}}}
			season, weeks, err := s.poolFinishedWeeks(t.Context(), 2026)
			if err != nil || season != tc.season || !slices.Equal(weeks, tc.finished) {
				t.Fatalf("got %d %v %v", season, weeks, err)
			}
		})
	}
}

func TestNativePoolTotalsCacheAndArchiveInvalidation(t *testing.T) {
	t.Parallel()
	f := scoreCacheLeague(t, true)
	f.s.lineupEnv = poolSchedule{lineupEnvFake: &lineupEnvFake{}, seasons: map[int][]leagueweek.Week{2026: {{Number: 1, Games: []leagueweek.Game{{Kickoff: t0().Add(-time.Hour), Final: true}}}}}}
	records := mustRecords(t, f.s)
	reading := func() commissioner.NativeAssets {
		return commissioner.NativeAssets{League: bracketLeague, Snapshot: &commissioner.RawNativeSnapshot{Season: 2026}, Pool: []commissioner.NativePoolPlayer{{Player: "1001", Points: "—"}}}
	}
	first := reading()
	if err := f.s.nativePoolPoints(t.Context(), &first, records); err != nil {
		t.Fatal(err)
	}
	key := f.s.poolKey
	if key == "" {
		t.Fatal("no cache key")
	}
	// A retained total is read without re-running the incomplete-evidence scorer.
	f.s.poolPoints["1001"] = 123000
	second := reading()
	if err := f.s.nativePoolPoints(t.Context(), &second, records); err != nil {
		t.Fatal(err)
	}
	if second.Pool[0].Points != "12.3" || f.s.poolKey != key {
		t.Fatal(second.Pool)
	}
	putHeldSeasonWeeks(t, f.s, 1, t0().Add(time.Minute))
	third := reading()
	if err := f.s.nativePoolPoints(t.Context(), &third, records); err != nil {
		t.Fatal(err)
	}
	if third.Pool[0].Points != "—" || f.s.poolKey == key {
		t.Fatal("stale total survived new NFL statistics", third.Pool)
	}
}
