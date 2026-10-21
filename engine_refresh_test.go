package main

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
)

func TestEngineRunsWithLoadedInputsAndEmitsOnce(t *testing.T) {
	t.Parallel()
	a := targetTestApp(t)
	if err := a.core.Rulebook.Initialize(a.core.Ctx, weekRules{end: "17"}); err != nil {
		t.Fatal(err)
	}
	a.core.Params = params.New(a.core.Pools)
	if err := a.core.Params.Initialize(a.core.Ctx); err != nil {
		t.Fatal(err)
	}
	lk, err := normalize.NewLookup([]players.RawPlayer{
		{ID: "0042", Name: "Loaded, Rookie", Position: "WR", Team: "BUF"},
		{ID: "0043", Name: "Held, Free", Position: "WR", Team: "BUF", Status: "R"},
	})
	if err != nil {
		t.Fatal(err)
	}
	a.core.Lookup, a.core.HasLookup = lk, true
	free, err := playerid.New("0043")
	if err != nil {
		t.Fatal(err)
	}
	rostered, err := playerid.New("0042")
	if err != nil {
		t.Fatal(err)
	}
	a.core.SeasonFreeAgents.Value = []playerid.PlayerID{free, free, rostered}
	for _, batch := range []measures.Batch{
		{Source: "nflverse", Facts: []measures.Fact{{IDType: "mfl", ID: "0042", Field: "players.position", Raw: "WR"}}},
		{Source: "mfl", Facts: []measures.Fact{{IDType: "mfl", ID: "0042", Season: 2024,
			Field: "playerScores.score", Raw: "100"}}},
	} {
		if _, err := a.core.History.Ingest(a.core.Ctx, batch); err != nil {
			t.Fatal(err)
		}
	}
	var snaps []measures.Fact
	for week := 1; week <= 16; week++ {
		snaps = append(snaps, measures.Fact{IDType: "mfl", ID: "0042",
			Season: 2024, Week: week, Field: "snap_counts.offense_snaps", Raw: "40"})
	}
	if _, err := a.core.History.Ingest(a.core.Ctx, measures.Batch{Source: "nflverse", Facts: snaps}); err != nil {
		t.Fatal(err)
	}

	emitted := 0
	a.core.SeasonChanged = func(context.Context) { emitted++ }
	a.core.ModelMu.Lock()
	defer a.core.ModelMu.Unlock()
	a.checkEngine(a.core.Ctx)
	first, held, err := a.core.History.LatestRun(a.core.Ctx, a.core.Season, history.RunModel)
	if err != nil || !held || emitted != 1 {
		t.Fatalf("first: %+v %t %v events %d", first, held, err, emitted)
	}
	scores, err := a.core.History.ModelScores(a.core.Ctx, first.ID)
	if err != nil || len(scores) != 2 || scores[0].MFLID != "0042" || scores[1].MFLID != "0043" {
		t.Fatalf("scores: %+v %v", scores, err)
	}
	a.checkEngine(a.core.Ctx)
	if emitted != 1 {
		t.Fatal("fresh unchanged inputs scored")
	}
	a.core.ModelDirty.Store(true)
	a.checkEngine(a.core.Ctx)
	next, held, err := a.core.History.LatestRun(a.core.Ctx, a.core.Season, history.RunModel)
	if err != nil || !held || emitted != 2 || next.ID == first.ID {
		t.Fatalf("changed: %+v %v events %d", next, err, emitted)
	}
}

func TestBackgroundEngineSkipsWhileTheEngineRuns(t *testing.T) {
	t.Parallel()
	a := targetTestApp(t)
	a.core.Params = params.New(a.core.Pools)
	a.core.ModelMu.Lock()
	defer a.core.ModelMu.Unlock()
	done := make(chan struct{})
	go func() { a.refreshEngine(a.core.Ctx); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("background refresh waited on a running engine")
	}
}

func TestC07ListsEngineRunAndAge(t *testing.T) {
	t.Parallel()
	a := targetTestApp(t)
	nr := history.NewModelRun{Season: a.core.Season, AsOf: time.Now().Add(-2 * time.Hour), Engine: "fixture",
		InputsHash: "fixture", Params: history.ParamSet{Params: params.DefaultSet().Model()},
		Scores: []history.ModelScore{{MFLID: "0042", Position: "WR"}}}
	if _, _, err := a.core.History.WriteModelRun(a.core.Ctx, nr); err != nil {
		t.Fatal(err)
	}
	in, err := a.healthInput(a.core.Ctx)
	if err != nil {
		t.Fatal(err)
	}
	report := healthReport(in)
	for _, row := range report.Rows {
		if row.Cells[0] == "Engine run" {
			if !strings.Contains(row.Cells[1], whenText(nr.AsOf, time.Local)) ||
				!strings.Contains(row.Cells[1], sourceAge(healthRow{lastGood: nr.AsOf}, in.now)) {
				t.Fatalf("engine date or age missing: %+v", row)
			}
			return
		}
	}
	t.Fatal("C-07 has no engine run")
}

func TestEngineRefreshIsWiredAfterInputs(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		path  string
		calls []string
	}{
		{"refresh_app.go", []string{"a.launchRefresh(parent)", "a.refreshWeeksInBackground(parent)"}},
		{"week_app.go", []string{"a.refreshSeason(ctx)", "a.refreshFinishedScores(ctx)", "a.refreshEngine(parent)", "case <-ticker.C:", "a.launchRefresh(parent)"}},
		{"engine_refresh.go", []string{"a.loadSignals(ctx)", "a.loadSeasonPoints(ctx)", "a.refreshFreeAgents(ctx)", "a.checkEngine(ctx)"}},
	} {
		body, err := os.ReadFile(tc.path)
		if err != nil {
			t.Fatal(err)
		}
		from := 0
		for _, call := range tc.calls {
			index := strings.Index(string(body[from:]), call)
			if index < 0 {
				t.Fatalf("%s: missing or out-of-order %s", tc.path, call)
			}
			from += index + len(call)
		}
	}
}

func TestSourceHealthEventOnFailedManualRefresh(t *testing.T) {
	t.Parallel()
	a := targetTestApp(t)
	events := 0
	a.core.SeasonChanged = func(context.Context) { events++ }
	var err error
	a.core.MFLClient, err = mfl.New("", 1000, mfl.WithTransport(&directoryDown{}))
	if err != nil {
		t.Fatal(err)
	}
	a.TargetDataRefresh("league")
	if events != 1 {
		t.Fatal("failed refresh did not notify source health", events)
	}
}
