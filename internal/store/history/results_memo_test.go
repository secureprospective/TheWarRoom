package history

import (
	"reflect"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

// TestResultsMemoReusesUntilALedgerWrite: a repeated results read is served from the memo, and the first
// read after a score change shows the change.
func TestResultsMemoReusesUntilALedgerWrite(t *testing.T) {
	t.Parallel()
	f := scoreCacheLeague(t, true)
	read := func() commissioner.NativeResults {
		t.Helper()
		out, err := f.s.NativeResults(t.Context(), bracketLeague, 2026, 1)
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	first := read()
	if hits, _ := f.s.results.stats(); hits != 0 {
		t.Fatalf("first read hit the memo %d times", hits)
	}
	again := read()
	if hits, _ := f.s.results.stats(); hits != 1 || !reflect.DeepEqual(again.Games, first.Games) {
		t.Fatalf("repeat read was not reused: hits=%d", hits)
	}
	home := first.Games[0].Home
	raw := scoreCommand(t, commissioner.ScoreFranchise, commissioner.RawScoreChange{Season: 2026, Week: 1,
		Franchise: home, Delta: pointsDelta(15001)}, revision(t, f.s), "memo-write")
	runScoreCommand(t, f.s, raw, "creator")
	after := read()
	if reflect.DeepEqual(after.Scores, first.Scores) {
		t.Fatalf("read after a score change still shows the old scores: %+v", first.Games[0])
	}
	if after.Revision == first.Revision {
		t.Fatal("read after a write kept the old revision")
	}
}
