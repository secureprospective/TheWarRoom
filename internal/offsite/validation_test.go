package offsite

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"filippo.io/age"
)

func TestAuthenticatedArchiveRejectsMissingMemberAndWrongDigest(t *testing.T) {
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	stage := t.TempDir()
	if err := os.Mkdir(filepath.Join(stage, "config"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"server.db", "checkpoint.json", "config/authenticator.key", "config/origin.key", "config/origin.pem", "config/endpoint-registry.csv", "config/backup.json"} {
		if err := os.WriteFile(filepath.Join(stage, name), []byte("original"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	m, err := manifest(stage, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"missing", "digest"} {
		t.Run(mode, func(t *testing.T) {
			var body bytes.Buffer
			sealed, err := age.Encrypt(&body, identity.Recipient())
			if err != nil {
				t.Fatal(err)
			}
			zipped := gzip.NewWriter(sealed)
			archive := tar.NewWriter(zipped)
			if mode == "digest" {
				if err := os.WriteFile(filepath.Join(stage, "server.db"), []byte("tampered"), 0600); err != nil {
					t.Fatal(err)
				}
				if err := writeArchive(archive, stage, m); err != nil {
					t.Fatal(err)
				}
			} else {
				encoded, err := json.Marshal(m)
				if err != nil {
					t.Fatal(err)
				}
				if err := archive.WriteHeader(&tar.Header{Name: "manifest.json", Mode: 0600, Size: int64(len(encoded))}); err != nil {
					t.Fatal(err)
				}
				if _, err := archive.Write(encoded); err != nil {
					t.Fatal(err)
				}
			}
			if err := archive.Close(); err != nil {
				t.Fatal(err)
			}
			if err := zipped.Close(); err != nil {
				t.Fatal(err)
			}
			if err := sealed.Close(); err != nil {
				t.Fatal(err)
			}
			dest := filepath.Join(t.TempDir(), "rejected")
			if _, err := Restore(bytes.NewReader(body.Bytes()), identity, dest); err == nil {
				t.Fatal("invalid authenticated set accepted")
			}
			if _, err := os.Stat(dest); !os.IsNotExist(err) {
				t.Fatal("partial restored state survived rejection")
			}
		})
	}
}

func TestManifestRejectsTraversalDuplicatesAndBudget(t *testing.T) {
	good := RawManifest{Version: 1, Captured: time.Now().UTC(), Files: []RawEntry{{"server.db", 1, strings.Repeat("a", 64)}, {"checkpoint.json", 1, strings.Repeat("b", 64)}, {"config/authenticator.key", 1, strings.Repeat("c", 64)}}}
	for _, name := range []string{"config/origin.key", "config/origin.pem", "config/endpoint-registry.csv", "config/backup.json"} {
		good.Files = append(good.Files, RawEntry{name, 1, strings.Repeat("d", 64)})
	}
	for _, name := range []string{"../outside", "config/../outside", "config/a/b", "config/..", "config/."} {
		m := good
		m.Files = append([]RawEntry(nil), good.Files...)
		m.Files[2].Name = name
		if m.Validate() == nil {
			t.Fatalf("unsafe member accepted: %s", name)
		}
	}
	duplicate := good
	duplicate.Files = append(append([]RawEntry(nil), good.Files...), good.Files[0])
	if duplicate.Validate() == nil {
		t.Fatal("duplicate accepted")
	}
	large := good
	large.Files = append([]RawEntry(nil), good.Files...)
	large.Files[0].Size = MaxPlain
	if large.Validate() == nil {
		t.Fatal("oversized total accepted")
	}
}

func TestPrivateConfigurationAcceptsPublicRecipientOnly(t *testing.T) {
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	good := RawConfig{strings.Repeat("a", 32) + ".r2.cloudflarestorage.com", "league-backup", strings.Repeat("b", 32), strings.Repeat("c", 64), identity.Recipient().String()}
	if err := good.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"private key", "host trick", "http endpoint", "empty credentials"} {
		t.Run(mode, func(t *testing.T) {
			config := good
			switch mode {
			case "private key":
				config.Recipient = identity.String()
			case "host trick":
				config.Endpoint += ".attacker.example"
			case "http endpoint":
				config.Endpoint = "http://" + config.Endpoint
			case "empty credentials":
				config.SecretKey = ""
			}
			if config.Validate() == nil {
				t.Fatal("invalid settings accepted")
			}
		})
	}
	//nolint:gosec // Synthetic repeated-letter credentials exercise the private settings boundary.
	body, err := json.Marshal(good)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "backup.json")
	if err := os.WriteFile(path, body, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadConfig(path); err != nil {
		t.Fatal(err)
	}
	//nolint:gosec // Deliberately insecure fixture must be refused by ReadConfig.
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadConfig(path); err == nil {
		t.Fatal("public credentials accepted")
	}
}
