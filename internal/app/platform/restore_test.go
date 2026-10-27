package platform

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/db"
)

func TestBackupCLIListsSortedManifestDirectoriesOnly(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"z-set", "a-set", "incomplete"} {
		path := filepath.Join(dir, "backups", name)
		if err := os.MkdirAll(path, 0o700); err != nil {
			t.Fatal(err)
		}
		if name != "incomplete" {
			if err := os.WriteFile(filepath.Join(path, "manifest.json"), []byte("{}"), 0o600); err != nil {
				t.Fatal(err)
			}
		}
	}
	service := BackupCLI{Directory: func() (string, error) { return dir, nil }}
	var stdout, stderr bytes.Buffer
	if code := service.List(&stdout, &stderr); code != 0 || stdout.String() != "a-set\nz-set\n" || stderr.Len() != 0 {
		t.Fatal(code, stdout.String(), stderr.String())
	}
	service.Directory = func() (string, error) { return t.TempDir(), nil }
	stdout.Reset()
	if code := service.List(&stdout, &stderr); code != 0 || !strings.HasPrefix(stdout.String(), "no backup sets in ") {
		t.Fatal(code, stdout.String())
	}
}

func TestBackupCLILockRefusalBeforeBackupAccessAndPrivateLog(t *testing.T) {
	config := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", config)
	dir := t.TempDir()
	lock, err := AcquireInstanceLock(filepath.Join(dir, "thewarroom-dev.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer ReleaseInstanceLock(lock)
	service := BackupCLI{Directory: func() (string, error) { return dir, nil }, DevBuild: true,
		Backups: func(string) (*db.Backups, error) { t.Fatal("prepared restore before lock admission"); return nil, nil },
	}
	var stdout, stderr bytes.Buffer
	if code := service.Restore("set", &stdout, &stderr); code != 1 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "Restore needs the app closed") || strings.Contains(stderr.String(), "already running") {
		t.Fatal(code, stdout.String(), stderr.String())
	}
	stderr.Reset()
	RestoreFailure(&stderr, errors.New("secret technical cause"), "public recovery")
	if !strings.Contains(stderr.String(), "public recovery") || strings.Contains(stderr.String(), "secret technical cause") {
		t.Fatal(stderr.String())
	}
	paths, err := filepath.Glob(filepath.Join(config, "TheWarRoom", "logs", "restore-*.log"))
	if err != nil || len(paths) != 2 {
		t.Fatal(paths, err)
	}
	found := false
	for _, path := range paths {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		found = found || strings.Contains(string(body), "secret technical cause")
	}
	if !found {
		t.Fatal("private cause missing from logs")
	}
}

func TestBackupCLIPreparationFailureDoesNotRestore(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	service := BackupCLI{Directory: func() (string, error) { return t.TempDir(), nil },
		Backups: func(string) (*db.Backups, error) { return nil, context.Canceled },
	}
	var stdout, stderr bytes.Buffer
	if code := service.Restore("../set", &stdout, &stderr); code != 1 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "could not prepare") {
		t.Fatal(code, stderr.String())
	}
}
