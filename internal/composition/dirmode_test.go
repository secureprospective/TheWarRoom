package composition

import (
	"log"
	"os"
	"path/filepath"
	"testing"
)

// The data dir holds the league databases and logs: owner-only, and an older 0750 dir is tightened.
func TestConfigDirAndLogsAreOwnerOnly(t *testing.T) {
	cfg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)
	if err := os.MkdirAll(filepath.Join(cfg, "TheWarRoom"), 0o750); err != nil {
		t.Fatal(err)
	}
	dir, err := configDir()
	if err != nil {
		t.Fatalf("configDir: %v", err)
	}
	if err := setupLogging(dir); err != nil {
		t.Fatalf("setupLogging: %v", err)
	}
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	for _, d := range []string{dir, filepath.Join(dir, "logs")} {
		info, err := os.Stat(d)
		if err != nil {
			t.Fatal(err)
		}
		if mode := info.Mode().Perm(); mode != 0o700 {
			t.Errorf("%s mode = %v, want 0700", d, mode)
		}
	}
	logs, err := filepath.Glob(filepath.Join(dir, "logs", "thewarroom-*.log"))
	if err != nil || len(logs) != 1 {
		t.Fatalf("log files = %v, %v", logs, err)
	}
	if info, err := os.Stat(logs[0]); err != nil || info.Mode().Perm() != 0o600 {
		t.Errorf("log file mode = %v, %v, want 0600", info.Mode().Perm(), err)
	}
}
