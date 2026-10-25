package offsite

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"slices"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type R2 struct {
	client *minio.Client
	bucket string
}

func NewR2(config RawConfig) (*R2, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	client, err := minio.New(config.Endpoint, &minio.Options{Creds: credentials.NewStaticV4(config.AccessKey, config.SecretKey, ""), Secure: true, Region: "auto", BucketLookup: minio.BucketLookupPath})
	if err != nil {
		return nil, fmt.Errorf("initialize R2 client")
	}
	return &R2{client, config.Bucket}, nil
}

func (r *R2) upload(ctx context.Context, name, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open encrypted set: %w", err)
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("inspect encrypted set: %w", err)
	}
	if info.Size() > MaxObject {
		return fmt.Errorf("encrypted backup is larger than 128 MiB; writes stay paused until backup capacity is resolved")
	}
	_, err = r.client.PutObject(ctx, r.bucket, name, f, info.Size(), minio.PutObjectOptions{ContentType: "application/octet-stream", DisableMultipart: true, DisableContentSha256: true})
	if err != nil {
		return fmt.Errorf("R2 upload failed; check the bucket and backup token")
	}
	return nil
}

func (r *R2) verify(ctx context.Context, name, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open upload verification: %w", err)
	}
	defer func() { _ = f.Close() }()
	local := sha256.New()
	size, err := io.Copy(local, f)
	if err != nil {
		return fmt.Errorf("hash upload: %w", err)
	}
	remote, err := r.client.GetObject(ctx, r.bucket, name, minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("R2 verification could not read the backup")
	}
	defer func() { _ = remote.Close() }()
	digest := sha256.New()
	n, err := io.Copy(digest, io.LimitReader(remote, MaxObject+1))
	if err != nil || n != size || hex.EncodeToString(local.Sum(nil)) != hex.EncodeToString(digest.Sum(nil)) {
		return fmt.Errorf("R2 verification failed: backup is missing or changed")
	}
	return nil
}

// prune keeps the newest twelve sets plus one per UTC day for thirty days. A set is
// one atomic encrypted object; foreign keys and failed verification never trigger deletion.
func (r *R2) prune(ctx context.Context, now time.Time) error {
	var names []minio.ObjectInfo
	for object := range r.client.ListObjects(ctx, r.bucket, minio.ListObjectsOptions{Prefix: "sets/", Recursive: true}) {
		if object.Err != nil {
			return fmt.Errorf("R2 retention could not list backups")
		}
		if _, err := objectTime(object.Key); err == nil {
			names = append(names, object)
		}
	}
	return r.pruneObjects(ctx, names, now)
}

// Sets lists recovery set names, newest first. Foreign objects are ignored.
func (r *R2) Sets(ctx context.Context) ([]string, error) {
	var names []string
	for object := range r.client.ListObjects(ctx, r.bucket, minio.ListObjectsOptions{Prefix: "sets/", Recursive: true}) {
		if object.Err != nil {
			return nil, fmt.Errorf("could not list backups in R2; check the bucket and token")
		}
		if _, err := objectTime(object.Key); err == nil {
			names = append(names, object.Key)
		}
	}
	slices.Sort(names)
	slices.Reverse(names)
	return names, nil
}

// Open streams one recovery set for Restore, which bounds and authenticates it.
func (r *R2) Open(ctx context.Context, name string) (io.ReadCloser, error) {
	if _, err := objectTime(name); err != nil {
		return nil, err
	}
	object, err := r.client.GetObject(ctx, r.bucket, name, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("could not download backup %s from R2", name)
	}
	return object, nil
}
