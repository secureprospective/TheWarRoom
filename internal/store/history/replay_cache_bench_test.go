//go:build leaguebench

package history

import (
	"maps"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

// Rule-only traffic isolates replay length, not a full-season scoring workload.
func BenchmarkReplayCache100k(b *testing.B) {
	path := filepath.Join(b.TempDir(), "replay.db")
	s, pools := openBigLeague(b, path)
	seedLeagueWrite(b, s, 100_000)
	if err := pools.Close(); err != nil {
		b.Fatal(err)
	}
	s = nil
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	start := time.Now()
	s, pools = openBigLeague(b, path)
	records, err := s.nativeRecords(b.Context(), bigLeagueID)
	if err != nil {
		b.Fatal(err)
	}
	restart := time.Since(start)
	want := commissioner.Replay(commissioner.WithoutClosedBooks(records))
	got := commissioner.Replay(records)
	if len(got.Active) != 100_000 || got.Name != want.Name || !maps.Equal(got.Active, want.Active) {
		b.Fatal("cold replay differs")
	}
	records = nil
	runtime.GC()
	runtime.ReadMemStats(&after)
	retained := int64(after.HeapAlloc) - int64(before.HeapAlloc)
	b.ReportAllocs()
	b.ResetTimer()
	samples := make([]time.Duration, 0, b.N)
	for b.Loop() {
		start := time.Now()
		r, err := s.nativeRecords(b.Context(), bigLeagueID)
		if err != nil {
			b.Fatal(err)
		}
		if len(commissioner.Replay(r).Active) != 100_000 {
			b.Fatal("wrong warm revision")
		}
		samples = append(samples, time.Since(start))
	}
	b.StopTimer()
	slices.Sort(samples)
	b.ReportMetric(float64(samples[(len(samples)-1)*95/100].Nanoseconds()), "p95-ns/read")
	b.ReportMetric(float64(restart.Nanoseconds()), "restart-ns")
	b.ReportMetric(float64(retained), "retained-B")
	runtime.KeepAlive(pools)
}
