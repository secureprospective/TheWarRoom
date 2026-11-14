package offsite

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"

	"filippo.io/age"
	"github.com/minio/minio-go/v7"
)

// Raw captures (the raw archive's fetched bodies and the blobs a transfer brought) never ride in a
// reference set: with them the set passed the 128 MiB object cap. Each body is uploaded once, encrypted
// to the backup recipient, as its own object named by its sha256; a body is immutable, so an object
// once read back never needs uploading again. Retention never prunes them: a set names its bodies.

// EvidencePrefix is where raw capture objects live in the bucket, apart from every kind of set.
const EvidencePrefix = "evidence/"

// EvidenceName is the object name of one raw capture.
func EvidenceName(sha string) string { return EvidencePrefix + sha + ".age" }

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
	name := EvidenceName(sha)
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

// OpenEvidence streams one raw capture object for restore.
func (r *R2) OpenEvidence(ctx context.Context, sha string) (io.ReadCloser, error) {
	object, err := r.client.GetObject(ctx, r.bucket, EvidenceName(sha), minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("could not download raw capture %s from R2", sha)
	}
	return object, nil
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
