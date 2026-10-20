package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/modelrun"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
	"github.com/secureprospective/TheWarRoom/internal/store/rulebook"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// This opt-in benchmark reads disposable offline copies, never the app's live databases.
func TestOfflineEngineSnapshotTiming(t *testing.T) {
	t.Parallel()
	dir := os.Getenv("TWR_ENGINE_BENCH_DIR")
	if dir == "" {
		t.Skip("requires disposable offline snapshot")
	}
	ctx := context.Background()
	hp, err := db.Open(ctx, filepath.Join(dir, "history-bench.db"), history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := hp.Close(); err != nil {
			t.Error(err)
		}
	}()
	lp, err := db.Open(ctx, filepath.Join(dir, "thewarroom-bench.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := lp.Close(); err != nil {
			t.Error(err)
		}
	}()
	reg, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	h := history.New(hp, reg)
	if err := h.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	mirror := state.NewMirror(lp, rulebook.New(lp))
	if err := mirror.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := root.Close(); err != nil {
			t.Error(err)
		}
	}()
	body, err := root.ReadFile("players-bench.json")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := players.Parse(ctx, body)
	if err != nil {
		t.Fatal(err)
	}
	lookup, err := normalize.NewLookup(raw)
	if err != nil {
		t.Fatal(err)
	}
	owned := map[string]bool{}
	for _, fid := range mirror.Franchises() {
		roster, _ := mirror.Roster(fid)
		for _, p := range roster {
			owned[p.MFLID] = true
		}
	}
	// A stress universe, not a claim that MFL's held free-agent export lists all these players.
	var free []string
	for _, p := range raw {
		if _, known := lookup.Facts(p.ID); known && !owned[p.ID] {
			free = append(free, p.ID)
		}
	}
	runner, err := modelrun.New(mirror.Reader(), lookup, h, "offline-benchmark")
	if err != nil {
		t.Fatal(err)
	}
	spec := modelrun.Spec{Season: mirror.Season(), AsOf: time.Now(), LastWeek: 17, Params: params.DefaultSet()}
	before, err := runner.Run(ctx, spec)
	if err != nil {
		t.Fatal(err)
	}
	old, err := h.ModelScores(ctx, before.RunID)
	if err != nil {
		t.Fatal(err)
	}
	spec.FreeAgents = free
	started := time.Now()
	after, err := runner.Run(ctx, spec)
	elapsed := time.Since(started)
	if err != nil {
		t.Fatal(err)
	}
	next, err := h.ModelScores(ctx, after.RunID)
	if err != nil {
		t.Fatal(err)
	}
	fullBefore, fullAfter := map[string]history.ModelScore{}, map[string]history.ModelScore{}
	for _, s := range old {
		fullBefore[s.MFLID] = s
	}
	for _, s := range next {
		if owned[s.MFLID] {
			fullAfter[s.MFLID] = s
		}
	}
	a, err := json.Marshal(fullBefore)
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(fullAfter)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Fatal("full stored roster map changed")
	}
	t.Logf("offline populated snapshot: %s; %d rostered / %d free valued; %d rostered / %d free excluded; %d roster values byte-identical",
		elapsed, after.RosteredScored, after.FreeAgentScored, after.RosteredExcluded, after.FreeAgentExcluded, len(fullBefore))
}
