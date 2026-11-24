package history

import (
	"bytes"
	"encoding/json"
	"testing"
)

// TestFrozenResultsAnswerUnchanged: the kept (frozen) results answer encodes to the same JSON as a reading
// scored afresh with nothing kept, for the same league state.
func TestFrozenResultsAnswerUnchanged(t *testing.T) {
	t.Parallel()
	kept := scoreCacheLeague(t, true)
	readResults(t, kept)
	hit := readResults(t, kept)
	if !hit.Frozen() {
		t.Fatal("the kept answer was not frozen")
	}
	fresh := readResults(t, scoreCacheLeague(t, false))
	if fresh.Frozen() {
		t.Fatal("a reading with nothing kept was frozen")
	}
	want, err := json.Marshal(fresh)
	if err != nil {
		t.Fatal(err)
	}
	for name, encode := range map[string]func() ([]byte, error){
		"marshal": func() ([]byte, error) { return json.Marshal(hit) },
		"encoded": hit.EncodedJSON,
	} {
		got, err := encode()
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("%s: frozen answer differs:\n got %s\nwant %s", name, got, want)
		}
	}
	if len(hit.Scores) == 0 || len(hit.Scores[0].Starters) == 0 {
		t.Fatal("fixture has no starters to compare")
	}
}
