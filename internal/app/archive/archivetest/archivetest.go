// Package archivetest compiles the reduced league archive for tests: the Lab's pinned inputs
// in internal/lab/testdata, a few seasons of the real league, compiled exactly as the app does.
// The root package's tests and internal/app/archive's tests share it.
package archivetest

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/facts"
)

// Cutoff is the snapshot cutoff the tests show for the reduced archive.
const Cutoff = "Oct 11, 2026"

// Facts compiles the reduced archive. It takes a few seconds; callers that need it often cache it.
func Facts(t testing.TB) facts.Facts {
	t.Helper()
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("archivetest: no source path")
	}
	testdata := filepath.Join(filepath.Dir(here), "..", "..", "..", "lab", "testdata")
	compiled, err := facts.Compile(t.Context(), inputs(t, filepath.Join(testdata, "facts", "inputs.json.gz")), notes(t, filepath.Join(testdata, "note_readings.csv")))
	if err != nil {
		t.Fatal(err)
	}
	return compiled
}

func inputs(t testing.TB, path string) []facts.ResolvedInput {
	t.Helper()
	raw, err := os.ReadFile(path)
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
	if err := gz.Close(); err != nil {
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
	out := []facts.ResolvedInput{}
	for _, in := range fixture.Inputs {
		var b []byte
		for _, token := range in.Tokens {
			b = append(b, fixture.Dictionary[token]...)
		}
		out = append(out, facts.ResolvedInput{File: in.File, SHA256: in.SHA256, Bytes: in.Bytes, Body: b})
	}
	return out
}

func notes(t testing.TB, path string) []facts.RawNoteReading {
	t.Helper()
	curated, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := facts.ParseCurated(curated)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}
