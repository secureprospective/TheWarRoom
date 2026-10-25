package releasetrust

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/mod/semver"
)

// RawDesktopRelease is a signed target binding a version to one native executable.
// Publish desktop-<os>-<arch>.json alongside the executable, not an archive.
type RawDesktopRelease struct {
	Version string `json:"version"`
	Target  string `json:"target"`
	SHA256  string `json:"sha256"`
}

func (r RawDesktopRelease) Validate() error {
	digest, err := hex.DecodeString(r.SHA256)
	if !semver.IsValid(r.Version) || semver.Canonical(r.Version) != r.Version ||
		!safeTarget(r.Target) || len(digest) != sha256.Size || err != nil || strings.ToLower(r.SHA256) != r.SHA256 {
		return fmt.Errorf("the signed desktop release is unreadable")
	}
	return nil
}

func decodeRelease(data []byte) (RawDesktopRelease, error) {
	var raw RawDesktopRelease
	if len(data) > 4096 {
		return raw, fmt.Errorf("the signed desktop release is too large")
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(&raw); err != nil {
		return raw, fmt.Errorf("read desktop release: %w", err)
	}
	if err := d.Decode(&struct{}{}); err != io.EOF {
		return raw, fmt.Errorf("one desktop release is required")
	}
	return raw, raw.Validate()
}

// Stage re-verifies fresh TUF metadata on every attempt, including after interruption.
// Only the private update cache is written; the running app and league files are untouched.
// The caller holds the desktop instance lock for this cache's entire lifetime.
func (c *Client) Stage(installed string) (string, error) {
	c.stageMu.Lock()
	defer c.stageMu.Unlock()
	if !semver.IsValid(installed) {
		return "", fmt.Errorf("this development build cannot install owner updates")
	}
	path, err := c.Download("desktop-" + runtime.GOOS + "-" + runtime.GOARCH + ".json")
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read verified desktop release: %w", err)
	}
	release, err := decodeRelease(data)
	if err != nil {
		return "", err
	}
	if err := c.admitRelease(installed, release.Version); err != nil {
		return "", err
	}
	path, err = c.Download(release.Target)
	if err != nil {
		return "", err
	}
	data, err = os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read verified executable: %w", err)
	}
	digest := sha256.Sum256(data)
	if hex.EncodeToString(digest[:]) != release.SHA256 {
		return "", fmt.Errorf("the executable does not match the signed desktop release")
	}
	name := "staged-" + release.SHA256
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	staged := filepath.Join(c.directory, name)
	if err := atomicUpdateFile(staged, data, 0700); err != nil {
		return "", err
	}
	// Remember accepted application versions as well as TUF metadata versions.
	if err := atomicUpdateFile(filepath.Join(c.directory, "desktop-version"), []byte(release.Version), 0600); err != nil {
		return "", err
	}
	return staged, nil
}

func (c *Client) admitRelease(installed, candidate string) error {
	if semver.Compare(candidate, installed) <= 0 {
		return fmt.Errorf("the update is not newer than this copy of The War Room")
	}
	saved, err := os.ReadFile(filepath.Join(c.directory, "desktop-version"))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read remembered desktop version: %w", err)
	}
	if !semver.IsValid(string(saved)) || semver.Compare(candidate, string(saved)) < 0 {
		return fmt.Errorf("the update is older than an already accepted update; keep this copy and try again later")
	}
	return nil
}

func atomicUpdateFile(path string, data []byte, mode os.FileMode) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".staging-*")
	if err != nil {
		return fmt.Errorf("prepare staged update: %w", err)
	}
	defer func() { _ = os.Remove(f.Name()) }()
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return fmt.Errorf("write staged update: %w", err)
	}
	if err := f.Chmod(mode); err != nil {
		_ = f.Close()
		return fmt.Errorf("protect staged update: %w", err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return fmt.Errorf("save staged update: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close staged update: %w", err)
	}
	if err := os.Rename(f.Name(), path); err != nil {
		return fmt.Errorf("finish staged update: %w", err)
	}
	return nil
}
