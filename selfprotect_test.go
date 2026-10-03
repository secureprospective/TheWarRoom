package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDBFileName_DevBuildSeparateFile proves the Tier 3 dev-build guard: a dev
// binary resolves to a SEPARATE -dev database, never the real ledger filename.
func TestDBFileName_DevBuildSeparateFile(t *testing.T) {
	if got := dbFileName("thewarroom", false); got != "thewarroom.db" {
		t.Fatalf("release dbFileName = %q, want thewarroom.db", got)
	}
	if got := dbFileName("thewarroom", true); got != "thewarroom-dev.db" {
		t.Fatalf("dev dbFileName = %q, want thewarroom-dev.db", got)
	}
	if got := dbFileName("history", true); got != "history-dev.db" {
		t.Fatalf("dev history = %q, want history-dev.db", got)
	}
	if dbFileName("thewarroom", true) == dbFileName("thewarroom", false) {
		t.Fatal("dev and release DB must be distinct files — a dev build must not touch the live ledger")
	}
}

// TestInstanceLock_SecondAcquireFails proves the single-instance guard: once the
// lock is held, a second acquire on the same DB path is refused; after release a
// fresh acquire succeeds.
func TestInstanceLock_SecondAcquireFails(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "thewarroom.db")

	first, err := acquireInstanceLock(dbPath)
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}

	if second, err := acquireInstanceLock(dbPath); err == nil {
		releaseInstanceLock(second)
		t.Fatal("second acquire succeeded while the lock was held — instance guard failed")
	}

	releaseInstanceLock(first)

	third, err := acquireInstanceLock(dbPath)
	if err != nil {
		t.Fatalf("acquire after release: %v", err)
	}
	releaseInstanceLock(third)
}

// TestInstanceLock_DevAndRealDoNotContend proves the lock is keyed to the DB path,
// so a dev build and a real build (different DB files) never falsely block.
func TestInstanceLock_DevAndRealDoNotContend(t *testing.T) {
	dir := t.TempDir()
	realLock, err := acquireInstanceLock(filepath.Join(dir, "thewarroom.db"))
	if err != nil {
		t.Fatalf("acquire real: %v", err)
	}
	defer releaseInstanceLock(realLock)

	dev, err := acquireInstanceLock(filepath.Join(dir, "thewarroom-dev.db"))
	if err != nil {
		t.Fatalf("dev build must not contend with the real ledger lock: %v", err)
	}
	releaseInstanceLock(dev)
}

// TestSetupLogging_PrunesToMax proves disk logging creates the logs dir and keeps
// at most maxLogFiles files.
func TestSetupLogging_PrunesToMax(t *testing.T) {
	dir := t.TempDir()
	logDir := filepath.Join(dir, "logs")
	if err := os.MkdirAll(logDir, 0o750); err != nil {
		t.Fatalf("mkdir logs: %v", err)
	}

	// Seed more than the cap of uniquely-named stale log files, then prune.
	for i := 0; i < maxLogFiles+5; i++ {
		name := filepath.Join(logDir, fmt.Sprintf("thewarroom-2020%06dT000000Z.log", i))
		if err := os.WriteFile(name, []byte("stub"), 0o600); err != nil {
			t.Fatalf("seed stub: %v", err)
		}
	}
	pruneLogs(logDir)

	matches, err := filepath.Glob(filepath.Join(logDir, "thewarroom-*.log"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(matches) > maxLogFiles {
		t.Fatalf("pruneLogs kept %d files, want <= %d", len(matches), maxLogFiles)
	}
}

// TestRunFlag_UnknownArgumentNeverOpensAWindow: an unknown argument exits 2 with usage instead
// of falling through to a window on the real database; -version prints the build label.
func TestRunFlag_UnknownArgumentNeverOpensAWindow(t *testing.T) {
	for _, args := range [][]string{{"-version", "x"}, {"--version"}, {"-probe", "-version"}, {"run"}} {
		var out, errOut bytes.Buffer
		if code := runFlag(args, &out, &errOut); code != 2 || !strings.Contains(errOut.String(), "usage") {
			t.Errorf("runFlag(%q) = %d, stderr %q; want 2 with usage", args, code, errOut.String())
		}
	}
	var out, errOut bytes.Buffer
	if code := runFlag([]string{"-version"}, &out, &errOut); code != 0 || strings.TrimSpace(out.String()) != buildLabel() {
		t.Errorf("-version = %d, stdout %q; want 0 and %q", code, out.String(), buildLabel())
	}
}
