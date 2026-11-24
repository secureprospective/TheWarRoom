package mflmirror

import (
	"context"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
)

// fakeMFL is a recorded-shape Source that counts calls by export; no live MFL is ever called.
type fakeMFL struct{ calls map[string]int }

func (f *fakeMFL) total() int {
	return f.calls["freeAgents"] + f.calls["players"] + f.calls["franchises"]
}

func (f *fakeMFL) FetchFreeAgents(context.Context, string, int) ([]string, error) {
	f.calls["freeAgents"]++
	return []string{"0531", "13593"}, nil
}

func (f *fakeMFL) FetchFranchises(context.Context, string, int) ([]Franchise, error) {
	f.calls["franchises"]++
	return []Franchise{{ID: "0001", Name: "Alpha"}, {ID: "0002", Name: "Beta"}}, nil
}

func (f *fakeMFL) FetchPlayers(context.Context, string, int) ([]byte, error) {
	f.calls["players"]++
	return []byte(`[{"ID":"0531","Name":"Doe, John","Position":"WR","Team":"BUF"},{"ID":"13593","Name":"Roe, Rich","Position":"RB","Team":"FA"}]`), nil
}

func (f *fakeMFL) FetchLeagueData(context.Context, string, int) (LeagueData, error) {
	f.calls["leagueData"]++
	return LeagueData{}, nil
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

func TestMirrorTeamChoiceIsKeyedBySeatAndHeldList(t *testing.T) {
	now := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	store := newTestStore(t, &now)
	ctx := context.Background()
	if err := store.SetTeam(ctx, "franchise.01", "14432", "0001"); err == nil {
		t.Fatal("team accepted before any list was held")
	}
	if _, err := store.Refresh(ctx, &fakeMFL{calls: map[string]int{}}, "14432"); err != nil {
		t.Fatal(err)
	}
	if err := store.SetTeam(ctx, "franchise.01", "14432", "9999"); err == nil {
		t.Fatal("team outside the held list accepted")
	}
	if err := store.SetTeam(ctx, "", "14432", "0001"); err == nil {
		t.Fatal("team saved without a seat")
	}
	if err := store.SetTeam(ctx, "franchise.01", "14432", "0002"); err != nil {
		t.Fatal(err)
	}
	if team, ok, err := store.Team(ctx, "franchise.01", "14432"); err != nil || !ok || team != "0002" {
		t.Fatalf("own team = %q %v %v", team, ok, err)
	}
	if _, ok, _ := store.Team(ctx, "franchise.02", "14432"); ok {
		t.Fatal("another seat read this seat's choice")
	}
	if _, ok, _ := store.Team(ctx, "franchise.01", "55555"); ok {
		t.Fatal("a choice made for one MFL league applies to another")
	}
}
