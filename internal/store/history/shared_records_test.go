package history

import (
	"context"
	"reflect"
	"slices"
	"sync"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/deepclone"
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
