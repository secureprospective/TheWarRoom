package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/serverversion"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

func oldUpgradeFile(t *testing.T, keepRelease bool) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "server.db")
	if err := migrateServer(t.Context(), path); err != nil {
		t.Fatal(err)
	}
	pools, err := db.Open(t.Context(), path, history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	registry, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	records, err := leagueservice.FixtureRecords()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := history.New(pools, registry).ExecuteCommissioner(t.Context(), records[0].Command, "fixture.creator", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := accounts.New(accountstore.New(pools)).LocalBootstrap(t.Context(), accounts.RawCredentials{Username: "commissioner", Password: "upgrade-test-password"}); err != nil {
		t.Fatal(err)
	}
	if _, err := accountstore.New(pools).ResetSignIns(t.Context(), "upgrade drill"); err != nil {
		t.Fatal(err)
	}
	if !keepRelease {
		if _, err := pools.Write().ExecContext(t.Context(), "DROP TABLE server_release"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pools.Write().ExecContext(t.Context(), "DROP TABLE league_roster_restore; DROP TABLE league_recovery_log; PRAGMA user_version=7"); err != nil {
		t.Fatal(err)
	}
	if err := pools.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestUpgradeBacksUpMigratesAndVerifiesMatchedCheckpoint(t *testing.T) {
	path := oldUpgradeFile(t, false)
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := runUpgrade(t.Context(), []string{"-db", path, "-check"}, io.Discard); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(original, after) {
		t.Fatal("check changed older file")
	}
	if err := servePreflight(t.Context(), path); !errors.Is(err, serverversion.ErrOlderFile) {
		t.Fatalf("serve silently migrated older file: %v", err)
	}
	save := filepath.Join(t.TempDir(), "before")
	var output bytes.Buffer
	if err := runUpgrade(t.Context(), []string{"-db", path, "-save", save}, &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "Upgraded") {
		t.Fatal(output.String())
	}
	if err := upgradeReady(t.Context(), path); err != nil {
		t.Fatal(err)
	}
	plan, err := serverversion.Preflight(t.Context(), filepath.Join(save, "server.db"))
	if err != nil || plan.State != serverversion.Older {
		t.Fatalf("saved original = %+v %v", plan, err)
	}
	body, err := os.ReadFile(filepath.Join(save, "checkpoint.json"))
	if err != nil {
		t.Fatal(err)
	}
	var checkpoint rawCheckpoint
	if err := json.Unmarshal(body, &checkpoint); err != nil || checkpoint.Ledger.Size == 0 || checkpoint.Security.Size == 0 {
		t.Fatalf("empty upgrade evidence: %+v %v", checkpoint, err)
	}
	for _, file := range []string{path, filepath.Join(save, "server.db")} {
		if err := runCheckpoint(t.Context(), []string{"-db", file, "-check", filepath.Join(save, "checkpoint.json")}, io.Discard); err != nil {
			t.Fatal(err)
		}
	}
}

func TestUpgradeFailureRestoresSavedLeague(t *testing.T) {
	// The old file intentionally has a conflicting future table: migration must fail after snapshot.
	path := oldUpgradeFile(t, true)
	save := filepath.Join(t.TempDir(), "before")
	err := runUpgrade(t.Context(), []string{"-db", path, "-save", save}, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "put back") {
		t.Fatalf("failed upgrade not restored: %v", err)
	}
	plan, err := serverversion.Preflight(t.Context(), path)
	if err != nil || plan.File.Schema != 7 {
		t.Fatalf("failed migration changed schema: %+v %v", plan, err)
	}
	if err := runCheckpoint(t.Context(), []string{"-db", path, "-check", filepath.Join(save, "checkpoint.json")}, io.Discard); err != nil {
		t.Fatal(err)
	}
}

func TestUpgradeRefusesNewerSchemaAndBinaryOnlyDowngradeUntouched(t *testing.T) {
	for _, schema := range []bool{false, true} {
		path := filepath.Join(t.TempDir(), "server.db")
		if err := migrateServer(t.Context(), path); err != nil {
			t.Fatal(err)
		}
		pools, err := db.Open(t.Context(), path, history.SchemaVersion())
		if err != nil {
			t.Fatal(err)
		}
		if schema {
			_, err = pools.Write().ExecContext(t.Context(), "PRAGMA user_version=999")
		} else {
			err = pools.StampServerRelease(t.Context(), serverversion.Current().Release+1, time.Now())
		}
		if err != nil {
			t.Fatal(err)
		}
		if err := pools.Close(); err != nil {
			t.Fatal(err)
		}
		original, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		save := filepath.Join(t.TempDir(), "must-not-exist")
		for _, check := range []bool{false, true} {
			args := []string{"-db", path, "-save", save}
			if check {
				args = append(args, "-check")
			}
			if err := runUpgrade(t.Context(), args, io.Discard); !errors.Is(err, serverversion.ErrNewerFile) {
				t.Fatalf("newer file upgrade = %v", err)
			}
		}
		if _, err := openHostedFile(t.Context(), path); !errors.Is(err, serverversion.ErrNewerFile) {
			t.Fatalf("newer file serve = %v", err)
		}
		after, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(original, after) {
			t.Fatal("newer file changed")
		}
		afterInfo, err := os.Stat(path)
		if err != nil || !info.ModTime().Equal(afterInfo.ModTime()) {
			t.Fatal("newer file touched")
		}
		if _, err := os.Stat(save); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("refusal created backup folder")
		}
		for _, side := range []string{"-wal", "-shm", "-journal"} {
			if _, err := os.Lstat(path + side); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("refusal created %s", side)
			}
		}
	}
}

func TestUpgradeBackupFailureDoesNotMigrate(t *testing.T) {
	path := oldUpgradeFile(t, false)
	err := runUpgrade(t.Context(), []string{"-db", path, "-save", filepath.Join(t.TempDir(), "before"), "-backup-config", filepath.Join(t.TempDir(), "backup.json")}, io.Discard)
	if err == nil {
		t.Fatal("upgrade proceeded without verified configured backup")
	}
	plan, err := serverversion.Preflight(t.Context(), path)
	if err != nil || plan.File.Schema != 7 || plan.State != serverversion.Older {
		t.Fatalf("backup failure migrated original: %+v %v", plan, err)
	}
}
