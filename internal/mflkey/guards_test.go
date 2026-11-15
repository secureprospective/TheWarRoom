package mflkey

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMutationTimeouts(t *testing.T) {
	for _, operation := range []string{"set", "delete"} {
		t.Run(operation, func(t *testing.T) {
			release, finished := make(chan struct{}), make(chan struct{})
			block := func() error { <-release; close(finished); return nil }
			s := New("league", "2026")
			s.set = func(string, string, string) error { return block() }
			s.delete = func(string, string) error { return block() }
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
			defer cancel()
			var err error
			if operation == "set" {
				err = s.Set(ctx, "candidate")
			} else {
				err = s.Delete(ctx)
			}
			if !errors.Is(err, ErrTimeout) {
				t.Fatal(err)
			}
			close(release)
			<-finished
		})
	}
}

func TestForLeagueIsolatesCredentials(t *testing.T) {
	t.Parallel()
	store := NewMemory("100", "2026")
	first, second := store.ForLeague("101"), store.ForLeague("102")
	if err := first.Set(t.Context(), Key("first-fixture-key")); err != nil {
		t.Fatal(err)
	}
	if err := second.Set(t.Context(), Key("second-fixture-key")); err != nil {
		t.Fatal(err)
	}
	got, err := store.ForLeague("101").Get(t.Context())
	if err != nil || got.Reveal() != "first-fixture-key" {
		t.Fatal("first league lost its key")
	}
	got, err = store.ForLeague("102").Get(t.Context())
	if err != nil || got.Reveal() != "second-fixture-key" {
		t.Fatal("second league lost its key")
	}
	if _, err := store.Get(t.Context()); !errors.Is(err, ErrNoKey) {
		t.Fatal("credential crossed leagues")
	}
	if err := first.Delete(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := second.Get(t.Context()); err != nil {
		t.Fatal("deleting one key affected another league")
	}
}
