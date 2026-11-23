package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/model"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// The commands carry a desktop model run to a stopped hosted server's reference file, once.
func TestExportModelRunThenImportModelScores(t *testing.T) {
	ctx := t.Context()
	src := filepath.Join(t.TempDir(), "history.db")
	pools, err := db.Open(ctx, src, history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	reg, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	store := history.New(pools, reg)
	if err := store.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.WriteModelRun(ctx, history.NewModelRun{Season: 2026, AsOf: time.Date(2026, 10, 29, 0, 0, 0, 0, time.UTC),
		Params: history.ParamSet{Params: map[string]float64{"a": 1}, Measures: []string{}}, Engine: "test", InputsHash: "h",
		Scores: []history.ModelScore{{MFLID: "0800", Position: "QB", Value: model.Value{DynastyPPG: 9}}, {MFLID: "1002", Position: "WR"}}}); err != nil {
		t.Fatal(err)
	}
	if err := pools.Close(); err != nil {
		t.Fatal(err)
	}
	exported := filepath.Join(t.TempDir(), "run.json")
	var out bytes.Buffer
	if err := runExportModelRun(ctx, []string{"-db", src, "-out", exported}, &out); err != nil || !strings.Contains(out.String(), "2 scores") {
		t.Fatalf("export: %v %q", err, out.String())
	}
	dir := t.TempDir()
	files, err := openSignInFiles(ctx, filepath.Join(dir, "server.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := files.Close(); err != nil {
		t.Fatal(err)
	}
	args := []string{"-db", filepath.Join(dir, "server.db"), "-file", exported}
	out.Reset()
	if err := runImportModelScores(ctx, args, &out); err != nil || !strings.Contains(out.String(), "Imported run 1, season 2026, 2 scores") {
		t.Fatalf("import: %v %q", err, out.String())
	}
	out.Reset()
	if err := runImportModelScores(ctx, args, &out); err != nil || !strings.Contains(out.String(), "already held") {
		t.Fatalf("second import: %v %q", err, out.String())
	}
}
