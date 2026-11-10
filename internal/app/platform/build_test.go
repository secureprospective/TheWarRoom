package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
) // TestDBFileName_DevBuildSeparateFile proves the Tier 3 dev-build guard: a dev

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
