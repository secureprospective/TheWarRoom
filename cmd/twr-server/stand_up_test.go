package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	catalog "github.com/secureprospective/TheWarRoom/internal/commandcatalog"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

func TestStandUpCommandLocalDoorAndResult(t *testing.T) {
	t.Parallel()
	registry, err := catalog.NewOperatorRegistry()
	if err != nil {
		t.Fatal(err)
	}
	definition := registry.Entries()[0]
	if err := definition.Definition.ValidatePublication(); err != nil {
		t.Fatal(err)
	}
	if definition.Console[catalog.Execute] == nil || definition.Native != nil || definition.Definition.Permission != "offline-operator" {
		t.Fatal("offline recovery lost its explicit console boundary")
	}
	source := filepath.Join(t.TempDir(), "history.db")
	pools, err := db.Open(t.Context(), source, history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	ledger := history.New(pools, &measures.Registry{})
	if err := ledger.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile("../../internal/commissioner/testdata/native-assets.wire.json")
	if err != nil {
		t.Fatal(err)
	}
	var held commissioner.NativeAssets
	if err := json.Unmarshal(body, &held); err != nil {
		t.Fatal(err)
	}
	if held.Snapshot == nil {
		t.Fatal("missing held snapshot")
	}
	commands := []commissioner.RawCommand{
		{League: "native.cli", Command: commissioner.Create, Args: commissioner.RawArgs{Name: "Recovery command proof", Preset: commissioner.PresetLegacy, Season: held.Snapshot.Season}, Request: "create", Reason: "Recovery proof", Effective: "now", Visibility: "league"},
		{League: "native.cli", Command: commissioner.NativeSeed, Args: commissioner.RawArgs{Native: held.Snapshot}, Expected: 1, Request: "seed", Reason: "Recovery proof", Effective: "now", Visibility: "league"},
	}
	for _, command := range commands {
		if _, err := ledger.ExecuteCommissioner(t.Context(), command, "creator", nil); err != nil {
			t.Fatal(err)
		}
	}
	if err := pools.Close(); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "recovered")
	args := []string{"stand-up", "-from-db", source, "-league", "native.cli", "-to-dir", dest}
	var output bytes.Buffer
	if err := run(t.Context(), args, &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "Rosters and contracts restored and checked exactly") || !strings.Contains(output.String(), "read-only") {
		t.Fatal(output.String())
	}
	p, err := db.Open(t.Context(), filepath.Join(dest, "history.db"), history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = p.Close() }()
	var logged bool
	if err := p.Read().QueryRowContext(t.Context(), `SELECT EXISTS(SELECT 1 FROM league_recovery_log WHERE league='native.cli' AND message LIKE 'Rosters and contracts restored%')`).Scan(&logged); err != nil || !logged {
		t.Fatalf("recovery was not logged: %v", err)
	}
	if err := run(t.Context(), args, &output); err == nil {
		t.Fatal("console recovery overwrote an existing destination")
	}
	for _, bad := range [][]string{{"stand-up"}, {"stand-up", "unexpected"}} {
		if err := run(t.Context(), bad, &output); err == nil {
			t.Fatal("invalid console input accepted")
		}
	}
}
