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

	"github.com/secureprospective/TheWarRoom/internal/facts"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/store/history"

	sourcesapp "github.com/secureprospective/TheWarRoom/internal/app/sources"
)

func TestSourcesBindingGolden(t *testing.T) {
	a := targetTestApp(t)
	a.dataDir = t.TempDir()
	a.crosswalk = CrosswalkReport{OK: true, LoadedAt: "2026-10-11T18:00:00Z", Links: 1000, Promoted: 12,
		Rates: []CrosswalkRate{{Position: "QB", Rostered: 4, RosteredMatched: 3}}, Total: CrosswalkRate{Position: "ALL", Rostered: 4, RosteredMatched: 3},
		Unmatched: []CrosswalkMiss{{MFLID: "0042", Name: "Waiting", Position: "QB", Reason: "no NFL id yet"}}}
	ds, err := a.TargetDataSources()
	if err != nil {
		t.Fatal(err)
	}
	ds.Health.AsOf = "As of: fixture clock"
	// The not-ready load exercises the binding without a network request.
	stopped := &App{started: make(chan struct{}), startupErr: errors.New("fixture startup failed")}
	close(stopped.started)
	values := map[string][]byte{
		"LoadCrosswalk":       sourcesWire(t, stopped.LoadCrosswalk()),
		"GetCrosswalkReport":  sourcesWire(t, a.GetCrosswalkReport()),
		"TargetCrosswalk":     sourcesWire(t, a.TargetCrosswalk()),
		"TargetDataSources":   sourcesWire(t, ds),
		"TargetDataRefresh":   sourcesWire(t, a.TargetDataRefresh("unknown")),
		"SetLeagueArchiveDir": sourcesWire(t, a.SetLeagueArchiveDir("relative")),
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

func TestSourcesWireTwins(t *testing.T) {
	t.Parallel()
	c := sourcesapp.CrosswalkReport{OK: true, Error: "error", LoadedAt: "at", Links: 12, Promoted: 3,
		Rates:     []sourcesapp.CrosswalkRate{{Position: "QB", Rostered: 4, RosteredMatched: 3, FreeAgents: 2, FreeAgentsMatched: 1}},
		Total:     sourcesapp.CrosswalkRate{Position: "ALL", Rostered: 8, RosteredMatched: 7, FreeAgents: 6, FreeAgentsMatched: 5},
		Unmatched: []sourcesapp.CrosswalkMiss{{MFLID: "0042", Name: "Name", Team: "BUF", Position: "QB", Franchise: "0001", Reason: "reason"}}}
	s := sourcesapp.SignalsReport{OK: true, Error: "error", StartedAt: "start", FinishedAt: "finish", CrosswalkNote: "note",
		Loads: []sourcesapp.SignalLoad{{Feed: "feed", Season: 2026, Status: "loaded", Rows: 1, Facts: 2, Added: 3, Unresolved: 4, Missing: []string{"missing"}, Error: "error"}}}
	for _, c := range []sourcesapp.CrosswalkReport{c, {}, {Rates: []sourcesapp.CrosswalkRate{}, Unmatched: []sourcesapp.CrosswalkMiss{}}} {
		if string(sourcesWire(t, c)) != string(sourcesWire(t, wireCrosswalk(c))) || string(sourcesWire(t, c)) != string(sourcesWire(t, serviceCrosswalk(wireCrosswalk(c)))) {
			t.Fatal("crosswalk twin lost a field or slice shape")
		}
	}
	for _, s := range []sourcesapp.SignalsReport{s, {}, {Loads: []sourcesapp.SignalLoad{}}} {
		if string(sourcesWire(t, s)) != string(sourcesWire(t, wireSignals(s))) || string(sourcesWire(t, s)) != string(sourcesWire(t, serviceSignals(wireSignals(s)))) {
			t.Fatal("signals twin lost a field or slice shape")
		}
	}
}

func TestSourcesCompileRetainsRootLock(t *testing.T) {
	a := targetTestApp(t)
	a.factsMu.Lock()
	defer a.factsMu.Unlock()
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
	a.crosswalk = CrosswalkReport{OK: true, LoadedAt: "2026-10-11T18:00:00Z", Links: 1000,
		Rates: []CrosswalkRate{{Position: "WR", Rostered: 4, RosteredMatched: 3}},
		Total: CrosswalkRate{Position: "ALL", Rostered: 4, RosteredMatched: 3},
		Unmatched: []CrosswalkMiss{{MFLID: "0816", Name: "Gosnell, Stephen", Team: "MIN", Position: "WR",
			Franchise: "Cincinnati Bengals", Reason: "no NFL id yet"}}}
	c := a.TargetCrosswalk()
	wire := struct {
		Data      DataSources   `json:"data"`
		Crosswalk Report        `json:"crosswalk"`
		Params    ParamsResult  `json:"params"`
		Result    DataActResult `json:"result"`
	}{
		Data: DataSources{DatabaseChecks: (&App{}).databaseChecksReport(), Health: wireReport(sourcesapp.HealthReport(sourcesapp.HealthInput{Reg: reg, Now: at, Zone: time.UTC,
			Sources: []history.SourceHealth{{Source: leagueArchiveSource, State: history.SourceActive, LastSuccess: at}}})),
			Signals: wireReport(sourcesapp.SignalFilesReport(sourcesapp.SignalsView{Feeds: []sourcesapp.FeedView{{Feed: "stats_player_week", Source: "nflverse",
				LastLoaded: "2026-10-11T18:00:00Z", Fresh: true, CoverageSeason: 2026,
				Seasons:  []sourcesapp.SeasonCount{{Season: 2026, Players: 1300, Waiting: 37}},
				Coverage: []sourcesapp.PositionShare{{Position: "WR", Rostered: 1454, WithData: 1105}}}}}, sourcesapp.SignalsReport{}, time.UTC)),
			Facts: wireReport(sourcesapp.FactsReport(sourcesapp.FactsInput{LabBuilt: "2026-10-06T15:57:20+00:00", LabThrough: "2026-10-06T01:42:57+00:00", Zone: time.UTC})),
			Checks: wireReport(sourcesapp.ChecksReport(sourcesapp.FactsInput{Zone: time.UTC, Latest: &history.SnapshotRecord{BuiltAt: at}, Results: []facts.CheckResult{
				{Name: "Win-loss records match MFL standings", Detail: "MFL counted playoff games in the 2013 standings.",
					Checked: 448, Agreed: 446, Explained: 1, Review: 1, Passed: true, Items: []facts.CheckItem{
						{Key: "2013/0001", Words: "2013 Buffalo Bills: MFL standings 11-3-0 count its playoff games (regular season 11-2-0).",
							Class: facts.ClassExplained, Reason: "In 2013 MFL counted playoff games in the standings."},
						{Key: "2014/0009", Words: "2014 Houston Texans: MFL standings 9-4-0, weekly games 9-3-0.",
							Class: facts.ClassReview, Reason: "Possibly a commissioner adjustment."}}},
				{Name: "Lineup scores equal MFL player scores", Checked: 202954, Agreed: 202954, Passed: true}}})),
			ArchiveDir: "/fixture/raw", PhotosOn: true, Scopes: wireScopes(sourcesapp.DataScopes()), Words: sourcesapp.DataWords()},
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

func TestCrosswalkReportKeepsLeadingZeroIDsAndGoWording(t *testing.T) {
	t.Parallel()
	a := targetTestApp(t)
	if r := a.TargetCrosswalk(); r.Status != "empty" || !strings.Contains(r.Note, "No crosswalk loaded") {
		t.Fatalf("empty crosswalk = %+v", r)
	}
	a.crosswalk = CrosswalkReport{OK: true, LoadedAt: "2026-10-11T18:00:00Z", Links: 1000, Promoted: 12,
		Rates:     []CrosswalkRate{{Position: "QB", Rostered: 4, RosteredMatched: 3, FreeAgents: 0}},
		Total:     CrosswalkRate{Position: "ALL", Rostered: 4, RosteredMatched: 3},
		Unmatched: []CrosswalkMiss{{MFLID: "0042", Name: "Waiting", Team: "BUF", Position: "QB", Franchise: "0001", Reason: "no NFL id yet"}}}
	r := a.TargetCrosswalk()
	if r.Note != "1,000 links; 12 waiting facts resolved." || r.Rows[0].Cells[1] != "3 of 4 · 75.0%" ||
		r.Rows[0].Cells[2] != "None" || r.Rows[2].Cells[0] != "Waiting" || r.Rows[2].Cells[1] != "QB · BUF" {
		t.Fatalf("crosswalk = %+v", r)
	}
}
