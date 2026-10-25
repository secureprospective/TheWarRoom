package offsite

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"filippo.io/age"
)

// ErrRestoreKey means the identity cannot unlock the set: the wrong restore key, not damage.
var ErrRestoreKey = errors.New("the restore key does not unlock this backup")

// Restore decrypts into a new private staging directory, never into live state. The
// offline restore caller owns the identity and must verify checkpoints/reset auth before activation.
func Restore(source io.Reader, identity *age.X25519Identity, dest string) (manifest RawManifest, err error) {
	if err := os.Mkdir(dest, 0700); err != nil {
		return manifest, fmt.Errorf("create new restore folder: %w", err)
	}
	defer func() {
		if err != nil {
			_ = os.RemoveAll(dest)
		}
	}()
	plain, err := age.Decrypt(io.LimitReader(source, MaxObject+1), identity)
	var mismatch *age.NoIdentityMatchError
	if errors.As(err, &mismatch) {
		return manifest, ErrRestoreKey
	}
	if err != nil {
		return manifest, fmt.Errorf("unlock recovery set: %w", err)
	}
	zipped, err := gzip.NewReader(plain)
	if err != nil {
		return manifest, fmt.Errorf("open recovery archive: %w", err)
	}
	defer func() { _ = zipped.Close() }()
	bounded := &io.LimitedReader{R: zipped, N: MaxPlain + (1 << 20) + 1}
	archive := tar.NewReader(bounded)
	manifest, err = readManifest(archive)
	if err != nil {
		return manifest, err
	}
	for _, entry := range manifest.Files {
		if err := restoreMember(archive, entry, dest); err != nil {
			return manifest, err
		}
	}
	if _, err := archive.Next(); err != io.EOF {
		return manifest, fmt.Errorf("unexpected or damaged recovery member")
	}
	// Consume gzip trailers and the age authentication tag, even after tar's end markers.
	if _, err := io.Copy(io.Discard, bounded); err != nil {
		return manifest, fmt.Errorf("verify recovery archive: %w", err)
	}
	if bounded.N == 0 {
		return manifest, fmt.Errorf("recovery archive exceeds its size limit")
	}
	if _, err := io.Copy(io.Discard, plain); err != nil {
		return manifest, fmt.Errorf("verify encrypted recovery set: %w", err)
	}
	return manifest, nil
}
func readManifest(archive *tar.Reader) (RawManifest, error) {
	var m RawManifest
	header, err := archive.Next()
	if err != nil || header.Name != "manifest.json" || header.Typeflag != tar.TypeReg || header.Size > 16384 {
		return m, fmt.Errorf("recovery manifest is missing or damaged")
	}
	if err := json.NewDecoder(archive).Decode(&m); err != nil {
		return m, fmt.Errorf("decode recovery manifest: %w", err)
	}
	return m, m.Validate()
}
func restoreMember(archive *tar.Reader, e RawEntry, dest string) error {
	header, err := archive.Next()
	if err != nil || header.Name != e.Name || header.Typeflag != tar.TypeReg || header.Size != e.Size {
		return fmt.Errorf("recovery member is missing or changed")
	}
	path := filepath.Join(dest, filepath.FromSlash(e.Name))
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create recovery settings folder: %w", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("create recovery member: %w", err)
	}
	defer func() { _ = f.Close() }()
	digest := sha256.New()
	n, err := io.Copy(io.MultiWriter(f, digest), archive)
	if err != nil || n != e.Size || hex.EncodeToString(digest.Sum(nil)) != e.SHA256 {
		return fmt.Errorf("recovery member digest does not match")
	}
	if err := f.Sync(); err != nil {
		return fmt.Errorf("sync recovered member: %w", err)
	}
	return nil
}
