package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"

	archiveapp "github.com/secureprospective/TheWarRoom/internal/app/archive"
	"github.com/secureprospective/TheWarRoom/internal/app/archive/archivetest"
	"github.com/secureprospective/TheWarRoom/internal/app/report"
)

// Only serial tests call this helper. The on-disk snapshot is immutable after its first compile.
func reducedArchive(t *testing.T) archiveapp.Snapshot {
	t.Helper()
	path := suiteFixturePath(t, "reduced-archive.json")
	body, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		f := archivetest.Facts(t)
		body, err = json.Marshal(archiveapp.Snapshot{SHA: f.Fingerprint(), Facts: f})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, body, 0o600); err != nil {
			t.Fatal(err)
		}
	} else if err != nil {
		t.Fatal(err)
	}
	var fixture archiveapp.Snapshot
	if err := json.Unmarshal(body, &fixture); err != nil {
		t.Fatal(err)
	}
	fixture.Cutoff = archivetest.Cutoff
	return fixture
}

// TestArchiveWireMatchesGolden freezes the Archive wire contract in testdata/wire/archive: what
// the frontend receives, field for field. The bound method opens on a fresh history (no compiled
// snapshot); the rest build on the reduced archive, before and after a franchise is chosen.
// Regenerate with TWR_WRITE_WIRE=1 go test -run TestArchiveWireMatchesGolden . and review the diff.
func TestArchiveWireMatchesGolden(t *testing.T) {
	a := targetTestApp(t)
	bound, err := a.TargetArchive(0, "", "")
	if err != nil {
		t.Fatal(err)
	}
	checkArchiveWire(t, "unimported", bound)
	cache := reducedArchive(t)
	for _, c := range []struct {
		name, team, other string
		season            int
		stale             bool
	}{
		{name: "league"},
		{name: "season-stale", season: 2015, stale: true},
		{name: "franchise", team: "0001", other: "0002"},
	} {
		b, err := buildArchive(cache, c.stale, c.season, c.team, c.other)
		if err != nil {
			t.Fatal(c.name, err)
		}
		checkArchiveWire(t, c.name, b)
	}
}

func checkArchiveWire(t *testing.T, name string, b ArchiveBoard) {
	t.Helper()
	got, err := json.MarshalIndent(b, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')
	path := "testdata/wire/archive/" + name + ".json"
	if os.Getenv("TWR_WRITE_WIRE") == "1" {
		if err := os.MkdirAll("testdata/wire/archive", 0o750); err != nil {
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
		t.Errorf("%s differs from the Archive binding's output; the wire contract changed", path)
	}
}

// TestReportWireTwinsMatch proves wireBoard and wireChoices lose nothing: a board with every
// field set, and one left empty, encode the same before and after conversion. A field added to
// one type set and not the other, a renamed JSON tag, or a field the converter skips fails here.
func TestReportWireTwinsMatch(t *testing.T) {
	t.Parallel()
	var full report.Board
	n := 0
	fillWire(reflect.ValueOf(&full).Elem(), &n)
	choices := []report.Choice{{Value: "0001", Label: "One"}}
	for _, c := range []struct {
		name       string
		built, got any
	}{
		{"full board", full, wireBoard(full)},
		{"empty board", report.Board{}, wireBoard(report.Board{})},
		{"empty report", report.Board{Reports: []report.Report{{}}}, wireBoard(report.Board{Reports: []report.Report{{}}})},
		{"choices", choices, wireChoices(choices)},
		{"nil choices", []report.Choice(nil), wireChoices(nil)},
	} {
		want, err := json.Marshal(c.built)
		if err != nil {
			t.Fatal(err)
		}
		got, err := json.Marshal(c.got)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(want, got) {
			t.Errorf("%s: wire twin encodes differently\n built %s\n wire  %s", c.name, want, got)
		}
	}
}

// fillWire sets every field reachable from v to a distinct non-zero value.
func fillWire(v reflect.Value, n *int) {
	*n++
	kind := v.Kind()
	if kind == reflect.Struct {
		for i := range v.NumField() {
			fillWire(v.Field(i), n)
		}
		return
	}
	if kind == reflect.Pointer {
		v.Set(reflect.New(v.Type().Elem()))
		fillWire(v.Elem(), n)
		return
	}
	if kind == reflect.Slice {
		v.Set(reflect.MakeSlice(v.Type(), 2, 2))
		for i := range v.Len() {
			fillWire(v.Index(i), n)
		}
		return
	}
	scalars := map[reflect.Kind]func(){
		reflect.String:  func() { v.SetString(fmt.Sprint("text ", *n)) },
		reflect.Float64: func() { v.SetFloat(float64(*n) + 0.25) },
		reflect.Int:     func() { v.SetInt(int64(*n)) },
		reflect.Int64:   func() { v.SetInt(int64(*n)) },
		reflect.Bool:    func() { v.SetBool(true) },
	}
	set, ok := scalars[kind]
	if !ok {
		panic("fillWire: add a case for " + kind.String())
	}
	set()
}
