package composition

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	archiveapp "github.com/secureprospective/TheWarRoom/internal/app/archive"
	"github.com/secureprospective/TheWarRoom/internal/app/archive/archivetest"
	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
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

// Root retains a small composition golden; feature cases live beside the API.
func TestArchiveCompositionWireMatchesGolden(t *testing.T) {
	a := targetTestApp(t)
	bound, err := archiveapp.New(archiveHost{a.core}).TargetArchive(0, "", "")
	if err != nil {
		t.Fatal(err)
	}
	checkGolden(t, "archive", "unimported", bound)
}

func (a *App) archiveOpponent(team string) string { return (archiveHost{a.core}).ArchiveOpponent(team) }

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

func TestArchiveWireDuringRefreshPublication(t *testing.T) {
	a, transport, _, _ := seasonTestApp(t)
	a.core.Week.Number = 5
	if err := a.core.KeyStore.Set(a.core.Ctx, "fixture-key"); err != nil {
		t.Fatal(err)
	}
	entered, release := make(chan struct{}), make(chan struct{})
	var err error
	a.core.MFLClient, err = mfl.New("api", 10000,
		mfl.WithTransport(blockedSeasonTrade{base: transport, entered: entered, release: release}),
		mfl.WithKeySource(a.core.KeyStore.Get))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(a.core.Ctx)
	defer cancel()
	done := make(chan struct{})
	go func() { a.refreshSeason(ctx); close(done) }()
	defer func() { cancel(); close(release); <-done }()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("refresh did not reach fixture I/O")
	}
	// The real refresh has published lineups and holds RefreshMu during pending-trade I/O.
	read := make(chan error, 1)
	go func() { _, err := archiveapp.New(archiveHost{a.core}).TargetArchive(0, "0001", ""); read <- err }()
	select {
	case err := <-read:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("archive read blocked behind refresh I/O")
	}
	a.core.SeasonMu.Lock()
	published := a.core.SeasonLineups
	a.core.SeasonMu.Unlock()
	if published.FetchedAt.IsZero() || len(published.Value.Matchups) == 0 {
		t.Fatal("lineups were not published")
	}
	match := published.Value.Matchups[0]
	b, err := archiveapp.New(archiveHost{a.core}).TargetArchive(0, match.Home, "")
	if err != nil || b.Other != match.Away {
		t.Fatal(b.Other, err)
	}
}
