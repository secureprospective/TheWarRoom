package history

import (
	"testing"
)

func TestWholeStreamEventsReadOnlyTheAppendedTail(t *testing.T) {
	t.Parallel()
	s, _, p := newStore(t)
	writer := New(p, s.Registry()) // another store appends, as a second process would
	for i, id := range []string{"one", "two"} {
		if _, err := writer.AppendEvent(t.Context(), eventFixture(id), int64(i)); err != nil {
			t.Fatal(err)
		}
	}
	first, err := s.Events(t.Context(), "legacy", "contracts", 0)
	if err != nil || len(first) != 2 {
		t.Fatalf("first read %d %v", len(first), err)
	}
	*first[0].Payload.MoneyCents = 1 // callers own their copy
	if _, err := writer.AppendEvent(t.Context(), eventFixture("three"), 2); err != nil {
		t.Fatal(err)
	}
	second, err := s.Events(t.Context(), "legacy", "contracts", 0)
	if err != nil || len(second) != 3 || second[2].ID != "three" || *second[0].Payload.MoneyCents != 123456789 {
		t.Fatalf("second read %+v %v", second, err)
	}
	key := eventStreamKey{"legacy", "contracts"}
	s.eventsMu.Lock()
	held := s.events[key]
	if held.revision != 3 || len(held.events) != 3 {
		s.eventsMu.Unlock()
		t.Fatalf("held %d events at revision %d", len(held.events), held.revision)
	}
	held.lastID = "replaced" // a restored database no longer has the held newest entry
	held.events[0].ID = "stale"
	s.events[key] = held
	s.eventsMu.Unlock()
	third, err := s.Events(t.Context(), "legacy", "contracts", 0)
	if err != nil || len(third) != 3 || third[0].ID != "one" {
		t.Fatalf("re-read after replaced prefix %+v %v", third, err)
	}
	empty, err := s.Events(t.Context(), "legacy", "none", 0)
	if err != nil || len(empty) != 0 {
		t.Fatalf("empty stream %+v %v", empty, err)
	}
}
