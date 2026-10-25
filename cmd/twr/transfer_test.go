package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

func exportCLIFixture(t *testing.T) (string, string, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "source.db")
	pools, err := db.Open(t.Context(), path, history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	registry, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	store := history.New(pools, registry)
	if err := store.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := accountstore.New(pools).Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	raw := commissioner.RawCommand{League: "native.cli", Command: commissioner.Create, Args: commissioner.RawArgs{Name: "CLI move", Preset: commissioner.PresetEmpty, Season: 2026, FranchiseCount: 2}, Reason: "Create the test league", Request: "create", Effective: "now", Visibility: "league"}
	if _, err := store.ExecuteCommissioner(t.Context(), raw, "creator", nil); err != nil {
		t.Fatal(err)
	}
	evidence := history.RawFootballEvidence{Version: 1, Season: 2026, Source: "commissioner reference import", AsOf: time.Now().UTC(), Players: map[string]history.RawFootballPlayer{"0099": {Identity: "reference.99", Facts: nativecheck.PlayerFacts{Name: "QB", Position: leaguerules.QB, Team: "DAL"}}}, Weeks: []leagueweek.Week{{Number: 1, Games: []leagueweek.Game{{Teams: [2]string{"DAL", "NYG"}, Kickoff: time.Now().UTC()}}}}}
	body, err := json.Marshal(evidence)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(body)
	digest := hex.EncodeToString(hash[:])
	if err := store.ImportFootball(t.Context(), body, digest); err != nil {
		t.Fatal(err)
	}
	if err := pools.Close(); err != nil {
		t.Fatal(err)
	}
	key := filepath.Join(dir, "key")
	if err := os.WriteFile(key, []byte(strings.Repeat("ab", 32)+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	return path, digest, key
}

func TestLeagueExportCLISaveCheckAndRefuseOverwrite(t *testing.T) {
	path, digest, key := exportCLIFixture(t)
	output := filepath.Join(filepath.Dir(path), "league.twr")
	args := []string{"league-export", "save", "--db", path, "--league", "native.cli", "--source-generation", "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee", "--football", digest, "--key-file", key, "--output", output}
	var messages bytes.Buffer
	e := env{out: &messages}
	root := e.root()
	root.SetArgs(args)
	if err := root.ExecuteContext(t.Context()); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(output)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("export not private", err)
	}
	before, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	root = e.root()
	root.SetArgs(args)
	if err := root.ExecuteContext(t.Context()); err == nil {
		t.Fatal("overwrote an existing export")
	}
	after, err := os.ReadFile(output)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("refused overwrite changed file", err)
	}
	root = e.root()
	root.SetArgs([]string{"league-export", "check", "--file", output, "--key-file", key})
	if err := root.ExecuteContext(t.Context()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(messages.String(), "Nothing was imported or activated.") {
		t.Fatal("missing authority warning")
	}
}

func TestLeagueExportCLIRejectsMissingDatabaseAndPublicKeyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing.db")
	key := filepath.Join(dir, "key")
	if err := os.WriteFile(key, []byte(strings.Repeat("ab", 32)), 0644); err != nil { //nolint:gosec // Deliberately public key file proves the command refuses unsafe permissions.
		t.Fatal(err)
	}
	if _, err := readExportKey(key); err == nil {
		t.Fatal("public key file accepted")
	}
	if err := os.Chmod(key, 0600); err != nil {
		t.Fatal(err)
	}
	root := env{out: &bytes.Buffer{}}.root()
	root.SetArgs([]string{"league-export", "save", "--db", path, "--key-file", key})
	if err := root.ExecuteContext(t.Context()); err == nil {
		t.Fatal("missing database accepted")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("missing database was created")
	}
}
