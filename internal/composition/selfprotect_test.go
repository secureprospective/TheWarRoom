package composition

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPackageSmokeRejectsInvalidDirectoryAndHeldLock(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	for _, path := range []string{"relative", filepath.Join(directory, "missing")} {
		var stderr bytes.Buffer
		if code := packageSmoke(path, Embedded{}, &stderr); code != 2 {
			t.Fatalf("invalid directory returned %d: %s", code, &stderr)
		}
	}
	lock, err := acquireInstanceLock(filepath.Join(directory, "smoke.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer releaseInstanceLock(lock)
	var stderr bytes.Buffer
	if code := packageSmoke(directory, Embedded{}, &stderr); code != 1 || !strings.Contains(stderr.String(), "another instance") {
		t.Fatalf("second smoke returned %d: %s", code, &stderr)
	}
	if _, err := os.Stat(filepath.Join(directory, "ready")); !os.IsNotExist(err) {
		t.Fatalf("rejected smoke wrote readiness: %v", err)
	}
	if _, err := os.Stat(filepath.Join(directory, "smoke.db")); !os.IsNotExist(err) {
		t.Fatalf("rejected smoke opened a database: %v", err)
	}
}

func TestNativeStartupRefusesUnsafeOriginBeforeWindow(t *testing.T) {
	var stderr bytes.Buffer
	if code := runNative("https://user:sentinel@localhost", "native.test", Embedded{}, &stderr); code != 2 {
		t.Fatalf("unsafe native startup = %d", code)
	}
	if strings.Contains(stderr.String(), "sentinel") || !strings.Contains(stderr.String(), "HTTPS origin") {
		t.Fatalf("unsafe startup diagnostic: %q", stderr.String())
	}
	// A missing native argument must not fall through to local app startup.
	stderr.Reset()
	var stdout bytes.Buffer
	if code := runFlag([]string{"-native", "https://localhost"}, &stdout, &stderr); code != 2 {
		t.Fatalf("incomplete native startup = %d", code)
	}
}
