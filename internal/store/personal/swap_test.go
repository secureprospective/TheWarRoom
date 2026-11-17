package personal

import (
	"context"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

func TestWatchSwap(t *testing.T) {
	ctx := context.Background()
	pools, err := db.Open(ctx, filepath.Join(t.TempDir(), "app-dev.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = pools.Close() }()
	store := New(pools)
	if err := store.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)
	watch := func(mine, id string) {
		t.Helper()
		if err := store.Change(ctx, RawChange{Mine: mine, Kind: "watch", Entity: "player", ID: id, Action: "add"}, at); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range []string{"0816", "12626", "15711"} {
		watch("0025", id)
	}
	watch("0007", "0816")
	if err := store.Change(ctx, RawChange{Mine: "0025", Kind: "draft", Entity: "player", ID: "0816", Action: "add"}, at); err != nil {
		t.Fatal(err)
	}
	order := func(mine, kind string) []string {
		t.Helper()
		entries, err := store.Read(ctx, mine)
		if err != nil {
			t.Fatal(err)
		}
		var ids []string
		for _, e := range entries {
			if e.Kind == kind {
				ids = append(ids, e.ID)
			}
		}
		return ids
	}
	swap := func(mine, id, with string) error {
		return store.Change(ctx, RawChange{Mine: mine, Kind: "watch", Entity: "player", ID: id, With: with, Action: "swap"}, at)
	}
	tests := []struct {
		name, mine, id, with string
		wantErr              bool
		want                 []string
	}{
		{"swap skips the player between", "0025", "15711", "0816", false, []string{"15711", "12626", "0816"}},
		{"swap back restores", "0025", "0816", "15711", false, []string{"0816", "12626", "15711"}},
		{"unlisted partner is refused", "0025", "0816", "9999", true, []string{"0816", "12626", "15711"}},
		{"another owner's list is not mine", "0007", "0816", "12626", true, []string{"0816", "12626", "15711"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := swap(tt.mine, tt.id, tt.with); (err != nil) != tt.wantErr {
				t.Fatalf("swap err = %v, wantErr %v", err, tt.wantErr)
			}
			if got := order("0025", "watch"); !slices.Equal(got, tt.want) {
				t.Fatalf("watch order = %v, want %v", got, tt.want)
			}
			if got := order("0007", "watch"); !slices.Equal(got, []string{"0816"}) {
				t.Fatalf("other owner's list changed: %v", got)
			}
			if got := order("0025", "draft"); !slices.Equal(got, []string{"0816"}) {
				t.Fatalf("draft list changed: %v", got)
			}
		})
	}
}

func TestHostedEntriesAreScopedToAuthenticatedFranchise(t *testing.T) {
	ctx := t.Context()
	pools, err := db.Open(ctx, filepath.Join(t.TempDir(), "league.db"), history.LeagueSchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = pools.Close() }()
	if err := history.CreateLeagueFile(ctx, pools, history.LeagueFile{League: "native.test"}); err != nil {
		t.Fatal(err)
	}
	if found, err := db.HasTable(ctx, pools, "personal_entries"); err != nil || found {
		t.Fatalf("league file has desktop personal_entries = %v, %v", found, err)
	}
	store := NewLeague(pools)
	if err := store.Initialize(ctx); err == nil {
		t.Fatal("hosted store tried to initialize the desktop schema")
	}
	addHostedEntry(ctx, t, store, "franchise.01", RawChange{Mine: "franchise.02", Kind: "watch", Entity: "player", ID: "0816", Action: "add"})
	addHostedEntry(ctx, t, store, "franchise.01", RawChange{Mine: "franchise.02", Kind: "note", Entity: "player", ID: "0816", Text: "private", Action: "add"})
	assertHostedEntryCount(ctx, t, store, "franchise.01", 2)
	assertHostedEntryCount(ctx, t, store, "franchise.02", 0)
	notesA, err := store.PlayerNotesForFranchise(ctx, "franchise.01", "0816")
	if err != nil || notesA.Count != 1 || notesA.Notes[0].Text != "private" {
		t.Fatalf("franchise A notes = %+v, %v", notesA, err)
	}
	notesB, err := store.PlayerNotesForFranchise(ctx, "franchise.02", "0816")
	if err != nil || notesB.Count != 0 {
		t.Fatalf("franchise B read A's notes: %+v, %v", notesB, err)
	}
	addHostedEntry(ctx, t, store, "franchise.02", RawChange{Mine: "franchise.01", Kind: "watch", Entity: "player", ID: "0816", Action: "remove"})
	assertHostedEntryCount(ctx, t, store, "franchise.01", 2)
	addHostedEntry(ctx, t, store, "franchise.02", RawChange{Mine: "franchise.01", Kind: "watch", Entity: "player", ID: "0816", Action: "add"})
	assertHostedEntryCount(ctx, t, store, "franchise.02", 1)
	assertHostedEntryCount(ctx, t, store, "franchise.01", 2)
}

func TestHostedDraftQueueReplaceIsOrderedAtomicAndPrivate(t *testing.T) {
	ctx := t.Context()
	pools, err := db.Open(ctx, filepath.Join(t.TempDir(), "league.db"), history.LeagueSchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = pools.Close() }()
	if err := history.CreateLeagueFile(ctx, pools, history.LeagueFile{League: "native.test"}); err != nil {
		t.Fatal(err)
	}
	store := NewLeague(pools)
	at := time.Date(2026, 11, 17, 15, 0, 0, 0, time.UTC)
	for _, players := range [][]string{{"0099", "0100", "0101"}, {"0101", "0099", "0100"}} {
		if err := store.ReplaceDraftQueue(ctx, "franchise.01", players, at); err != nil {
			t.Fatal(err)
		}
	}
	read := func(franchise string) []string {
		t.Helper()
		entries, err := store.ReadFranchise(ctx, franchise)
		if err != nil {
			t.Fatal(err)
		}
		var ids []string
		for _, entry := range entries {
			if entry.Kind == "draft" {
				ids = append(ids, entry.ID)
			}
		}
		return ids
	}
	want := []string{"0101", "0099", "0100"}
	if got := read("franchise.01"); !slices.Equal(got, want) {
		t.Fatalf("replaced draft queue = %v, want %v", got, want)
	}
	if got := read("franchise.02"); len(got) != 0 {
		t.Fatalf("another franchise read the queue: %v", got)
	}
	if err := store.ReplaceDraftQueue(ctx, "franchise.01", []string{"0101", "0101"}, at); err == nil {
		t.Fatal("duplicate player queue was accepted")
	}
	if got := read("franchise.01"); !slices.Equal(got, want) {
		t.Fatalf("invalid replacement changed queue = %v, want %v", got, want)
	}
	if err := store.ReplaceDraftQueue(ctx, "franchise.01", nil, at); err != nil {
		t.Fatal(err)
	}
	if got := read("franchise.01"); len(got) != 0 {
		t.Fatalf("empty replacement did not clear queue: %v", got)
	}
}

func addHostedEntry(ctx context.Context, t *testing.T, store *Store, franchiseID string, change RawChange) {
	t.Helper()
	at := time.Date(2026, 11, 17, 15, 0, 0, 0, time.UTC)
	if err := store.ChangeForFranchise(ctx, franchiseID, change, at); err != nil {
		t.Fatal(err)
	}
}

func assertHostedEntryCount(ctx context.Context, t *testing.T, store *Store, franchiseID string, want int) {
	t.Helper()
	entries, err := store.ReadFranchise(ctx, franchiseID)
	if err != nil || len(entries) != want {
		t.Fatalf("franchise %s has %+v, %v; want %d entries", franchiseID, entries, err, want)
	}
}
