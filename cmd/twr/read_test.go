package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/secureprospective/TheWarRoom/internal/serverclient"
)

func TestTwrReadPrintsTheServersData(t *testing.T) {
	e, league, out, address := newTestEnv(t)
	if err := e.runTwr(t, testLink, "connect", "--address", address, "--league", "native.test"); err != nil {
		t.Fatal(err)
	}
	scope := filepath.Join(t.TempDir(), "scope.json")
	if err := os.WriteFile(scope, []byte(`{"season":2026}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var want bytes.Buffer
	if err := json.Indent(&want, []byte(fakeReadData), "", "  "); err != nil {
		t.Fatal(err)
	}
	want.WriteByte('\n')

	out.Reset()
	if err := e.runTwr(t, "", "read", "standings.read", scope); err != nil {
		t.Fatal(err)
	}
	if out.String() != want.String() {
		t.Fatalf("read printed %q, want %q", out.String(), want.String())
	}
	out.Reset()
	if err := e.runTwr(t, `{"season":2026}`, "read", "standings.read", "-"); err != nil {
		t.Fatal(err)
	}
	if out.String() != want.String() {
		t.Fatalf("stdin read printed %q, want %q", out.String(), want.String())
	}
	if len(league.reads) != 2 || league.reads[0].Command != "standings.read" || league.reads[0].Scope.League != "native.test" || league.reads[0].Scope.Season != 2026 {
		t.Fatalf("reads = %+v", league.reads)
	}
}

func TestTwrReadRefusesNonReadCommandsWithoutContactingTheLeague(t *testing.T) {
	e, league, _, address := newTestEnv(t)
	if err := e.runTwr(t, testLink, "connect", "--address", address, "--league", "native.test"); err != nil {
		t.Fatal(err)
	}
	before := league.requests
	for _, command := range []string{"ownership.correct", ".read", "standings.read.bad"} {
		err := e.runTwr(t, "", "read", command)
		if err == nil || !strings.Contains(err.Error(), ".read") {
			t.Fatalf("read %q = %v, want a plain refusal", command, err)
		}
	}
	serverSide, clientSide := mcp.NewInMemoryTransports()
	serverSession, err := e.mcpServer().Connect(t.Context(), serverSide, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = serverSession.Close() }()
	session, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil).Connect(t.Context(), clientSide, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = session.Close() }()
	result, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: "league_read", Arguments: map[string]any{"command": "preview.change"}})
	if err != nil || !result.IsError {
		t.Fatalf("MCP non-read = %+v (%v)", result, err)
	}
	if league.requests != before || len(league.reads) != 0 || len(league.dispatched) != 0 {
		t.Fatalf("non-read reached the league: requests %d -> %d, reads %+v", before, league.requests, league.reads)
	}
}

func TestTwrMCPLeagueReadReturnsTheData(t *testing.T) {
	e, league, _, address := newTestEnv(t)
	if err := e.runTwr(t, testLink, "connect", "--address", address, "--league", "native.test"); err != nil {
		t.Fatal(err)
	}
	serverSide, clientSide := mcp.NewInMemoryTransports()
	serverSession, err := e.mcpServer().Connect(t.Context(), serverSide, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = serverSession.Close() }()
	session, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil).Connect(t.Context(), clientSide, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = session.Close() }()
	result, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: "league_read",
		Arguments: map[string]any{"command": "standings.read", "scope": map[string]any{"season": 2026}}})
	if err != nil || result.IsError {
		t.Fatalf("league_read = %+v (%v)", result, err)
	}
	if got := result.Content[0].(*mcp.TextContent).Text; got != fakeReadData {
		t.Fatalf("league_read text = %q, want %q", got, fakeReadData)
	}
	if len(league.reads) != 1 || league.reads[0].Command != "standings.read" || league.reads[0].Scope.League != "native.test" || league.reads[0].Scope.Season != 2026 {
		t.Fatalf("reads = %+v", league.reads)
	}
	// An unknown scope field is refused the same way the CLI refuses it.
	result, err = session.CallTool(t.Context(), &mcp.CallToolParams{Name: "league_read",
		Arguments: map[string]any{"command": "standings.read", "scope": map[string]any{"bogus": 1}}})
	if err != nil || !result.IsError || len(league.reads) != 1 {
		t.Fatalf("unknown scope field = %+v (%v) reads %d", result, err, len(league.reads))
	}
}

func TestTwrReadRefusalExitsNonZeroWithTheMessage(t *testing.T) {
	e, league, _, address := newTestEnv(t)
	if err := e.runTwr(t, testLink, "connect", "--address", address, "--league", "native.test"); err != nil {
		t.Fatal(err)
	}
	league.refuseReads = true
	err := e.runTwr(t, "", "read", "standings.read")
	if err == nil || !serverclient.IsRefused(err) || !strings.Contains(err.Error(), "may not cover it") {
		t.Fatalf("refused read = %v, want the league's refusal", err)
	}
	serverSide, clientSide := mcp.NewInMemoryTransports()
	serverSession, err := e.mcpServer().Connect(t.Context(), serverSide, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = serverSession.Close() }()
	session, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil).Connect(t.Context(), clientSide, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = session.Close() }()
	result, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: "league_read", Arguments: map[string]any{"command": "standings.read"}})
	if err != nil || !result.IsError || !strings.Contains(result.Content[0].(*mcp.TextContent).Text, "may not cover it") {
		t.Fatalf("MCP refused read = %+v (%v)", result, err)
	}
}
