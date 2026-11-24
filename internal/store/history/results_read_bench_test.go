package history

import (
	"encoding/json"
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

// TestResultsStandingsReadCost serves results.read and standings.read the way hosted phones poll between
// score changes: 16 readers per CPU, the memo warm, each answer encoded to JSON as workflowJSON does. Opt-in: TWR_BENCH=1; run
// with GOMAXPROCS=2 to match the Lab.
func TestResultsStandingsReadCost(t *testing.T) {
	if os.Getenv("TWR_BENCH") == "" {
		t.Skip("set TWR_BENCH=1 to measure")
	}
	f := scoreCacheLeague(t, true)
	hosted := commissioner.WithCredentialAdmission(t.Context(), nil)
	reads := map[string]func() (any, error){
		"results.read":   func() (any, error) { return f.s.NativeResults(hosted, bracketLeague, 2026, 1) },
		"standings.read": func() (any, error) { return f.s.NativeStandings(hosted, bracketLeague, 2026) },
	}
	for name, read := range reads {
		r := testing.Benchmark(func(b *testing.B) {
			b.ReportAllocs()
			b.SetParallelism(16)
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					out, err := read()
					if err != nil {
						b.Error(err)
						return
					}
					if _, err := encodeLikeWorkflow(out); err != nil {
						b.Error(err)
						return
					}
				}
			})
		})
		t.Logf("%s parallel: %s %s", name, r.String(), r.MemString())
	}
}

// encodeLikeWorkflow encodes an answer the way commandcatalog's workflowJSON does.
func encodeLikeWorkflow(v any) ([]byte, error) {
	if encoded, ok := v.(interface{ EncodedJSON() ([]byte, error) }); ok {
		return encoded.EncodedJSON()
	}
	return json.Marshal(v)
}

// TestStandingsReadBench runs standings.read as hosted phones poll it, five times: memo warm, 16 readers per CPU,
// each answer encoded as workflowJSON does. Opt-in: TWR_BENCH=1, with GOMAXPROCS=2.
func TestStandingsReadBench(t *testing.T) {
	if os.Getenv("TWR_BENCH") == "" {
		t.Skip("set TWR_BENCH=1 to measure")
	}
	f := scoreCacheLeague(t, true)
	hosted := commissioner.WithCredentialAdmission(t.Context(), nil)
	for run := range 5 {
		r := testing.Benchmark(func(b *testing.B) {
			b.ReportAllocs()
			b.SetParallelism(16)
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					out, err := f.s.NativeStandings(hosted, bracketLeague, 2026)
					if err != nil {
						b.Error(err)
						return
					}
					if _, err := encodeLikeWorkflow(out); err != nil {
						b.Error(err)
						return
					}
				}
			})
		})
		t.Logf("standings.read run %d: %s %s", run, r.String(), r.MemString())
	}
}
