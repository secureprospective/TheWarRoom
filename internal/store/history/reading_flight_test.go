package history

import (
	"bytes"
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

// gatedReading counts runs and holds every run until released, so callers are known to overlap.
type gatedReading struct {
	runs    atomic.Int32
	started chan struct{}
	release chan struct{}
}

func newGatedReading() *gatedReading {
	return &gatedReading{started: make(chan struct{}, 64), release: make(chan struct{})}
}

func (g *gatedReading) read(ctx context.Context) (time.Time, error) {
	g.runs.Add(1)
	g.started <- struct{}{}
	<-g.release
	return time.Unix(int64(g.runs.Load()), 0), ctx.Err()
}

func TestShareReadingRunsConcurrentIdenticalReadingsOnce(t *testing.T) {
	t.Parallel()
	var flights readingFlights
	g := newGatedReading()
	const n = 12
	results := make([]time.Time, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			got, err := shareReading(t.Context(), &flights, "NativeStandings", "native.x|2026", g.read)
			if err != nil {
				t.Error(err)
			}
			results[i] = got
		})
	}
	<-g.started
	waitFor(t, func() bool { return flightWaiters(&flights) == n })
	close(g.release)
	wg.Wait()
	if got := g.runs.Load(); got != 1 {
		t.Fatalf("%d concurrent identical readings ran %d times, want 1", n, got)
	}
	for i, r := range results {
		if !r.Equal(results[0]) {
			t.Fatalf("caller %d read %v, the flight read %v: joiners must share its result and clock", i, r, results[0])
		}
	}
	if got := flights.inFlight("NativeStandings"); got != 0 {
		t.Fatalf("%d readings still held after completion, want none kept", got)
	}
	// Completed readings are not memoized: the next caller reads afresh.
	g2 := newGatedReading()
	close(g2.release)
	if _, err := shareReading(t.Context(), &flights, "NativeStandings", "native.x|2026", g2.read); err != nil {
		t.Fatal(err)
	}
	if g2.runs.Load() != 1 {
		t.Fatal("a reading after completion joined a finished flight")
	}
}

func TestShareReadingNeverJoinsDifferentViewers(t *testing.T) {
	t.Parallel()
	var flights readingFlights
	g := newGatedReading()
	at := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	viewers := []context.Context{
		t.Context(),
		commissioner.WithRecipient(t.Context(), "owner-1", at),
		commissioner.WithRecipient(t.Context(), "owner-2", at),
		commissioner.WithRecipient(t.Context(), "owner-1", at.Add(time.Second)),
		commissioner.WithCredentialAdmission(t.Context(), nil), // hosted: never shared, even with itself
		commissioner.WithCredentialAdmission(t.Context(), nil),
	}
	var wg sync.WaitGroup
	for _, ctx := range viewers {
		wg.Go(func() {
			_, _ = shareReading(ctx, &flights, "NativeOdds", "native.x|2026", g.read)
		})
	}
	for range viewers {
		<-g.started
	}
	close(g.release)
	wg.Wait()
	if got := g.runs.Load(); int(got) != len(viewers) {
		t.Fatalf("%d viewers with different visibility ran %d readings, want one each", len(viewers), got)
	}
}

func TestShareReadingCancelsOnlyWhenEveryCallerLeaves(t *testing.T) {
	t.Parallel()
	var flights readingFlights
	g := newGatedReading()
	leaving, leave := context.WithCancel(t.Context())
	done := make(chan error, 2)
	go func() {
		_, err := shareReading(leaving, &flights, "NativeResults", "w1", g.read)
		done <- err
	}()
	<-g.started
	go func() {
		_, err := shareReading(t.Context(), &flights, "NativeResults", "w1", g.read)
		done <- err
	}()
	waitFor(t, func() bool { return flightWaiters(&flights) == 2 })
	leave()
	if err := <-done; err == nil {
		t.Fatal("the caller that left was not told so")
	}
	close(g.release)
	if err := <-done; err != nil {
		t.Fatalf("the caller still waiting lost its reading: %v", err)
	}
}

func flightWaiters(f *readingFlights) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, fl := range f.flights {
		n += fl.waiters
	}
	return n
}

func waitFor(t *testing.T, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !ok() {
		if time.Now().After(deadline) {
			t.Fatal("timed out")
		}
		time.Sleep(time.Millisecond)
	}
}

func TestSharedStandingsMatchUnsharedReading(t *testing.T) {
	t.Parallel()
	f := heldSeasonLeague(t)
	want, err := f.s.readNativeStandings(t.Context(), bracketLeague, 2026)
	if err != nil {
		t.Fatal(err)
	}
	wantJSON, _ := json.Marshal(want)
	const n = 8
	got := make([][]byte, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			reading, err := f.s.NativeStandings(t.Context(), bracketLeague, 2026)
			if err != nil {
				t.Error(err)
			}
			got[i], _ = json.Marshal(reading)
		})
	}
	wg.Wait()
	for i := range got {
		if !bytes.Equal(got[i], wantJSON) {
			t.Fatalf("shared reading %d differs from the unshared reading", i)
		}
	}
	if runs := f.s.flights.runs("NativeStandings"); runs < 1 || runs > n {
		t.Fatalf("%d readings run for %d callers", runs, n)
	}
}
