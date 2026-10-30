package releasetrust

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/theupdateframework/go-tuf/v2/metadata"
)

func publishServer(t *testing.T, r *repository, directory string, version int64, custom string, expired bool) {
	t.Helper()
	targets := metadata.Targets(time.Now().Add(time.Hour))
	targets.Signed.Version = version
	info, err := metadata.TargetFile().FromBytes("server.tar.gz", []byte("untrusted executable bytes; never execute"), "sha256")
	if err != nil {
		t.Fatal(err)
	}
	if custom != "" {
		raw := json.RawMessage(custom)
		info.Custom = &raw
	}
	targets.Signed.Targets["server.tar.gz"] = info
	writeServerFile(t, filepath.Join(directory, fmt.Sprintf("%d.targets.json", version)), signed(t, targets, r.keys["targets"]...))
	snapshot := metadata.Snapshot(time.Now().Add(time.Hour))
	snapshot.Signed.Version = version
	snapshot.Signed.Meta["targets.json"] = metadata.MetaFile(version)
	writeServerFile(t, filepath.Join(directory, fmt.Sprintf("%d.snapshot.json", version)), signed(t, snapshot, r.keys["snapshot"]...))
	timestamp := metadata.Timestamp(time.Now().Add(time.Hour))
	if expired {
		timestamp.Signed.Expires = time.Now().Add(-time.Hour)
	}
	timestamp.Signed.Version = version
	timestamp.Signed.Meta["snapshot.json"] = metadata.MetaFile(version)
	writeServerFile(t, filepath.Join(directory, "timestamp.json"), signed(t, timestamp, r.keys["timestamp"]...))
}

func writeServerFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
}

func serverCacheState(t *testing.T, directory string) map[string]string {
	t.Helper()
	root, err := os.OpenRoot(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	state := make(map[string]string)
	err = filepath.WalkDir(directory, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name, err := filepath.Rel(directory, path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		state[name] = info.Mode().String()
		if !entry.IsDir() {
			data, err := root.ReadFile(name)
			if err != nil {
				return err
			}
			state[name] += string(data)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return state
}

func TestVerifyServerBundle(t *testing.T) {
	for _, name := range []string{"good", "byte-flipped", "truncated", "wrong-root", "expired-timestamp", "metadata-rollback", "missing-custom-release", "lower-release", "equal-release", "higher-release", "invalid-custom-release"} {
		t.Run(name, func(t *testing.T) {
			r := newRepository(t)
			delivery, cache := t.TempDir(), privateServerCache(t)
			root := filepath.Join(t.TempDir(), "root.json")
			writeServerFile(t, root, r.files["/metadata/1.root.json"])
			bundle := filepath.Join(delivery, "server.tar.gz")
			original := []byte("untrusted executable bytes; never execute")
			writeServerFile(t, bundle, original)
			publishServer(t, r, delivery, 2, `{"twr_server_release":5}`, false)
			if _, _, err := VerifyServerBundle(root, cache, delivery, bundle, 5); err != nil {
				t.Fatal(err)
			}
			switch name {
			case "byte-flipped":
				changed := bytes.Clone(original)
				changed[0] ^= 1
				writeServerFile(t, bundle, changed)
			case "truncated":
				writeServerFile(t, bundle, original[:3])
			case "wrong-root":
				other := newRepository(t)
				writeServerFile(t, root, other.files["/metadata/1.root.json"])
				// Wrong trust is meaningful at bootstrap; an established cache owns its saved root.
				cache = privateServerCache(t)
			case "expired-timestamp":
				publishServer(t, r, delivery, 3, `{"twr_server_release":5}`, true)
			case "metadata-rollback":
				publishServer(t, r, delivery, 1, `{"twr_server_release":5}`, false)
			case "missing-custom-release":
				publishServer(t, r, delivery, 3, "", false)
			case "lower-release":
				publishServer(t, r, delivery, 3, `{"twr_server_release":4}`, false)
			case "higher-release":
				publishServer(t, r, delivery, 3, `{"twr_server_release":6}`, false)
			case "invalid-custom-release":
				publishServer(t, r, delivery, 3, `{"twr_server_release":"five"}`, false)
			}
			before := serverCacheState(t, cache)
			bundleBefore, err := os.ReadFile(bundle)
			if err != nil {
				t.Fatal(err)
			}
			release, digest, err := VerifyServerBundle(root, cache, delivery, bundle, 5)
			success := name == "good" || name == "equal-release" || name == "higher-release"
			if success {
				want := 5
				if name == "higher-release" {
					want = 6
				}
				sum := sha256.Sum256(original)
				if err != nil || release != want || digest != hex.EncodeToString(sum[:]) {
					t.Fatalf("result %d %s %v", release, digest, err)
				}
			} else {
				if err == nil || release != 0 || digest != "" {
					t.Fatalf("accepted refusal: %d %s %v", release, digest, err)
				}
				// New authenticated metadata must survive later application-policy refusal.
				switch {
				case strings.Contains(name, "custom-release") || name == "lower-release":
					assertServerTargetsVersion(t, cache, 3)
				case name == "wrong-root":
					for _, role := range []string{"timestamp", "snapshot", "targets"} {
						if _, err := os.Stat(filepath.Join(cache, "metadata", role+".json")); !os.IsNotExist(err) {
							t.Fatalf("wrong root saved untrusted %s: %v", role, err)
						}
					}
				case !reflect.DeepEqual(before, serverCacheState(t, cache)):
					t.Fatal("refusal changed established cache")
				}
			}
			after, err := os.ReadFile(bundle)
			if err != nil || !bytes.Equal(bundleBefore, after) {
				t.Fatal("bundle changed")
			}
		})
	}
}

func privateServerCache(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "cache")
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	return path
}

func assertServerTargetsVersion(t *testing.T, cache string, version int64) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(cache, "metadata", "targets.json"))
	if err != nil {
		t.Fatal(err)
	}
	targets, err := metadata.Targets().FromBytes(data)
	if err != nil || targets.Signed.Version != version {
		t.Fatalf("remembered targets: %v %v", targets, err)
	}
}

func TestVerifyServerBundleRootRotation(t *testing.T) {
	r, next := newRepository(t), newRepository(t)
	directory, cache := t.TempDir(), privateServerCache(t)
	root, bundle := filepath.Join(directory, "root.json"), filepath.Join(directory, "server.tar.gz")
	writeServerFile(t, root, r.files["/metadata/1.root.json"])
	writeServerFile(t, bundle, []byte("untrusted executable bytes; never execute"))
	publishServer(t, r, directory, 1, `{"twr_server_release":5}`, false)
	if _, _, err := VerifyServerBundle(root, cache, directory, bundle, 5); err != nil {
		t.Fatal(err)
	}
	next.root.Signed.Version = 2
	for _, role := range []string{"targets", "snapshot", "timestamp"} {
		next.root.Signed.Roles[role] = r.root.Signed.Roles[role]
		for _, id := range r.root.Signed.Roles[role].KeyIDs {
			next.root.Signed.Keys[id] = r.root.Signed.Keys[id]
		}
	}
	signers := append([]signature.Signer{}, r.keys["root"][:2]...)
	signers = append(signers, next.keys["root"][:2]...)
	rotated := signed(t, next.root, signers...)
	rotationPath := filepath.Join(directory, "2.root.json")
	writeServerFile(t, rotationPath, rotated)
	if _, _, err := VerifyServerBundle(root, cache, directory, bundle, 5); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(rotationPath); err != nil {
		t.Fatal(err)
	}
	if _, _, err := VerifyServerBundle(root, cache, directory, bundle, 5); err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(filepath.Join(cache, "metadata", "root.json"))
	if err != nil || !bytes.Equal(saved, rotated) {
		t.Fatal("rotated trust did not survive restart")
	}
}
