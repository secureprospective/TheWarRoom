package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func backupFixture(t *testing.T) (*Backups, [3]*Pools) {
	t.Helper()
	dir := t.TempDir()
	files := [3]BackupFile{{"thewarroom.db", MainVersion()}, {"history.db", MainVersion()}, {"whatif.db", MainVersion()}}
	b, err := NewBackups(dir, "test", files)
	if err != nil {
		t.Fatal(err)
	}
	var pools [3]*Pools
	for i, file := range files {
		p, err := Open(t.Context(), filepath.Join(dir, file.Name))
		if err != nil {
			t.Fatal(err)
		}
		pools[i] = p
		t.Cleanup(func() {
			if err := p.Close(); err != nil {
				t.Error(err)
			}
		})
		if _, err := p.Write().ExecContext(t.Context(), "CREATE TABLE marker (value TEXT); INSERT INTO marker VALUES ('old')"); err != nil {
			t.Fatal(err)
		}
	}
	return b, pools
}

func backupAt() time.Time { return time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC) }

func TestMatchedBackupWaitsForEveryWriter(t *testing.T) {
	b, pools := backupFixture(t)
	tx, err := pools[2].Write().BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(t.Context(), "UPDATE marker SET value='matched'"); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { _, err := b.Capture(t.Context(), pools, "daily", backupAt()); done <- err }()
	// Wait until capture holds the first writer and is blocked on the last.
	deadline := time.Now().Add(5 * time.Second)
	for pools[0].Write().Stats().InUse == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if pools[0].Write().Stats().InUse == 0 {
		t.Fatal("capture did not acquire main writer")
	}
	select {
	case err := <-done:
		t.Fatalf("capture did not wait: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	entries, _ := os.ReadDir(filepath.Join(b.dir, "backups"))
	if len(entries) != 0 {
		t.Fatal("VACUUM started before every writer stopped")
	}
	for _, p := range pools[:2] {
		// A queued writer must not slip between the three VACUUMs.
		ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
		_, err := p.Write().ExecContext(ctx, "UPDATE marker SET value='torn'")
		cancel()
		if err == nil {
			t.Fatal("writer escaped backup barrier")
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	sets, _, err := b.sets(t.Context())
	if err != nil || len(sets) != 1 {
		t.Fatalf("sets: %v %v", sets, err)
	}
	for i, file := range b.files {
		p, err := sql.Open("sqlite", "file:"+filepath.Join(sets[0].path, file.Name)+"?mode=ro&immutable=1")
		if err != nil {
			t.Fatal(err)
		}
		var value string
		err = p.QueryRowContext(t.Context(), "SELECT value FROM marker").Scan(&value)
		if closeErr := p.Close(); closeErr != nil {
			t.Fatal(closeErr)
		}
		want := "old"
		if i == 2 {
			want = "matched"
		}
		if err != nil || value != want {
			t.Fatalf("file %s: %q %v", file.Name, value, err)
		}
	}
	if set, err := b.Capture(t.Context(), pools, "daily", backupAt().Add(time.Hour)); err != nil || set != "" {
		t.Fatalf("second daily: %q %v", set, err)
	}
}

func TestRestoreRefusesInvalidSets(t *testing.T) {
	cases := []string{"version", "missing", "missing-entry", "extra", "extra-entry", "size", "sha", "newer", "actual-newer", "duplicate", "separator", "dotdot", "symlink", "unknown-json", "trailing-json"}
	for _, test := range cases {
		t.Run(test, func(t *testing.T) {
			b, pools := backupFixture(t)
			set, err := b.Capture(t.Context(), pools, "daily", backupAt())
			if err != nil {
				t.Fatal(err)
			}
			for _, p := range pools {
				if err := p.Close(); err != nil {
					t.Fatal(err)
				}
			}
			var m RawBackupManifest
			if err := readJSON(filepath.Join(set, "manifest.json"), &m); err != nil {
				t.Fatal(err)
			}
			switch test {
			case "version":
				m.ManifestVersion = 2
			case "missing":
				if err := os.Remove(filepath.Join(set, m.Files[0].Name)); err != nil {
					t.Fatal(err)
				}
			case "missing-entry":
				m.Files = m.Files[:2]
			case "extra-entry":
				m.Files = append(m.Files, m.Files[0])
			case "extra":
				if err := os.WriteFile(filepath.Join(set, "foreign.db"), []byte("foreign"), 0600); err != nil {
					t.Fatal(err)
				}
			case "size":
				m.Files[0].Size++
			case "sha":
				m.Files[0].SHA256 = strings.Repeat("0", 64)
			case "newer", "actual-newer":
				p, err := Open(t.Context(), filepath.Join(set, m.Files[0].Name))
				if err != nil {
					t.Fatal(err)
				}
				if _, err := p.Write().ExecContext(t.Context(), "PRAGMA user_version=999"); err != nil {
					t.Fatal(err)
				}
				if err := p.Close(); err != nil {
					t.Fatal(err)
				}
				entry, err := backupEntry(t.Context(), filepath.Join(set, m.Files[0].Name), m.Files[0].Name)
				if err != nil {
					t.Fatal(err)
				}
				if test == "actual-newer" {
					entry.UserVersion = 0
				}
				m.Files[0] = entry
			case "duplicate":
				m.Files[1] = m.Files[0]
			case "separator":
				m.Files[0].Name = "../thewarroom.db"
			case "dotdot":
				m.Files[0].Name = ".."
			case "symlink":
				path := filepath.Join(set, m.Files[0].Name)
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.Join(b.dir, m.Files[0].Name), path); err != nil {
					t.Fatal(err)
				}
			}
			if err := durableJSON(filepath.Join(set, "manifest.json"), m); err != nil {
				t.Fatal(err)
			}
			if test == "unknown-json" || test == "trailing-json" {
				path := filepath.Join(filepath.Clean(set), "manifest.json")
				body, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if test == "unknown-json" {
					body = append([]byte(`{"foreign":1,`), body[1:]...)
				} else {
					body = append(body, []byte(` {}`)...)
				}
				if err := os.WriteFile(path, body, 0600); err != nil { //nolint:gosec // path is the manifest of this test's generated temporary backup, never decoded input.
					t.Fatal(err)
				}
			}
			if err := b.Restore(t.Context(), set, backupAt().Add(time.Hour)); err == nil {
				t.Fatalf("accepted %s", test)
			}
			if _, err := os.Stat(filepath.Join(b.dir, "restore.journal")); !os.IsNotExist(err) {
				t.Fatalf("invalid set touched live files: %v", err)
			}
			assertMarkers(t, b, "old")
		})
	}
}

func assertMarkers(t *testing.T, b *Backups, want string) {
	t.Helper()
	for _, file := range b.files {
		p, err := Open(t.Context(), filepath.Join(b.dir, file.Name))
		if err != nil {
			t.Fatal(err)
		}
		var value string
		err = p.Read().QueryRowContext(t.Context(), "SELECT value FROM marker").Scan(&value)
		if closeErr := p.Close(); closeErr != nil {
			t.Fatal(closeErr)
		}
		if err != nil || value != want {
			t.Fatalf("%s=%q want %q: %v", file.Name, value, want, err)
		}
	}
}

func TestRestoreRecoversEveryJournalStep(t *testing.T) {
	steps := []string{"prepared", "moving", "moved:thewarroom.db", "moved:history.db", "moved:whatif.db", "installing", "installed:thewarroom.db", "installed:history.db", "installed:whatif.db", "complete", "success"}
	for _, step := range steps {
		t.Run(step, func(t *testing.T) {
			b, pools := backupFixture(t)
			set, err := b.Capture(t.Context(), pools, "daily", backupAt())
			if err != nil {
				t.Fatal(err)
			}
			for _, p := range pools {
				if _, err := p.Write().ExecContext(t.Context(), "UPDATE marker SET value='current'"); err != nil {
					t.Fatal(err)
				}
				if err := p.Close(); err != nil {
					t.Fatal(err)
				}
			}
			reached := false
			b.crash = func(at string) error {
				if at == step {
					reached = true
					return fmt.Errorf("injected crash at %s", at)
				}
				return nil
			}
			err = b.Restore(t.Context(), set, backupAt().Add(time.Hour))
			if step != "success" && (err == nil || !reached) {
				t.Fatalf("crash hook %s: %v", step, err)
			}
			if step == "success" && err != nil {
				t.Fatal(err)
			}
			// A fresh coordinator, not the crashed process's memory, performs startup recovery.
			fresh, err := NewBackups(b.dir, b.build, b.files)
			if err != nil {
				t.Fatal(err)
			}
			if err := fresh.Recover(t.Context()); err != nil {
				t.Fatal(err)
			}
			if err := fresh.Recover(t.Context()); err != nil {
				t.Fatal(err)
			}
			want := "current"
			if step == "complete" || step == "success" {
				want = "old"
			}
			assertMarkers(t, fresh, want)
			if _, err := os.Stat(filepath.Join(b.dir, "restore.journal")); !os.IsNotExist(err) {
				t.Fatalf("journal survives: %v", err)
			}
			if step == "success" {
				sets, reports, err := fresh.sets(t.Context())
				if err != nil || len(reports) != 0 || len(sets) != 2 {
					t.Fatalf("safety set: %v %v %v", sets, reports, err)
				}
			}
		})
	}
}

func TestBackupRetentionSixtyDays(t *testing.T) {
	b, pools := backupFixture(t)
	// Capture once; hard links make a 60-day valid-manifest calendar cheap without
	// replacing the production validation/pruning path with a mock.
	seed, err := b.Capture(t.Context(), pools, "daily", backupAt())
	if err != nil {
		t.Fatal(err)
	}
	var manifest RawBackupManifest
	if err := readJSON(filepath.Join(seed, "manifest.json"), &manifest); err != nil {
		t.Fatal(err)
	}
	var all []backupSet
	for day := 0; day < 60; day++ {
		for _, kind := range []string{"daily", "pre-migration", "pre-restore"} {
			at := backupAt().AddDate(0, 0, -day)
			path := filepath.Join(b.dir, "backups", fmt.Sprintf("%s-%02d", kind, day))
			if err := os.Mkdir(path, 0700); err != nil {
				t.Fatal(err)
			}
			for _, file := range b.files {
				if err := os.Link(filepath.Join(seed, file.Name), filepath.Join(path, file.Name)); err != nil {
					t.Fatal(err)
				}
			}
			m := manifest
			m.Kind, m.CapturedAt = kind, at
			if err := durableJSON(filepath.Join(path, "manifest.json"), m); err != nil {
				t.Fatal(err)
			}
			all = append(all, backupSet{path, m})
		}
	}
	if err := os.RemoveAll(seed); err != nil {
		t.Fatal(err)
	}
	foreign := filepath.Join(b.dir, "backups", "foreign")
	if err := os.Mkdir(foreign, 0700); err != nil {
		t.Fatal(err)
	}
	protected := filepath.Join(b.dir, "backups", "daily-59")
	reports, err := b.Prune(t.Context(), protected)
	if err != nil || len(reports) != 1 || !strings.Contains(reports[0], "foreign") {
		t.Fatalf("foreign report: %v %v", reports, err)
	}
	wantDays := map[int]bool{0: true, 1: true, 2: true, 3: true, 4: true, 5: true, 6: true, 9: true, 16: true, 23: true, 30: true, 59: true}
	// Oct 14 (seventh daily) is Wednesday: prior ISO weeks' newest days are Oct 11, 4, Sep 27, 20.
	for _, set := range all {
		day := int(backupAt().Sub(set.manifest.CapturedAt) / (24 * time.Hour))
		want := day < 5
		if set.manifest.Kind == "daily" {
			want = wantDays[day]
		}
		_, err := os.Stat(set.path)
		if (err == nil) != want {
			t.Errorf("retention %s: exists=%v want=%v", filepath.Base(set.path), err == nil, want)
		}
	}
	if _, err := os.Stat(foreign); err != nil {
		t.Fatal("foreign directory deleted")
	}
}

func TestPreMigrationMatchedSetBeforeChanges(t *testing.T) {
	b, pools := backupFixture(t)
	for _, p := range pools {
		if err := p.Close(); err != nil {
			t.Fatal(err)
		}
	}
	set, err := b.BeforeMigrations(t.Context(), backupAt())
	if err != nil || set == "" {
		t.Fatalf("pre-migration capture: %q %v", set, err)
	}
	m, err := b.verify(t.Context(), set)
	if err != nil || m.Kind != "pre-migration" || len(m.Files) != 3 {
		t.Fatalf("manifest: %+v %v", m, err)
	}
	versions := []int{m.Files[0].UserVersion, m.Files[1].UserVersion, m.Files[2].UserVersion}
	if !reflect.DeepEqual(versions, []int{0, 0, 0}) {
		t.Fatal(versions)
	}
	assertMarkers(t, b, "old")
}

func TestBackupManifestValidation(t *testing.T) {
	b, pools := backupFixture(t)
	set, err := b.Capture(t.Context(), pools, "daily", backupAt())
	if err != nil {
		t.Fatal(err)
	}
	original, err := b.verify(t.Context(), set)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []string{"version", "missing", "extra", "duplicate", "newer", "separator", "dotdot"} {
		t.Run(test, func(t *testing.T) {
			m := original
			m.Files = append([]RawBackupEntry(nil), original.Files...)
			switch test {
			case "version":
				m.ManifestVersion++
			case "missing":
				m.Files = m.Files[:2]
			case "extra":
				m.Files = append(m.Files, m.Files[0])
			case "duplicate":
				m.Files[1] = m.Files[0]
			case "newer":
				m.Files[0].UserVersion = 999
			case "separator":
				m.Files[0].Name = `history/evil.db`
			case "dotdot":
				m.Files[0].Name = `..`
			}
			if err := m.Validate(b.files); err == nil {
				t.Fatalf("accepted %s manifest", test)
			}
		})
	}
}
