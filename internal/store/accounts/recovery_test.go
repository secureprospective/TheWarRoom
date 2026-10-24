package accounts

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"path/filepath"
	"testing"

	identity "github.com/secureprospective/TheWarRoom/internal/accounts"
)

func TestRecoveryResetPersistsAcrossReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "server.db")
	store, pools := openStore(t, path)
	credential := fixtureCredential(t)
	if err := store.Insert(t.Context(), credential, true); err != nil {
		t.Fatal(err)
	}
	key := bytes.Repeat([]byte{42}, 32)
	service := identity.NewRecovery(identity.New(store), store, sha256.Sum256(key))
	raw := identity.RawConsoleRecovery{Username: credential.Username, Password: "new-password", Reason: "Offline test"}
	if err := service.ConsoleRecover(t.Context(), raw, key); err != nil {
		t.Fatal(err)
	}
	if err := pools.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, _ := openStore(t, path)
	generation, err := reopened.AuthGeneration(t.Context(), credential.ID)
	if err != nil || generation != 1 {
		t.Fatalf("revocation did not persist: %d %v", generation, err)
	}
	if _, err := identity.New(reopened).Authenticate(t.Context(), identity.RawCredentials{Username: credential.Username, Password: raw.Password}); err != nil {
		t.Fatal(err)
	}
	if err := reopened.ResetAccount(t.Context(), credential.Username, credential.PasswordHash, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ""); !errors.Is(err, identity.ErrRecoveryRefused) {
		t.Fatalf("unissued code admitted: %v", err)
	}
	var audits int
	if err := reopened.pools.Read().QueryRowContext(t.Context(), `SELECT count(*) FROM account_security_audit`).Scan(&audits); err != nil || audits != 1 {
		t.Fatalf("audit did not persist: %d %v", audits, err)
	}
}
