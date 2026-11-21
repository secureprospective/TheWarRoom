package main

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

func TestESPNNewsArchiveRoundTrip(t *testing.T) {
	ctx := t.Context()
	pools, err := db.Open(ctx, filepath.Join(t.TempDir(), "reference.db"), history.ReferenceSchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = pools.Close() }()
	if err := history.PrepareReferenceFile(ctx, pools); err != nil {
		t.Fatal(err)
	}
	reg, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	store := history.New(pools, reg)
	arch := espnNewsArchive{store: store}
	key := "https://site.api.espn.com/apis/site/v2/sports/football/nfl/news?limit=50"
	wantAt := time.Date(2026, 11, 21, 13, 0, 0, 0, time.UTC)
	payload := []byte(`{"articles":[]}`)

	if err := arch.SaveESPNBody(ctx, key, payload, wantAt); err != nil {
		t.Fatal(err)
	}
	got, gotAt, ok, err := arch.LoadESPNBody(ctx, key)
	if err != nil || !ok {
		t.Fatalf("LoadESPNBody ok=%v err=%v", ok, err)
	}
	if string(got) != string(payload) || !gotAt.Equal(wantAt) {
		t.Fatalf("round-trip mismatch: body=%q at=%v", string(got), gotAt)
	}
}
