package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	catalog "github.com/secureprospective/TheWarRoom/internal/commandcatalog"
)

func TestBrowserGenerationFreshnessAndRefusal(t *testing.T) {
	if err := runBrowser("../../src/app/commands/browser.gen.ts", true); err != nil {
		t.Fatal(err)
	}
	commands := catalog.BrowserCommands()
	first, err := generateBrowser(commands)
	if err != nil {
		t.Fatal(err)
	}
	second, err := generateBrowser(commands)
	if err != nil || !bytes.Equal(first, second) {
		t.Fatal("browser generation is not deterministic", err)
	}
	for _, invalid := range [][]catalog.BrowserCommand{
		append(commands, commands[0]), {{ID: "", Label: "Missing name"}}, {{ID: "missing.label"}},
	} {
		if body, err := generateBrowser(invalid); err == nil || body != nil {
			t.Fatal("invalid browser inventory generated a door")
		}
	}
	path := t.TempDir() + "/browser.gen.ts"
	if err := os.WriteFile(path, []byte("stale"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := runBrowser(path, true); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatal("stale browser publication accepted", err)
	}
	held, err := os.ReadFile(path)
	if err != nil || string(held) != "stale" {
		t.Fatal("freshness check wrote the file", err)
	}
}
