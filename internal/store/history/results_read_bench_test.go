package history

import (
	"os"
	"testing"
)

// TestResultsReadCost measures one repeated results read on a 32-team league with the score cache on, the
// shape of every phone polling the Live tab. Opt-in: TWR_BENCH=1.
func TestResultsReadCost(t *testing.T) {
	if os.Getenv("TWR_BENCH") == "" {
		t.Skip("set TWR_BENCH=1 to measure")
	}
	f := scoreCacheLeague(t, true)
	ctx := t.Context()
	if _, err := f.s.NativeResults(ctx, bracketLeague, 2026, 1); err != nil {
		t.Fatal(err)
	}
	res := testing.Benchmark(func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, err := f.s.NativeResults(ctx, bracketLeague, 2026, 1); err != nil {
				b.Fatal(err)
			}
		}
	})
	t.Logf("results.read: %s %s", res.String(), res.MemString())
}
