package releasetrust

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/theupdateframework/go-tuf/v2/metadata"
)

// RawServerRelease is the signed server bundle's monotonic release number.
type RawServerRelease struct {
	Release int `json:"twr_server_release"`
}

func (r RawServerRelease) Validate() error {
	if r.Release < 1 {
		return fmt.Errorf("the signed server release number is missing or invalid")
	}
	return nil
}

// VerifyServerBundle authenticates local delivery files without installing or running them.
// The caller exclusively owns cache for the duration. Authenticated TUF metadata is
// retained even if a later bundle check fails: clearing it would permit rollback.
func VerifyServerBundle(rootPath, cache, metadataDir, bundle string, installed int) (int, string, error) {
	if rootPath == "" || cache == "" || metadataDir == "" || bundle == "" || installed < 1 || !safeTarget(filepath.Base(bundle)) {
		return 0, "", fmt.Errorf("choose the trusted root, private cache, delivery folder, bundle and installed release")
	}
	root, err := readLimited(rootPath, 512000)
	if err != nil {
		return 0, "", fmt.Errorf("read installed release trust: %w", err)
	}
	if err := validateRoot(root); err != nil {
		return 0, "", err
	}
	if err := protectServerCache(cache); err != nil {
		return 0, "", err
	}
	c := &Client{bootstrap: Bootstrap{Root: root, Config: RawConfig{MetadataURL: "/"}}, directory: cache, fetcher: localMetadata{directory: metadataDir}}
	up, err := c.updater()
	if err != nil {
		return 0, "", err
	}
	if err := up.Refresh(); err != nil {
		return 0, "", fmt.Errorf("verify delivered release metadata: %w", err)
	}
	info, err := up.GetTargetInfo(filepath.Base(bundle))
	if err != nil {
		return 0, "", fmt.Errorf("find signed server bundle: %w", err)
	}
	release, err := serverRelease(info.Custom, installed)
	if err != nil {
		return 0, "", err
	}
	digest, err := verifyBundleBytes(info, bundle)
	if err != nil {
		return 0, "", err
	}
	return release, digest, nil
}

func serverRelease(custom *json.RawMessage, installed int) (int, error) {
	if custom == nil {
		return 0, fmt.Errorf("the signed server release number is missing")
	}
	var raw RawServerRelease
	if err := json.Unmarshal(*custom, &raw); err != nil {
		return 0, fmt.Errorf("read signed server release: %w", err)
	}
	if err := raw.Validate(); err != nil {
		return 0, err
	}
	if raw.Release < installed {
		return 0, fmt.Errorf("the bundle is older than the installed server")
	}
	return raw.Release, nil
}

func verifyBundleBytes(info *metadata.TargetFiles, path string) (string, error) {
	if info.Length < 1 || info.Length > 512*1024*1024 || len(info.Hashes["sha256"]) != sha256.Size {
		return "", fmt.Errorf("the signed bundle size or fingerprint is invalid")
	}
	data, err := readLimited(path, info.Length)
	if err != nil {
		return "", err
	}
	if err := info.VerifyLengthHashes(data); err != nil {
		return "", fmt.Errorf("verify server bundle bytes: %w", err)
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:]), nil
}

type localMetadata struct{ directory string }

func (f localMetadata) DownloadFile(address string, maxLength int64, _ time.Duration) ([]byte, error) {
	name := strings.TrimPrefix(address, "/")
	if !safeTarget(name) {
		return nil, fmt.Errorf("invalid delivery metadata name")
	}
	data, err := readLimited(filepath.Join(f.directory, name), maxLength)
	if errors.Is(err, os.ErrNotExist) {
		return nil, &metadata.ErrDownloadHTTP{StatusCode: 404, URL: address}
	}
	return data, err
}

func readLimited(path string, maxLength int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open release file: %w", err)
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("check release file: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() > maxLength {
		return nil, fmt.Errorf("release file is not a regular file within the size limit")
	}
	data, err := io.ReadAll(io.LimitReader(f, maxLength+1))
	if err != nil {
		return nil, fmt.Errorf("read release file: %w", err)
	}
	if int64(len(data)) > maxLength {
		return nil, fmt.Errorf("release file exceeds the size limit")
	}
	return data, nil
}

func protectServerCache(directory string) error {
	if err := os.Mkdir(directory, 0700); err != nil && !errors.Is(err, os.ErrExist) {
		return fmt.Errorf("create private release cache: %w", err)
	}
	for _, path := range []string{directory, filepath.Join(directory, "metadata"), filepath.Join(directory, "targets")} {
		info, err := os.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("check release cache: %w", err)
		}
		if !info.IsDir() || info.Mode().Perm() != 0700 {
			return fmt.Errorf("release cache folders must be private real directories")
		}
	}
	return nil
}
