package personal

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
)

func TestRestartAndDraftOrder(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "app-dev.db")
	pools, err := db.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	store := New(pools)
	if err := store.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
	change := func(kind, id, action string) {
		t.Helper()
		r := RawChange{Mine: "0025", Kind: kind, Entity: "player", ID: id, Action: action}
		if err := store.Change(ctx, r, at); err != nil {
			t.Fatal(err)
		}
	}
	for _, kind := range []string{"watch", "bait", "draft"} {
		change(kind, "0816", "add")
	}
	change("watch", "0816", "add")
	change("draft", "12626", "add")
	change("draft", "15711", "add")
	change("draft", "15711", "up")
	change("draft", "0816", "down")
	for i, entity := range []string{"player", "franchise"} {
		r := RawChange{Mine: "0025", Kind: "note", Entity: entity, ID: []string{"0816", "0025"}[i], Action: "add", Text: "Plain <text>\nSecond line"}
		if err := store.Change(ctx, r, at); err != nil {
			t.Fatal(err)
		}
	}
	before, err := store.Read(ctx, "0025")
	if err != nil {
		t.Fatal(err)
	}
	if len(before) != 7 {
		t.Fatalf("idempotent markers and both notes: %+v", before)
	}
	if err := pools.Close(); err != nil {
		t.Fatal(err)
	}
	pools, err = db.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := pools.Close(); err != nil {
			t.Error(err)
		}
	}()
	store = New(pools)
	if err := store.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	after, err := store.Read(ctx, "0025")
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("restart: %+v %v", after, err)
	}
	ids := []string{}
	for _, entry := range after {
		if entry.Kind == "draft" {
			ids = append(ids, entry.ID)
		}
	}
	if !reflect.DeepEqual(ids, []string{"15711", "0816", "12626"}) {
		t.Fatalf("reorder: %v", ids)
	}
	change("watch", "0816", "remove")
	after, err = store.Read(ctx, "0025")
	if err != nil || len(after) != 6 {
		t.Fatalf("remove: %+v %v", after, err)
	}
	other, err := store.Read(ctx, "0001")
	if err != nil || len(other) != 0 {
		t.Fatalf("franchise leak: %v %v", other, err)
	}
}

func TestRejectInvalidChanges(t *testing.T) {
	base := RawChange{Mine: "0025", Kind: "watch", Entity: "player", ID: "0816", Action: "add"}
	for _, mutate := range []func(*RawChange){
		func(r *RawChange) { r.Mine = "25'; DELETE" },
		func(r *RawChange) { r.Kind = "waiver" },
		func(r *RawChange) { r.ID = "816" },
		func(r *RawChange) { r.Entity = "pick" },
		func(r *RawChange) { r.Action = "submit" },
		func(r *RawChange) { r.Action = "up" },
		func(r *RawChange) { r.Kind = "note"; r.Text = " " },
		func(r *RawChange) { r.Kind = "note"; r.Text = "A\x00B" },
	} {
		r := base
		mutate(&r)
		if err := r.Validate(); err == nil {
			t.Fatalf("accepted invalid input %+v", r)
		}
	}
}
