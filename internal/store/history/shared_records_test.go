package history

import (
	"context"
	"reflect"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/deepclone"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
)

// ledgerView is the part of a held record a reading may see: everything but the private memos.
type ledgerView struct {
	Receipt   commissioner.Receipt
	Command   commissioner.RawCommand
	Principal string
	Canonical string
}

func ledgerViews(records []commissioner.Record) []ledgerView {
	out := make([]ledgerView, len(records))
	for i, r := range records {
		out[i] = deepclone.Of(ledgerView{r.Receipt, r.Command, r.Principal, r.Canonical})
	}
	return out
}

// Readings share one snapshot per revision. A caller that rewrites, appends to or reorders its slice
// must not change what the next caller reads.
func TestSharedRecordsCallerCannotChangeAnother(t *testing.T) {
	t.Parallel()
	f := heldSeasonLeague(t)
	first, err := f.s.nativeRecords(t.Context(), bracketLeague)
	if err != nil || len(first) < 2 {
		t.Fatalf("records: %d, %v", len(first), err)
	}
	want := ledgerViews(first)
	first[0] = commissioner.Record{}
	first[1].Principal = "changed"
	slices.Reverse(first)
	_ = append(first[:1], commissioner.Record{})
	second, err := f.s.nativeRecords(t.Context(), bracketLeague)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ledgerViews(second), want) {
		t.Fatal("one caller changed the records another caller read")
	}
}

// A caller writing its slice during concurrent reads must not change the cached revision or race readers.
func TestSharedRecordsConcurrentReadersAndWriterAreIsolated(t *testing.T) {
	t.Parallel()
	f := heldSeasonLeague(t)
	baseline, err := f.s.nativeRecords(t.Context(), bracketLeague)
	if err != nil || len(baseline) < 2 {
		t.Fatalf("records: %d, %v", len(baseline), err)
	}
	want := ledgerViews(baseline)
	const readers, iterations = 4, 12
	start := make(chan struct{})
	var wg sync.WaitGroup
	for range readers {
		wg.Go(func() {
			<-start
			for range iterations {
				records, err := f.s.nativeRecords(t.Context(), bracketLeague)
				if err != nil {
					t.Error(err)
					return
				}
				if !reflect.DeepEqual(ledgerViews(records), want) {
					t.Error("concurrent caller saw another caller's slice write")
					return
				}
			}
		})
	}
	wg.Go(func() {
		<-start
		for range iterations {
			records, err := f.s.nativeRecords(t.Context(), bracketLeague)
			if err != nil {
				t.Error(err)
				return
			}
			records[0] = commissioner.Record{}
			records[1].Principal = "changed"
			slices.Reverse(records)
			_ = append(records[:1], commissioner.Record{})
		}
	})
	close(start)
	wg.Wait()
	got, err := f.s.nativeRecords(t.Context(), bracketLeague)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ledgerViews(got), want) {
		t.Fatal("concurrent caller changed the cached ledger revision")
	}
}

// The shared snapshot is read-only by contract: every scoring reading must leave it exactly as it was.
func TestSharedRecordsStayUnchangedByReadings(t *testing.T) {
	t.Parallel()
	f := heldSeasonLeague(t)
	if _, err := f.s.nativeRecords(t.Context(), bracketLeague); err != nil {
		t.Fatal(err)
	}
	f.s.recordsMu.Lock()
	held := f.s.records[bracketLeague].records
	f.s.recordsMu.Unlock()
	want := ledgerViews(held)
	if _, err := f.s.NativeStandings(t.Context(), bracketLeague, 2026); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.NativeResults(t.Context(), bracketLeague, 2026, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.NativeBracket(t.Context(), bracketLeague, 2026); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ledgerViews(held), want) {
		t.Fatal("a reading wrote into the shared ledger snapshot")
	}
}

// Joiners of one flight each get their own copy of its result.
func TestJoinedReadingsCannotChangeEachOther(t *testing.T) {
	t.Parallel()
	var flights readingFlights
	release, started := make(chan struct{}), make(chan struct{})
	var once sync.Once
	read := func(_ context.Context) (map[string][]int, error) {
		once.Do(func() { close(started) })
		<-release
		return map[string][]int{"a": {1, 2}}, nil
	}
	const n = 6
	got := make([]map[string][]int, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			v, err := shareReading(t.Context(), &flights, "NativeStandings", "native.x|2026", func(c context.Context) (map[string][]int, error) { return read(c) })
			if err != nil {
				t.Error(err)
			}
			got[i] = v
		})
	}
	<-started
	waitFor(t, func() bool { return flightWaiters(&flights) == n })
	close(release)
	wg.Wait()
	got[0]["a"][0] = 99
	got[0]["b"] = nil
	for i := 1; i < n; i++ {
		if !reflect.DeepEqual(got[i], map[string][]int{"a": {1, 2}}) {
			t.Fatalf("caller %d sees another caller's write: %v", i, got[i])
		}
	}
}

func TestLedgerWriteDoesNotJoinPreviousReadingFlight(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		read func(context.Context, *Store) (int64, error)
	}{
		{"NativeResults", func(ctx context.Context, s *Store) (int64, error) {
			out, err := s.NativeResults(ctx, bracketLeague, 2026, 1)
			return out.Revision, err
		}},
		{"NativeStandings", func(ctx context.Context, s *Store) (int64, error) {
			out, err := s.NativeStandings(ctx, bracketLeague, 2026)
			return out.Revision, err
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := bracketTest(t)
			baseline := readResults(t, f)
			entered, release := make(chan struct{}), make(chan struct{})
			unblock := sync.OnceFunc(func() { close(release) })
			original := f.s.evidence
			var blocked atomic.Bool
			f.s.evidence = func(ctx context.Context, scope source.RawNativeScope) (source.NativeEvidence, error) {
				if blocked.CompareAndSwap(false, true) {
					close(entered)
					<-release
				}
				return original(ctx, scope)
			}
			var oldRevision int64
			var oldErr error
			var wg sync.WaitGroup
			wg.Go(func() { oldRevision, oldErr = tc.read(t.Context(), f.s) })
			defer func() {
				unblock()
				wg.Wait()
			}()
			select {
			case <-entered:
			case <-time.After(5 * time.Second):
				t.Fatal("pre-write reading did not reach score evidence")
			}

			changeScore(t, f, baseline.Games[0].Home, "flight-write")
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			newRevision, err := tc.read(ctx, f.s)
			if err != nil || newRevision != baseline.Revision+1 {
				t.Fatalf("post-write reading while the old flight is held: revision %d, err %v; want revision %d",
					newRevision, err, baseline.Revision+1)
			}
			unblock()
			wg.Wait()
			if oldErr != nil || oldRevision != baseline.Revision {
				t.Fatalf("pre-write reading: revision %d, err %v; want revision %d", oldRevision, oldErr, baseline.Revision)
			}
		})
	}
}
