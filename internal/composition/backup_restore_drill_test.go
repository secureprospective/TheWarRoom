package composition

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	commissionerapp "github.com/secureprospective/TheWarRoom/internal/app/commissioner"
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
	if err := a.seatStore.Initialize(t.Context()); err != nil {
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

// snapshotInto copies all three databases into another home. This fixture has no concurrent
// writers. It exercises SQLite snapshots, not a production matched-set restore or crash recovery.
func snapshotInto(t *testing.T, source *App) *App {
	t.Helper()
	home := t.TempDir()
	for _, file := range source.databaseFiles() {
		if _, err := file.pools.Write().ExecContext(t.Context(), "VACUUM INTO ?", filepath.Join(home, dbFileName(file.name, isDevBuild()))); err != nil {
			t.Fatal(err)
		}
	}
	return restoreDrillApp(t, home)
}

func restoreDrillLeague(t *testing.T) (*App, commissioner.RawCommand) {
	t.Helper()
	source := restoreDrillApp(t, t.TempDir())
	raw := commissioner.RawCommand{League: "native.restore", Command: commissioner.Create,
		Args:   commissioner.RawArgs{Name: "Restore drill", Preset: commissioner.PresetEmpty, Season: 2026, FranchiseCount: 4},
		Reason: "Start the restore drill", Request: "create", Effective: "now", Visibility: "league"}
	executeCommissionerTest(t, source, raw)
	people, err := newCommissionerAPI(source).TargetCommissionerPeople(raw.League)
	if err != nil || !people.MayChangeRules {
		t.Fatalf("source commissioner missing: %+v %v", people, err)
	}
	return source, raw
}

func requireRole(t *testing.T, a *App, league, role string) {
	t.Helper()
	people, err := newCommissionerAPI(a).TargetCommissionerPeople(league)
	if err != nil || people.YourRole != role || people.MayChangeRules != (role == "Commissioner") {
		t.Fatalf("want %s: %+v %v", role, people, err)
	}
}

// A restore into another folder or OS user is a new local identity. The commissioner's local
// sign-in, made before the backup, links the new identity back to the same account.
func TestRestoreOnAnotherComputerKeepsCommissionerWithSignIn(t *testing.T) {
	source, raw := restoreDrillLeague(t)
	api := newCommissionerAPI(source)
	if status, err := api.TargetCommissionerSeat(); err != nil || status.HasAccount {
		t.Fatalf("fresh install already has a sign-in: %+v %v", status, err)
	}
	if status, err := api.TargetCommissionerSeatSignIn("Commish", "league-password"); err != nil || status.Username != "commish" {
		t.Fatalf("protect seat: %+v %v", status, err)
	}
	before, err := api.TargetCommissionerRules(raw.League)
	if err != nil {
		t.Fatal(err)
	}
	restored := snapshotInto(t, source)
	sourcePrincipal, _ := source.commissionerPrincipal()
	restoredPrincipal, _ := restored.commissionerPrincipal()
	if sourcePrincipal == restoredPrincipal {
		t.Fatal("the drill must restore under a different local identity")
	}
	after, err := newCommissionerAPI(restored).TargetCommissionerRules(raw.League)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("native rules changed after snapshot restore: %v", err)
	}
	requireRole(t, restored, raw.League, "Visitor")
	seat := newCommissionerAPI(restored)
	if status, err := seat.TargetCommissionerSeat(); err != nil || !status.HasAccount || status.Username != "" {
		t.Fatalf("restored copy before sign-in: %+v %v", status, err)
	}
	// A copy without the password links nothing; it cannot create a second sign-in either.
	if _, err := seat.TargetCommissionerSeatSignIn("commish", "guessed-password"); err == nil {
		t.Fatal("wrong password linked the restored copy")
	}
	if _, err := seat.TargetCommissionerSeatSignIn("stranger", "own-password"); err == nil {
		t.Fatal("a second local account was created on a restored copy")
	}
	requireRole(t, restored, raw.League, "Visitor")
	if status, err := seat.TargetCommissionerSeatSignIn("COMMISH", "league-password"); err != nil || status.Username != "commish" {
		t.Fatalf("claim seat: %+v %v", status, err)
	}
	requireRole(t, restored, raw.League, "Commissioner")
	grant := commissioner.RawCommand{League: raw.League, Command: "commish.franchise.edit",
		Args:   commissioner.RawArgs{People: &commissioner.RawPeople{Franchise: "franchise.01", Name: "Restored", Abbreviation: "RES"}},
		Reason: "Prove restored authority", Request: "restored-edit", Expected: 2, Effective: "now", Visibility: "league"}
	executeCommissionerTest(t, restored, grant)
	// The source computer keeps its own link and its role.
	requireRole(t, source, raw.League, "Commissioner")
}

// Without the commissioner's sign-in, a stranger's copy of the data stays a Visitor even after
// the stranger creates a sign-in of their own.
func TestStrangersRestoredCopyWithoutSignInStaysVisitor(t *testing.T) {
	source, raw := restoreDrillLeague(t)
	restored := snapshotInto(t, source)
	requireRole(t, restored, raw.League, "Visitor")
	if _, err := newCommissionerAPI(restored).TargetCommissionerSeatSignIn("stranger", "own-password"); err != nil {
		t.Fatal(err)
	}
	requireRole(t, restored, raw.League, "Visitor")
	principal, err := source.commissionerPrincipal()
	if err != nil {
		t.Fatal(err)
	}
	held, err := restored.core.History.CommissionerPeople(t.Context(), raw.League, principal)
	if err != nil || !held.MayChangeRules {
		t.Fatalf("snapshot lost stored commissioner rights: %+v %v", held, err)
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
	rules, err := newCommissionerAPI(a).TargetCommissionerRules(raw.League)
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
	if _, err := a.core.Backups.Restore(t.Context(), set, at.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	restored := restoreDrillApp(t, dir)
	after := readNativeTest(t, restored, "native.restore")
	if !reflect.DeepEqual(before, after) {
		t.Fatal("native roster changed after same-home restore")
	}
	afterRules, err := newCommissionerAPI(restored).TargetCommissionerRules("native.restore")
	if err != nil || !reflect.DeepEqual(rules, afterRules) {
		t.Fatalf("rules changed: %v", err)
	}
	people, err := newCommissionerAPI(restored).TargetCommissionerPeople("native.restore")
	if err != nil || !people.MayChangeRules || people.YourRole == "Visitor" {
		t.Fatalf("same-home commissioner lost rights: %+v %v", people, err)
	}
	leagues, err := newCommissionerAPI(restored).TargetCommissionerLeagues()
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

func TestRepeatedOldSchemaRestoreStartsAndRetainsSafetyCopies(t *testing.T) {
	dir := t.TempDir()
	b, err := newDatabaseBackups(dir)
	if err != nil {
		t.Fatal(err)
	}
	var pools [3]*db.Pools
	for i, base := range []string{"thewarroom", "history", "whatif"} {
		p, err := db.Open(t.Context(), filepath.Join(dir, dbFileName(base, isDevBuild())))
		if err != nil {
			t.Fatal(err)
		}
		pools[i] = p
		t.Cleanup(func() { _ = p.Close() })
		if _, err := p.Write().ExecContext(t.Context(), "CREATE TABLE before_migration (id TEXT); INSERT INTO before_migration VALUES ('retained')"); err != nil {
			t.Fatal(err)
		}
	}
	at := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
	source, err := b.Capture(t.Context(), pools, "daily", at)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range pools {
		if err := p.Close(); err != nil {
			t.Fatal(err)
		}
	}
	var copies []string
	for attempt := range 2 {
		if _, err := b.Restore(t.Context(), source, at.Add(time.Duration(attempt+1)*time.Hour)); err != nil {
			t.Fatal(err)
		}
		a := restoreDrillApp(t, dir)
		for _, file := range a.databaseFiles() {
			backup := file.pools.MigrationFindings().Backup
			if backup == "" {
				t.Fatalf("%s has no migration safety copy", file.name)
			}
			copies = append(copies, backup)
			var held string
			if err := file.pools.Read().QueryRowContext(t.Context(), "SELECT id FROM before_migration").Scan(&held); err != nil || held != "retained" {
				t.Fatalf("%s lost data: %q %v", file.name, held, err)
			}
		}
		a.shutdown(t.Context())
	}
	seen := make(map[string]bool)
	for _, copy := range copies {
		if seen[copy] {
			t.Fatalf("migration backup reused: %s", copy)
		}
		seen[copy] = true
		info, err := os.Stat(copy)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0600 {
			t.Fatalf("%s mode=%v", copy, info.Mode())
		}
	}
	entries, err := os.ReadDir(filepath.Join(dir, "backups"))
	if err != nil {
		t.Fatal(err)
	}
	counts := make(map[string]int)
	for _, entry := range entries {
		set := filepath.Join(dir, "backups", entry.Name())
		body, err := os.ReadFile(filepath.Join(set, "manifest.json"))
		if err != nil {
			t.Fatal(err)
		}
		var manifest db.RawBackupManifest
		if err := json.Unmarshal(body, &manifest); err != nil {
			t.Fatal(err)
		}
		counts[manifest.Kind]++
		for _, file := range manifest.Files {
			info, err := os.Stat(filepath.Join(set, file.Name))
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode().Perm() != 0600 {
				t.Fatalf("%s/%s mode=%v", entry.Name(), file.Name, info.Mode())
			}
			if manifest.Kind == "pre-migration" && file.UserVersion != 0 {
				t.Fatalf("safety copy already migrated: %+v", file)
			}
		}
	}
	if counts["daily"] != 1 || counts["pre-migration"] != 2 || counts["pre-restore"] != 2 {
		t.Fatalf("safety copies: %v", counts)
	}
}

// Issue 14, 16, 18: Back up now needs the local sign-in, and its set restores onto a new
// computer whose profile was never opened, where the app's stores then start from it.
func TestBackUpNowRestoresIntoNeverOpenedProfile(t *testing.T) {
	source, raw := restoreDrillLeague(t)
	api := newCommissionerAPI(source)
	if _, err := api.TargetBackupNow(); !errors.Is(err, commissionerapp.ErrBackupNeedsSignIn) {
		t.Fatalf("backup without a sign-in: %v", err)
	}
	if entries, _ := os.ReadDir(filepath.Join(source.core.Paths.DataDir, "backups")); len(entries) != 0 {
		t.Fatalf("a refused backup wrote %d sets", len(entries))
	}
	if _, err := api.TargetCommissionerSeatSignIn("Commish", "league-password"); err != nil {
		t.Fatal(err)
	}
	saved, err := api.TargetBackupNow()
	if err != nil || !strings.HasPrefix(saved.Set, "manual-") || !strings.Contains(saved.Message, saved.Set) {
		t.Fatalf("back up now: %+v %v", saved, err)
	}
	before, err := api.TargetCommissionerRules(raw.League)
	if err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	copied := filepath.Join(home, "backups", saved.Set)
	if err := os.MkdirAll(copied, 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{dbFileName("thewarroom", isDevBuild()), dbFileName("history", isDevBuild()),
		dbFileName("whatif", isDevBuild()), "manifest.json"} {
		body, err := os.ReadFile(filepath.Join(source.core.Paths.DataDir, "backups", saved.Set, name))
		if err != nil {
			t.Fatal(err)
		}
		//nolint:gosec // both paths are this test's temp dirs and fixed member names
		if err := os.WriteFile(filepath.Join(copied, name), body, 0600); err != nil {
			t.Fatal(err)
		}
	}
	b, err := newDatabaseBackups(home)
	if err != nil {
		t.Fatal(err)
	}
	safety, err := b.Restore(t.Context(), copied, time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC))
	if err != nil || safety != "" {
		t.Fatalf("restore into a never-opened profile: safety=%q %v", safety, err)
	}
	restored := restoreDrillApp(t, home)
	after, err := newCommissionerAPI(restored).TargetCommissionerRules(raw.League)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("restored rules differ: %v", err)
	}
	requireRole(t, restored, raw.League, "Visitor")
	if _, err := newCommissionerAPI(restored).TargetCommissionerSeatSignIn("commish", "league-password"); err != nil {
		t.Fatal(err)
	}
	requireRole(t, restored, raw.League, "Commissioner")
}

// Issue 17: a refused password reads as the refusal alone, with no internal prefix.
func TestWrongPasswordMessageIsPlain(t *testing.T) {
	source, _ := restoreDrillLeague(t)
	api := newCommissionerAPI(source)
	if _, err := api.TargetCommissionerSeatSignIn("commish", "league-password"); err != nil {
		t.Fatal(err)
	}
	_, err := newCommissionerAPI(snapshotInto(t, source)).TargetCommissionerSeatSignIn("commish", "guessed")
	if err == nil || err.Error() != accounts.ErrInvalidCredentials.Error() {
		t.Fatalf("wrong password: %v", err)
	}
}
