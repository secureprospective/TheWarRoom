package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/db"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
)

func TestOfflineConsoleRecovery(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "server.db")
	pools, err := db.Open(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = pools.Close() }()
	store := accountstore.New(pools)
	if err := store.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	credentials := accounts.RawCredentials{Username: "owner", Password: "old-password"}
	account, err := accounts.New(store).LocalBootstrap(t.Context(), credentials)
	if err != nil {
		t.Fatal(err)
	}
	key := bytes.Repeat([]byte{42}, 32)
	keyPath := filepath.Join(dir, "offline-key")
	if err := os.WriteFile(keyPath, key, 0600); err != nil {
		t.Fatal(err)
	}
	args := []string{"-accounts-db", path, "-offline-key-file", keyPath, "-offline-key-sha256", fmt.Sprintf("%x", sha256.Sum256(key))}
	input := `{"username":"owner","password":"new-password","reason":"Lost factors in fixture"}`
	var output bytes.Buffer
	for _, bad := range []string{input + ` {}`, strings.Replace(input, `"reason"`, `"unexpected"`, 1)} {
		if err := runRecovery(t.Context(), args, strings.NewReader(bad), &output); err == nil {
			t.Fatal("invalid request accepted")
		}
	}
	//nolint:gosec // Deliberately insecure fixture verifies rejection of exposed offline keys.
	if err := os.Chmod(keyPath, 0644); err != nil {
		t.Fatal(err)
	}
	if err := runRecovery(t.Context(), args, strings.NewReader(input), &output); err == nil {
		t.Fatal("world-readable offline key accepted")
	}
	if err := os.Chmod(keyPath, 0600); err != nil {
		t.Fatal(err)
	}
	if err := runRecovery(t.Context(), args, strings.NewReader(input), &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "Account recovered") || strings.Contains(output.String(), "new-password") {
		t.Fatalf("unsafe or missing response: %s", output.String())
	}
	recovered, err := accounts.New(store).Authenticate(t.Context(), accounts.RawCredentials{Username: "owner", Password: "new-password"})
	if err != nil || recovered != account {
		t.Fatalf("recovery login/identity: %v", err)
	}
	var count int
	if err := pools.Read().QueryRowContext(t.Context(), `SELECT count(*) FROM account_security_audit WHERE operation = 'offline_key_reset'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("audit count %d: %v", count, err)
	}
}
