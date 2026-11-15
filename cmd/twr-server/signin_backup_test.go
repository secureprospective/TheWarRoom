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
		reference  bool
	}{
		{"backup", 12, false, false},
		{"synced", 3, true, false},
		{"backup", 12, false, true},
		{"synced", 3, true, true},
	} {
		t.Run(fmt.Sprintf("%s/reference=%v", tc.step, tc.reference), func(t *testing.T) {
			s := newOneFileServer(t)
			move, want := signInMove, 1
			if tc.reference {
				settleBackupMove(t, s.path, signInMove)
				move, want = referenceMove, 2
			}
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
				child.Env = append(os.Environ(), "TWR_BACKUP_KILL_STEP="+tc.step, "TWR_BACKUP_KILL_FILE="+move.file)
				output, err := child.CombinedOutput()
				var exited *exec.ExitError
				if !errors.As(err, &exited) || exited.ExitCode() != -1 {
					t.Fatalf("child not killed: %v %s", err, output)
				}
				assertConversionBackup(t, s.path, want)
			}
			if leftover != "" {
				for _, name := range []string{leftover, leftover + "-journal"} {
					if _, err := os.Lstat(name); !errors.Is(err, os.ErrNotExist) {
						t.Fatalf("incomplete leftover retained: %s %v", name, err)
					}
				}
			}
			settleBackupMove(t, s.path, move)
			assertConversionBackup(t, s.path, want)
			s.assertConverted(t)
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
	move := signInMove
	if os.Getenv("TWR_BACKUP_KILL_FILE") == referenceFileName {
		move = referenceMove
	}
	settleBackupMove(t, "server.db", move)
	t.Fatal("conversion did not reach the kill point")
}

func settleBackupMove(t *testing.T, path string, move fileMove) {
	t.Helper()
	pools, err := db.Open(t.Context(), path, history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = pools.Close() }()
	if err := move.settle(t.Context(), pools, path); err != nil {
		t.Fatal(err)
	}
}

func assertConversionBackup(t *testing.T, path string, want int) {
	t.Helper()
	matches, err := filepath.Glob(fmt.Sprintf("%s.pre-v%d-*.bak", path, history.SchemaVersion()))
	if err != nil || len(matches) != want {
		t.Fatalf("pre-change backups=%v err=%v; want %d", matches, err, want)
	}
	for _, match := range matches {
		p, err := db.OpenReadOnly(t.Context(), match, history.SchemaVersion())
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = p.Close() }()
		findings, err := db.Check(t.Context(), p.Read())
		if err != nil || len(findings.Integrity) != 1 || findings.Integrity[0] != "ok" || len(findings.ForeignKeys) != 0 {
			t.Fatalf("backup check: %+v %v", findings, err)
		}
	}
}
