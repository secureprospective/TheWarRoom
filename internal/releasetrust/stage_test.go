package releasetrust

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func desktopRepository(t *testing.T, version string) *repository {
	t.Helper()
	r := newRepository(t)
	publishDesktop(t, r, version, 2, "")
	return r
}

func publishDesktop(t *testing.T, r *repository, version string, metadataVersion int64, expired string) {
	t.Helper()
	payload := []byte("#!/bin/sh\nprintf 'updated desktop opened'\n")
	digest := sha256.Sum256(payload)
	raw := RawDesktopRelease{Version: version, Target: "desktop-executable", SHA256: hex.EncodeToString(digest[:])}
	data, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	r.payloads["desktop-"+runtime.GOOS+"-"+runtime.GOARCH+".json"] = data
	r.payloads[raw.Target] = payload
	r.publish(t, metadataVersion, expired)
}

func TestValidStagedDesktopLaunchAndInterruptedRecovery(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX launch fixture; Windows packaging is a separate drill")
	}
	r := desktopRepository(t, "v1.1.0")
	directory := t.TempDir()
	path, err := r.client(t, directory).Stage("v1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	// A crash after staging, or a damaged staged file, cannot authorize its launch on retry.
	if err := os.WriteFile(path, []byte("interrupted or corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, ".staging-interrupted"), []byte("partial"), 0600); err != nil {
		t.Fatal(err)
	}
	path, err = r.client(t, directory).Stage("v1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command(path).Output()
	if err != nil || string(output) != "updated desktop opened" {
		t.Fatalf("staged launch: %q %v", output, err)
	}
}

func TestStageRejectsUnsafeUpdatesWithoutNativeWrites(t *testing.T) {
	for _, failure := range []string{"corruption", "root", "timestamp", "snapshot", "targets", "downgrade", "remembered-downgrade", "manifest"} {
		t.Run(failure, func(t *testing.T) {
			r := desktopRepository(t, "v1.1.0")
			directory := t.TempDir()
			native := filepath.Join(t.TempDir(), "league.db")
			before := []byte("native league data must never change")
			if err := os.WriteFile(native, before, 0600); err != nil {
				t.Fatal(err)
			}
			installed := "v1.0.0"
			switch failure {
			case "corruption":
				for key := range r.files {
					if strings.HasSuffix(key, ".desktop-executable") {
						r.files[key] = []byte("corrupt")
					}
				}
			case "root":
				r = newRepository(t)
				// Existing root expiry coverage exercises the same Download boundary.
				r.root.Signed.Expires = r.root.Signed.Expires.AddDate(-1, 0, 0)
				r.files["/metadata/1.root.json"] = signed(t, r.root, r.keys["root"][:2]...)
			case "timestamp", "snapshot", "targets":
				publishDesktop(t, r, "v1.1.0", 2, failure)
			case "downgrade":
				installed = "v1.2.0"
			case "remembered-downgrade":
				if err := os.WriteFile(filepath.Join(directory, "desktop-version"), []byte("v1.2.0"), 0600); err != nil {
					t.Fatal(err)
				}
			case "manifest":
				r.payloads["desktop-"+runtime.GOOS+"-"+runtime.GOARCH+".json"] = []byte(`{"version":"v1.1.0","target":"../escape","sha256":"no"}`)
				r.publish(t, 2, "")
			}
			if path, err := r.client(t, directory).Stage(installed); err == nil || path != "" {
				t.Fatal("unsafe release staged")
			}
			files, err := filepath.Glob(filepath.Join(directory, "staged-*"))
			if err != nil || len(files) != 0 {
				t.Fatalf("rejected update staged: %v %v", files, err)
			}
			after, err := os.ReadFile(native)
			if err != nil || string(after) != string(before) {
				t.Fatal("native data changed")
			}
		})
	}
}

func TestRememberedVersionRejectsNewlySignedDowngradeAfterRestart(t *testing.T) {
	r := desktopRepository(t, "v1.2.0")
	directory := t.TempDir()
	if _, err := r.client(t, directory).Stage("v1.0.0"); err != nil {
		t.Fatal(err)
	}
	publishDesktop(t, r, "v1.1.0", 3, "")
	if path, err := r.client(t, directory).Stage("v1.0.0"); err == nil || path != "" {
		t.Fatal("new signatures authorized an application downgrade")
	}
	saved, err := os.ReadFile(filepath.Join(directory, "desktop-version"))
	if err != nil || string(saved) != "v1.2.0" {
		t.Fatal("rejection changed the application floor")
	}
}

func TestInterruptedDownloadRetriesWithRetainedTrust(t *testing.T) {
	r := desktopRepository(t, "v1.1.0")
	directory := t.TempDir()
	var name string
	var payload []byte
	for key, data := range r.files {
		if strings.HasSuffix(key, ".desktop-executable") {
			name, payload = key, data
			delete(r.files, key)
			break
		}
	}
	if _, err := r.client(t, directory).Stage("v1.0.0"); err == nil {
		t.Fatal("interrupted download succeeded")
	}
	r.files[name] = payload
	if _, err := r.client(t, directory).Stage("v1.0.0"); err != nil {
		t.Fatalf("restart failed to recover: %v", err)
	}
}
