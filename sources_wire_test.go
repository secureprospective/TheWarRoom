package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	raceapp "github.com/secureprospective/TheWarRoom/internal/app/race"
	"github.com/secureprospective/TheWarRoom/internal/app/report"

	"github.com/secureprospective/TheWarRoom/internal/facts"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/store/history"

	"github.com/secureprospective/TheWarRoom/internal/app/core"
	sourcesapp "github.com/secureprospective/TheWarRoom/internal/app/sources"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
)

func TestSourcesBindingGolden(t *testing.T) {
	a := targetTestApp(t)
	a.core.Paths.DataDir = t.TempDir()
	a.sourcesHeld.PublishCrosswalk(sourcesapp.CrosswalkReport{OK: true, LoadedAt: "2026-10-11T18:00:00Z", Links: 1000, Promoted: 12,
		Rates: []sourcesapp.CrosswalkRate{{Position: "QB", Rostered: 4, RosteredMatched: 3}}, Total: sourcesapp.CrosswalkRate{Position: "ALL", Rostered: 4, RosteredMatched: 3},
		Unmatched: []sourcesapp.CrosswalkMiss{{MFLID: "0042", Name: "Waiting", Position: "QB", Reason: "no NFL id yet"}}})
	ds, err := sourcesapp.New(a.core, a.sourcesHeld).TargetDataSources()
	if err != nil {
		t.Fatal(err)
	}
	ds.Health.AsOf = "As of: fixture clock"
	// The not-ready load exercises the binding without a network request.
	stopped := &App{raceHeld: &raceapp.Held{}, sourcesHeld: &sourcesapp.Held{}, core: &core.State{Started: make(chan struct{}), StartupErr: errors.New("fixture startup failed")}}
	close(stopped.core.Started)
	values := map[string][]byte{
		"LoadCrosswalk":       sourcesWire(t, sourcesapp.New(stopped.core, stopped.sourcesHeld).LoadCrosswalk()),
		"GetCrosswalkReport":  sourcesWire(t, sourcesapp.New(a.core, a.sourcesHeld).GetCrosswalkReport()),
		"TargetCrosswalk":     sourcesWire(t, sourcesapp.New(a.core, a.sourcesHeld).TargetCrosswalk()),
		"TargetDataSources":   sourcesWire(t, ds),
		"TargetDataRefresh":   sourcesWire(t, sourcesapp.New(a.core, a.sourcesHeld).TargetDataRefresh("unknown")),
		"SetLeagueArchiveDir": sourcesWire(t, sourcesapp.New(a.core, a.sourcesHeld).SetLeagueArchiveDir("relative")),
	}
	for name, body := range values {
		path := filepath.Join("testdata", "wire", "sources", name+".json")
		if os.Getenv("TWR_WRITE_WIRE") == "1" {
			if e := os.MkdirAll(filepath.Dir(path), 0o750); e != nil {
				t.Fatal(e)
			}
			if e := os.WriteFile(path, body, 0o600); e != nil {
				t.Fatal(e)
			}
		}
		held, e := os.ReadFile(path)
		if e != nil {
			t.Fatal(e)
		}
		if string(held) != string(body) {
			t.Errorf("%s binding wire changed", name)
		}
	}
}

func sourcesWire[T any](t *testing.T, value T) []byte {
	t.Helper()
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return append(b, '\n')
}

func healthReport(in sourcesHealthInput) Report {
	return wireReport(sourcesapp.HealthReport(in.HealthInput))
}
func whenText(at time.Time, zone *time.Location) string {
	return at.In(zone).Format("Jan 2, 2006, 3:04 PM")
}

type healthRow struct{ lastGood time.Time }

func sourceAge(row healthRow, now time.Time) string {
	return sourcesapp.PlainAge(now.Sub(row.lastGood))
}

func TestSourcesCompileRetainsRootLock(t *testing.T) {
	a := targetTestApp(t)
	a.core.FactsMu.Lock()
	defer a.core.FactsMu.Unlock()
	msg, err := a.compileFacts(context.Background())
	if err != nil || msg != "The checked facts are already compiling." {
		t.Fatalf("compile while held = %q, %v", msg, err)
	}
}

func TestDataSourcesWireMatchesGo(t *testing.T) {
	t.Parallel()
	reg, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 10, 11, 18, 0, 0, 0, time.UTC)
	a := targetTestApp(t)
	a.sourcesHeld.PublishCrosswalk(sourcesapp.CrosswalkReport{OK: true, LoadedAt: "2026-10-11T18:00:00Z", Links: 1000,
		Rates: []sourcesapp.CrosswalkRate{{Position: "WR", Rostered: 4, RosteredMatched: 3}},
		Total: sourcesapp.CrosswalkRate{Position: "ALL", Rostered: 4, RosteredMatched: 3},
		Unmatched: []sourcesapp.CrosswalkMiss{{MFLID: "0816", Name: "Gosnell, Stephen", Team: "MIN", Position: "WR",
			Franchise: "Cincinnati Bengals", Reason: "no NFL id yet"}}})
	c := sourcesapp.New(a.core, a.sourcesHeld).TargetCrosswalk()
	wire := struct {
		Data      sourcesapp.DataSources `json:"data"`
		Crosswalk report.Report          `json:"crosswalk"`
		Params    ParamsResult           `json:"params"`
		Result    DataActResult          `json:"result"`
	}{
		Data: sourcesapp.DataSources{DatabaseChecks: (&App{core: &core.State{}}).databaseChecksReading(), Health: sourcesapp.HealthReport(sourcesapp.HealthInput{Reg: reg, Now: at, Zone: time.UTC,
			Sources: []history.SourceHealth{{Source: sourcesapp.LeagueArchiveSource, State: history.SourceActive, LastSuccess: at}}}),
			Signals: sourcesapp.SignalFilesReport(sourcesapp.SignalsView{Feeds: []sourcesapp.FeedView{{Feed: "stats_player_week", Source: "nflverse",
				LastLoaded: "2026-10-11T18:00:00Z", Fresh: true, CoverageSeason: 2026,
				Seasons:  []sourcesapp.SeasonCount{{Season: 2026, Players: 1300, Waiting: 37}},
				Coverage: []sourcesapp.PositionShare{{Position: "WR", Rostered: 1454, WithData: 1105}}}}}, sourcesapp.SignalsReport{}, time.UTC),
			Facts: sourcesapp.FactsReport(sourcesapp.FactsInput{LabBuilt: "2026-10-06T15:57:20+00:00", LabThrough: "2026-10-06T01:42:57+00:00", Zone: time.UTC}),
			Checks: sourcesapp.ChecksReport(sourcesapp.FactsInput{Zone: time.UTC, Latest: &history.SnapshotRecord{BuiltAt: at}, Results: []facts.CheckResult{
				{Name: "Win-loss records match MFL standings", Detail: "MFL counted playoff games in the 2013 standings.",
					Checked: 448, Agreed: 446, Explained: 1, Review: 1, Passed: true, Items: []facts.CheckItem{
						{Key: "2013/0001", Words: "2013 Buffalo Bills: MFL standings 11-3-0 count its playoff games (regular season 11-2-0).",
							Class: facts.ClassExplained, Reason: "In 2013 MFL counted playoff games in the standings."},
						{Key: "2014/0009", Words: "2014 Houston Texans: MFL standings 9-4-0, weekly games 9-3-0.",
							Class: facts.ClassReview, Reason: "Possibly a commissioner adjustment."}}},
				{Name: "Lineup scores equal MFL player scores", Checked: 202954, Agreed: 202954, Passed: true}}}),
			ArchiveDir: "/fixture/raw", PhotosOn: true, Scopes: sourcesapp.DataScopes(), Words: sourcesapp.DataWords()},
		Crosswalk: c, Params: ParamsResult{OK: true, Params: []ParamView{{Key: "test.weight", Label: "Test weight", Group: "Other settings", Position: "WR", Value: 0.5, Default: 0.5, Min: 0, Max: 1, Description: "Test weight", Overridden: true}}},
		Result: DataActResult{OK: true, Summary: "Looked at 8 files: 0 new, 8 already held."},
	}
	body, err := json.MarshalIndent(wire, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	body = append(body, '\n')
	path := filepath.Join("frontend", "src", "app", "datasources", "wire.testdata.json")
	if os.Getenv("TWR_WRITE_WIRE") == "1" {
		if err := os.WriteFile(path, body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	held, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != string(held) {
		t.Fatal("data sources wire stale; regenerate with TWR_WRITE_WIRE=1")
	}
}

func TestEveryEngineParameterHasGoPlainWordsAndOrderedAreas(t *testing.T) {
	t.Parallel()
	a := targetTestApp(t)
	a.core.Params = params.New(a.core.Pools)
	if err := a.core.Params.Initialize(a.core.Ctx); err != nil {
		t.Fatal(err)
	}
	result := a.GetParams()
	if !result.OK || len(result.Params) < 100 {
		t.Fatalf("params = %+v", result)
	}
	previous := ""
	for _, p := range result.Params {
		if p.Label == "" || p.Group == "" || p.Group == "Other settings" ||
			strings.ContainsAny(p.Label, "._") || strings.Contains(p.Label, "model") {
			t.Errorf("parameter %s has no plain label and area: %q / %q", p.Key, p.Label, p.Group)
		}
		if p.Group < previous {
			t.Errorf("areas not grouped: %q after %q", p.Group, previous)
		}
		previous = p.Group
		if p.Key == "captier.cold_ceiling_pct" && p.Label != "Cold salary tier ceiling (% of cap)" {
			t.Errorf("cold ceiling = %q", p.Label)
		}
		if p.Key == "l4.breakout.athletic_lift" && (p.Label != "how far athletic rating lifts breakout age and age trajectory" ||
			p.Group != "Scouting adjustment · Breakout (inert, no run reads it)") {
			t.Errorf("breakout words = %q / %q", p.Label, p.Group)
		}
	}
}

func TestSourcesManifestUsesOneHolderPerApp(t *testing.T) {
	first, second := NewApp(), NewApp()
	if first.sourcesHeld == nil || second.sourcesHeld == nil || first.sourcesHeld == second.sourcesHeld {
		t.Fatal("holder must be allocated once per App in common construction")
	}
	close(first.core.Started)
	close(second.core.Started)
	boundAPI := func(app *App) *sourcesapp.API {
		t.Helper()
		for _, bound := range bindings(app) {
			if api, ok := bound.(*sourcesapp.API); ok {
				return api
			}
		}
		t.Fatal("Sources API not bound")
		return nil
	}
	first.sourcesHeld.PublishCrosswalk(sourcesapp.CrosswalkReport{OK: true, Links: 42})
	if boundAPI(first).GetCrosswalkReport().Links != 42 || boundAPI(second).GetCrosswalkReport().OK {
		t.Fatal("bound API did not use its App's common holder")
	}
}
