package main

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/draftreports"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
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
	at := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	pending := draftFixturePick(t, 1, 2, "0001", "0532", 0)
	pending.Player, pending.At = nil, time.Time{}
	skipped := draftFixturePick(t, 1, 1, "0002", "0533", 100)
	skipped.Player, skipped.Skipped, skipped.Comments = nil, true, "From One"
	checkDraftWire(t, "reports-stale", buildDraftReports(2026,
		heldSeasonFeed[[]draftreports.Pick]{value: []draftreports.Pick{pending, skipped}, fetchedAt: at, refreshError: "timeout"},
		heldSeasonFeed[[]draftreports.FuturePick]{value: []draftreports.FuturePick{}, fetchedAt: at, readPending: true},
		nil, map[string]string{"0001": "One"}, nil, "Engine points unavailable.", "Next draft: no date set by the commissioner.", time.UTC))
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
