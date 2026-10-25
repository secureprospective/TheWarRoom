package main

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"filippo.io/age"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/offsite"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

func TestCapturedHostedSetCarriesCompatibleCheckpoint(t *testing.T) {
	pools, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "server.db"), history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = pools.Close() }()
	if _, err := hostedServices(t.Context(), pools, make([]byte, 32), ""); err != nil {
		t.Fatal(err)
	}
	config := t.TempDir()
	for _, name := range []string{"authenticator.key", "origin.pem", "origin.key", "endpoint-registry.csv", "backup.json"} {
		if err := os.WriteFile(filepath.Join(config, name), []byte("fixture"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	// The restore-side key is never passed to the hosted capture API.
	offline, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	sealed, _, err := offsite.Capture(t.Context(), pools, config, t.TempDir(), offline.Recipient().String(), hostedCheckpoint)
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(sealed)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	dest := filepath.Join(t.TempDir(), "restored")
	if _, err := offsite.Restore(f, offline, dest); err != nil {
		t.Fatal(err)
	}
	if err := runCheckpoint(t.Context(), []string{"-db", filepath.Join(dest, "server.db"), "-check", filepath.Join(dest, "checkpoint.json")}, io.Discard); err != nil {
		t.Fatalf("saved snapshot/checkpoint not matched: %v", err)
	}
}
