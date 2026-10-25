package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

func TestCheckpointRoundTripAndMismatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "server.db")
	pools, err := db.Open(t.Context(), path, history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	registry, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	if err := history.New(pools, registry).Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := accountstore.New(pools).Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := pools.Close(); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := run(t.Context(), []string{"checkpoint", "-db", path}, &out); err != nil {
		t.Fatal(err)
	}
	var saved rawCheckpoint
	if err := json.Unmarshal(out.Bytes(), &saved); err != nil || saved.Version != 1 ||
		saved.Ledger.Source != "league_event" || saved.Security.Source != "account_security_audit" {
		t.Fatalf("checkpoint %s: %v", out.String(), err)
	}
	file := filepath.Join(dir, "checkpoint.json")
	if err := os.WriteFile(file, out.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := run(t.Context(), []string{"checkpoint", "-db", path, "-check", file}, &out); err != nil ||
		!strings.Contains(out.String(), "matches") {
		t.Fatalf("matching checkpoint refused: %q %v", out.String(), err)
	}

	// A checkpoint the server's history cannot reproduce: a later copy, or history rewritten since.
	saved.Ledger.Size, saved.Ledger.Hash = saved.Ledger.Size+1, strings.Repeat("0", 64)
	body, err := json.Marshal(saved)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, body, 0o600); err != nil {
		t.Fatal(err)
	}
	err = run(t.Context(), []string{"checkpoint", "-db", path, "-check", file}, &out)
	if !errors.Is(err, errCheckpointMismatch) || !strings.Contains(err.Error(), "league history") {
		t.Fatalf("mismatch accepted: %v", err)
	}
	if err := run(t.Context(), []string{"checkpoint", "-db", filepath.Join(dir, "missing.db")}, &out); err == nil {
		t.Fatal("missing database accepted")
	}
	if _, err := os.Stat(filepath.Join(dir, "missing.db")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("checkpoint created a database")
	}
}
