package composition

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	commissionerapp "github.com/secureprospective/TheWarRoom/internal/app/commissioner"
)

func TestPersistedIdentity(t *testing.T) {
	dir := t.TempDir()
	want, err := commissionerapp.LocalPrincipal(dir, os.Getuid())
	if err != nil {
		t.Fatal(err)
	}
	for _, step := range []string{"fresh", "second", "moved"} {
		t.Run(step, func(t *testing.T) {
			if step == "moved" {
				moved := filepath.Join(t.TempDir(), "profile")
				if err := os.Rename(dir, moved); err != nil {
					t.Fatal(err)
				}
				dir = moved
			}
			got, err := persistedIdentity(dir)
			if err != nil || got != want {
				t.Fatalf("identity = %q, %v; want %q", got, err, want)
			}
			info, err := os.Stat(identityFile(dir))
			if err != nil || info.Mode().Perm() != 0o600 {
				t.Fatalf("identity permissions: %v %v", info, err)
			}
		})
	}
}

func TestRelinkIdentityFlag(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir, err := configDir()
	if err != nil {
		t.Fatal(err)
	}
	app := restoreDrillApp(t, dir)
	original := filepath.Join(t.TempDir(), "original")
	principal, err := commissionerapp.LocalPrincipal(original, os.Getuid())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.core.History.ExecuteCommissioner(t.Context(), restoreCLILeague("native.relink"), principal, nil); err != nil {
		t.Fatal(err)
	}
	app.shutdown(t.Context())
	for _, tc := range []struct {
		name, path string
		code       int
	}{{"wrong", original + "wrong", 1}, {"matching", original, 0}} {
		t.Run(tc.name, func(t *testing.T) {
			var out, errOut bytes.Buffer
			code := runFlag([]string{"--relink-identity-from", tc.path}, &out, &errOut)
			if code != tc.code {
				t.Fatalf("code %d: %s %s", code, out.String(), errOut.String())
			}
			body, err := os.ReadFile(identityFile(dir))
			if tc.code != 0 {
				if !os.IsNotExist(err) || !strings.Contains(errOut.String(), "does not match") {
					t.Fatalf("wrong path wrote identity or no message: %q %v %s", body, err, errOut.String())
				}
				return
			}
			if err != nil || string(body) != principal {
				t.Fatalf("identity %q %v", body, err)
			}
		})
	}
}
