package history

import (
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

// TestResultsReadCost measures results reads on a 32-team league with the score cache on: one phone
// polling (sequential) and the burst after a score change, when 20 phones re-read the same week at once.
// Opt-in: TWR_BENCH=1.
func TestResultsReadCost(t *testing.T) {
	if os.Getenv("TWR_BENCH") == "" {
		t.Skip("set TWR_BENCH=1 to measure")
	}
	f := scoreCacheLeague(t, true)
	ctx := t.Context()
	first := readResults(t, f)
	seq := testing.Benchmark(func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, err := f.s.NativeResults(ctx, bracketLeague, 2026, 1); err != nil {
				b.Fatal(err)
			}
		}
	})
	t.Logf("results.read sequential: %s %s", seq.String(), seq.MemString())
	hosted := commissioner.WithCredentialAdmission(ctx, nil) // server requests never join shareReading
	writes := 0
	burst := testing.Benchmark(func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			writes++
			changeScore(t, f, first.Games[0].Home, fmt.Sprintf("bench-%d", writes)) //nolint:contextcheck // the write uses the test context, outside the timed burst
			b.StartTimer()
			var wg sync.WaitGroup
			for range 20 {
				wg.Go(func() {
					if _, err := f.s.NativeResults(hosted, bracketLeague, 2026, 1); err != nil {
						t.Error(err)
					}
				})
			}
			wg.Wait()
		}
	})
	t.Logf("results.read burst of 20 after a write: %s per burst", burst.String())
}
