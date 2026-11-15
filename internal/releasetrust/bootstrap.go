// Package releasetrust verifies public update configuration and signed release files.
package releasetrust

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"

	"github.com/theupdateframework/go-tuf/v2/metadata"
)

// RawConfig is the complete public routing configuration, never server configuration.
type RawConfig struct {
	ServerURL   string `json:"server_url"`
	MetadataURL string `json:"metadata_url"`
	TargetsURL  string `json:"targets_url"`
}

// Validate accepts public HTTPS names without credentials, pins or private routing.
func (r RawConfig) Validate() error {
	for _, value := range []string{r.ServerURL, r.MetadataURL, r.TargetsURL} {
		u, err := url.Parse(value)
		if err != nil {
			return fmt.Errorf("read public update address: %w", err)
		}
		host := u.Hostname()
		if u.Scheme != "https" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" ||
			(u.Port() != "" && u.Port() != "443") || net.ParseIP(host) != nil || !publicName(host) ||
			!repositoryPath(u.EscapedPath()) {
			return fmt.Errorf("update addresses must contain only public HTTPS names and repository paths")
		}
	}
	return nil
}

func publicName(host string) bool {
	if !strings.Contains(host, ".") {
		return false
	}
	for _, suffix := range []string{".local", ".localhost", ".internal", ".lan"} {
		if strings.HasSuffix(host, suffix) {
			return false
		}
	}
	for _, label := range strings.Split(host, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, c := range label {
			if !publicNameCharacter(c) {
				return false
			}
		}
	}
	return len(host) <= 253
}

// DecodeConfig rejects extra fields so deployment secrets cannot ride in client configuration.
func DecodeConfig(data []byte) (RawConfig, error) {
	var raw RawConfig
	if len(data) > 4096 {
		return raw, fmt.Errorf("public update configuration is too large")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&raw); err != nil {
		return raw, fmt.Errorf("read public update configuration: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return RawConfig{}, fmt.Errorf("public update configuration must contain one document")
	}
	if err := raw.Validate(); err != nil {
		return RawConfig{}, err
	}
	return raw, nil
}

// Bootstrap carries only public routing and an independently authenticated public root.
// Obtain it from the reviewed installer, never by downloading trust from the update host.
type Bootstrap struct {
	Config RawConfig
	Root   []byte
}

// Validate checks the public root's shape and threshold with the maintained TUF library.
func (b Bootstrap) Validate() error {
	if err := b.Config.Validate(); err != nil {
		return err
	}
	return validateRoot(b.Root)
}

func validateRoot(data []byte) error {
	if len(data) == 0 || len(data) > 512000 {
		return fmt.Errorf("update trust root is missing or too large")
	}
	root, err := metadata.Root().FromBytes(data)
	if err != nil {
		return fmt.Errorf("read update trust root: %w", err)
	}
	for _, key := range root.Signed.Keys {
		if key == nil || key.Type != metadata.KeyTypeEd25519 || key.Scheme != metadata.KeySchemeEd25519 || len(key.Value.UnrecognizedFields) != 0 {
			return fmt.Errorf("update trust root must contain only public Ed25519 keys")
		}
	}
	role := root.Signed.Roles[metadata.ROOT]
	if role == nil || role.Threshold != 2 || len(role.KeyIDs) != 3 || !root.Signed.ConsistentSnapshot {
		return fmt.Errorf("update trust requires two of three root signatures and consistent snapshots")
	}
	if err := root.VerifyDelegate(metadata.ROOT, root); err != nil {
		return fmt.Errorf("verify update trust root: %w", err)
	}
	return nil
}

func repositoryPath(path string) bool {
	return path == "" || path == "/" || path == "/metadata/" || path == "/targets/"
}

func publicNameCharacter(c rune) bool {
	return c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-'
}
