package offsite

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"filippo.io/age"
	"github.com/secureprospective/TheWarRoom/internal/db"
)

type RawEntry struct {
	Name   string `json:"name"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}
type RawManifest struct {
	Version  int        `json:"version"`
	Captured time.Time  `json:"captured"`
	Files    []RawEntry `json:"files"`
}

func (m RawManifest) Validate() error {
	if m.Version != 1 || m.Captured.IsZero() || len(m.Files) < 3 || len(m.Files) > 32 {
		return fmt.Errorf("invalid recovery manifest")
	}
	seen := map[string]bool{}
	var total int64
	for _, e := range m.Files {
		if err := e.Validate(); err != nil {
			return err
		}
		if seen[e.Name] {
			return fmt.Errorf("duplicate recovery member")
		}

		total += e.Size
		seen[e.Name] = true
	}
	for _, name := range []string{"server.db", "checkpoint.json", "config/authenticator.key", "config/origin.key", "config/origin.pem", "config/endpoint-registry.csv", "config/backup.json"} {
		if !seen[name] {
			return fmt.Errorf("recovery set is missing required server settings")
		}
	}
	if total > MaxPlain {
		return fmt.Errorf("incomplete or oversized recovery set")
	}
	return nil
}

func (e RawEntry) Validate() error {
	valid := e.Name == "server.db" || e.Name == "checkpoint.json" || (strings.HasPrefix(e.Name, "config/") && filepath.Base(e.Name) == strings.TrimPrefix(e.Name, "config/") && filepath.Base(e.Name) != "." && filepath.Base(e.Name) != "..")
	digest, err := hex.DecodeString(e.SHA256)
	if !valid || e.Size < 0 || e.Size > MaxPlain || err != nil || len(digest) != sha256.Size {
		return fmt.Errorf("invalid recovery member")
	}
	return nil
}

// Capture encrypts a matched set to a public recipient. stage must be private and disposable.
// Configuration must be immutable for the running server (the installed sandbox binds it read-only).
func Capture(ctx context.Context, pools *db.Pools, configDir, stage, recipient string, checkpoint func(context.Context, string) ([]byte, error)) (string, time.Time, error) {
	at := time.Now().UTC()
	err := pools.CaptureHosted(ctx, filepath.Join(stage, "server.db"), func() error {
		return copyConfig(configDir, filepath.Join(stage, "config"))
	})
	if err != nil {
		return "", time.Time{}, fmt.Errorf("capture recovery set: %w", err)
	}
	body, err := checkpoint(ctx, filepath.Join(stage, "server.db"))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("verify captured history: %w", err)
	}
	if err := os.WriteFile(filepath.Join(stage, "checkpoint.json"), body, 0600); err != nil {
		return "", time.Time{}, fmt.Errorf("save recovery checkpoint: %w", err)
	}
	path, err := encryptArchive(stage, recipient, at)
	return path, at, err
}

func copyConfig(source, dest string) error {
	if err := os.Mkdir(dest, 0700); err != nil {
		return fmt.Errorf("stage recovery settings: %w", err)
	}
	entries, err := os.ReadDir(source)
	if err != nil {
		return fmt.Errorf("read recovery settings: %w", err)
	}
	if len(entries) > 28 {
		return fmt.Errorf("too many server configuration files")
	}
	for _, e := range entries {
		if !e.Type().IsRegular() {
			return fmt.Errorf("server configuration must contain regular files only")
		}
		f, err := os.Open(filepath.Join(source, e.Name()))
		if err != nil {
			return fmt.Errorf("read recovery setting: %w", err)
		}
		body, err := io.ReadAll(io.LimitReader(f, 1<<20+1))
		_ = f.Close()
		if err != nil || len(body) > 1<<20 {
			return fmt.Errorf("server configuration file cannot be captured")
		}
		if err := os.WriteFile(filepath.Join(dest, e.Name()), body, 0600); err != nil {
			return fmt.Errorf("stage recovery setting: %w", err)
		}
	}
	return nil
}

func manifest(stage string, at time.Time) (RawManifest, error) {
	m := RawManifest{Version: 1, Captured: at}
	root, err := os.OpenRoot(stage)
	if err != nil {
		return m, fmt.Errorf("open recovery staging root: %w", err)
	}
	defer func() { _ = root.Close() }()
	err = filepath.WalkDir(stage, func(path string, e os.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("inspect recovery members: %w", err)
		}
		if e.IsDir() {
			return nil
		}
		name, err := filepath.Rel(stage, path)
		if err != nil {
			return fmt.Errorf("name recovery member: %w", err)
		}
		// SQLite sidecars from checkpoint verification are not authoritative snapshot members.
		if name != "server.db" && name != "checkpoint.json" && !strings.HasPrefix(name, "config"+string(filepath.Separator)) {
			return nil
		}
		f, err := root.Open(name)
		if err != nil {
			return fmt.Errorf("read recovery member: %w", err)
		}
		defer func() { _ = f.Close() }()
		h := sha256.New()
		n, err := io.Copy(h, io.LimitReader(f, MaxPlain+1))
		if err != nil {
			return fmt.Errorf("digest recovery member: %w", err)
		}
		m.Files = append(m.Files, RawEntry{filepath.ToSlash(name), n, hex.EncodeToString(h.Sum(nil))})
		return nil
	})
	if err != nil {
		return m, fmt.Errorf("inspect recovery set: %w", err)
	}
	return m, m.Validate()
}

func encryptArchive(stage, recipient string, at time.Time) (string, error) {
	m, err := manifest(stage, at)
	if err != nil {
		return "", err
	}
	key, err := age.ParseX25519Recipient(recipient)
	if err != nil {
		return "", fmt.Errorf("read backup public recipient: %w", err)
	}
	path := filepath.Join(stage, "set.age")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return "", fmt.Errorf("create encrypted backup: %w", err)
	}
	defer func() { _ = f.Close() }()
	sealed, err := age.Encrypt(f, key)
	if err != nil {
		return "", fmt.Errorf("encrypt backup: %w", err)
	}
	zipped := gzip.NewWriter(sealed)
	archive := tar.NewWriter(zipped)
	if err := writeArchive(archive, stage, m); err != nil {
		return "", err
	}
	for _, closer := range []io.Closer{archive, zipped, sealed} {
		if err := closer.Close(); err != nil {
			return "", fmt.Errorf("finish encrypted backup: %w", err)
		}
	}
	if err := f.Sync(); err != nil {
		return "", fmt.Errorf("sync encrypted backup: %w", err)
	}
	return path, nil
}

func writeArchive(archive *tar.Writer, stage string, m RawManifest) error {
	body, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("encode recovery manifest: %w", err)
	}
	if err := archive.WriteHeader(&tar.Header{Name: "manifest.json", Mode: 0600, Size: int64(len(body))}); err != nil {
		return fmt.Errorf("archive recovery manifest: %w", err)
	}
	if _, err := archive.Write(body); err != nil {
		return fmt.Errorf("archive recovery manifest: %w", err)
	}
	for _, e := range m.Files {
		if err := archive.WriteHeader(&tar.Header{Name: e.Name, Mode: 0600, Size: e.Size}); err != nil {
			return fmt.Errorf("archive recovery member: %w", err)
		}
		if err := archiveFile(archive, filepath.Join(stage, filepath.FromSlash(e.Name))); err != nil {
			return err
		}
	}
	return nil
}
func archiveFile(dest io.Writer, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open recovery member: %w", err)
	}
	defer func() { _ = f.Close() }()
	if _, err := io.Copy(dest, f); err != nil {
		return fmt.Errorf("archive recovery member: %w", err)
	}
	return nil
}
