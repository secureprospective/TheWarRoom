package core_test

import (
	"reflect"
	"sync"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/app/core/coretest"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func TestCommissionerPortsKeepAtomicCreationAndDetachedReplay(t *testing.T) {
	t.Parallel()
	state := coretest.New(t).Build()
	command := commissioner.RawCommand{League: "native.port", Command: commissioner.Create,
		Args: commissioner.RawArgs{Name: "Port league", Preset: commissioner.PresetEmpty, Season: 2026,
			FranchiseCount: 4},
		Request: "create", Reason: "Start the league", Effective: "now", Visibility: "league"}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, request := range []string{"one", "two"} {
		wg.Go(func() {
			raw := command
			raw.Request = request
			_, err := state.ExecuteCommissioner(t.Context(), raw, "creator", nil)
			results <- err
		})
	}
	wg.Wait()
	close(results)
	successes := 0
	for err := range results {
		if err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("%d creations landed at one version", successes)
	}
	leagues, err := state.NativeLeagues(t.Context())
	if err != nil || len(leagues) != 1 {
		t.Fatal(leagues, err)
	}
	leagues[0].Name = "mutated"
	again, err := state.NativeLeagues(t.Context())
	if err != nil || again[0].Name != command.Args.Name {
		t.Fatal(again, err)
	}
	people, err := state.CommissionerPeople(t.Context(), command.League, "creator")
	if err != nil || len(people.Seats) == 0 {
		t.Fatal(people, err)
	}
	original := people.Seats[0]
	people.Seats[0].Name = "mutated"
	reread, err := state.CommissionerPeople(t.Context(), command.League, "creator")
	if err != nil || !reflect.DeepEqual(reread.Seats[0], original) {
		t.Fatal(reread, err)
	}
	events, err := state.Events(t.Context(), command.League, commissioner.Stream, 0)
	if err != nil || len(events) != 1 {
		t.Fatal(events, err)
	}
	events[0].Payload.Commissioner.Principal = "mutated"
	replay, err := state.Events(t.Context(), command.League, commissioner.Stream, 0)
	if err != nil || replay[0].Payload.Commissioner.Principal != "creator" {
		t.Fatal(replay, err)
	}
}
