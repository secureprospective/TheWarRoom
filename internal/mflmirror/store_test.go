package mflmirror

import (
	"context"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
)

// fakeMFL is a recorded-shape Source that counts calls by export; no live MFL is ever called.
type fakeMFL struct{ calls map[string]int }

func (f *fakeMFL) total() int { return f.calls["freeAgents"] + f.calls["players"] }

func (f *fakeMFL) FetchFreeAgents(context.Context, string, int) ([]string, error) {
	f.calls["freeAgents"]++
	return []string{"0531", "13593"}, nil
}

func (f *fakeMFL) FetchPlayers(context.Context, string, int) ([]byte, error) {
	f.calls["players"]++
	return []byte(`[{"ID":"0531","Name":"Doe, John","Position":"WR","Team":"BUF"},{"ID":"13593","Name":"Roe, Rich","Position":"RB","Team":"FA"}]`), nil
}

func (f *fakeMFL) Directory([]byte) (Directory, error) { return nil, nil }

func newTestStore(t *testing.T, clock *time.Time) *Store {
	t.Helper()
	pools, err := db.OpenScratch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	return NewStore(pools, func() time.Time { return *clock })
}

func TestMirrorRefreshThrottle(t *testing.T) {
	now := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	store := newTestStore(t, &now)
	fake := &fakeMFL{calls: map[string]int{}}
	var f Source = fake
	ctx := context.Background()

	first, err := store.Refresh(ctx, f, "14432")
	if err != nil || !first.Fetched || len(first.Held.FreeAgents) != 2 || len(first.Held.PlayersJSON) == 0 {
		t.Fatalf("first refresh = %+v, %v", first, err)
	}
	calls := fake.total()
	if calls == 0 {
		t.Fatal("first refresh made no calls")
	}
	now = now.Add(4 * time.Minute)
	second, err := store.Refresh(ctx, f, "14432")
	if err != nil {
		t.Fatal(err)
	}
	if second.Fetched || fake.total() != calls {
		t.Fatalf("second refresh inside the window called MFL: fetched=%v calls %d -> %d", second.Fetched, calls, fake.total())
	}
	if second.Age != 4*time.Minute || len(second.Held.FreeAgents) != 2 {
		t.Fatalf("held copy age = %v, agents = %v", second.Age, second.Held.FreeAgents)
	}
	now = now.Add(2 * time.Minute)
	third, err := store.Refresh(ctx, f, "14432")
	if err != nil || !third.Fetched {
		t.Fatalf("refresh after the window = %+v, %v", third, err)
	}
}

func TestMirrorPlayersHeldForADay(t *testing.T) {
	now := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	store := newTestStore(t, &now)
	fake := &fakeMFL{calls: map[string]int{}}
	ctx := context.Background()
	if _, err := store.Refresh(ctx, fake, "14432"); err != nil {
		t.Fatal(err)
	}
	now = now.Add(10 * time.Minute)
	if _, err := store.Refresh(ctx, fake, "14432"); err != nil {
		t.Fatal(err)
	}
	if fake.calls["players"] != 1 || fake.calls["freeAgents"] != 2 {
		t.Fatalf("exports = %v; players once, free agents twice expected", fake.calls)
	}
}

func TestMirrorRegistration(t *testing.T) {
	now := time.Now()
	store := newTestStore(t, &now)
	ctx := context.Background()
	if _, ok, err := store.Registration(ctx); err != nil || ok {
		t.Fatalf("empty registration = %v, %v", ok, err)
	}
	if err := store.Register(ctx, Registration{MFLLeague: "14432", Franchise: "0007"}); err != nil {
		t.Fatal(err)
	}
	r, ok, err := store.Registration(ctx)
	if err != nil || !ok || r.MFLLeague != "14432" || r.Franchise != "0007" {
		t.Fatalf("registration = %+v %v %v", r, ok, err)
	}
}
