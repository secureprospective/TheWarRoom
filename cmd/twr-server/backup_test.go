package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"filippo.io/age"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/offsite"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
)

// openTestServer opens a fresh two-file hosted server in dir, as a first start does.
func openTestServer(t *testing.T, dir string) stateFiles {
	t.Helper()
	files, err := openSignInFiles(t.Context(), filepath.Join(dir, "server.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = files.Close() })
	return files
}

// Each kind of set carries a checkpoint its own restore accepts.
func TestCapturedHostedSetCarriesCompatibleCheckpoint(t *testing.T) {
	files := openTestServer(t, t.TempDir())
	if _, err := hostedServices(t.Context(), files, make([]byte, 32), ""); err != nil {
		t.Fatal(err)
	}
	config := t.TempDir()
	for _, name := range []string{"authenticator.key", "origin.pem", "origin.key", "endpoint-registry.csv", "backup.json"} {
		if err := os.WriteFile(filepath.Join(config, name), []byte("fixture"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	// The restore-side key is never passed to the hosted capture API.
	offline, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		kind       offsite.SetKind
		pools      *db.Pools
		checkpoint func(context.Context, string) ([]byte, error)
		verify     func(dir string) error
	}{
		{offsite.LeagueSet, files.league, leagueSetCheckpoint, func(dir string) error {
			return checkCheckpoint(t.Context(), filepath.Join(dir, "checkpoint.json"), filepath.Join(dir, "server.db"), &discard{})
		}},
		{offsite.SignInSet, files.signIns, signInSetCheckpoint, func(dir string) error {
			saved, err := readCheckpoint(filepath.Join(dir, "checkpoint.json"))
			if err != nil {
				return err
			}
			return checkAnchors(t.Context(), saved, "", signInSource{path: filepath.Join(dir, signInFileName), version: accountstore.ServerVersion()})
		}},
	} {
		t.Run(tc.kind.File, func(t *testing.T) {
			sealed, _, err := offsite.CaptureSet(t.Context(), tc.kind, tc.pools, config, t.TempDir(), offline.Recipient().String(), tc.checkpoint)
			if err != nil {
				t.Fatal(err)
			}
			f, err := os.Open(sealed)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = f.Close() }()
			dest := filepath.Join(t.TempDir(), "restored")
			manifest, err := offsite.Restore(f, offline, dest)
			if err != nil {
				t.Fatal(err)
			}
			if !manifest.Holds(tc.kind.File) {
				t.Fatalf("set does not hold %s", tc.kind.File)
			}
			if err := tc.verify(dest); err != nil {
				t.Fatalf("saved snapshot/checkpoint not matched: %v", err)
			}
		})
	}
}

type discard struct{}

func (*discard) Write(p []byte) (int, error) { return len(p), nil }
