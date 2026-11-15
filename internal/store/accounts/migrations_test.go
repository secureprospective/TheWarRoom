package accounts

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/db"
)

func accountsVersion(t *testing.T, pools *db.Pools) int {
	t.Helper()
	var version int
	if err := pools.Read().QueryRowContext(t.Context(), `SELECT version FROM schema_version WHERE name=?`, SchemaName).Scan(&version); err != nil {
		t.Fatal(err)
	}
	return version
}

func TestAccountsSchemaFreshFileIsVersioned(t *testing.T) {
	_, pools := openStore(t, filepath.Join(t.TempDir(), "server.db"))
	if got := accountsVersion(t, pools); got != SchemaVersion() {
		t.Fatalf("accounts version = %d, want %d", got, SchemaVersion())
	}
	var user int
	if err := pools.Read().QueryRowContext(t.Context(), `PRAGMA user_version`).Scan(&user); err != nil || user != 0 {
		t.Fatalf("accounts must not stamp the shared user_version: %d %v", user, err)
	}
}

// A file written by a build that created the tables at runtime adopts v1 without rewriting rows.
func TestAccountsSchemaExistingFileAdopts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "server.db")
	pools, err := db.Open(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := pools.BeginWrite(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if err := adoptAccounts(t.Context(), tx); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	old := New(pools)
	if err := old.initializeAudit(t.Context()); err != nil {
		t.Fatal(err)
	}
	credential := fixtureCredential(t)
	if err := old.Insert(t.Context(), credential, false); err != nil {
		t.Fatal(err)
	}
	var epoch string
	if err := pools.Read().QueryRowContext(t.Context(), `SELECT epoch FROM account_auth_epoch`).Scan(&epoch); err != nil {
		t.Fatal(err)
	}
	if err := pools.Close(); err != nil {
		t.Fatal(err)
	}
	store, reopened := openStore(t, path)
	if got := accountsVersion(t, reopened); got != 1 {
		t.Fatalf("adopted version = %d", got)
	}
	held, err := store.Credential(t.Context(), credential.Username)
	if err != nil || held != credential {
		t.Fatalf("credential = %+v err=%v", held, err)
	}
	var after string
	if err := reopened.Read().QueryRowContext(t.Context(), `SELECT epoch FROM account_auth_epoch`).Scan(&after); err != nil || after != epoch {
		t.Fatalf("auth epoch rewritten: %q -> %q (%v)", epoch, after, err)
	}
	if _, err := store.VerifyAudit(t.Context()); err != nil {
		t.Fatal(err)
	}
	if backups, _ := filepath.Glob(path + ".pre-v*.bak"); len(backups) != 0 {
		t.Fatalf("adoption is idempotent DDL and takes no backup: %v", backups)
	}
}

func TestAccountsSchemaNewerVersionRefused(t *testing.T) {
	path := filepath.Join(t.TempDir(), "server.db")
	_, pools := openStore(t, path)
	if _, err := pools.Write().ExecContext(t.Context(), `UPDATE schema_version SET version=? WHERE name=?`, SchemaVersion()+1, SchemaName); err != nil {
		t.Fatal(err)
	}
	err := New(pools).Initialize(t.Context())
	if err == nil || !strings.Contains(err.Error(), "NEWER") {
		t.Fatalf("newer accounts schema accepted: %v", err)
	}
}
