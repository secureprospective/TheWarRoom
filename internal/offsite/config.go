// Package offsite captures encrypted hosted recovery sets and verifies off-host coverage.
package offsite

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"

	"filippo.io/age"
)

const MaxObject = 128 << 20
const MaxPlain = 512 << 20

// RawConfig contains only the public age recipient, never a private restore identity.
type RawConfig struct {
	Endpoint  string `json:"endpoint"`
	Bucket    string `json:"bucket"`
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
	Recipient string `json:"recipient"`
}

func (r RawConfig) String() string   { return "[backup configuration redacted]" }
func (r RawConfig) GoString() string { return r.String() }
func (r RawConfig) Validate() error {
	host := regexp.MustCompile(`^[a-f0-9]{32}(\.(eu|us|fedramp))?\.r2\.cloudflarestorage\.com$`)
	bucket := regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,61}[a-z0-9]$`)
	if !host.MatchString(r.Endpoint) || !bucket.MatchString(r.Bucket) || len(r.AccessKey) != 32 || len(r.SecretKey) != 64 {
		return fmt.Errorf("backup settings need the R2 endpoint, bucket and S3 credentials")
	}
	if _, err := age.ParseX25519Recipient(r.Recipient); err != nil {
		return fmt.Errorf("backup needs an age public recipient, not the private restore key")
	}
	return nil
}

func ReadConfig(path string) (RawConfig, error) {
	var r RawConfig
	f, err := os.Open(path)
	if err != nil {
		return r, fmt.Errorf("open private backup settings: %w", err)
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 || info.Size() > 8192 {
		return r, fmt.Errorf("backup settings must be a small private file (0600)")
	}
	decoder := json.NewDecoder(io.LimitReader(f, 8193))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&r); err != nil {
		return RawConfig{}, fmt.Errorf("backup settings are not valid JSON")
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		return RawConfig{}, fmt.Errorf("backup settings must contain one JSON object")
	}
	if err := r.Validate(); err != nil {
		return RawConfig{}, err
	}
	return r, nil
}
