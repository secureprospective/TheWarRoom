package db

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMigrationBackupPrecedesStepAndPreservesData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	p, err := Open(t.Context(), path, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = p.Close() }()
	if _, err := p.Write().ExecContext(t.Context(), "CREATE TABLE held (id TEXT PRIMARY KEY, value BLOB); INSERT INTO held VALUES ('0001', x'000102ff')"); err != nil {
		t.Fatal(err)
	}
	called := false
	result, err := p.Migrate(t.Context(), func(ctx context.Context, tx *sql.Tx) error {
		called = true
		matches, err := filepath.Glob(path + ".pre-v0-*.bak")
		if err != nil || len(matches) != 1 {
			t.Fatalf("backups=%v err=%v", matches, err)
		}
		backup, err := sql.Open("sqlite", "file:"+matches[0]+"?mode=ro")
		if err != nil {
			return err
		}
		defer func() { _ = backup.Close() }()
		var value []byte
		if err := backup.QueryRowContext(ctx, "SELECT value FROM held WHERE id='0001'").Scan(&value); err != nil {
			return err
		}
		if !bytes.Equal(value, []byte{0, 1, 2, 255}) {
			t.Fatalf("backup data = %v", value)
		}
		var version int
		if err := backup.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
			return err
		}
		if version != 0 {
			t.Fatalf("backup version = %d", version)
		}
		_, err = tx.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS held (id TEXT PRIMARY KEY, value BLOB)")
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if !called || !strings.HasPrefix(result.Backup, path+".pre-v0-") {
		t.Fatalf("called=%v result=%+v", called, result)
	}
	var value []byte
	if err := p.Read().QueryRowContext(t.Context(), "SELECT value FROM held WHERE id='0001'").Scan(&value); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(value, []byte{0, 1, 2, 255}) {
		t.Fatalf("held data changed: %v", value)
	}
	assertVersion(t, p, 1)
	if len(result.After.Integrity) != 1 || result.After.Integrity[0] != "ok" {
		t.Fatalf("integrity findings = %+v", result)
	}
	called = false
	result, err = p.Migrate(t.Context(), func(context.Context, *sql.Tx) error { called = true; return nil })
	if err != nil || called || result.Backup != "" {
		t.Fatalf("repeat migration: called=%v result=%+v err=%v", called, result, err)
	}
}

func TestMigrationFailingStepRollsBackOnlyThatStep(t *testing.T) {
	p, err := Open(t.Context(), filepath.Join(t.TempDir(), "rollback.db"), 2)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = p.Close() }()
	failure := errors.New("injected failure")
	_, err = p.Migrate(t.Context(), func(ctx context.Context, tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, "CREATE TABLE held (id TEXT); INSERT INTO held VALUES ('retained')")
		return err
	}, func(ctx context.Context, tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "UPDATE held SET id='lost'; CREATE TABLE incomplete (id TEXT)"); err != nil {
			return err
		}
		return failure
	})
	if !errors.Is(err, failure) {
		t.Fatalf("failure = %v", err)
	}
	assertVersion(t, p, 1)
	var value string
	if err := p.Read().QueryRowContext(t.Context(), "SELECT id FROM held").Scan(&value); err != nil {
		t.Fatal(err)
	}
	if value != "retained" {
		t.Fatalf("failed step changed data: %q", value)
	}
	var count int
	if err := p.Read().QueryRowContext(t.Context(), "SELECT count(*) FROM sqlite_master WHERE name='incomplete'").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("failed step left DDL behind")
	}
}

func TestMigrationReturnsExistingForeignKeyFindings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "orphan.db")
	fixture, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.ExecContext(t.Context(), "CREATE TABLE parent (id TEXT PRIMARY KEY); CREATE TABLE child (id TEXT REFERENCES parent(id)); INSERT INTO child VALUES ('absent'); CREATE TABLE checked (n INTEGER CHECK(n>0)); PRAGMA ignore_check_constraints=ON; INSERT INTO checked VALUES (-1)"); err != nil {
		t.Fatal(err)
	}
	if err := fixture.Close(); err != nil {
		t.Fatal(err)
	}
	p, err := Open(t.Context(), path, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = p.Close() }()
	result, err := p.Migrate(t.Context(), func(context.Context, *sql.Tx) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	for _, findings := range []CheckFindings{result.Before, result.After} {
		if len(findings.Integrity) != 1 || !strings.Contains(findings.Integrity[0], "CHECK constraint failed in checked") {
			t.Fatalf("integrity findings = %+v", findings)
		}
		if len(findings.ForeignKeys) != 1 {
			t.Fatalf("FK findings = %+v", result)
		}
		f := findings.ForeignKeys[0]
		if f.Table != "child" || f.Parent != "parent" || !f.RowID.Valid || f.RowID.Int64 != 1 || f.Constraint != 0 {
			t.Fatalf("finding = %+v", f)
		}
	}
	assertVersion(t, p, 1)
}

func TestOpenUsesCallerFileVersionBeforeWriting(t *testing.T) {
	for _, maxVersion := range []int{1, 2} {
		path := filepath.Join(t.TempDir(), "future.db")
		fixture, err := sql.Open("sqlite", path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fixture.ExecContext(t.Context(), "PRAGMA user_version=3; CREATE TABLE future_only (id TEXT)"); err != nil {
			t.Fatal(err)
		}
		if err := fixture.Close(); err != nil {
			t.Fatal(err)
		}
		before, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		p, err := Open(t.Context(), path, maxVersion)
		if p != nil {
			_ = p.Close()
		}
		if err == nil || !strings.Contains(err.Error(), "NEWER version") {
			t.Fatalf("refusal = %v", err)
		}
		after, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(before, after) {
			t.Fatal("newer database changed")
		}
	}
}

func TestMigrationBackupsRetainNewestFiveIncludingLegacy(t *testing.T) {
	path := filepath.Join(t.TempDir(), "backup.db")
	p, err := Open(t.Context(), path, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = p.Close() }()
	legacy := path + ".pre-v0.bak"
	if err := os.WriteFile(legacy, []byte("keep existing recovery snapshot"), 0600); err != nil {
		t.Fatal(err)
	}
	old := time.Unix(1, 0)
	if err := os.Chtimes(legacy, old, old); err != nil {
		t.Fatal(err)
	}
	foreign := path + ".pre-vnotes.bak"
	if err := os.WriteFile(foreign, []byte("foreign"), 0600); err != nil {
		t.Fatal(err)
	}
	link := path + ".pre-v99-123.bak"
	if err := os.Symlink(foreign, link); err != nil {
		t.Fatal(err)
	}
	failure := errors.New("retry migration")
	var backups []string
	for i := range 7 {
		result, err := p.Migrate(t.Context(), func(context.Context, *sql.Tx) error { return failure })
		if !errors.Is(err, failure) {
			t.Fatalf("attempt %d: %v", i, err)
		}
		backups = append(backups, result.Backup)
		at := old.Add(time.Duration(i+1) * time.Hour)
		if err := os.Chtimes(result.Backup, at, at); err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			body, err := os.ReadFile(legacy)
			if err != nil || string(body) != "keep existing recovery snapshot" {
				t.Fatalf("legacy changed: %q %v", body, err)
			}
		}
	}
	assertVersion(t, p, 0)
	for i, backup := range append([]string{legacy}, backups...) {
		info, err := os.Stat(backup)
		if i < 3 {
			if !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("old backup retained: %s %v", backup, err)
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0600 {
			t.Fatalf("backup mode: %v", info.Mode())
		}
		version, err := fileUserVersion(t.Context(), backup)
		if err != nil || version != 0 {
			t.Fatalf("backup version: %d %v", version, err)
		}
	}
	for _, preserved := range []string{foreign, link} {
		if _, err := os.Lstat(preserved); err != nil {
			t.Fatal(err)
		}
	}
}

func assertVersion(t *testing.T, p *Pools, want int) {
	t.Helper()
	var got int
	if err := p.Read().QueryRowContext(t.Context(), "PRAGMA user_version").Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("user_version=%d, want %d", got, want)
	}
}

func TestFailedMigrationBackupLeavesPriorCopiesAndSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "backup.db")
	p, err := Open(t.Context(), path, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = p.Close() }()
	prior, err := p.backupMigration(t.Context(), 0)
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(prior)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := p.backupMigration(ctx, 0); !errors.Is(err, context.Canceled) {
		t.Fatalf("backup failure: %v", err)
	}
	matches, err := filepath.Glob(path + ".pre-v*.bak")
	if err != nil || len(matches) != 1 || matches[0] != prior {
		t.Fatalf("failed capture leaked or removed copies: %v %v", matches, err)
	}
	after, err := os.ReadFile(prior)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("prior backup changed: %v", err)
	}
	assertVersion(t, p, 0)
}
