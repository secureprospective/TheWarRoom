package accounts

import (
	"errors"
	"path/filepath"
	"sync"
	"testing"

	identity "github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/db"
)

func openStore(t *testing.T, path string) (*Store, *db.Pools) {
	t.Helper()
	pools, err := db.Open(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	store := New(pools)
	if err := store.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	return store, pools
}

func fixtureCredential(t *testing.T) identity.Credential {
	t.Helper()
	hash, err := identity.HashPassword("sentinel-password-fixture")
	if err != nil {
		t.Fatal(err)
	}
	return identity.Credential{Account: identity.Account{ID: "12345678-1234-4234-8234-123456789abc", Username: "owner"}, PasswordHash: hash}
}

func TestUniqueUsernamesAndImmutableIdentity(t *testing.T) {
	store, pools := openStore(t, filepath.Join(t.TempDir(), "server.db"))
	first := fixtureCredential(t)
	if err := store.Insert(t.Context(), first, false); err != nil {
		t.Fatal(err)
	}
	duplicate := first
	duplicate.ID = "87654321-1234-4234-8234-123456789abc"
	if err := store.Insert(t.Context(), duplicate, false); !errors.Is(err, identity.ErrUsernameTaken) {
		t.Fatalf("duplicate: %v", err)
	}
	for _, query := range []string{
		`UPDATE accounts SET id = '87654321-1234-4234-8234-123456789abc'`,
		`UPDATE accounts SET username = 'replacement'`,
		`DELETE FROM accounts`,
		`INSERT OR REPLACE INTO accounts SELECT id, username, password_hash FROM accounts`,
	} {
		if _, err := pools.Write().ExecContext(t.Context(), query); err == nil {
			t.Fatal("account identity changed")
		}
	}
	stored, err := store.Credential(t.Context(), "owner")
	if err != nil || stored != first {
		t.Fatalf("refusal changed credentials: %v", err)
	}
}

func TestConcurrentBootstrapAndReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "server.db")
	store, pools := openStore(t, path)
	credential := fixtureCredential(t)
	var wait sync.WaitGroup
	results := make(chan error, 8)
	for range 8 {
		wait.Go(func() { results <- store.Insert(t.Context(), credential, true) })
	}
	wait.Wait()
	close(results)
	wins := 0
	for err := range results {
		if err == nil {
			wins++
		} else if !errors.Is(err, identity.ErrBootstrapClosed) {
			t.Fatal(err)
		}
	}
	if wins != 1 {
		t.Fatalf("bootstrap winners: %d", wins)
	}
	if err := pools.Close(); err != nil {
		t.Fatal(err)
	}
	store, _ = openStore(t, path)
	stored, err := store.Credential(t.Context(), credential.Username)
	if err != nil || stored != credential {
		t.Fatalf("reopen: %v", err)
	}
	if err := store.Insert(t.Context(), credential, true); !errors.Is(err, identity.ErrBootstrapClosed) {
		t.Fatalf("bootstrap reopened: %v", err)
	}
	account, err := identity.New(store).Authenticate(t.Context(), identity.RawCredentials{Username: "OWNER", Password: "sentinel-password-fixture"})
	if err != nil || account != credential.Account {
		t.Fatalf("reopened account login: %v", err)
	}
}

func TestInvalidCredentialsPreserveStoreAndCorruptionFailsClosed(t *testing.T) {
	store, pools := openStore(t, filepath.Join(t.TempDir(), "server.db"))
	first := fixtureCredential(t)
	for _, bad := range []identity.Credential{
		{Account: first.Account, PasswordHash: "malformed"},
		{Account: identity.Account{ID: "bad-id", Username: "owner"}, PasswordHash: first.PasswordHash},
		{Account: identity.Account{ID: first.ID, Username: "OWNER"}, PasswordHash: first.PasswordHash},
	} {
		if err := store.Insert(t.Context(), bad, false); err == nil {
			t.Fatal("invalid credential stored")
		}
	}
	if _, err := store.Credential(t.Context(), "owner"); !errors.Is(err, identity.ErrNotFound) {
		t.Fatalf("invalid insert changed state: %v", err)
	}
	if err := store.Insert(t.Context(), first, true); err != nil {
		t.Fatal(err)
	}
	if _, err := pools.Write().ExecContext(t.Context(), `UPDATE accounts SET password_hash = ? WHERE username = ?`, "malformed", "owner"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Credential(t.Context(), "owner"); err == nil {
		t.Fatal("corrupt stored hash returned")
	}
	if _, err := identity.New(store).Authenticate(t.Context(), identity.RawCredentials{Username: "owner", Password: "sentinel-password-fixture"}); err == nil {
		t.Fatal("corrupt stored hash authenticated")
	}
}
