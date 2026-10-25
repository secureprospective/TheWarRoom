package releasetrust

import (
	"bytes"
	"crypto"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/theupdateframework/go-tuf/v2/metadata"
)

type repository struct {
	root     *metadata.Metadata[metadata.RootType]
	keys     map[string][]signature.Signer
	files    map[string][]byte
	payloads map[string][]byte
}

func newRepository(t *testing.T) *repository {
	t.Helper()
	r := &repository{root: metadata.Root(time.Now().Add(24 * time.Hour)), keys: make(map[string][]signature.Signer), files: make(map[string][]byte)}
	for _, role := range []string{"root", "targets", "snapshot", "timestamp"} {
		count := 1
		if role == "root" {
			count = 3
		}
		for range count {
			_, private, err := ed25519.GenerateKey(nil)
			if err != nil {
				t.Fatal(err)
			}
			signer, err := signature.LoadSigner(private, crypto.Hash(0))
			if err != nil {
				t.Fatal(err)
			}
			key, err := metadata.KeyFromPublicKey(private.Public())
			if err != nil {
				t.Fatal(err)
			}
			if err := r.root.Signed.AddKey(key, role); err != nil {
				t.Fatal(err)
			}
			r.keys[role] = append(r.keys[role], signer)
		}
	}
	r.root.Signed.Roles["root"].Threshold = 2
	r.files["/metadata/1.root.json"] = signed(t, r.root, r.keys["root"][:2]...)
	r.payloads = map[string][]byte{"app.tar.gz": []byte("reviewed application"), "client-config.json": configBytes(t)}
	r.publish(t, 1, "")
	return r
}

func signed[T metadata.Roles](t *testing.T, m *metadata.Metadata[T], signers ...signature.Signer) []byte {
	t.Helper()
	m.Signatures = nil
	for _, signer := range signers {
		if _, err := m.Sign(signer); err != nil {
			t.Fatal(err)
		}
	}
	data, err := m.ToBytes(false)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func (r *repository) publish(t *testing.T, version int64, expired string) {
	t.Helper()
	expiry := func(role string) time.Time {
		if role == expired {
			return time.Now().Add(-time.Hour)
		}
		return time.Now().Add(time.Hour)
	}
	targets := metadata.Targets(expiry("targets"))
	targets.Signed.Version = version
	for name, payload := range r.payloads {
		info, err := metadata.TargetFile().FromBytes(name, payload, "sha256")
		if err != nil {
			t.Fatal(err)
		}
		targets.Signed.Targets[name] = info
		r.files["/targets/"+hex.EncodeToString(info.Hashes["sha256"])+"."+name] = payload
	}
	data := signed(t, targets, r.keys["targets"]...)
	r.files[fmt.Sprintf("/metadata/%d.targets.json", version)] = data
	snapshot := metadata.Snapshot(expiry("snapshot"))
	snapshot.Signed.Version = version
	snapshot.Signed.Meta["targets.json"] = metadata.MetaFile(version)
	data = signed(t, snapshot, r.keys["snapshot"]...)
	r.files[fmt.Sprintf("/metadata/%d.snapshot.json", version)] = data
	timestamp := metadata.Timestamp(expiry("timestamp"))
	timestamp.Signed.Version = version
	timestamp.Signed.Meta["snapshot.json"] = metadata.MetaFile(version)
	r.files["/metadata/timestamp.json"] = signed(t, timestamp, r.keys["timestamp"]...)
}

func configBytes(t *testing.T) []byte {
	t.Helper()
	data, err := json.Marshal(RawConfig{ServerURL: "https://league.example.org", MetadataURL: "https://updates.example.org/metadata/", TargetsURL: "https://updates.example.org/targets/"})
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func (r *repository) DownloadFile(address string, maxLength int64, _ time.Duration) ([]byte, error) {
	u, err := url.Parse(address)
	if err != nil {
		return nil, err
	}
	data, ok := r.files[u.Path]
	if !ok {
		return nil, &metadata.ErrDownloadHTTP{StatusCode: 404, URL: address}
	}
	if int64(len(data)) > maxLength {
		return nil, fmt.Errorf("oversized test response")
	}
	return data, nil
}

func (r *repository) client(t *testing.T, directory string) *Client {
	t.Helper()
	cfg, err := DecodeConfig(configBytes(t))
	if err != nil {
		t.Fatal(err)
	}
	c, err := New(Bootstrap{Config: cfg, Root: r.files["/metadata/1.root.json"]}, directory)
	if err != nil {
		t.Fatal(err)
	}
	c.fetcher = r
	return c
}

func TestSignedArtifactsAndPublicConfig(t *testing.T) {
	r := newRepository(t)
	c := r.client(t, t.TempDir())
	for _, name := range []string{"app.tar.gz", "client-config.json"} {
		path, err := c.Download(name)
		if err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if name == "client-config.json" {
			if _, err := DecodeConfig(data); err != nil {
				t.Fatal(err)
			}
		} else if string(data) != "reviewed application" {
			t.Fatal("wrong artifact")
		}
	}
}

func TestTamperRejected(t *testing.T) {
	for _, path := range []string{"/metadata/timestamp.json", "/metadata/1.snapshot.json", "/metadata/1.targets.json", "artifact", "config"} {
		t.Run(path, func(t *testing.T) {
			r := newRepository(t)
			name := "app.tar.gz"
			if path == "artifact" || path == "config" {
				if path == "config" {
					name = "client-config.json"
				}
				for key := range r.files {
					if bytes.HasSuffix([]byte(key), []byte("."+name)) {
						r.files[key] = []byte("corrupt")
					}
				}
			} else {
				r.files[path] = bytes.ReplaceAll(r.files[path], []byte(`"version":1`), []byte(`"version":9`))
			}
			c := r.client(t, t.TempDir())
			if file, err := c.Download(name); err == nil || file != "" {
				t.Fatal("tamper accepted")
			}
			entries, err := os.ReadDir(filepath.Join(c.directory, "targets"))
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 0 {
				t.Fatal("unverified artifact written")
			}
		})
	}
}

func TestExpiredMetadataRejected(t *testing.T) {
	for _, role := range []string{"root", "timestamp", "snapshot", "targets"} {
		t.Run(role, func(t *testing.T) {
			r := newRepository(t)
			if role == "root" {
				r.root.Signed.Expires = time.Now().Add(-time.Hour)
				r.files["/metadata/1.root.json"] = signed(t, r.root, r.keys["root"][:2]...)
			} else {
				r.publish(t, 1, role)
			}
			if path, err := r.client(t, t.TempDir()).Download("app.tar.gz"); err == nil || path != "" {
				t.Fatal("expired metadata accepted")
			}
		})
	}
}

func TestRollbackRejectedAfterRestart(t *testing.T) {
	for _, role := range []string{"timestamp", "snapshot", "targets"} {
		t.Run(role, func(t *testing.T) {
			r := newRepository(t)
			oldTimestamp := append([]byte(nil), r.files["/metadata/timestamp.json"]...)
			r.publish(t, 2, "")
			directory := t.TempDir()
			if _, err := r.client(t, directory).Download("app.tar.gz"); err != nil {
				t.Fatal(err)
			}
			saved, err := os.ReadFile(filepath.Join(directory, "metadata", "targets.json"))
			if err != nil {
				t.Fatal(err)
			}
			switch role {
			case "timestamp":
				r.files["/metadata/timestamp.json"] = oldTimestamp
			case "snapshot":
				stamp := metadata.Timestamp(time.Now().Add(time.Hour))
				stamp.Signed.Version = 3
				stamp.Signed.Meta["snapshot.json"] = metadata.MetaFile(1)
				r.files["/metadata/timestamp.json"] = signed(t, stamp, r.keys["timestamp"]...)
			case "targets":
				snap := metadata.Snapshot(time.Now().Add(time.Hour))
				snap.Signed.Version = 3
				snap.Signed.Meta["targets.json"] = metadata.MetaFile(1)
				r.files["/metadata/3.snapshot.json"] = signed(t, snap, r.keys["snapshot"]...)
				stamp := metadata.Timestamp(time.Now().Add(time.Hour))
				stamp.Signed.Version = 3
				stamp.Signed.Meta["snapshot.json"] = metadata.MetaFile(3)
				r.files["/metadata/timestamp.json"] = signed(t, stamp, r.keys["timestamp"]...)
			}
			if path, err := r.client(t, directory).Download("app.tar.gz"); err == nil || path != "" {
				t.Fatal("rollback accepted")
			}
			after, err := os.ReadFile(filepath.Join(directory, "metadata", "targets.json"))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(saved, after) {
				t.Fatal("trusted targets replaced on rejection")
			}
		})
	}
}

func TestThresholdRootRotation(t *testing.T) {
	for _, counts := range [][2]int{{1, 2}, {2, 1}, {2, 2}} {
		t.Run(fmt.Sprint(counts), func(t *testing.T) {
			r, next := newRepository(t), newRepository(t)
			// Replace root keys, retain release-role keys.
			next.root.Signed.Version = 2
			for _, role := range []string{"targets", "snapshot", "timestamp"} {
				next.root.Signed.Roles[role] = r.root.Signed.Roles[role]
				for _, id := range r.root.Signed.Roles[role].KeyIDs {
					next.root.Signed.Keys[id] = r.root.Signed.Keys[id]
				}
			}
			signers := append([]signature.Signer{}, r.keys["root"][:counts[0]]...)
			signers = append(signers, next.keys["root"][:counts[1]]...)
			r.files["/metadata/2.root.json"] = signed(t, next.root, signers...)
			directory := t.TempDir()
			_, err := r.client(t, directory).Download("app.tar.gz")
			if counts == [2]int{2, 2} {
				if err != nil {
					t.Fatal(err)
				}
				// Restart with the old installer root and no rotation on the server.
				delete(r.files, "/metadata/2.root.json")
				if _, err := r.client(t, directory).Download("app.tar.gz"); err != nil {
					t.Fatal(err)
				}
				data, err := os.ReadFile(filepath.Join(directory, "metadata", "root.json"))
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(data, signed(t, next.root, signers...)) {
					t.Fatal("saved root regressed")
				}
			} else if err == nil {
				t.Fatal("insufficient root threshold accepted")
			}
		})
	}
}

func TestBootstrapRejectsSecretsAndPins(t *testing.T) {
	for _, extra := range []string{`"tunnel_token":"secret"`, `"leaf_pin":"sha256:pin"`, `"password":"secret"`} {
		data := append([]byte(nil), configBytes(t)...)
		data = append(data[:len(data)-1], []byte(","+extra+"}")...)
		if _, err := DecodeConfig(data); err == nil {
			t.Fatal("extra config field accepted")
		}
	}
	for _, address := range []string{"http://league.example.org", "https://user:pass@league.example.org", "https://192.168.1.1", "https://host.lan", "https://league.example.org/?token=secret"} {
		raw := RawConfig{ServerURL: address, MetadataURL: address, TargetsURL: address}
		if raw.Validate() == nil {
			t.Fatal("non-public address accepted")
		}
	}
	if err := publicRedirect(&http.Request{URL: &url.URL{Scheme: "http", Host: "updates.example.org"}}, nil); err == nil {
		t.Fatal("insecure redirect accepted")
	}
}

func TestDamagedCacheDoesNotBootstrapAgain(t *testing.T) {
	r := newRepository(t)
	directory := t.TempDir()
	if err := os.Mkdir(filepath.Join(directory, "metadata"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := r.client(t, directory).Download("app.tar.gz"); err == nil {
		t.Fatal("damaged cache silently reset")
	}
}

func TestBootstrapRejectsInvalidRoot(t *testing.T) {
	r := newRepository(t)
	cfg, err := DecodeConfig(configBytes(t))
	if err != nil {
		t.Fatal(err)
	}
	root := append([]byte(nil), r.files["/metadata/1.root.json"]...)
	for _, data := range [][]byte{nil, []byte("not metadata"), bytes.ReplaceAll(root, []byte(`"version":1`), []byte(`"version":9`))} {
		if _, err := New(Bootstrap{Config: cfg, Root: data}, t.TempDir()); err == nil {
			t.Fatal("invalid root accepted")
		}
	}
	for _, name := range []string{"../escape", "app/escape", "..", "https://host/file", ""} {
		c := r.client(t, t.TempDir())
		if _, err := c.Download(name); err == nil {
			t.Fatal("unsafe target path accepted")
		}
		if _, err := os.Stat(filepath.Join(c.directory, "metadata")); !os.IsNotExist(err) {
			t.Fatal("invalid target wrote cache")
		}
	}
}
