package leagueservice

import (
	"reflect"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/governance"
)

// eventRecords is the pre-11-08 hosted path: every record rebuilt from the full event list.
func eventRecords(t *testing.T, f *dueFixture, league string) []commissioner.Record {
	t.Helper()
	events, err := f.store.Events(t.Context(), league, commissioner.Stream, 0)
	if err != nil {
		t.Fatal(err)
	}
	out := make([]commissioner.Record, 0, len(events))
	for _, e := range events {
		out = append(out, *e.Payload.Commissioner)
	}
	return out
}

// The hosted reader serves the shared cached snapshot; it must equal the event-built records, and the
// people state and replay folded from it, at every revision, including after a new write.
func TestStoredHostRecordsEqualEventRecords(t *testing.T) {
	t.Parallel()
	f := newDueFixture(t)
	check := func() {
		t.Helper()
		held := releaseRoleHistory(t, f, dueLeague)
		built := eventRecords(t, f, dueLeague)
		if len(held) != len(built) {
			t.Fatalf("held %d records, events %d", len(held), len(built))
		}
		for i := range built {
			if !reflect.DeepEqual(held[i].Command, built[i].Command) || held[i].Receipt != built[i].Receipt ||
				held[i].Principal != built[i].Principal || held[i].Canonical != built[i].Canonical {
				t.Fatalf("record %d differs", i)
			}
		}
		a, errA := commissioner.GovernanceState(held)
		b, errB := commissioner.GovernanceState(built)
		if (errA == nil) != (errB == nil) || !reflect.DeepEqual(a, b) {
			t.Fatalf("people state differs: %v / %v", errA, errB)
		}
		if !reflect.DeepEqual(commissioner.Replay(held), commissioner.Replay(built)) {
			t.Fatal("replay differs")
		}
	}
	check()
	check() // second read hits the cache
	f.execute(t, "creator", commissioner.CaretakerSeat, commissioner.RawArgs{People: &commissioner.RawPeople{Principal: "creator", Franchise: "franchise.03"}})
	check()
	raw := f.raw(t, "commish.role.grant", commissioner.RawArgs{People: &commissioner.RawPeople{Principal: "deputy.one", Role: governance.CoCommissioner}})
	raw.Visibility = "private"
	if _, err := f.store.ExecuteCommissioner(t.Context(), raw, "creator", nil); err != nil {
		t.Fatal(err)
	}
	check()
}
