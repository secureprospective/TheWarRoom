package main

import (
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
)

func restoreDrillApp(t *testing.T, dir string) *App {
	t.Helper()
	a := &App{ctx: t.Context(), dataDir: dir, started: make(chan struct{})}
	_, hist, err := a.openDatabases(t.Context(), dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.shutdown(t.Context()) })
	if err := hist.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	for _, p := range []*db.Pools{a.pools, a.whatifPools} {
		if err := p.InitializeMain(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
	hist.SetClock(func() time.Time { return time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC) })
	a.history = hist
	close(a.started)
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
	held, err := restored.history.CommissionerPeople(t.Context(), raw.League, principal)
	if err != nil || !held.MayChangeRules {
		t.Fatalf("snapshot lost stored commissioner rights rather than local identity: %+v %v", held, err)
	}
}
