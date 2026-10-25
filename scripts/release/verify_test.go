package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestVerifyRequiresAuthenticatedBootstrap(t *testing.T) {
	var output bytes.Buffer
	if err := run(nil, &output); err == nil {
		t.Fatal("missing bootstrap accepted")
	}
	if output.Len() != 0 {
		t.Fatal("reported success without verification")
	}
}

func TestVerifyRejectsSecretConfigBeforeCacheWrite(t *testing.T) {
	directory := t.TempDir()
	root := filepath.Join(directory, "root.json")
	config := filepath.Join(directory, "config.json")
	cache := filepath.Join(directory, "cache")
	if err := os.WriteFile(root, []byte("not a trusted root"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(config, []byte(`{"tunnel_token":"sentinel"}`), 0600); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := run([]string{"-root", root, "-config", config, "-cache", cache, "-target", "app.tar.gz"}, &output); err == nil {
		t.Fatal("secret configuration accepted")
	}
	if _, err := os.Stat(cache); !os.IsNotExist(err) {
		t.Fatal("invalid bootstrap wrote cache")
	}
	if output.Len() != 0 {
		t.Fatal("reported success without verification")
	}
}
