package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/composition"
)

func TestEveryGeneratedDoorIsFreshAndRejectsDrift(t *testing.T) {
	registry, err := composition.CommandRegistry()
	if err != nil {
		t.Fatal(err)
	}
	first, err := generateDoors(registry)
	if err != nil {
		t.Fatal(err)
	}
	second, err := generateDoors(registry)
	if err != nil {
		t.Fatal(err)
	}
	if err := runDoors("../../..", true); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	for path, body := range first {
		if !bytes.Equal(body, second[path]) {
			t.Fatalf("nondeterministic %s", path)
		}
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for path, body := range first {
		t.Run(path, func(t *testing.T) {
			full := filepath.Join(root, path)
			if err := os.WriteFile(full, []byte("stale"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := runDoors(root, true); err == nil || !strings.Contains(err.Error(), "stale command door "+full) {
				t.Fatalf("wrong drift failure: %v", err)
			}
			held, err := os.ReadFile(full)
			if err != nil || string(held) != "stale" {
				t.Fatal("freshness check changed the file", err)
			}
			if err := os.WriteFile(full, body, 0o600); err != nil {
				t.Fatal(err)
			}
		})
	}
}
