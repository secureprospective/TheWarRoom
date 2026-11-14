package main

import (
	"bytes"
	"compress/gzip"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/offsite"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

func TestReferenceRestoreReportsMissingOwnedPlayer(t *testing.T) {
	h := newTwoFileHost(t)
	lacks, err := referenceLacks(t.Context(), referencePath(h.dir+"/server.db"), referenceNeed{
		players: []string{"player." + strings.Repeat("f", 32)},
	})
	if err != nil || lacks != "it lacks 1 player(s) the leagues own" {
		t.Fatalf("missing player not reported: %q (%v)", lacks, err)
	}
}

// A league pinned to football this server does not hold no longer stops the server: it is served by a
// store with no football, the other league keeps its own, the dashboard names it, and once the
// evidence is back the league gets it by itself at the next look (at most once a minute).
func TestMissingPinnedReferenceStopsHostedSelection(t *testing.T) {
	h := newTwoFileHost(t)
	_, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	ctx := t.Context()
	body, digest := footballEvidence(t, "Later") // football this server has never held
	var packed bytes.Buffer
	writer := gzip.NewWriter(&packed)
	if _, err := writer.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	stored := packed.Bytes()
	pins := newLeaguePins(h.files.leagues)
	now := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	pins.now = func() time.Time { return now }
	if err := pins.selectPins(ctx, "", []accountstore.DirectoryEntry{{League: "native.hosted", Football: digest}}); err != nil {
		t.Fatalf("a missing pin stopped the server: %v", err)
	}
	if pin := pins.pin(t.Context(), "native.hosted"); !pin.Missing || pin.Football != nil || pin.Digest != digest {
		t.Fatalf("missing pin = %+v", pin)
	}
	if pin := pins.pin(t.Context(), "native.other"); pin.Missing || pin.Digest != "" {
		t.Fatalf("another league moved: %+v", pin)
	}
	if got := pins.missingLeagues(); len(got) != 1 || !strings.Contains(got[0], "native.hosted") {
		t.Fatalf("dashboard names %v", got)
	}
	if err := history.PutArchiveBody(ctx, h.files.reference, digest, stored); err != nil {
		t.Fatal(err)
	}
	if pin := pins.pin(t.Context(), "native.hosted"); !pin.Missing {
		t.Fatal("looked again before a minute had passed")
	}
	now = now.Add(footballRetry)
	pin := pins.pin(t.Context(), "native.hosted")
	if pin.Missing || pin.Football == nil || pin.Football.Digest() != digest {
		t.Fatalf("returned evidence not picked up: %+v", pin)
	}
	if got := pins.missingLeagues(); len(got) != 0 {
		t.Fatalf("dashboard still names %v", got)
	}
}

// A running server changes one league's pin through its private socket: the league's next request uses
// it, the other league does not move, and a version the reference file lacks is refused with nothing
// changed.
func TestPinFootballLiveThroughSocket(t *testing.T) {
	ctx := t.Context()
	h := newTwoFileHost(t)
	_, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	digest, _ := importFootball(t, h.files, "Pinned")
	signIns := signInStore(h.files)
	if err := recordDirectory(ctx, hostedLeague(t, h.files), signIns); err != nil {
		t.Fatal(err)
	}
	pins := newLeaguePins(h.files.leagues)
	if err := pins.selectPins(ctx, "", nil); err != nil {
		t.Fatal(err)
	}
	socket := filepath.Join(t.TempDir(), "status.sock")
	stop, err := startStatusSocket(ctx, socket, statusSource(h.files.signIns, pins, hostedBackups{
		league:    func() (offsite.Status, bool) { return offsite.Status{}, false },
		reference: func() (offsite.Status, bool) { return offsite.Status{}, false },
		signIns:   func() (offsite.Status, bool) { return offsite.Status{}, false },
	}), pinRoute(pins, signIns))
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	var output bytes.Buffer
	h.files.signIns.SetWriteGuard(func() error { return errors.New("changes are paused until off-site backups catch up") })
	if err := runPinFootball(ctx, []string{"-status-socket", socket, "-league", "native.hosted", "-football", digest}, &output); err == nil ||
		!strings.Contains(err.Error(), "paused") {
		t.Fatalf("the backup guard did not hold a live pin change: %v", err)
	}
	if pin := pins.pin(t.Context(), "native.hosted"); pin.Digest != "" {
		t.Fatalf("guarded pin changed the league: %+v", pin)
	}
	h.files.signIns.SetWriteGuard(nil)
	if err := runPinFootball(ctx, []string{"-status-socket", socket, "-league", "native.hosted", "-football", strings.Repeat("e", 64)}, &output); err == nil {
		t.Fatal("a version the reference file lacks was accepted")
	}
	if pin := pins.pin(t.Context(), "native.hosted"); pin.Digest != "" {
		t.Fatalf("refused pin changed the league: %+v", pin)
	}
	if err := runPinFootball(ctx, []string{"-status-socket", socket, "-league", "native.hosted", "-football", digest}, &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "uses it from the league's next request") {
		t.Fatalf("console words: %s", output.String())
	}
	if pin := pins.pin(t.Context(), "native.hosted"); pin.Digest != digest || pin.Football == nil {
		t.Fatalf("live pin not applied: %+v", pin)
	}
	if pin := pins.pin(t.Context(), "native.other"); pin.Digest != "" {
		t.Fatalf("another league moved: %+v", pin)
	}
	directory, err := signIns.Directory(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range directory {
		if entry.League == "native.hosted" && entry.Football != digest {
			t.Fatalf("directory row not written: %+v", entry)
		}
	}
}
