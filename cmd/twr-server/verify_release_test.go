package main

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/theupdateframework/go-tuf/v2/metadata"
)

func TestVerifyReleaseCommand(t *testing.T) {
	for _, tc := range []struct {
		name    string
		args    []string
		want    string
		refused bool
	}{
		{"help", []string{"-help"}, "-installed-release", false},
		{"missing-options", nil, "provide -metadata", true},
		{"bad-option", []string{"-unknown"}, "correct the options", true},
		{"missing-delivery", []string{"-root", "missing-root", "-cache", t.TempDir(), "-metadata", t.TempDir(), "-bundle", "missing.tar.gz", "-installed-release", "5"}, "keep the installed server", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			err := run(context.Background(), append([]string{"verify-release"}, tc.args...), &out)
			if tc.refused {
				if err == nil || !strings.Contains(err.Error(), tc.want) || strings.Contains(err.Error(), "\n") || out.Len() != 0 {
					t.Fatalf("refusal: %q %v", out.String(), err)
				}
			} else if err != nil || !strings.Contains(out.String(), tc.want) {
				t.Fatalf("help: %q %v", out.String(), err)
			}
		})
	}
}

func TestVerifyReleaseCommandSignedBundle(t *testing.T) {
	directory := t.TempDir()
	cache := filepath.Join(directory, "cache")
	payload := []byte("candidate bundle must never be executed")
	root := metadata.Root(time.Now().Add(time.Hour))
	var signers []signature.Signer
	for range 3 {
		_, key, err := ed25519.GenerateKey(nil)
		if err != nil {
			t.Fatal(err)
		}
		signer, err := signature.LoadSigner(key, crypto.Hash(0))
		if err != nil {
			t.Fatal(err)
		}
		public, err := metadata.KeyFromPublicKey(key.Public())
		if err != nil {
			t.Fatal(err)
		}
		if err := root.Signed.AddKey(public, "root"); err != nil {
			t.Fatal(err)
		}
		if len(signers) == 0 {
			for _, role := range []string{"targets", "snapshot", "timestamp"} {
				if err := root.Signed.AddKey(public, role); err != nil {
					t.Fatal(err)
				}
			}
		}
		signers = append(signers, signer)
	}
	root.Signed.Roles["root"].Threshold = 2
	writeVerificationMetadata(t, directory, "root.json", root, signers[:2]...)
	targets := metadata.Targets(time.Now().Add(time.Hour))
	info, err := metadata.TargetFile().FromBytes("server.tar.gz", payload, "sha256")
	if err != nil {
		t.Fatal(err)
	}
	custom := json.RawMessage(`{"twr_server_release":5}`)
	info.Custom = &custom
	targets.Signed.Targets["server.tar.gz"] = info
	writeVerificationMetadata(t, directory, "1.targets.json", targets, signers[0])
	snapshot := metadata.Snapshot(time.Now().Add(time.Hour))
	snapshot.Signed.Meta["targets.json"] = metadata.MetaFile(1)
	writeVerificationMetadata(t, directory, "1.snapshot.json", snapshot, signers[0])
	timestamp := metadata.Timestamp(time.Now().Add(time.Hour))
	timestamp.Signed.Meta["snapshot.json"] = metadata.MetaFile(1)
	writeVerificationMetadata(t, directory, "timestamp.json", timestamp, signers[0])
	bundle := filepath.Join(directory, "server.tar.gz")
	if err := os.WriteFile(bundle, payload, 0600); err != nil {
		t.Fatal(err)
	}
	args := []string{"verify-release", "-root", filepath.Join(directory, "root.json"), "-cache", cache, "-metadata", directory, "-bundle", bundle, "-installed-release", "5"}
	var out bytes.Buffer
	if err := run(context.Background(), args, &out); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(payload)
	if want := fmt.Sprintf("verified release 5 sha256 %x\n", sum); out.String() != want {
		t.Fatalf("output %q, want %q", out.String(), want)
	}
	if err := os.WriteFile(bundle, []byte("tampered"), 0600); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := run(context.Background(), args, &out); err == nil || out.Len() != 0 {
		t.Fatalf("tampered release: %q %v", out.String(), err)
	}
}

func writeVerificationMetadata[T metadata.Roles](t *testing.T, directory, name string, value *metadata.Metadata[T], signers ...signature.Signer) {
	t.Helper()
	for _, signer := range signers {
		if _, err := value.Sign(signer); err != nil {
			t.Fatal(err)
		}
	}
	data, err := value.ToBytes(false)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, name), data, 0600); err != nil {
		t.Fatal(err)
	}
}
