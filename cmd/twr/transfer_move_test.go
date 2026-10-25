package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

func TestLeagueMoveCLIConfirmedCutoverAndAbort(t *testing.T) {
	source, football, key := exportCLIFixture(t)
	target := filepath.Join(t.TempDir(), "target.db")
	pools, err := db.Open(t.Context(), target, history.SchemaVersion())
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
	e := env{out: &out}
	run := func(args ...string) error { cmd := e.root(); cmd.SetArgs(args); return cmd.ExecuteContext(t.Context()) }
	file := filepath.Join(t.TempDir(), "move.twr")
	if err := run("league-export", "save", "--db", source, "--league", "native.cli", "--source-generation", "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee", "--football", football, "--key-file", key, "--output", file); err != nil {
		t.Fatal(err)
	}
	if err := run("league-import", "--db", target, "--file", file, "--key-file", key); err != nil {
		t.Fatal(err)
	}
	flags := []string{"--source-db", source, "--target-db", target, "--file", file, "--key-file", key, "--commissioner", "creator"}
	if err := run(append([]string{"league-move", "seal"}, flags...)...); err == nil {
		t.Fatal("missing confirmation accepted")
	}
	flags = append(flags, "--confirm")
	for _, action := range []string{"seal", "abort", "seal", "activate", "activate"} {
		out.Reset()
		if err := run(append([]string{"league-move", action}, flags...)...); err != nil {
			t.Fatal(action, err)
		}
	}
	if !strings.Contains(out.String(), "read-only copy") || strings.Contains(out.String(), "native.cli") {
		t.Fatal("not plain owner wording", out.String())
	}
	if err := run(append([]string{"league-move", "abort"}, flags...)...); err == nil {
		t.Fatal("activated target allowed source abort")
	}
	server, closeServer, err := openMoveStore(t.Context(), target)
	if err != nil {
		t.Fatal(err)
	}
	command := commissioner.RawCommand{League: "native.cli", Command: commissioner.RulesSet, Args: commissioner.RawArgs{Field: "cap.amount", Value: json.RawMessage(`100000000000`)}, Expected: 1, Request: "server.change", Reason: "Server change before moving back", Effective: "now", Visibility: "league"}
	if _, err := server.ExecuteCommissioner(t.Context(), command, "creator", nil); err != nil {
		closeServer()
		t.Fatal(err)
	}
	before, _, _, err := server.CaptureTransfer(t.Context(), "native.cli", []string{football})
	closeServer()
	if err != nil {
		t.Fatal(err)
	}
	reverse := filepath.Join(t.TempDir(), "reverse.twr")
	if err := run("league-export", "save", "--db", target, "--league", "native.cli", "--football", football, "--key-file", key, "--output", reverse); err != nil {
		t.Fatal("export stored generation", err)
	}
	reverseFlags := []string{"--source-db", target, "--target-db", source, "--file", reverse, "--key-file", key, "--commissioner", "creator", "--confirm"}
	for _, action := range []string{"seal", "abort", "seal", "activate"} {
		if err := run(append([]string{"league-move", action}, reverseFlags...)...); err != nil {
			t.Fatal("reverse", action, err)
		}
	}
	desktop, closeDesktop, err := openMoveStore(t.Context(), source)
	if err != nil {
		t.Fatal(err)
	}
	defer closeDesktop()
	after, _, _, err := desktop.CaptureTransfer(t.Context(), "native.cli", []string{football})
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("CLI reverse lost server changes", err)
	}

}
