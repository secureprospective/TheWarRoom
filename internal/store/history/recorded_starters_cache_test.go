package history

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

// A finished week's starters are built once and reused until the TTL; a new key or expiry rebuilds.
func TestRecordedStartersCacheReusesFinishedWeeks(t *testing.T) {
	var c recordedStartersCache
	builds := 0
	at := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)
	read := func(key string, now time.Time) commissioner.NativeResults {
		out := commissioner.NativeResults{Scores: []commissioner.FranchiseWeekScore{
			{Franchise: "0001", RecordedResult: "W"}, {Franchise: "0002"},
		}}
		c.apply(t.Context(), key, now, &out, func() {
			builds++
			out.Scores[0].Starters = []commissioner.StarterScore{{Player: "13604"}}
		})
		return out
	}
	first := read("wk3", at)
	again := read("wk3", at.Add(time.Minute))
	if builds != 1 {
		t.Fatalf("builds = %d, want 1 (second read reuses)", builds)
	}
	if len(again.Scores[0].Starters) != 1 || again.Scores[0].Starters[0].Player != "13604" {
		t.Fatalf("reused starters = %+v", again.Scores[0].Starters)
	}
	if again.Scores[1].Starters != nil {
		t.Fatalf("a franchise without a recorded result got starters")
	}
	again.Scores[0].Starters[0].Player = "changed"
	if read("wk3", at.Add(2*time.Minute)).Scores[0].Starters[0].Player != "13604" || first.Scores[0].Starters[0].Player != "13604" {
		t.Fatalf("callers share starters memory")
	}
	read("wk4", at)
	read("wk3", at.Add(recordedStartersTTL+time.Second))
	if builds != 3 {
		t.Fatalf("builds = %d, want 3 (new key and expiry rebuild)", builds)
	}
}

func TestRecordedStartersCacheSkipsUnrecordedAndCancelled(t *testing.T) {
	var c recordedStartersCache
	at := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)

	recorded := commissioner.NativeResults{Scores: []commissioner.FranchiseWeekScore{
		{Franchise: "0001", RecordedResult: "W"},
	}}
	finishedBuilds := 0
	c.apply(t.Context(), "wk1", at, &recorded, func() {
		finishedBuilds++
		recorded.Scores[0].Starters = []commissioner.StarterScore{{Player: "13604"}}
	})

	// Unrecorded (live/future) weeks must not write entries or evict cached finished weeks.
	liveBuilds := 0
	for i := range 300 {
		live := commissioner.NativeResults{Scores: []commissioner.FranchiseWeekScore{
			{Franchise: "0001"}, {Franchise: "0002"},
		}}
		c.apply(t.Context(), fmt.Sprintf("live-%d", i), at, &live, func() {
			liveBuilds++
		})
	}
	if liveBuilds != 0 {
		t.Fatalf("unrecorded weeks ran build %d times, want 0", liveBuilds)
	}
	hits, misses := c.stats()
	if hits != 0 || misses != 1 {
		t.Fatalf("unrecorded weeks mutated stats: hits=%d misses=%d, want 0/1", hits, misses)
	}

	// The finished week cached earlier must still hit without rebuilding.
	recheck := commissioner.NativeResults{Scores: []commissioner.FranchiseWeekScore{
		{Franchise: "0001", RecordedResult: "W"},
	}}
	c.apply(t.Context(), "wk1", at.Add(time.Minute), &recheck, func() {
		finishedBuilds++
	})
	if finishedBuilds != 1 || len(recheck.Scores[0].Starters) != 1 {
		t.Fatalf("finished week evicted by unrecorded reads: builds=%d starters=%+v",
			finishedBuilds, recheck.Scores[0].Starters)
	}

	// A build whose context ends early must not cache partial or empty starters.
	cancelCtx, cancel := context.WithCancel(t.Context())
	partial := commissioner.NativeResults{Scores: []commissioner.FranchiseWeekScore{
		{Franchise: "0001", RecordedResult: "W"},
	}}
	c.apply(cancelCtx, "wk2", at, &partial, func() {
		cancel()
	})
	full := commissioner.NativeResults{Scores: []commissioner.FranchiseWeekScore{
		{Franchise: "0001", RecordedResult: "W"},
	}}
	rebuilt := 0
	c.apply(t.Context(), "wk2", at.Add(time.Second), &full, func() {
		rebuilt++
		full.Scores[0].Starters = []commissioner.StarterScore{{Player: "14100"}}
	})
	if rebuilt != 1 || len(full.Scores[0].Starters) != 1 || full.Scores[0].Starters[0].Player != "14100" {
		t.Fatalf("cancelled build was cached: rebuilt=%d starters=%+v", rebuilt, full.Scores[0].Starters)
	}
}

func TestRecordedStartersCacheDeduplicatesInFlight(t *testing.T) {
	var c recordedStartersCache
	at := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)
	started := make(chan struct{})
	release := make(chan struct{})
	builds := 0
	out1 := commissioner.NativeResults{Scores: []commissioner.FranchiseWeekScore{
		{Franchise: "0001", RecordedResult: "W"},
	}}
	out2 := commissioner.NativeResults{Scores: []commissioner.FranchiseWeekScore{
		{Franchise: "0001", RecordedResult: "W"},
	}}
	var wg sync.WaitGroup
	wg.Go(func() {
		c.apply(t.Context(), "wk5", at, &out1, func() {
			builds++
			close(started)
			<-release
			out1.Scores[0].Starters = []commissioner.StarterScore{{Player: "15000"}}
		})
	})
	<-started
	wg.Go(func() {
		c.apply(t.Context(), "wk5", at, &out2, func() {
			builds++
		})
	})
	time.Sleep(15 * time.Millisecond)
	close(release)
	wg.Wait()

	if builds != 1 {
		t.Fatalf("concurrent apply ran build %d times, want 1", builds)
	}
	if len(out2.Scores[0].Starters) != 1 || out2.Scores[0].Starters[0].Player != "15000" {
		t.Fatalf("waiter starters = %+v, want 15000", out2.Scores[0].Starters)
	}
	hits, misses := c.stats()
	if hits != 1 || misses != 1 {
		t.Fatalf("stats = hits:%d misses:%d, want 1/1", hits, misses)
	}
}
