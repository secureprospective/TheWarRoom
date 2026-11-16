package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

func TestStressCheck(t *testing.T) {
	t.Parallel()
	for _, split := range []bool{false, true} {
		name := "single"
		if split {
			name = "split"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			path := filepath.Join(root, "server.db")
			ledger := path
			if split {
				emptyDatabase(t, path)
				ledger = filepath.Join(root, "leagues", "legacy", "league.db")
				if err := os.MkdirAll(filepath.Dir(ledger), 0700); err != nil {
					t.Fatal(err)
				}
			}
			seedLedger(t, ledger)
			assertStressReport(t, path, ledger)
		})
	}
}

func emptyDatabase(t *testing.T, path string) {
	t.Helper()
	pool, err := db.Open(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	if err := pool.Close(); err != nil {
		t.Fatal(err)
	}
}

func seedLedger(t *testing.T, path string) {
	t.Helper()
	pool, err := db.Open(t.Context(), path, history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := pool.Close(); err != nil {
			t.Error(err)
		}
	}()
	if err := history.MigrateLeague(t.Context(), pool); err != nil {
		t.Fatal(err)
	}
	event := history.LeagueEvent{ID: "known", League: "legacy", Stream: commissioner.Stream,
		EffectiveAt: time.Now(), Actor: "owner", Scope: "league", Kind: "test",
		Payload: history.EventPayload{Version: 1}, InputHash: strings.Repeat("a", 64), CorrelationID: "recipient.known"}
	if _, err := history.New(pool, nil).AppendEvent(t.Context(), event, 0); err != nil {
		t.Fatal(err)
	}
}

func assertStressReport(t *testing.T, path, ledger string) {
	t.Helper()
	before, err := os.ReadFile(ledger)
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := run(t.Context(), path, &output); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"integrity ok\n", "req \"owner\" \"recipient.known\" 1\n", "head legacy commissioner 1\n"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("missing %q in %q", want, output.String())
		}
	}
	after, err := os.ReadFile(ledger)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("database changed: %v", err)
	}
	for _, suffix := range []string{"-wal", "-shm"} {
		if _, err := os.Stat(ledger + suffix); !os.IsNotExist(err) {
			t.Errorf("sidecar %s created: %v", suffix, err)
		}
	}
}
