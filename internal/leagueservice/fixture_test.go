package leagueservice

import (
	"context"
	"reflect"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func TestFixtureUsesSharedEngine(t *testing.T) {
	records, err := FixtureRecords()
	if err != nil {
		t.Fatal(err)
	}
	want, err := commissioner.Reading("native.https-fixture", "fixture.observer", records, fixtureObserver{})
	if err != nil {
		t.Fatal(err)
	}
	got, err := FixtureLedger(t.Context())
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("fixture reading: %+v %v", got, err)
	}
	records[0].Command.Args.Name = "changed by caller"
	again, err := FixtureLedger(t.Context())
	if err != nil || !reflect.DeepEqual(again, want) {
		t.Fatalf("fixture mutated: %+v %v", again, err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := FixtureLedger(ctx); err == nil {
		t.Fatal("cancellation ignored")
	}
}
