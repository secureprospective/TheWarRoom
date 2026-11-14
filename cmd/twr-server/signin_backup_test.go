package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

func TestConversionBackupKilledRetry(t *testing.T) {
	for _, tc := range []struct {
		step       string
		kills      int
		incomplete bool
	}{
		{"backup", 12, false},
		{"synced", 3, true},
	} {
		t.Run(tc.step, func(t *testing.T) {
			s := newOneFileServer(t)
			var leftover string
			if tc.incomplete {
				leftover = fmt.Sprintf("%s.pre-v%d-12345.bak", s.path, history.SchemaVersion())
				for _, name := range []string{leftover, leftover + "-journal"} {
					if err := os.WriteFile(name, []byte("interrupted"), 0600); err != nil {
						t.Fatal(err)
					}
				}
			}
			executable, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			for range tc.kills {
				child := exec.CommandContext(t.Context(), executable, "-test.run=^TestConversionBackupKillChild$")
				child.Dir = filepath.Dir(s.path)
				child.Env = append(os.Environ(), "TWR_BACKUP_KILL_STEP="+tc.step)
				output, err := child.CombinedOutput()
				var exited *exec.ExitError
				if !errors.As(err, &exited) || exited.ExitCode() != -1 {
					t.Fatalf("child not killed: %v %s", err, output)
				}
				assertConversionBackup(t, s.path)
			}
			if leftover != "" {
				for _, name := range []string{leftover, leftover + "-journal"} {
					if _, err := os.Lstat(name); !errors.Is(err, os.ErrNotExist) {
						t.Fatalf("incomplete leftover retained: %s %v", name, err)
					}
				}
			}
			s.assertConverted(t)
			assertConversionBackup(t, s.path)
		})
	}
}

func TestConversionBackupKillChild(t *testing.T) {
	if os.Getenv("TWR_BACKUP_KILL_STEP") == "" {
		t.Skip("subprocess helper")
	}
	conversionStep = func(step, _ string) {
		if step != os.Getenv("TWR_BACKUP_KILL_STEP") {
			return
		}
		process, err := os.FindProcess(os.Getpid())
		if err != nil {
			t.Fatal(err)
		}
		if err := process.Kill(); err != nil {
			t.Fatal(err)
		}
		select {} // Kill is asynchronous on some platforms; never continue the conversion.
	}
	files, err := openSignInFiles(t.Context(), "server.db")
	if err != nil {
		t.Fatal(err)
	}
	_ = files.Close()
	t.Fatal("conversion did not reach the kill point")
}

func assertConversionBackup(t *testing.T, path string) {
	t.Helper()
	matches, err := filepath.Glob(fmt.Sprintf("%s.pre-v%d-*.bak", path, history.SchemaVersion()))
	if err != nil || len(matches) != 1 {
		t.Fatalf("pre-change backups=%v err=%v; want one", matches, err)
	}
	p, err := db.OpenReadOnly(t.Context(), matches[0], history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = p.Close() }()
	findings, err := db.Check(t.Context(), p.Read())
	if err != nil || len(findings.Integrity) != 1 || findings.Integrity[0] != "ok" || len(findings.ForeignKeys) != 0 {
		t.Fatalf("backup check: %+v %v", findings, err)
	}
}
