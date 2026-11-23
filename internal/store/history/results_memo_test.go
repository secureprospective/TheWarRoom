package history

import (
	"fmt"
	"reflect"
	"sync"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

// changeScore records a franchise score change in week 1, the kind of write that makes every phone re-read.
func changeScore(t *testing.T, f *bracketFixture, franchise, request string) {
	t.Helper()
	raw := scoreCommand(t, commissioner.ScoreFranchise, commissioner.RawScoreChange{Season: 2026, Week: 1,
		Franchise: franchise, Delta: pointsDelta(15001)}, revision(t, f.s), request)
	runScoreCommand(t, f.s, raw, "creator")
}

func readResults(t *testing.T, f *bracketFixture) commissioner.NativeResults {
	t.Helper()
	out, err := f.s.NativeResults(t.Context(), bracketLeague, 2026, 1)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// TestResultsMemoReusesUntilALedgerWrite: a repeated results read is served from the memo, and the first
// read after a score change shows the change.
func TestResultsMemoReusesUntilALedgerWrite(t *testing.T) {
	t.Parallel()
	f := scoreCacheLeague(t, true)
	first := readResults(t, f)
	if hits, _, _ := f.s.results.stats(); hits != 0 {
		t.Fatalf("first read hit the memo %d times", hits)
	}
	again := readResults(t, f)
	if hits, _, _ := f.s.results.stats(); hits != 1 || !reflect.DeepEqual(again.Games, first.Games) {
		t.Fatalf("repeat read was not reused: hits=%d", hits)
	}
	changeScore(t, f, first.Games[0].Home, "memo-write")
	after := readResults(t, f)
	if reflect.DeepEqual(after.Scores, first.Scores) {
		t.Fatalf("read after a score change still shows the old scores: %+v", first.Games[0])
	}
	if after.Revision == first.Revision {
		t.Fatal("read after a write kept the old revision")
	}
}

// TestStandingsMemoReusesUntilALedgerWrite is the same contract for the season's standings.
func TestStandingsMemoReusesUntilALedgerWrite(t *testing.T) {
	t.Parallel()
	f := scoreCacheLeague(t, true)
	read := func() commissioner.NativeStandings {
		t.Helper()
		out, err := f.s.NativeStandings(t.Context(), bracketLeague, 2026)
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	first := read()
	again := read()
	if hits, _, runs := f.s.standings.stats(); hits != 1 || runs != 1 || !reflect.DeepEqual(again, first) {
		t.Fatalf("repeat standings read was not reused: hits=%d runs=%d", hits, runs)
	}
	changeScore(t, f, readResults(t, f).Games[0].Home, "standings-write")
	if after := read(); after.Revision == first.Revision || reflect.DeepEqual(after, first) {
		t.Fatal("standings after a score change were not read afresh")
	}
}

// TestReadingBurstAfterAWriteComputesOnce: right after a score change every phone re-reads the same week at
// once; the reading is computed once and each caller gets its own copy.
func TestReadingBurstAfterAWriteComputesOnce(t *testing.T) {
	t.Parallel()
	f := scoreCacheLeague(t, true)
	first := readResults(t, f)
	if _, err := f.s.NativeStandings(t.Context(), bracketLeague, 2026); err != nil {
		t.Fatal(err)
	}
	changeScore(t, f, first.Games[0].Home, "burst-write")
	_, _, resultsBefore := f.s.results.stats()
	_, _, standingsBefore := f.s.standings.stats()
	const readers = 20
	// Hosted requests never join shareReading's flights, as on the server.
	hosted := commissioner.WithCredentialAdmission(t.Context(), nil)
	start := make(chan struct{})
	outs := make([]commissioner.NativeResults, readers)
	errs := make([]error, 2*readers)
	var wg sync.WaitGroup
	for i := range readers {
		wg.Go(func() {
			<-start
			outs[i], errs[i] = f.s.NativeResults(hosted, bracketLeague, 2026, 1)
		})
		wg.Go(func() {
			<-start
			_, errs[readers+i] = f.s.NativeStandings(hosted, bracketLeague, 2026)
		})
	}
	close(start)
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	_, _, resultsAfter := f.s.results.stats()
	_, _, standingsAfter := f.s.standings.stats()
	if got := fmt.Sprint(resultsAfter-resultsBefore, standingsAfter-standingsBefore); got != "1 1" {
		t.Fatalf("burst of %d readers computed results, standings %s times, want 1 1", readers, got)
	}
	if &outs[0].Scores[0] == &outs[1].Scores[0] || reflect.DeepEqual(outs[0].Scores, first.Scores) {
		t.Fatal("burst readers share one copy or read the old scores")
	}
}
