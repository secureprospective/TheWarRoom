package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/draft/drafttest"
	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/draftreports"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// TestDraftWireMatchesGolden freezes the Draft wire contract in testdata/wire/draft: what the
// frontend receives from every Draft binding, field for field. The bound methods run on a fresh
// app (nothing held, so no clock or time zone reaches the output); the held states build in UTC.
// Regenerate with TWR_WRITE_WIRE=1 go test -run TestDraftWireMatchesGolden . and review the diff.
func TestDraftWireMatchesGolden(t *testing.T) {
	t.Parallel()
	a := targetTestApp(t)
	board, err := a.TargetDraftReports()
	if err != nil {
		t.Fatal(err)
	}
	checkDraftWire(t, "reports-unfetched", board)
	checkDraftWire(t, "reports-held", testDraftBoard(t))
	checkDraftWire(t, "reports-stale", draftFixtureBoard(drafttest.Stale(t)))
	checkDraftWire(t, "reports-empty", buildDraftReports(2026, heldSeasonFeed[[]draftreports.Pick]{},
		heldSeasonFeed[[]draftreports.FuturePick]{refreshError: "down"}, nil, nil, nil, "", "", time.UTC))
	nfl, err := a.TargetNFLDraft()
	if err != nil {
		t.Fatal(err)
	}
	checkDraftWire(t, "nfl-unheld", nfl)
	checkDraftWire(t, "nfl-held", nflDraftReport(2026, heldSeasonFeed[[]players.RawPlayer]{value: []players.RawPlayer{
		{ID: "17463", Name: "Simpson, Ty", Position: "QB", Team: "LAR", DraftYear: "2026", DraftRound: "1", DraftPick: "13"},
		{ID: "17462", Name: "Mendoza, Fernando", Position: "QB", Team: "LVR", DraftYear: "2026", DraftRound: "1", DraftPick: "1"},
		{ID: "0816", Name: "Old rookie", Position: "WR", Team: "IND", DraftYear: "2025", DraftRound: "1", DraftPick: "2"},
		{ID: "0817", Name: "Undrafted", Position: "WR", Team: "IND", DraftYear: "2026"},
	}}))
	scope := RawDraftScope{From: 2017, To: 2026, Seasons: 3, Group: "DL"}
	checkDraftWire(t, "lab", a.TargetDraftLab(scope))
	scope.Segments, scope.NFL = []string{"r1a", "r3"}, []string{"n1"}
	checkDraftWire(t, "lab-filtered", a.TargetDraftLab(scope))
	scope.From, scope.To = 2100, 2101
	checkDraftWire(t, "lab-thin", a.TargetDraftLab(scope))
	scope.Seasons = 5
	checkDraftWire(t, "lab-invalid", a.TargetDraftLab(scope))
	errs := map[string]string{}
	if err := a.TargetOpenMFLDraftRoom(); err != nil {
		errs["openDraftRoom"] = err.Error()
	}
	checkDraftWire(t, "errors", errs)
}

// draftFixtureBoard is the draft reports board of a drafttest fixture, through the root converter.
func draftFixtureBoard(d drafttest.Draft) ReportBoard {
	held := func(f report.Feed) (time.Time, string, bool) { return f.FetchedAt, f.RefreshError, f.ReadPending }
	results := heldSeasonFeed[[]draftreports.Pick]{value: d.Picks}
	results.fetchedAt, results.refreshError, results.readPending = held(d.Read)
	future := heldSeasonFeed[[]draftreports.FuturePick]{value: d.Future}
	future.fetchedAt, future.refreshError, future.readPending = held(d.FutureRead)
	return buildDraftReports(2026, results, future, d.Names, d.Franchises, d.Values, d.ValueNote, d.Next, time.UTC)
}

func testDraftBoard(t *testing.T) ReportBoard {
	t.Helper()
	return draftFixtureBoard(drafttest.Held(t))
}

func reportByID(t *testing.T, board ReportBoard, id string) Report {
	t.Helper()
	for _, r := range board.Reports {
		if r.ID == id {
			return r
		}
	}
	t.Fatal(id)
	return Report{}
}

func checkDraftWire(t *testing.T, name string, v any) {
	t.Helper()
	got, err := json.MarshalIndent(v, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')
	path := "testdata/wire/draft/" + name + ".json"
	if os.Getenv("TWR_WRITE_WIRE") == "1" {
		if err := os.MkdirAll("testdata/wire/draft", 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil { //nolint:gosec // a test fixture
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(want, got) {
		t.Errorf("%s differs from the Draft binding's output; the wire contract changed", path)
	}
}

func TestDraftRefreshKeepsEachLastGoodCopy(t *testing.T) {
	t.Parallel()
	a, transport, _, _ := seasonTestApp(t)
	a.refreshDraftReports(a.ctx)
	if len(a.seasonDraftResults.value) != 160 || len(a.seasonFuturePicks.value) != 320 {
		t.Fatal("feeds not filled")
	}
	transport.failure = "draftResults"
	a.refreshDraftReports(a.ctx)
	if len(a.seasonDraftResults.value) != 160 || a.seasonDraftResults.refreshError == "" || a.seasonFuturePicks.refreshError != "" {
		t.Fatal("last good copy not retained independently")
	}
}

func TestDraftWireFixtureMatchesGo(t *testing.T) {
	t.Parallel()
	path := filepath.Join("frontend", "src", "app", "draftreports", "wire.testdata.json")
	body, err := json.MarshalIndent(testDraftBoard(t), "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	body = append(body, '\n')
	if os.Getenv("TWR_WRITE_WIRE") == "1" {
		if err := os.WriteFile(path, body, 0600); err != nil {
			t.Fatal(err)
		}
	}
	held, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(held) != string(body) {
		t.Fatal("draft wire fixture drift; regenerate with TWR_WRITE_WIRE=1")
	}
}

func TestDraftHeldReadingRefreshCacheAndMFLLink(t *testing.T) {
	t.Parallel()
	a, transport, _, _ := seasonTestApp(t)
	if err := a.TargetOpenMFLDraftRoom(); err == nil {
		t.Fatal("opened undiscovered host")
	}
	if _, err := a.TargetDraftReports(); err != nil || len(transport.requests) != 0 {
		t.Fatal("reading fetched", err)
	}
	if err := a.TargetDraftReportsRefresh(false); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(transport.requests, []string{"draftResults", "futureDraftPicks"}) {
		t.Fatal(transport.requests)
	}
	board, err := a.TargetDraftReports()
	if err != nil || len(reportByID(t, board, "M-063").Rows) != 160 || !strings.Contains(board.MFLURL, "O=17") {
		t.Fatal(board, err)
	}
	if err := a.TargetDraftReportsRefresh(false); err != nil || len(transport.requests) != 2 {
		t.Fatal("fresh copy refetched", err)
	}
	opened := ""
	a.openURL = func(_ context.Context, url string) { opened = url }
	if err := a.TargetOpenMFLDraftRoom(); err != nil || opened != board.MFLURL || len(transport.requests) != 2 {
		t.Fatal("link submitted or fetched", opened, err)
	}
	if err := a.TargetDraftReportsRefresh(true); err != nil || len(transport.requests) != 4 {
		t.Fatal("force refresh skipped", err)
	}
}

func TestDraftArchiveFallbackKeepsOriginalAsOf(t *testing.T) {
	t.Parallel()
	a, transport, _, _ := seasonTestApp(t)
	at := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	recordSeasonArchive(t, a, "draftResults", "", transport.bodies["draftResults"], at)
	transport.failure = "draftResults"
	a.refreshDraftReports(a.ctx)
	board, err := a.TargetDraftReports()
	if err != nil {
		t.Fatal(err)
	}
	results := reportByID(t, board, "M-063")
	if results.Status != "stale" || len(results.Rows) != 160 || !strings.Contains(results.AsOf, reportTime(at, time.Local)) ||
		!strings.Contains(results.Note, "last known data") || reportByID(t, board, "M-068").Status != "ok" {
		t.Fatal(results)
	}
}

func TestDraftValueNotePlainWords(t *testing.T) {
	t.Parallel()
	a, _, _, _ := seasonTestApp(t)
	at := time.Date(2026, 10, 5, 13, 44, 0, 0, time.Local)
	run := history.Run{ID: 3, AsOf: at, Engine: "secret-version"}
	want := "Engine run: October 5, 2026, 1:44 PM."
	if got := a.draftValueNote(run); got != want {
		t.Fatal(got)
	}
	run.MissingMeasures = []string{"context.birth_date"}
	got := a.draftValueNote(run)
	if !strings.Contains(got, "Missing player information:") || strings.Contains(got, "context.birth_date") || strings.Contains(got, "secret-version") || !strings.Contains(got, "Date of birth") || got == want {
		t.Fatal(got)
	}
}
