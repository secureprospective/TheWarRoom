package leagueservice

import (
	"reflect"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

type replayHost struct{ fixtureChangeHost }

func (replayHost) CommissionerPrincipal() (string, error) { return "creator", nil }

func TestServiceReplayCacheColdWarmAndRestart(t *testing.T) {
	f := newDueFixture(t)
	check := func() {
		t.Helper()
		service := New(replayHost{fixtureChangeHost{Store: f.store}})
		events := f.events(t)
		records := make([]commissioner.Record, len(events))
		for i, event := range events {
			records[i] = *event.Payload.Commissioner
		}
		state, err := commissioner.GovernanceState(records)
		if err != nil {
			t.Fatal(err)
		}
		want, err := commissioner.Reading(dueLeague, "creator", records, commissioner.GovernanceAuthorizer{State: state})
		if err != nil {
			t.Fatal(err)
		}
		for range 2 {
			got, err := service.TargetCommissionerLedger(dueLeague)
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("service/direct replay differ: %v", err)
			}
		}
		if len(f.events(t)) != len(events) {
			t.Fatal("read appended an event")
		}
	}
	check()
	note := f.execute(t, "creator", commissioner.NotePost, commissioner.RawArgs{Note: &commissioner.RawNote{Text: "A correction target"}})
	check()
	f.execute(t, "creator", commissioner.Revert, commissioner.RawArgs{Entry: note.Entry})
	check()
	if err := f.pools.Close(); err != nil {
		t.Fatal(err)
	}
	f.open(t)
	check()
}
