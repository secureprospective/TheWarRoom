package offsite

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"sync"

	"filippo.io/age"
	"github.com/minio/minio-go/v7"
)

// Raw captures (the raw archive's fetched bodies and the blobs a transfer brought) never ride in a
// reference set: with them the set passed the 128 MiB object cap. Each body is uploaded once, encrypted
// to the backup recipient, as its own object named by the recipient and the body's sha256; a body is
// immutable, so an object once read back never needs uploading again. The recipient is in the name so a
// bucket that already holds a body sealed to another key (an earlier restore key, or another server's)
// never stands in for this server's own copy, which this server's restore key could not open.
// Retention never prunes them: a set names its bodies.

// EvidencePrefix is where raw capture objects live in the bucket, apart from every kind of set.
const EvidencePrefix = "evidence/"

// EvidenceName is the object name of one raw capture sealed to recipient.
func EvidenceName(recipient, sha string) string {
	return EvidencePrefix + recipientTag(recipient) + "/" + sha + ".age"
}

// LegacyEvidenceName is where releases before the recipient was in the name put a raw capture;
// restore still reads it when the recipient's own copy is not there.
func LegacyEvidenceName(sha string) string { return EvidencePrefix + sha + ".age" }

// recipientTag names a backup recipient without spelling it out: 16 hex digits of its sha256.
func recipientTag(recipient string) string {
	sum := sha256.Sum256([]byte(recipient))
	return hex.EncodeToString(sum[:8])
}

// Evidence uploads raw captures once each. It remembers what this process has read back.
type Evidence struct {
	r2        *R2
	recipient string
	mu        sync.Mutex
	known     map[string]bool
}

// NewEvidence uploads to r2, encrypted to recipient.
func NewEvidence(r2 *R2, recipient string) *Evidence {
	return &Evidence{r2: r2, recipient: recipient, known: map[string]bool{}}
}

// Put uploads one body, as stored, under its archive key unless its object is already in the bucket;
// it then reads the object back. The restore proves the bodies against the set's checkpoint.
func (e *Evidence) Put(ctx context.Context, sha string, body []byte) error {
	e.mu.Lock()
	known := e.known[sha]
	e.mu.Unlock()
	if known {
		return nil
	}
	name := EvidenceName(e.recipient, sha)
	// Any failure to find the object uploads it again: a body is immutable, so a second copy is harmless.
	if _, err := e.r2.client.StatObject(ctx, e.r2.bucket, name, minio.StatObjectOptions{}); err == nil {
		e.remember(sha)
		return nil
	}
	sealed, err := SealEvidence(body, e.recipient)
	if err != nil {
		return err
	}
	if _, err := e.r2.client.PutObject(ctx, e.r2.bucket, name, bytes.NewReader(sealed), int64(len(sealed)),
		minio.PutObjectOptions{ContentType: "application/octet-stream", DisableMultipart: true, DisableContentSha256: true}); err != nil {
		return fmt.Errorf("R2 upload failed; check the bucket and backup token")
	}
	if err := e.readBack(ctx, name, sealed); err != nil {
		return err
	}
	e.remember(sha)
	return nil
}

// SealEvidence encrypts one raw capture to the backup recipient.
func SealEvidence(body []byte, recipient string) ([]byte, error) {
	key, err := age.ParseX25519Recipient(recipient)
	if err != nil {
		return nil, fmt.Errorf("read backup public recipient: %w", err)
	}
	var sealed bytes.Buffer
	w, err := age.Encrypt(&sealed, key)
	if err != nil {
		return nil, fmt.Errorf("encrypt raw capture: %w", err)
	}
	if _, err := w.Write(body); err != nil {
		return nil, fmt.Errorf("encrypt raw capture: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("encrypt raw capture: %w", err)
	}
	if sealed.Len() > MaxObject {
		return nil, fmt.Errorf("a raw capture is larger than 128 MiB")
	}
	return sealed.Bytes(), nil
}

func (e *Evidence) remember(sha string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.known[sha] = true
}

func (e *Evidence) readBack(ctx context.Context, name string, sealed []byte) error {
	remote, err := e.r2.client.GetObject(ctx, e.r2.bucket, name, minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("R2 verification could not read a raw capture")
	}
	defer func() { _ = remote.Close() }()
	got, err := io.ReadAll(io.LimitReader(remote, MaxObject+1))
	if err != nil || !bytes.Equal(got, sealed) {
		return fmt.Errorf("R2 verification failed: a raw capture is missing or changed")
	}
	return nil
}

// OpenEvidence streams one raw capture object for restore: the copy sealed to recipient, or else the
// copy an earlier release left under the body's sha256 alone.
func (r *R2) OpenEvidence(ctx context.Context, recipient, sha string) (io.ReadCloser, error) {
	for _, name := range []string{EvidenceName(recipient, sha), LegacyEvidenceName(sha)} {
		object, err := r.client.GetObject(ctx, r.bucket, name, minio.GetObjectOptions{})
		if err != nil {
			break
		}
		// The first read of the object's details is the download itself: a missing object fails here.
		if _, err := object.Stat(); err != nil {
			_ = object.Close()
			continue
		}
		return object, nil
	}
	return nil, fmt.Errorf("could not download raw capture %s from R2", sha)
}

// UnsealEvidence decrypts one raw capture.
func UnsealEvidence(source io.Reader, identity *age.X25519Identity) ([]byte, error) {
	plain, err := age.Decrypt(io.LimitReader(source, MaxObject+1), identity)
	if err != nil {
		return nil, ErrRestoreKey
	}
	body, err := io.ReadAll(io.LimitReader(plain, MaxObject+1))
	if err != nil {
		return nil, fmt.Errorf("read raw capture: %w", err)
	}
	return body, nil
}
