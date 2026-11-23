package leagueservice

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

type workerFixedRank struct{ order []string }

func (r workerFixedRank) Rank(_ context.Context, _ string, _ int, pool []string) ([]string, error) {
	avail := make(map[string]bool, len(pool))
	for _, id := range pool {
		avail[id] = true
	}
	out := make([]string, 0, len(pool))
	for _, id := range r.order {
		if avail[id] {
			out = append(out, id)
		}
	}
	return out, nil
}

func TestDueWorkerAutoPicksThenClosesExhaustedDraft(t *testing.T) {
	t.Parallel()
	const league = "native.workerauto"
	clock := &dueTestClock{}
	clock.instant.Store(time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC).UnixNano())
	pools, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "history.db"), history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	store := history.New(pools, &measures.Registry{})
	if err := store.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	store.SetClock(clock.Now)
	facts := nativecheck.Facts{
		"1001": {Name: "Alpha", Position: leaguerules.QB, Team: "KC"},
		"1002": {Name: "Beta", Position: leaguerules.WR, Team: "MIN"},
	}

	exec := func(cmd string, args commissioner.RawArgs) {
		t.Helper()
		events, err := store.Events(t.Context(), league, commissioner.Stream, 0)
		if err != nil {
			t.Fatal(err)
		}
		raw := commissioner.RawCommand{League: league, Command: cmd, Args: args,
			Expected: int64(len(events)), Request: fmt.Sprintf("w.%d.%s", len(events), cmd),
			Reason: "Setup", Effective: "now", Visibility: "league"}
		_, err = store.ExecuteCommissionerWithFacts(t.Context(), raw, "creator", nil, facts, time.UTC)
		if err != nil {
			t.Fatalf("%s: %v", cmd, err)
		}
	}

	exec(commissioner.Create, commissioner.RawArgs{Name: "Worker Auto",
		Preset: commissioner.PresetDefault32, Source: commissioner.SourceNative, Season: 2026, FranchiseCount: 2})
	exec(commissioner.RulesSet, commissioner.RawArgs{Field: "draft.startupRounds", Value: json.RawMessage("1")})
	exec(commissioner.DraftStartupOpen, commissioner.RawArgs{
		Draft: &commissioner.RawDraftArgs{Season: 2026, Startup: true}})
	clock.instant.Add(int64(time.Second))

	worker := NewDueWorkerWithRanking(history.NewOneFile(store),
		func() nativecheck.Facts { return facts }, clock, workerFixedRank{order: []string{"1001", "1002"}})
	if _, err := worker.step(t.Context()); err != nil {
		t.Fatalf("worker step: %v", err)
	}

	reading, err := store.NativeDraft(t.Context(), commissioner.RawNativeDraftScope{
		League: league, Season: 2026, Franchise: "franchise.01", At: clock.Now(), Startup: true,
	}, time.UTC, facts)
	if err != nil {
		t.Fatal(err)
	}
	if !reading.Draft.Exhausted {
		t.Fatalf("draft not exhausted after worker step; onClock = %+v", reading.Draft.OnClock)
	}
	if reading.Closed == nil || !reading.Closed.Automatic {
		t.Fatalf("draft not closed automatically in same worker step: %+v", reading.Closed)
	}
}

// pinnedFactsEnv is a league store's installed football evidence: the only place a hosted server holds
// player facts.
type pinnedFactsEnv struct{ facts nativecheck.Facts }

func (e pinnedFactsEnv) PlayerFacts() nativecheck.Facts { return e.facts }
func (e pinnedFactsEnv) LineupFacts(context.Context, int, int) (commissioner.LineupFacts, error) {
	return commissioner.LineupFacts{}, nil
}

// The hosted server hands the worker no facts; the due auto-pick must read the league's pinned ones.
func TestDueWorkerAutoPicksFromPinnedFactsWhenWorkerHasNone(t *testing.T) {
	t.Parallel()
	const league = "native.workerpinned"
	clock := &dueTestClock{}
	clock.instant.Store(time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC).UnixNano())
	pools, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "history.db"), history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	store := history.New(pools, &measures.Registry{})
	if err := store.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	store.SetClock(clock.Now)
	facts := nativecheck.Facts{
		"1001": {Name: "Alpha", Position: leaguerules.QB, Team: "KC"},
		"1002": {Name: "Beta", Position: leaguerules.WR, Team: "MIN"},
	}

	exec := func(cmd string, args commissioner.RawArgs) {
		t.Helper()
		events, err := store.Events(t.Context(), league, commissioner.Stream, 0)
		if err != nil {
			t.Fatal(err)
		}
		raw := commissioner.RawCommand{League: league, Command: cmd, Args: args,
			Expected: int64(len(events)), Request: fmt.Sprintf("w.%d.%s", len(events), cmd),
			Reason: "Setup", Effective: "now", Visibility: "league"}
		_, err = store.ExecuteCommissionerWithFacts(t.Context(), raw, "creator", nil, facts, time.UTC)
		if err != nil {
			t.Fatalf("%s: %v", cmd, err)
		}
	}

	exec(commissioner.Create, commissioner.RawArgs{Name: "Worker Auto",
		Preset: commissioner.PresetDefault32, Source: commissioner.SourceNative, Season: 2026, FranchiseCount: 2})
	exec(commissioner.RulesSet, commissioner.RawArgs{Field: "draft.startupRounds", Value: json.RawMessage("1")})
	exec(commissioner.DraftStartupOpen, commissioner.RawArgs{
		Draft: &commissioner.RawDraftArgs{Season: 2026, Startup: true}})
	clock.instant.Add(int64(time.Second))

	store.SetLineupEnvironment(pinnedFactsEnv{facts: facts})
	worker := NewDueWorkerWithRanking(history.NewOneFile(store),
		func() nativecheck.Facts { return nil }, clock, workerFixedRank{order: []string{"1001", "1002"}})
	if _, err := worker.step(t.Context()); err != nil {
		t.Fatalf("worker step: %v", err)
	}

	reading, err := store.NativeDraft(t.Context(), commissioner.RawNativeDraftScope{
		League: league, Season: 2026, Franchise: "franchise.01", At: clock.Now(), Startup: true,
	}, time.UTC, facts)
	if err != nil {
		t.Fatal(err)
	}
	if !reading.Draft.Exhausted {
		t.Fatalf("draft not exhausted after worker step; onClock = %+v", reading.Draft.OnClock)
	}
}
