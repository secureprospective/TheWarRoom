package history

import (
	"bytes"
	"encoding/json"
	"testing"
)

func standingsJSON(t *testing.T, f *bracketFixture) (json.RawMessage, bool) {
	t.Helper()
	out, err := f.s.NativeStandings(t.Context(), bracketLeague, 2026)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := out.EncodedJSON()
	if err != nil {
		t.Fatal(err)
	}
	viaMarshal, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(encoded, viaMarshal) {
		t.Fatalf("EncodedJSON and Marshal differ:\n%s\n%s", encoded, viaMarshal)
	}
	return encoded, out.Frozen()
}

// TestFrozenStandingsAnswerUnchanged: the kept (frozen) standings answer encodes to the same JSON as a reading
// scored afresh with nothing kept, before and after score writes (a write is visible on the very next read).
func TestFrozenStandingsAnswerUnchanged(t *testing.T) {
	t.Parallel()
	kept, fresh := scoreCacheLeague(t, true), scoreCacheLeague(t, false)
	var last []byte
	for step := range 3 {
		if step > 0 {
			for _, f := range []*bracketFixture{kept, fresh} {
				changeScore(t, f, readResults(t, f).Games[0].Home, "standings-step-"+string(rune('a'+step)))
			}
		}
		standingsJSON(t, kept) // fills the memo
		got, frozen := standingsJSON(t, kept)
		if !frozen {
			t.Fatalf("step %d: the kept answer was not frozen", step)
		}
		want, wasFrozen := standingsJSON(t, fresh)
		if wasFrozen {
			t.Fatalf("step %d: a reading with nothing kept was frozen", step)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("step %d: kept answer differs from a fresh read:\n got %s\nwant %s", step, got, want)
		}
		if step > 0 && bytes.Equal(got, last) {
			t.Fatalf("step %d: a score write left the standings answer unchanged (fixture proves nothing)", step)
		}
		last = got
	}
}
