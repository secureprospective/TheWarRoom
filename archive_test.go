package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/facts"
)

// reducedArchive compiles the reduced fixture archive the facts parity test uses.
func reducedArchive(t *testing.T) archiveCache {
	t.Helper()
	raw, err := os.ReadFile("internal/lab/testdata/facts/inputs.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	gz, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(gz)
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Dictionary []string `json:"dictionary"`
		Inputs     []struct {
			File   string `json:"file"`
			SHA256 string `json:"sha256"`
			Bytes  int64  `json:"bytes"`
			Tokens []int  `json:"tokens"`
		} `json:"inputs"`
	}
	if err := json.Unmarshal(body, &fixture); err != nil {
		t.Fatal(err)
	}
	inputs := []facts.ResolvedInput{}
	for _, in := range fixture.Inputs {
		var b []byte
		for _, token := range in.Tokens {
			b = append(b, fixture.Dictionary[token]...)
		}
		inputs = append(inputs, facts.ResolvedInput{File: in.File, SHA256: in.SHA256, Bytes: in.Bytes, Body: b})
	}
	curated, err := os.ReadFile("internal/lab/testdata/note_readings.csv")
	if err != nil {
		t.Fatal(err)
	}
	notes, err := facts.ParseCurated(curated)
	if err != nil {
		t.Fatal(err)
	}
	compiled, err := facts.Compile(t.Context(), inputs, notes)
	if err != nil {
		t.Fatal(err)
	}
	return archiveCache{sha: compiled.Fingerprint(), facts: compiled, cutoff: "Oct 11, 2026"}
}

func archiveReport(t *testing.T, b ArchiveBoard, id string) Report {
	t.Helper()
	for _, r := range b.Board.Reports {
		if r.ID == id {
			return r
		}
	}
	t.Fatal("missing report", id)
	return Report{}
}

func TestArchiveBeforeImportSaysHowToImport(t *testing.T) {
	b, err := buildArchive(archiveCache{}, false, 0, "", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range b.Board.Reports {
		if r.Status != "failed" || !strings.Contains(r.Note, "Import the league archive in Control Room › Data and sources") || len(r.Rows) != 0 {
			t.Fatal(r.ID, r.Status, r.Note)
		}
		if strings.Contains(strings.ToLower(r.Note), "refresh from mfl") {
			t.Fatal("before import, a report says to refresh from MFL", r.ID)
		}
	}
}

func TestArchiveReportsOnTheReducedArchive(t *testing.T) {
	cache := reducedArchive(t)
	b, err := buildArchive(cache, false, 0, "", "")
	if err != nil {
		t.Fatal(err)
	}
	history := archiveReport(t, b, "M-080")
	if history.Status != "ok" || len(history.Rows) == 0 {
		t.Fatal(history)
	}
	last := history.Rows[len(history.Rows)-1].Cells
	if last[1] != "Season in progress" {
		t.Fatal("the newest season is not called in progress", last)
	}
	rules := archiveReport(t, b, "M-182")
	for _, row := range rules.Rows {
		if strings.ContainsAny(strings.Join(row.Cells, " "), "{\"") {
			t.Fatal("raw MFL setting text in the rules report", row.Cells)
		}
	}
	if champions := archiveReport(t, b, "M-114"); champions.Status == "failed" || champions.Title != "League champions" {
		t.Fatal(champions)
	}
	for _, id := range []string{"M-104", "M-105", "M-023", "M-110", "M-019"} {
		if r := archiveReport(t, b, id); r.Status != "empty" || !strings.Contains(r.Note, "Choose") {
			t.Fatal(id, "without a franchise", r.Status, r.Note)
		}
	}
	team := b.Teams[0].Value
	other := b.Teams[1].Value
	chosen, err := buildArchive(cache, false, 0, team, other)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]string{}
	for _, id := range []string{"M-104", "M-105", "M-023", "M-110"} {
		r := archiveReport(t, chosen, id)
		shape := strings.Join(r.Columns, "|")
		if prior, ok := seen[shape]; ok {
			t.Fatal(id, "repeats", prior)
		}
		seen[shape] = id
	}
	for _, row := range archiveReport(t, chosen, "M-084").Rows {
		if strings.Contains(row.Cells[0], "wins") && strings.Contains(row.Cells[3], ".") {
			t.Fatal("a win count written with decimals", row.Cells)
		}
	}
	series := archiveReport(t, chosen, "M-019")
	for _, row := range series.Rows {
		if !strings.Contains(row.Cells[4], "–") {
			t.Fatal("a meeting without both scores", row.Cells)
		}
	}
}

func TestArchiveLineupInWords(t *testing.T) {
	raw := `{"count":"21","iop_starters":"8","position":[{"name":"QB","limit":"1"},{"limit":"1-3","name":"RB"}]}`
	if got := archiveLineup(raw); got != "QB 1 · RB 1–3 · 21 starters · 8 offensive" {
		t.Fatal(got)
	}
	if archiveLineup("") != "Not recorded" || archiveLineup("{") != "Not recorded" || archiveUnquote(`"8"`) != "8" {
		t.Fatal("unreadable settings")
	}
}

func TestTrophyTitleNamesATrophyOnce(t *testing.T) {
	for _, c := range [][3]string{
		{"NFC", "NFC Conference Champ", "NFC Conference Champ"},
		{"2015 Legacy Superbowl III", "2015 Legacy Champions", "2015 Legacy Superbowl III · 2015 Legacy Champions"},
		{"Division Champions", "", "Division Champions"},
	} {
		if got := trophyTitle(c[0], c[1]); got != c[2] {
			t.Errorf("trophyTitle(%q, %q) = %q, want %q", c[0], c[1], got, c[2])
		}
	}
}

func TestArchiveRetainsEveryNonPoolReport(t *testing.T) {
	b, err := buildArchive(archiveCache{}, false, 0, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if b.Board.Title != "Archive" {
		t.Fatal(b.Board.Title)
	}
	ids := []string{}
	for _, r := range b.Board.Reports {
		ids = append(ids, r.ID)
	}
	want := []string{"M-080", "M-114", "M-182", "M-104", "M-105", "M-023", "M-110", "M-019",
		"M-083", "M-084", "M-085", "M-086", "M-082", "M-073", "M-079", "M-081", "M-089"}
	if !slices.Equal(ids, want) {
		t.Fatalf("report ids = %v, want %v", ids, want)
	}
}
