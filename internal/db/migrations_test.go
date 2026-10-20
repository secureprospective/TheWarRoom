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
		backup, err := sql.Open("sqlite", "file:"+path+".pre-v0.bak?mode=ro")
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
	if !called || result.Backup != path+".pre-v0.bak" {
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
	if _, err := fixture.ExecContext(t.Context(), "CREATE TABLE parent (id TEXT PRIMARY KEY); CREATE TABLE child (id TEXT REFERENCES parent(id)); INSERT INTO child VALUES ('absent')"); err != nil {
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

func TestMigrationBackupCollisionStopsBeforeStep(t *testing.T) {
	path := filepath.Join(t.TempDir(), "backup.db")
	p, err := Open(t.Context(), path, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = p.Close() }()
	if err := os.WriteFile(path+".pre-v0.bak", []byte("keep existing recovery snapshot"), 0600); err != nil {
		t.Fatal(err)
	}
	called := false
	_, err = p.Migrate(t.Context(), func(context.Context, *sql.Tx) error { called = true; return nil })
	if err == nil || called {
		t.Fatalf("collision: called=%v err=%v", called, err)
	}
	assertVersion(t, p, 0)
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
