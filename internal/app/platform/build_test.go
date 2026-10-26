package platform

import (
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
