package main

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// Opt-in, offline measurement over a disposable SQLite backup, never the live database.
func TestArchiveMeasurement(t *testing.T) {
	t.Parallel()
	dir, path := os.Getenv("TWR_ARCHIVE_MEASURE_DIR"), os.Getenv("TWR_ARCHIVE_MEASURE_DB")
	if dir == "" || path == "" {
		t.Skip("archive measurement not requested")
	}
	if !strings.HasSuffix(path, "history-measure.db") {
		t.Fatal("measurement requires the disposable history-measure.db")
	}
	ctx := context.Background()
	pools, err := db.Open(ctx, path, history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := pools.Close(); err != nil {
			t.Error(err)
		}
	}()
	reg, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	store := history.New(pools, reg)
	if err := store.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	a := NewApp()
	a.history = store
	for i := range 2 {
		started := time.Now()
		rep, err := a.importArchiveFrom(ctx, dir)
		if err != nil {
			t.Fatal(err)
		}
		elapsed := time.Since(started)
		b, err := json.Marshal(rep)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("run %d: %s, %s", i+1, elapsed, b)
		if i == 1 && rep.Recorded != 0 {
			t.Fatal("second import was not idempotent")
		}
	}
	m, _, err := a.publishLabManifest(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("manifest %s: %d files, %d missing: %+v", m.SHA256, len(m.Entries), len(m.Missing()), m.Missing())
	if _, err := m.Resolve(ctx, store); err != nil {
		t.Logf("resolution: %v", err)
	}
}
