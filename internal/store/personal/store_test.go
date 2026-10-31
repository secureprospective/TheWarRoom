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
		func(r *RawChange) { r.Kind = "note"; r.Action = "remove" },
		func(r *RawChange) { r.Kind = "note"; r.Action = "remove"; r.At = "not-a-time" },
		func(r *RawChange) { r.At = "2026-10-11T12:00:00Z" },
	} {
		r := base
		mutate(&r)
		if err := r.Validate(); err == nil {
			t.Fatalf("accepted invalid input %+v", r)
		}
	}
}

func TestPlayerNoteThreads(t *testing.T) {
	player := "0816"
	t0 := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	note := func(text string) RawChange {
		return RawChange{Kind: "note", Entity: "player", ID: player, Action: "add", Text: text}
	}
	change := func(r RawChange) RawChange { r.Mine = "0025"; return r }
	remove := func(at time.Time) RawChange {
		return change(RawChange{Kind: "note", Entity: "player", ID: player, Action: "remove",
			At: at.UTC().Format(time.RFC3339Nano)})
	}
	watch := change(RawChange{Kind: "watch", Entity: "player", ID: player, Action: "add"})
	draft := change(RawChange{Kind: "draft", Entity: "player", ID: player, Action: "add"})
	cases := []struct {
		name  string
		steps []struct {
			change RawChange
			at     time.Time
		}
		readMine string
		want     []string
		kinds    map[string]int
	}{
		{
			name: "append keeps every dated note newest first",
			steps: []struct {
				change RawChange
				at     time.Time
			}{{change(note("first")), t0}, {change(note("second")), t0.Add(time.Minute)}, {change(note("third")), t0.Add(2 * time.Minute)}},
			want: []string{"third", "second", "first"},
		},
		{
			name: "delete one note by its date",
			steps: []struct {
				change RawChange
				at     time.Time
			}{{change(note("first")), t0}, {change(note("second")), t0.Add(time.Minute)},
				{remove(t0.Add(time.Minute)), t0.Add(2 * time.Minute)}, {change(note("third")), t0.Add(3 * time.Minute)}},
			want: []string{"third", "first"},
		},
		{
			name: "other kinds are unchanged by note changes",
			steps: []struct {
				change RawChange
				at     time.Time
			}{{watch, t0}, {draft, t0}, {change(note("kept")), t0}, {remove(t0), t0.Add(time.Minute)}},
			want:  []string{},
			kinds: map[string]int{"watch": 1, "draft": 1},
		},
		{
			name: "another franchise cannot read the thread",
			steps: []struct {
				change RawChange
				at     time.Time
			}{{change(note("private")), t0}},
			readMine: "0001",
			want:     []string{},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			pools, err := db.Open(ctx, filepath.Join(t.TempDir(), "app-dev.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := pools.Close(); err != nil {
					t.Error(err)
				}
			}()
			store := New(pools)
			if err := store.Initialize(ctx); err != nil {
				t.Fatal(err)
			}
			for _, step := range tt.steps {
				if err := store.Change(ctx, step.change, step.at); err != nil {
					t.Fatal(err)
				}
			}
			mine := tt.readMine
			if mine == "" {
				mine = "0025"
			}
			thread, err := store.PlayerNotes(ctx, mine, player)
			if err != nil {
				t.Fatal(err)
			}
			got := []string{}
			for _, entry := range thread.Notes {
				got = append(got, entry.Text)
			}
			if thread.Count != len(tt.want) || !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("thread: count=%d notes=%v want=%v", thread.Count, got, tt.want)
			}
			if tt.kinds == nil {
				return
			}
			entries, err := store.Read(ctx, "0025")
			if err != nil {
				t.Fatal(err)
			}
			seen := map[string]int{}
			for _, entry := range entries {
				seen[entry.Kind]++
			}
			for kind, count := range tt.kinds {
				if seen[kind] != count {
					t.Fatalf("%s changed: %v", kind, seen)
				}
			}
		})
	}
}
