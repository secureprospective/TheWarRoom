package personal

import (
	"context"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
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

func TestSwapValidation(t *testing.T) {
	base := RawChange{Mine: "0025", Kind: "watch", Entity: "player", ID: "0816", With: "12626", Action: "swap"}
	if err := base.Validate(); err != nil {
		t.Fatalf("valid swap refused: %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*RawChange)
	}{
		{"draft list cannot swap", func(r *RawChange) { r.Kind = "draft" }},
		{"bait list cannot swap", func(r *RawChange) { r.Kind = "bait" }},
		{"swap with self", func(r *RawChange) { r.With = r.ID }},
		{"partner not canonical", func(r *RawChange) { r.With = "816" }},
		{"partner missing", func(r *RawChange) { r.With = "" }},
		{"with on an add", func(r *RawChange) { r.Action = "add" }},
		{"with on a remove", func(r *RawChange) { r.Action = "remove" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := base
			tt.mutate(&r)
			if err := r.Validate(); err == nil {
				t.Fatalf("accepted %+v", r)
			}
		})
	}
}
