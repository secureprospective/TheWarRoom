package db

import (
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
)

// BackupFile specifies one fixed database name and the newest readable schema.
type BackupFile struct {
	Name       string
	MaxVersion int
}

type RawBackupEntry struct {
	Name        string `json:"name"`
	Size        int64  `json:"size"`
	SHA256      string `json:"sha256"`
	UserVersion int    `json:"user_version"`
}

type RawBackupManifest struct {
	ManifestVersion int              `json:"manifest_version"`
	Files           []RawBackupEntry `json:"files"`
	AppBuild        string           `json:"app_build"`
	CapturedAt      time.Time        `json:"captured_at"`
	Kind            string           `json:"kind"`
}

func (m RawBackupManifest) Validate(files [3]BackupFile) error {
	if m.ManifestVersion != 1 || len(m.Files) != 3 || m.AppBuild == "" || m.CapturedAt.IsZero() || m.CapturedAt.Location() != time.UTC {
		return fmt.Errorf("db: invalid backup manifest header")
	}
	if m.Kind != "daily" && m.Kind != "pre-migration" && m.Kind != "pre-restore" {
		return fmt.Errorf("db: invalid backup kind %q", m.Kind)
	}
	seen := make(map[string]bool)
	for _, entry := range m.Files {
		if err := entry.Validate(files); err != nil {
			return err
		}
		if seen[entry.Name] {
			return fmt.Errorf("db: duplicate backup entry %q", entry.Name)
		}
		seen[entry.Name] = true
	}
	return nil
}

func (entry RawBackupEntry) Validate(files [3]BackupFile) error {
	limit, ok := backupLimit(files, entry.Name)
	hash, err := hex.DecodeString(entry.SHA256)
	if !ok || entry.Size <= 0 || err != nil || len(hash) != sha256.Size || entry.UserVersion < 0 || entry.UserVersion > limit {
		return fmt.Errorf("db: invalid backup entry %q", entry.Name)
	}
	return nil
}

func backupLimit(files [3]BackupFile, name string) (int, bool) {
	if strings.ContainsAny(name, `/\`) || strings.Contains(name, "..") {
		return 0, false
	}
	for _, file := range files {
		if file.Name == name {
			return file.MaxVersion, true
		}
	}
	return 0, false
}

func readJSON[T any](path string, out *T) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("db: open metadata: %w", err)
	}
	defer func() { _ = f.Close() }()
	decoder := json.NewDecoder(io.LimitReader(f, 64*1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return fmt.Errorf("db: decode metadata: %w", err)
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		return fmt.Errorf("db: trailing metadata: %v", err)
	}
	return nil
}

func backupEntry(ctx context.Context, path, name string) (RawBackupEntry, error) {
	entry := RawBackupEntry{Name: name}
	info, err := os.Lstat(path)
	if err != nil {
		return entry, fmt.Errorf("db: stat backup: %w", err)
	}
	if !info.Mode().IsRegular() {
		return entry, fmt.Errorf("db: backup is not a regular file: %s", name)
	}
	f, err := os.Open(path)
	if err != nil {
		return entry, fmt.Errorf("db: read backup: %w", err)
	}
	defer func() { _ = f.Close() }()
	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return entry, fmt.Errorf("db: hash backup: %w", err)
	}
	entry.Size, entry.SHA256 = info.Size(), hex.EncodeToString(hash.Sum(nil))
	entry.UserVersion, err = probeVersion(ctx, path, "?mode=ro&immutable=1")
	return entry, err
}

func (b *Backups) verify(ctx context.Context, dir string) (RawBackupManifest, error) {
	var m RawBackupManifest
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() {
		return m, fmt.Errorf("db: backup set is not a directory: %s", dir)
	}
	if err := readJSON(filepath.Join(dir, "manifest.json"), &m); err != nil {
		return m, err
	}
	if err := m.Validate(b.files); err != nil {
		return m, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return m, fmt.Errorf("db: list backup: %w", err)
	}
	for _, e := range entries {
		if e.Name() == "manifest.json" {
			continue
		}
		if _, ok := backupLimit(b.files, e.Name()); !ok {
			return m, fmt.Errorf("db: extra backup file %q", e.Name())
		}
	}
	for _, want := range m.Files {
		got, err := backupEntry(ctx, filepath.Join(dir, want.Name), want.Name)
		if err != nil {
			return m, err
		}
		if got != want {
			return m, fmt.Errorf("db: backup size, sha256 or user_version mismatch: %s", want.Name)
		}
	}
	return m, nil
}

func syncDir(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("db: open directory: %w", err)
	}
	defer func() { _ = f.Close() }()
	if err := f.Sync(); err != nil {
		return fmt.Errorf("db: sync directory: %w", err)
	}
	return nil
}

func durableJSON[T any](path string, value T) error {
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("db: encode metadata: %w", err)
	}
	if err := os.WriteFile(path+".tmp", body, 0600); err != nil {
		return fmt.Errorf("db: write metadata: %w", err)
	}
	if err := syncFile(path + ".tmp"); err != nil {
		return err
	}
	if err := os.Rename(path+".tmp", path); err != nil {
		return fmt.Errorf("db: publish metadata: %w", err)
	}
	return syncDir(filepath.Dir(path))
}

func syncFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("db: open for sync: %w", err)
	}
	defer func() { _ = f.Close() }()
	if err := f.Sync(); err != nil {
		return fmt.Errorf("db: sync file: %w", err)
	}
	return nil
}
