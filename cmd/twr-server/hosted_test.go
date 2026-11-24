package main

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHostedInstanceLockRefusesSecondCopyBeforeDatabaseOpen(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "server.db")
	if err := os.WriteFile(dbPath, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	lock, err := lockHostedDatabase(dbPath, 0)
	if err != nil {
		t.Fatal(err)
	}
	args := []string{"-db", dbPath}
	err = startHosted(context.Background(), args, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "the server is running on "+dbPath+"; stop it first") {
		t.Fatalf("unexpected startup result: %v", err)
	}
	if info, err := os.Stat(dbPath); err != nil || info.Size() != 0 {
		t.Fatalf("refused start touched the database: %v", err)
	}
	if err := lock.Close(); err != nil {
		t.Fatal(err)
	}
	if err := startHosted(context.Background(), args, io.Discard); err == nil || strings.Contains(err.Error(), "the server is running") {
		t.Fatalf("startup did not pass lock after release: %v", err)
	}
}

func TestTrustedProxyFlagAcceptsOnlyLoopback(t *testing.T) {
	if hop, err := trustedProxy(""); err != nil || hop.IsValid() {
		t.Fatalf("default trusts a proxy: %v %v", hop, err)
	}
	if hop, err := trustedProxy("127.0.0.1"); err != nil || hop.String() != "127.0.0.1" {
		t.Fatalf("loopback hop refused: %v %v", hop, err)
	}
	for _, bad := range []string{"192.0.2.9", "0.0.0.0", "localhost", "fe80::1%lo"} {
		if _, err := trustedProxy(bad); err == nil {
			t.Errorf("%q accepted as a trusted hop", bad)
		}
	}
}

func TestHostedPublicHostFlagRequiresOneBareDNSName(t *testing.T) {
	config, err := parseHostedFlags([]string{"-public-host", "league.example.org"})
	if err != nil || config.publicHost == nil || *config.publicHost != "league.example.org" {
		t.Fatalf("valid public host config=%+v err=%v", config, err)
	}
	if _, err := parseHostedFlags([]string{"-public-host", "https://league.example.org"}); err == nil {
		t.Fatal("origin accepted as public host")
	}
}

func TestHostedBackupRequiresActualRuntimeSettings(t *testing.T) {
	dir := t.TempDir()
	files := []string{filepath.Join(dir, "authenticator.key"), filepath.Join(dir, "origin.pem"), filepath.Join(dir, "origin.key"), filepath.Join(dir, "endpoint-registry.csv")}
	if err := checkBackupFiles(dir, files); err != nil {
		t.Fatal(err)
	}
	files[0] = filepath.Join(t.TempDir(), "authenticator.key")
	if err := checkBackupFiles(dir, files); err == nil {
		t.Fatal("backup omitted the running authenticator key")
	}
}

// Console commands that say "server stopped" refuse while a server holds the database's lock; pin-football's
// -status-socket mode talks to the running server on purpose and is never refused for it.
func TestStoppedServerCommandsRefuseWhileServerRuns(t *testing.T) {
	dir := t.TempDir()
	db := filepath.Join(dir, "server.db")
	if err := os.WriteFile(db, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	held, err := lockStoppedServer(db)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = held.Close() }()
	file := filepath.Join(dir, "input")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	league, stage := "native.locked", filepath.Join(dir, "stage")
	for _, args := range [][]string{
		{"conversion-proof", "-db", db},
		{"create-league", "-db", db, "-league", league, "-name", "Locked", "-preset", "default_32", "-season", "2026"},
		{"delete-league", "-db", db, "-league", league},
		{"export-league", "-db", db, "-league", league, "-out", stage},
		{"import-league", "-db", db, "-in", file, "-stage", stage},
		{"finish-move", "-db", db, "-league", league},
		{"cancel-move", "-db", db, "-league", league},
		{"restore-league", "-db", db, "-league", league, "-stage", stage},
		{"import-model-scores", "-db", db, "-file", file},
		{"refresh-nfl-injuries", "-db", db, "-season", "2026"},
		{"refresh-nfl-schedule", "-db", db, "-league", league, "-season", "2026"},
		{"pin-football", "-db", db, "-league", league, "-football", strings.Repeat("e", 64)},
	} {
		if err := run(t.Context(), args, io.Discard); err == nil || !strings.Contains(err.Error(), "stop it first") {
			t.Errorf("%s with the server running: err = %v, want refused (stop it first)", args[0], err)
		}
	}
	socket := filepath.Join(dir, "status.sock")
	err = run(t.Context(), []string{"pin-football", "-status-socket", socket, "-league", league, "-football", strings.Repeat("e", 64)}, io.Discard)
	if err != nil && strings.Contains(err.Error(), "stop it first") {
		t.Fatalf("pin-football -status-socket refused for the lock: %v", err)
	}
}
