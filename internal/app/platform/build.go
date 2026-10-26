package platform

import (
	"fmt"
	"os"
	"path/filepath"
)

func BuildLabel(version, commit string) string {
	if commit == "" {
		return version
	}
	return version + " (" + commit + ")"
}

// ConfigDir creates or tightens the app data directory to owner-only.
func ConfigDir() (string, error) {
	cfg, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("user config dir: %w", err)
	}
	dir := filepath.Join(cfg, "TheWarRoom")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create data dir %q: %w", dir, err)
	}
	if err := os.Chmod(dir, 0o700); err != nil { //nolint:gosec // a directory needs its x bit; 0700 is owner-only
		return "", fmt.Errorf("restrict data dir %q: %w", dir, err)
	}
	return dir, nil
}

func IsDevBuild(version string) bool { return version == "dev" }

// DatabaseFileName isolates development databases from the production ledger.
func DatabaseFileName(base string, devBuild bool) string {
	if devBuild {
		return base + "-dev.db"
	}
	return base + ".db"
}
