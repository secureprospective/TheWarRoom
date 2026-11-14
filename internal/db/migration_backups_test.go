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

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

func TestMigrationBackupRetriesReuseVerifiedCopy(t *testing.T) {
	for _, retries := range []int{1, 12} {
		t.Run(fmtRetry(retries), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "server.db")
			var first string
			for i := range retries {
				p, err := Open(t.Context(), path, 1)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := p.Write().ExecContext(t.Context(), "CREATE TABLE IF NOT EXISTS held (id TEXT)"); err != nil {
					t.Fatal(err)
				}
				backup, err := p.BackupBeforeChange(t.Context())
				if err != nil {
					t.Fatal(err)
				}
				if i == 0 {
					first = backup
				}
				if backup != first {
					t.Fatalf("retry did not reuse copy: %s != %s", backup, first)
				}
				if err := p.Close(); err != nil {
					t.Fatal(err)
				}
			}
			assertOneMigrationBackup(t, path)
		})
	}
}

func fmtRetry(n int) string {
	if n == 1 {
		return "once"
	}
	return "twelve_restarts"
}

func assertOneMigrationBackup(t *testing.T, path string) string {
	t.Helper()
	matches, err := filepath.Glob(path + ".pre-v*.bak")
	if err != nil || len(matches) != 1 {
		t.Fatalf("backups=%v err=%v; want exactly one", matches, err)
	}
	valid, err := verifyMigrationBackup(t.Context(), matches[0])
	if err != nil || !valid {
		t.Fatalf("backup verification=%v err=%v", valid, err)
	}
	return matches[0]
}

func TestMigrationBackupRemovesIncompleteOnStart(t *testing.T) {
	for _, body := range [][]byte{nil, []byte("SQLite format 3\x00truncated")} {
		t.Run(stringName(body), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "server.db")
			leftover := path + ".pre-v0-123456.bak"
			for name, data := range map[string][]byte{leftover: body, leftover + "-journal": []byte("interrupted journal"), path + ".pre-vnotes.bak": []byte("foreign")} {
				if err := os.WriteFile(name, data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			p, err := Open(t.Context(), path, 1)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = p.Close() }()
			result, err := p.Migrate(t.Context(), func(ctx context.Context, tx *sql.Tx) error {
				for _, name := range []string{leftover, leftover + "-journal"} {
					if _, err := os.Lstat(name); !errors.Is(err, os.ErrNotExist) {
						t.Fatalf("leftover retained: %s %v", name, err)
					}
				}
				_, err := tx.ExecContext(ctx, "CREATE TABLE completed (id TEXT)")
				return err
			})
			if err != nil || result.Backup == "" {
				t.Fatalf("migration: %+v %v", result, err)
			}
			// The unrelated name is excluded from the exact backup count.
			if err := os.Remove(path + ".pre-vnotes.bak"); err != nil {
				t.Fatal(err)
			}
			assertOneMigrationBackup(t, path)
			assertVersion(t, p, 1)
		})
	}
}

func stringName(body []byte) string {
	if len(body) == 0 {
		return "empty"
	}
	return "truncated"
}

func TestMigrationBackupReplacementFailurePreservesEarlier(t *testing.T) {
	for _, fail := range []string{"canceled", "limited_writer"} {
		t.Run(fail, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "server.db")
			p, err := Open(t.Context(), path, 1)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = p.Close() }()
			old, err := p.BackupBeforeChange(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadFile(old)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := p.Write().ExecContext(t.Context(), "CREATE TABLE changed (value BLOB); INSERT INTO changed VALUES (zeroblob(8192))"); err != nil {
				t.Fatal(err)
			}
			ctx := t.Context()
			if fail == "canceled" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			} else {
				conn, err := p.Write().Conn(ctx)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := sqlite.Limit(conn, sqlite3.SQLITE_LIMIT_ATTACHED, 0); err != nil {
					t.Fatal(err)
				}
				if err := conn.Close(); err != nil {
					t.Fatal(err)
				}
			}
			_, err = p.BackupBeforeChange(ctx)
			if err == nil {
				t.Fatal("replacement unexpectedly succeeded")
			}
			if fail == "limited_writer" && !strings.Contains(err.Error(), "pre-migration backup: SQL logic error: too many attached databases") {
				t.Fatalf("failure did not come from replacement VACUUM: %v", err)
			}
			after, err := os.ReadFile(old)
			if err != nil || !bytes.Equal(before, after) {
				t.Fatalf("old backup changed: %v", err)
			}
			assertOneMigrationBackup(t, path)
		})
	}
}

func TestMigrationBackupDeduplicatesExistingCompleteCopies(t *testing.T) {
	path := filepath.Join(t.TempDir(), "server.db")
	p, err := Open(t.Context(), path, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = p.Close() }()
	first, err := p.BackupBeforeChange(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(first)
	if err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{"111", "222"} {
		if err := os.WriteFile(path+".pre-v0-"+suffix+".bak", body, 0600); err != nil { //nolint:gosec // Only numeric fixture suffixes under t.TempDir; no external path input.
			t.Fatal(err)
		}
	}
	if _, err := p.BackupBeforeChange(t.Context()); err != nil {
		t.Fatal(err)
	}
	assertOneMigrationBackup(t, p.path)
}
