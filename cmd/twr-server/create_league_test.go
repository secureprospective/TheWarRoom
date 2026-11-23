package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

const createdLeagueID = "native.rehtest"

func createLeagueArgs(path string, extra ...string) []string {
	return append([]string{"-db", path, "-league", createdLeagueID, "-name", "Rehearsal", "-preset", "default_32", "-season", "2026"}, extra...)
}

func emptyServerPath(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "server.db")
	if err := prepareFirstLoad(path); err != nil {
		t.Fatal(err)
	}
	return path
}

// A created league opens through the server's own store path with 32 empty franchises, the preset's
// rules and a starting snapshot, and its startup draft then opens.
func TestCreateLeagueMakesAnOpenableNativeLeague(t *testing.T) {
	ctx := t.Context()
	path := emptyServerPath(t)
	var out bytes.Buffer
	if err := runCreateLeague(ctx, createLeagueArgs(path), &out); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{createdLeagueID, "32 franchises", "pin-football", "refresh-nfl-schedule", "join-link"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("report lacks %q: %s", want, out.String())
		}
	}
	files := openTestServer(t, filepath.Dir(path))
	store, err := leagueStore(ctx, files, createdLeagueID)
	if err != nil {
		t.Fatal(err)
	}
	records, err := store.SharedCommissionerRecords(ctx, createdLeagueID)
	if err != nil {
		t.Fatal(err)
	}
	at := time.Now()
	directory, err := commissioner.NativeDirectoryReading(createdLeagueID, records, at)
	if err != nil {
		t.Fatal(err)
	}
	if len(directory.Franchises) != 32 {
		t.Fatalf("franchises = %d", len(directory.Franchises))
	}
	seats, err := leagueservice.JoinSeats{Store: store}.Franchises(ctx, createdLeagueID)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range seats {
		if f.Seat != "" {
			t.Fatalf("franchise %s is seated by %q", f.ID, f.Seat)
		}
	}
	if last := records[len(records)-1].Command.Command; last != commissioner.NativeSeed {
		t.Fatalf("last record %s, want the starting snapshot", last)
	}
	rules, err := commissioner.RulesNow(records, at)
	if err != nil {
		t.Fatal(err)
	}
	if !leaguerules.OLineOn(rules) {
		t.Fatal("the default_32 preset's O-Line rules are missing")
	}
	// A small player pool stands in for the server's football file.
	facts := nativecheck.Facts{"1000": {Name: "Pool Player", Position: leaguerules.WR}}
	open := commissioner.RawCommand{League: createdLeagueID, Command: commissioner.DraftStartupOpen,
		Args:     commissioner.RawArgs{Draft: &commissioner.RawDraftArgs{Season: 2026, Startup: true}},
		Expected: int64(len(records)), Request: "open", Reason: "test", Effective: "now", Visibility: "league"}
	if _, err := store.ExecuteCommissionerWithFacts(ctx, open, createLeaguePrincipal, nil, facts, time.UTC); err != nil {
		t.Fatalf("draft.startup.open: %v", err)
	}
}

func TestCreateLeagueRefusesAnExistingLeague(t *testing.T) {
	path := emptyServerPath(t)
	if err := runCreateLeague(t.Context(), createLeagueArgs(path), &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	err := runCreateLeague(t.Context(), createLeagueArgs(path), &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "already on this server") {
		t.Fatalf("second create: %v", err)
	}
}

func TestCreateLeagueRefusesAnUnknownPreset(t *testing.T) {
	path := emptyServerPath(t)
	args := []string{"-db", path, "-league", createdLeagueID, "-name", "X", "-preset", "no_such_preset", "-season", "2026"}
	err := runCreateLeague(t.Context(), args, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "default_32") {
		t.Fatalf("unknown preset: %v", err)
	}
	files := openTestServer(t, filepath.Dir(path))
	if len(files.leagues.leagues()) != 0 {
		t.Fatal("a refused create left a league file")
	}
}
