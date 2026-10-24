package history

import (
	"reflect"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/governance"
)

func TestDeputyOrderAtomicLedgerReplay(t *testing.T) {
	s, _, _ := newStore(t)
	if _, err := s.ExecuteCommissioner(t.Context(), createCommand(), "creator", nil); err != nil {
		t.Fatal(err)
	}
	for i, p := range []string{"A", "B", "C"} {
		executePeople(t, s, "creator", "commish.role.grant", commissioner.RawPeople{Principal: p, Role: governance.CoCommissioner}, int64(i+1))
	}
	raw := peopleCommand(commissioner.DeputyOrder, commissioner.RawPeople{Order: []governance.Principal{"C", "A", "B"}}, 4)
	raw.Visibility = "private"
	before, err := s.leagueRecords(t.Context(), "native.test")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		principal string
		order     []governance.Principal
	}{
		{"A", []governance.Principal{"C", "A", "B"}},
		{"creator", []governance.Principal{"C", "A"}},
		{"creator", []governance.Principal{"C", "A", "B", "extra"}},
		{"creator", []governance.Principal{"C", "A", "A"}},
		{"creator", []governance.Principal{"C", "A", "extra"}},
	} {
		bad := raw
		bad.Args.People = &commissioner.RawPeople{Order: tc.order}
		if _, err := s.ExecuteCommissioner(t.Context(), bad, tc.principal, nil); err == nil {
			t.Fatal("invalid order accepted")
		}
		after, err := s.leagueRecords(t.Context(), "native.test")
		if err != nil || !reflect.DeepEqual(before, after) {
			t.Fatalf("refusal changed ledger: %v", err)
		}
	}
	if _, err := s.ExecuteCommissioner(t.Context(), raw, "creator", nil); err != nil {
		t.Fatal(err)
	}
	events, err := s.Events(t.Context(), "native.test", commissioner.Stream, 0)
	if err != nil {
		t.Fatal(err)
	}
	records, err := commissionerRecords(events)
	if err != nil || len(records) != 5 {
		t.Fatalf("order must append exactly one record: %d %v", len(records), err)
	}
	for end := 4; end <= 5; end++ {
		state, err := commissioner.GovernanceState(records[:end])
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range []governance.Principal{"A", "B", "C"} {
			if !state.HasRole(p, governance.CoCommissioner) {
				t.Fatalf("%s lost powers at record %d", p, end)
			}
		}
	}
	state, err := commissioner.GovernanceState(records)
	if err != nil || !reflect.DeepEqual(state.CoCommissioners(), raw.Args.People.Order) {
		t.Fatalf("replayed order: %v", err)
	}
	judge, err := governance.AppealJudge(state, []governance.Principal{"creator"})
	if err != nil || judge != "C" {
		t.Fatalf("appeal priority: %s %v", judge, err)
	}
	executePeople(t, s, "creator", "commish.succession.stepdown", commissioner.RawPeople{Principal: "creator"}, 5)
	records, err = s.leagueRecords(t.Context(), "native.test")
	if err != nil {
		t.Fatal(err)
	}
	state, err = commissioner.GovernanceState(commissioner.WithoutClosedBooks(records))
	if err != nil || !state.HasRole("C", governance.Commissioner) {
		t.Fatalf("succession replay: %v", err)
	}
}
