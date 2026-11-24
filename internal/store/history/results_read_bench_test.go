package history

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

// benchDirectoryPlayers is how many synthetic players each carry a gsis and a pfr link in the
// benchmark fixture: 110,000 outside ids, matching the ~125k directory DynastyProcess loads on the Lab.
const benchDirectoryPlayers = 55000

// benchScoreCacheLeague is scoreCacheLeague with a realistic player directory so benchmarks see
// the real cost of reading outside ids.
func benchScoreCacheLeague(t *testing.T) *bracketFixture {
	t.Helper()
	f := scoreCacheLeague(t, true)
	seedBenchDirectory(t, f.s, benchDirectoryPlayers)
	return f
}

func seedBenchDirectory(t *testing.T, s *Store, players int) {
	t.Helper()
	ctx := t.Context()
	tx, err := s.pools.ReferencePools().BeginWrite(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()
	const seedSQL = `
WITH RECURSIVE seq(n) AS (
	SELECT 1 UNION ALL SELECT n + 1 FROM seq WHERE n < ?
)
INSERT INTO canonical_player(id) SELECT printf('player.%032d', n) FROM seq;
WITH RECURSIVE seq(n) AS (
	SELECT 1 UNION ALL SELECT n + 1 FROM seq WHERE n < ?
)
INSERT INTO player_external_id(provider, value, player)
SELECT 'mfl', printf('%06d', 200000 + n), printf('player.%032d', n) FROM seq
UNION ALL
SELECT 'gsis', printf('00-%07d', 200000 + n), printf('player.%032d', n) FROM seq
UNION ALL
SELECT 'pfr', printf('Bench%05d', n), printf('player.%032d', n) FROM seq;`
	if _, err := tx.ExecContext(ctx, seedSQL, players, players); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
}

// TestResultsReadCost measures results reads on a 32-team league with the score cache on: one phone
// polling (sequential) and the burst after a score change, when 20 phones re-read the same week at once.
// Opt-in: TWR_BENCH=1.
func TestResultsReadCost(t *testing.T) {
	if os.Getenv("TWR_BENCH") == "" {
		t.Skip("set TWR_BENCH=1 to measure")
	}
	f := benchScoreCacheLeague(t)
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
	f := benchScoreCacheLeague(t)
	hosted := commissioner.WithCredentialAdmission(t.Context(), nil)
	reads := []struct {
		name string
		read func() (any, error)
	}{
		{"results.read", func() (any, error) { return f.s.NativeResults(hosted, bracketLeague, 2026, 1) }},
		{"standings.read", func() (any, error) { return f.s.NativeStandings(hosted, bracketLeague, 2026) }},
	}
	for _, item := range reads {
		if _, err := item.read(); err != nil {
			t.Fatal(err)
		}
		seq := testing.Benchmark(func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				out, err := item.read()
				if err != nil {
					b.Fatal(err)
				}
				if _, err := encodeLikeWorkflow(out); err != nil {
					b.Fatal(err)
				}
			}
		})
		t.Logf("%s sequential: %s %s", item.name, seq.String(), seq.MemString())
		par := testing.Benchmark(func(b *testing.B) {
			b.ReportAllocs()
			b.SetParallelism(16)
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					out, err := item.read()
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
		t.Logf("%s parallel: %s %s", item.name, par.String(), par.MemString())
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
