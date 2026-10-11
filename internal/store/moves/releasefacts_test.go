package moves

import (
	"context"
	"reflect"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/envelope"
)

func storedRelease(t *testing.T, id string) (envelope.Envelope, envelope.ReleaseFacts) {
	t.Helper()
	e, f := testBid(t, id)
	spec := e.Receipt().Spec
	spec.Intent = envelope.CutIntent
	spec.Expected.RosterStatus = ""
	e, err := envelope.DraftRelease(bidAt(), spec, envelope.ReleaseCheck{OnRoster: true}, func() string { return id })
	if err != nil {
		t.Fatal(err)
	}
	return e, envelope.ReleaseFacts{Player: f.Player, Name: "Example", Season: 2026, Charge: 700000000, RostersStamp: "2026-10-10", ContractStamp: "cells", DeadCapStamp: "ledger"}
}
func TestReleaseMigrationRoundTripAndAtomicUniqueness(t *testing.T) {
	s, p := testStore(t)
	ctx := context.Background()
	bid, f := testBid(t, "existing")
	if err := s.SaveBid(ctx, bid, f); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Write().ExecContext(ctx, `DROP TABLE release_facts`); err != nil {
		t.Fatal(err)
	}
	if err := s.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get(ctx, "existing"); err != nil {
		t.Fatal("migration lost bid", err)
	}
	e, facts := storedRelease(t, "release-1")
	if err := s.SaveRelease(ctx, e, facts); err != nil {
		t.Fatal(err)
	}
	got, err := s.ReleaseFacts(ctx, e.ID())
	if err != nil || !reflect.DeepEqual(got, facts) {
		t.Fatalf("facts %+v %v", got, err)
	}
	duplicate, _ := storedRelease(t, "release-2")
	if err := s.SaveRelease(ctx, duplicate, facts); err == nil {
		t.Fatal("duplicate allowed")
	}
	if _, err := s.Get(ctx, duplicate.ID()); err == nil {
		t.Fatal("duplicate envelope not rolled back")
	}
	other := e.Receipt().Spec
	other.Intent = envelope.BuyoutIntent
	buyout, err := envelope.DraftRelease(bidAt(), other, envelope.ReleaseCheck{}, func() string { return "buyout" })
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SaveRelease(ctx, buyout, facts); err == nil {
		t.Fatal("cut and buyout duplicate allowed")
	}
}

func TestReleaseUniquenessSurvivesFranchiseChange(t *testing.T) {
	s, _ := testStore(t)
	ctx := context.Background()
	e, f := storedRelease(t, "first")
	if err := s.SaveRelease(ctx, e, f); err != nil {
		t.Fatal(err)
	}
	spec := e.Receipt().Spec
	spec.FranchiseID = "0002"
	next, err := envelope.DraftRelease(bidAt(), spec, envelope.ReleaseCheck{OnRoster: true}, func() string { return "other" })
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SaveRelease(ctx, next, f); err == nil {
		t.Fatal("same player has two open plans after franchise change")
	}
}
