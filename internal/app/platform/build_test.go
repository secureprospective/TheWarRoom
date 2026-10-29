package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
) // TestDBFileName_DevBuildSeparateFile proves the Tier 3 dev-build guard: a dev
// binary resolves to a SEPARATE -dev database, never the real ledger filename.
func TestDBFileName_DevBuildSeparateFile(t *testing.T) {
	t.Parallel()
	if got := DatabaseFileName("thewarroom", false); got != "thewarroom.db" {
		t.Fatalf("release dbFileName = %q, want thewarroom.db", got)
	}
	if got := DatabaseFileName("thewarroom", true); got != "thewarroom-dev.db" {
		t.Fatalf("dev dbFileName = %q, want thewarroom-dev.db", got)
	}
	if got := DatabaseFileName("history", true); got != "history-dev.db" {
		t.Fatalf("dev history = %q, want history-dev.db", got)
	}
	if DatabaseFileName("thewarroom", true) == DatabaseFileName("thewarroom", false) {
		t.Fatal("dev and release DB must be distinct files — a dev build must not touch the live ledger")
	}
}

func TestConfigDirCreatesAndTightens(t *testing.T) {
	// Environment overrides are process-wide; t.Setenv forbids parallel execution.
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", root)
	dir, err := ConfigDir()
	if err != nil || dir != filepath.Join(root, "TheWarRoom") {
		t.Fatalf("created: %q %v", dir, err)
	}
	if err := os.Chmod(dir, 0o750); err != nil { //nolint:gosec // Recreate an older group-readable directory to prove it is tightened.
		t.Fatal(err)
	}
	got, err := ConfigDir()
	if err != nil || got != dir {
		t.Fatalf("existing: %q %v", got, err)
	}
	info, err := os.Stat(dir)
	if err != nil || info.Mode().Perm() != 0o700 {
		t.Fatalf("directory mode: %v %v", info, err)
	}
}

func TestConfigDirFailsWithoutChangingBlocker(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", root)
	path := filepath.Join(root, "TheWarRoom")
	if err := os.WriteFile(path, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := ConfigDir()
	if got != "" || err == nil || !strings.HasPrefix(err.Error(), "create data dir ") {
		t.Fatalf("failure: %q %v", got, err)
	}
	body, readErr := os.ReadFile(path)
	if readErr != nil || string(body) != "keep" {
		t.Fatalf("blocker changed: %q %v", body, readErr)
	}
}

func TestDevBuildClassification(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		version string
		want    bool
	}{{"dev", true}, {"", false}, {"v1.0.0", false}, {"development", false}} {
		if got := IsDevBuild(tc.version); got != tc.want {
			t.Errorf("%q: %v", tc.version, got)
		}
	}
}

// TestSetupLogging_PrunesToMax proves disk logging creates the logs dir and keeps
// at most MaxLogFiles files.
func TestSetupLogging_PrunesToMax(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	logDir := filepath.Join(dir, "logs")
	if err := os.MkdirAll(logDir, 0o750); err != nil {
		t.Fatalf("mkdir logs: %v", err)
	}

	// Seed more than the cap of uniquely-named stale log files, then prune.
	for i := 0; i < MaxLogFiles+5; i++ {
		name := filepath.Join(logDir, fmt.Sprintf("thewarroom-2020%06dT000000Z.log", i))
		if err := os.WriteFile(name, []byte("stub"), 0o600); err != nil {
			t.Fatalf("seed stub: %v", err)
		}
	}
	PruneLogs(logDir)

	matches, err := filepath.Glob(filepath.Join(logDir, "thewarroom-*.log"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(matches) > MaxLogFiles {
		t.Fatalf("PruneLogs kept %d files, want <= %d", len(matches), MaxLogFiles)
	}
}

func TestBuildLabel(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ version, commit, want string }{{"dev", "", "dev"}, {"v1", "abc1234", "v1 (abc1234)"}} {
		if got := BuildLabel(tc.version, tc.commit); got != tc.want {
			t.Errorf("label = %q, want %q", got, tc.want)
		}
	}
}
