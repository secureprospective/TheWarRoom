package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	app "github.com/secureprospective/TheWarRoom/internal/app/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/app/offers"
	catalog "github.com/secureprospective/TheWarRoom/internal/commandcatalog"
)

func TestNativeClientCatalogParity(t *testing.T) {
	if err := run("../../src/app/commands/native.gen.ts", true); err != nil {
		t.Fatal(err)
	}
	commands := app.NativeCommands()
	first, err := generate(commands)
	if err != nil {
		t.Fatal(err)
	}
	second, err := generate(commands)
	if err != nil || !bytes.Equal(first, second) {
		t.Fatal("generation is not deterministic", err)
	}
	commands[0].Permission = catalog.Unset
	if body, err := generate(commands); err == nil || body != nil {
		t.Fatal("unset permission generated a client")
	}
}

func TestSchemaSubtreesSharedWithoutLosingConstraints(t *testing.T) {
	table := schemaTable{names: make(map[string]string)}
	schema := catalog.Schema{Type: "string"}
	first, err := table.intern(schema)
	if err != nil {
		t.Fatal(err)
	}
	repeated, err := table.intern(schema)
	if err != nil || first != repeated || strings.Count(table.out.String(), "const ") != 1 {
		t.Fatal("identical schemas were duplicated", err)
	}
	schema.Enum = []string{"allowed"}
	constrained, err := table.intern(schema)
	if err != nil || constrained == first || !strings.Contains(table.out.String(), `"allowed"`) {
		t.Fatal("distinct constraints were lost", err)
	}
	body, err := generate(app.NativeCommands())
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Count(body, []byte("Type:\"string\",\n  Enum:null,\n  Format:\"\",\n  Validator:\"\",\n  Nullable:false,\n  Fields:null,\n  Items:null")) != 1 {
		t.Fatal("unconstrained text schema was not shared across the catalog")
	}
}

func TestStaleGenerationRefusedWithoutWriting(t *testing.T) {
	path := t.TempDir() + "/native.gen.ts"
	const old = "stale"
	if err := os.WriteFile(path, []byte(old), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := run(path, true); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatal(err)
	}
	body, err := os.ReadFile(path)
	if err != nil || string(body) != old {
		t.Fatal("check changed the file", err)
	}
}

func TestMFLClientCatalogParity(t *testing.T) {
	body, err := generate(offers.MFLCommands())
	if err != nil {
		t.Fatal(err)
	}
	body = bytes.Replace(body, []byte("export const nativeCommands"), []byte("export const mflCommands"), 1)
	held, err := os.ReadFile("../../src/app/commands/mfl.gen.ts")
	if err != nil || !bytes.Equal(body, held) {
		t.Fatal("MFL catalog stale: run go run ./frontend/scripts/nativegen -surface mfl", err)
	}
}
