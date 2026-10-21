package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	marketapp "github.com/secureprospective/TheWarRoom/internal/app/market"
	raceapp "github.com/secureprospective/TheWarRoom/internal/app/race"

	"github.com/secureprospective/TheWarRoom/internal/app/core"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
)

func restoreDrillApp(t *testing.T, dir string) *App {
	t.Helper()
	a := &App{marketHeld: &marketapp.Held{}, raceHeld: &raceapp.Held{}, core: &core.State{Ctx: t.Context(), Paths: core.Paths{DataDir: dir}, Started: make(chan struct{})}}
	_, hist, err := a.openDatabases(t.Context(), dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.shutdown(t.Context()) })
	if err := hist.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	for _, p := range []*db.Pools{a.core.Pools, a.core.WhatifPools} {
		if err := p.InitializeMain(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
	hist.SetClock(func() time.Time { return time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC) })
	a.core.History = hist
	close(a.core.Started)
	return a
}

// This fixture has no concurrent writers. It exercises SQLite snapshots, not a production
// matched-set restore implementation or a crash-recovery protocol.
func TestBackupSnapshotDifferentHomeCommissionerRightsFinding(t *testing.T) {
	source := restoreDrillApp(t, t.TempDir())
	raw := commissioner.RawCommand{League: "native.restore", Command: commissioner.Create,
		Args:   commissioner.RawArgs{Name: "Restore drill", Preset: commissioner.PresetEmpty, Season: 2026, FranchiseCount: 4},
		Reason: "Start the restore drill", Request: "create", Effective: "now", Visibility: "league"}
	executeCommissionerTest(t, source, raw)
	before, err := source.TargetCommissionerRules(raw.League)
	if err != nil {
		t.Fatal(err)
	}
	people, err := source.TargetCommissionerPeople(raw.League)
	if err != nil || !people.MayChangeRules {
		t.Fatalf("source commissioner missing: %+v %v", people, err)
	}
	home := t.TempDir()
	for _, file := range source.databaseFiles() {
		if _, err := file.pools.Write().ExecContext(t.Context(), "VACUUM INTO ?", filepath.Join(home, dbFileName(file.name, isDevBuild()))); err != nil {
			t.Fatal(err)
		}
	}
	restored := restoreDrillApp(t, home)
	after, err := restored.TargetCommissionerRules(raw.League)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("native rules changed after snapshot restore: %v", err)
	}
	leagues, err := restored.TargetCommissionerLeagues()
	if err != nil || len(leagues) != 1 || leagues[0].ID != raw.League {
		t.Fatalf("native league missing: %+v %v", leagues, err)
	}
	people, err = restored.TargetCommissionerPeople(raw.League)
	if err != nil {
		t.Fatal(err)
	}
	if people.MayChangeRules || people.YourRole != "Visitor" {
		t.Fatalf("expected the documented folder-derived identity loss: %+v", people)
	}
	principal, err := source.commissionerPrincipal()
	if err != nil {
		t.Fatal(err)
	}
	held, err := restored.core.History.CommissionerPeople(t.Context(), raw.League, principal)
	if err != nil || !held.MayChangeRules {
		t.Fatalf("snapshot lost stored commissioner rights rather than local identity: %+v %v", held, err)
	}
}

func TestMatchedRestoreSameHomeKeepsCommissionerAndNativeRoster(t *testing.T) {
	dir := t.TempDir()
	a := restoreDrillApp(t, dir)
	raw := commissioner.RawCommand{League: "native.restore", Command: commissioner.Create,
		Args:   commissioner.RawArgs{Name: "Restore drill", Preset: commissioner.PresetLegacy, Season: 2026, FranchiseCount: 32},
		Reason: "Start the restore drill", Request: "create", Effective: "now", Visibility: "league"}
	executeCommissionerTest(t, a, raw)
	body, err := os.ReadFile("internal/commissioner/testdata/native-assets.wire.json")
	if err != nil {
		t.Fatal(err)
	}
	var wire commissioner.NativeAssets
	if err := json.Unmarshal(body, &wire); err != nil {
		t.Fatal(err)
	}
	principal, err := a.commissionerPrincipal()
	if err != nil {
		t.Fatal(err)
	}
	raw.Command, raw.Args, raw.Expected, raw.Request = commissioner.NativeSeed, commissioner.RawArgs{Native: wire.Snapshot}, 1, "seed"
	if _, err := a.core.History.ExecuteCommissioner(t.Context(), raw, principal, nil); err != nil {
		t.Fatal(err)
	}
	before := readNativeTest(t, a, raw.League)
	if before.Snapshot == nil || len(before.Snapshot.Players) == 0 {
		t.Fatal("fixture has no roster")
	}
	rules, err := a.TargetCommissionerRules(raw.League)
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	set, err := a.core.Backups.Capture(t.Context(), [3]*db.Pools{a.core.Pools, a.core.HistPools, a.core.WhatifPools}, "daily", at)
	if err != nil {
		t.Fatal(err)
	}
	raw.Command, raw.Args, raw.Expected, raw.Request = commissioner.Create, commissioner.RawArgs{Name: "After capture", Preset: commissioner.PresetEmpty, Season: 2026, FranchiseCount: 4}, 0, "later"
	raw.League = "native.later"
	executeCommissionerTest(t, a, raw)
	a.shutdown(t.Context())
	if err := a.core.Backups.Restore(t.Context(), set, at.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	restored := restoreDrillApp(t, dir)
	after := readNativeTest(t, restored, "native.restore")
	if !reflect.DeepEqual(before, after) {
		t.Fatal("native roster changed after same-home restore")
	}
	afterRules, err := restored.TargetCommissionerRules("native.restore")
	if err != nil || !reflect.DeepEqual(rules, afterRules) {
		t.Fatalf("rules changed: %v", err)
	}
	people, err := restored.TargetCommissionerPeople("native.restore")
	if err != nil || !people.MayChangeRules || people.YourRole == "Visitor" {
		t.Fatalf("same-home commissioner lost rights: %+v %v", people, err)
	}
	leagues, err := restored.TargetCommissionerLeagues()
	if err != nil || len(leagues) != 1 || leagues[0].ID != "native.restore" {
		t.Fatalf("restore did not rewind native leagues: %+v %v", leagues, err)
	}
}

func TestDatabaseMaintenanceCreatesDailySet(t *testing.T) {
	a := restoreDrillApp(t, t.TempDir())
	a.startDatabaseChecks()
	root := filepath.Join(a.core.Paths.DataDir, "backups")
	deadline := time.Now().Add(10 * time.Second)
	var entries []os.DirEntry
	for time.Now().Before(deadline) {
		entries, _ = os.ReadDir(root)
		if len(entries) == 1 && entries[0].Name()[0] != '.' {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if len(entries) != 1 || entries[0].Name()[0] == '.' {
		t.Fatal("maintenance did not publish a daily backup")
	}
	body, err := os.ReadFile(filepath.Join(root, entries[0].Name(), "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var captured db.RawBackupManifest
	if err := json.Unmarshal(body, &captured); err != nil {
		t.Fatal(err)
	}
	a.backupDatabases(t.Context(), captured.CapturedAt)
	entries, err = os.ReadDir(root)
	if err != nil || len(entries) != 1 {
		t.Fatalf("maintenance made duplicate daily sets: %v %v", entries, err)
	}
}

func TestStartupCapturesMatchedSetBeforeOpeningPools(t *testing.T) {
	dir := t.TempDir()
	for _, base := range []string{"thewarroom", "history", "whatif"} {
		p, err := db.Open(t.Context(), filepath.Join(dir, dbFileName(base, isDevBuild())))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := p.Write().ExecContext(t.Context(), "CREATE TABLE before_migration (id TEXT)"); err != nil {
			t.Fatal(err)
		}
		if err := p.Close(); err != nil {
			t.Fatal(err)
		}
	}
	a := &App{marketHeld: &marketapp.Held{}, raceHeld: &raceapp.Held{}, core: &core.State{Ctx: t.Context(), Paths: core.Paths{DataDir: dir}}}
	if _, _, err := a.openDatabases(t.Context(), dir); err != nil {
		t.Fatal(err)
	}
	defer a.shutdown(t.Context())
	entries, err := os.ReadDir(filepath.Join(dir, "backups"))
	if err != nil || len(entries) != 1 {
		t.Fatalf("startup omitted matched pre-migration set: %v %v", entries, err)
	}
	body, err := os.ReadFile(filepath.Join(dir, "backups", entries[0].Name(), "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m db.RawBackupManifest
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatal(err)
	}
	if m.Kind != "pre-migration" || len(m.Files) != 3 {
		t.Fatalf("startup backup: %+v", m)
	}
	for _, file := range m.Files {
		if file.UserVersion != 0 {
			t.Fatalf("migration ran before capture: %+v", file)
		}
	}
}
