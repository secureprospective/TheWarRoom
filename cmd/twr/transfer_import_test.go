package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

func TestLeagueImportCLIInactiveRetryAndMissingDestination(t *testing.T) {
	source, digest, key := exportCLIFixture(t)
	file := filepath.Join(filepath.Dir(source), "move.twr")
	var messages bytes.Buffer
	e := env{out: &messages}
	root := e.root()
	root.SetArgs([]string{"league-export", "save", "--db", source, "--league", "native.cli", "--source-generation", "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee", "--football", digest, "--key-file", key, "--output", file})
	if err := root.ExecuteContext(t.Context()); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "target.db")
	args := []string{"league-import", "--db", path, "--file", file, "--key-file", key}
	root = e.root()
	root.SetArgs(args)
	if err := root.ExecuteContext(t.Context()); err == nil {
		t.Fatal("missing destination accepted")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("created missing destination", err)
	}
	pools, err := db.Open(t.Context(), path, history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = pools.Close() }()
	registry, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	store := history.New(pools, registry)
	if err := store.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	var first string
	for i := range 2 {
		messages.Reset()
		root = e.root()
		root.SetArgs(args)
		if err := root.ExecuteContext(t.Context()); err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			first = messages.String()
		}
		if messages.String() != first || !strings.Contains(first, "is inactive") {
			t.Fatal("retry changed receipt or omitted inactive warning", messages.String())
		}
	}
	leagues, err := store.NativeLeagues(t.Context())
	if err != nil || len(leagues) != 0 {
		t.Fatal("import activated destination", leagues, err)
	}
}
