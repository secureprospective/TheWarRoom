package main

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/facts"
)

// TestArchiveReportsOnTheFullArchive compiles the facts from the full league archive (read only)
// and writes every Archive report as text, for reading real lines against MFL's own pages.
// Opt-in: TWR_ARCHIVE_PARITY=1 TWR_FACTS_ARCHIVE=<raw dir> TWR_ARCHIVE_SMOKE=<out file>.
func TestArchiveReportsOnTheFullArchive(t *testing.T) {
	root, out := os.Getenv("TWR_FACTS_ARCHIVE"), os.Getenv("TWR_ARCHIVE_SMOKE")
	if os.Getenv("TWR_ARCHIVE_PARITY") != "1" || root == "" || out == "" {
		t.Skip("opt-in")
	}
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
		Inputs []struct {
			File   string `json:"file"`
			SHA256 string `json:"sha256"`
		} `json:"inputs"`
	}
	if err := json.Unmarshal(body, &fixture); err != nil {
		t.Fatal(err)
	}
	archive, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = archive.Close() }()
	read := func(file string) facts.ResolvedInput {
		b, err := archive.ReadFile(filepath.FromSlash(file))
		if err != nil {
			t.Fatal(err)
		}
		return facts.ResolvedInput{File: file, Bytes: int64(len(b)), Body: b, SHA256: hexSum(b)}
	}
	inputs := []facts.ResolvedInput{}
	for _, in := range fixture.Inputs {
		inputs = append(inputs, read(in.File))
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
	pins, err := poolPins()
	if err != nil {
		t.Fatal(err)
	}
	poolInputs := []facts.ResolvedInput{}
	for _, p := range pins {
		poolInputs = append(poolInputs, read(p.File))
	}
	pools, err := facts.CompilePools(poolInputs)
	if err != nil {
		t.Fatal(err)
	}
	cache := archiveCache{sha: compiled.Fingerprint(), facts: compiled, pools: pools, cutoff: "full archive"}
	var text strings.Builder
	for _, choice := range []struct {
		season      int
		team, other string
	}{{0, "", ""}, {2015, "0001", "0002"}} {
		board, err := buildArchive(cache, false, choice.season, choice.team, choice.other)
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range board.Board.Reports {
			text.WriteString("\n== " + r.ID + " " + r.Title + " [" + r.Status + "] season " + strconv.Itoa(board.Season) + " team " + choice.team + "\n")
			text.WriteString(r.Source + " | " + r.Note + "\n" + strings.Join(r.Columns, " | ") + "\n")
			for _, row := range r.Summary {
				text.WriteString("  S: " + strings.Join(row.Cells, " | ") + "\n")
			}
			for _, row := range r.Rows {
				text.WriteString("  " + strings.Join(row.Cells, " | ") + "\n")
			}
		}
	}
	if err := os.WriteFile(out, []byte(text.String()), 0o600); err != nil { //nolint:gosec // opt-in report path the operator names
		t.Fatal(err)
	}
}

func hexSum(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
