package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
)

func restoreCLILeague(id string) commissioner.RawCommand {
	return commissioner.RawCommand{League: id, Command: commissioner.Create,
		Args:   commissioner.RawArgs{Name: "Restore " + id, Preset: commissioner.PresetEmpty, Season: 2026, FranchiseCount: 4},
		Reason: "Start the restore drill", Request: "create", Effective: "now", Visibility: "league"}
}

// The operator path end to end: list the sets, refuse while the app holds the lock, then restore
// with the app closed and reopen to the league as it was when the set was captured.
func TestRestoreFlagRestoresASetWithTheAppClosed(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir, err := configDir()
	if err != nil {
		t.Fatal(err)
	}
	a := restoreDrillApp(t, dir)
	executeCommissionerTest(t, a, restoreCLILeague("native.kept"))
	set, err := a.core.Backups.Capture(t.Context(), [3]*db.Pools{a.core.Pools, a.core.HistPools, a.core.WhatifPools}, "daily", time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	executeCommissionerTest(t, a, restoreCLILeague("native.after"))
	var out, errOut bytes.Buffer
	if code := runFlag([]string{"-backups"}, &out, &errOut); code != 0 || !strings.Contains(out.String(), filepath.Base(set)) {
		t.Fatalf("-backups = %d, %q %q; want the captured set listed", code, out.String(), errOut.String())
	}
	if code := runFlag([]string{"-restore", filepath.Base(set)}, &out, &errOut); code != 1 || !strings.Contains(errOut.String(), "app closed") {
		t.Fatalf("-restore with the app open = %d, %q; want a refusal", code, errOut.String())
	}
	a.shutdown(t.Context())
	out.Reset()
	if code := runFlag([]string{"-restore", filepath.Base(set)}, &out, &errOut); code != 0 || !strings.Contains(out.String(), "pre-restore") {
		t.Fatalf("-restore = %d, %q %q", code, out.String(), errOut.String())
	}
	leagues, err := newCommissionerAPI(restoreDrillApp(t, dir)).TargetCommissionerLeagues()
	if err != nil || len(leagues) != 1 || leagues[0].ID != "native.kept" {
		t.Fatalf("after restore leagues = %+v %v; want only the league held when the set was captured", leagues, err)
	}
}
