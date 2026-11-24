package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/secureprospective/TheWarRoom/internal/serverclient"
)

func TestTwrWorkflowPassesArgumentsAndModeAndRejectsNonObjects(t *testing.T) {
	e, league, out, address := newTestEnv(t)
	if err := e.runTwr(t, testLink, "connect", "--address", address, "--league", "native.test"); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := e.runTwr(t, "", "workflow", "draft.read", "--json",
		`{"week":4,"sort":"points","league":"native.other"}`); err != nil {
		t.Fatal(err)
	}
	if len(league.workflows) != 1 {
		t.Fatalf("workflows = %+v", league.workflows)
	}
	var args map[string]json.RawMessage
	if err := json.Unmarshal(league.workflows[0].Arguments, &args); err != nil {
		t.Fatal(err)
	}
	if league.workflows[0].Mode != "read" || league.workflows[0].Command != "draft.read" ||
		string(args["week"]) != "4" || string(args["sort"]) != `"points"` ||
		string(args["league"]) != `"native.test"` {
		t.Fatalf("read workflow = %+v args %s", league.workflows[0], league.workflows[0].Arguments)
	}
	before := len(league.workflows)
	if err := e.runTwr(t, "", "workflow", "draft.read", "--json", `[]`); err == nil ||
		!strings.Contains(err.Error(), "JSON object") || len(league.workflows) != before {
		t.Fatalf("non-object = %v; workflow calls %d", err, len(league.workflows))
	}
	if err := e.runTwr(t, "", "workflow", "draft.queue.set", "--mode", "execute",
		"--json", `{"players":[]}`); err != nil {
		t.Fatal(err)
	}
	if got := league.workflows[len(league.workflows)-1]; got.Mode != "execute" ||
		got.Command != "draft.queue.set" {
		t.Fatalf("execute workflow = %+v", got)
	}
}

func TestTwrWorkflowRefusalMessages(t *testing.T) {
	e, league, _, address := newTestEnv(t)
	if err := e.runTwr(t, testLink, "connect", "--address", address, "--league", "native.test"); err != nil {
		t.Fatal(err)
	}
	league.refuseReads = true
	league.refuseStatus = http.StatusConflict
	league.refuseContentType = "text/plain; charset=utf-8"
	league.refuseBody = "  sort is unavailable this week.\n"
	err := e.runTwr(t, "", "workflow", "draft.read")
	if err == nil || !serverclient.IsRefused(err) ||
		!strings.Contains(err.Error(), "sort is unavailable this week") {
		t.Fatalf("text refusal = %v", err)
	}
	league.refuseContentType = "application/json"
	league.refuseBody = ""
	err = e.runTwr(t, "", "workflow", "draft.read")
	if err == nil || !serverclient.IsRefused(err) ||
		!strings.Contains(err.Error(), serverclient.ErrRefused.Error()) {
		t.Fatalf("generic refusal = %v", err)
	}
	err = e.runTwr(t, "", "read", "standings.read")
	if err == nil || !serverclient.IsRefused(err) ||
		!strings.Contains(err.Error(), "the league refused this read") ||
		strings.Contains(err.Error(), "did not record this change") {
		t.Fatalf("read refusal wording = %v", err)
	}
}

func TestTwrMCPWorkflowListsAndCallsAgentWorkflow(t *testing.T) {
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
	session, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil).
		Connect(t.Context(), clientSide, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = session.Close() }()
	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, tool := range tools.Tools {
		if tool.Name == "league_workflow" {
			found = tool.Description != ""
		}
	}
	if !found {
		t.Fatalf("league_workflow missing or undescribed: %+v", tools.Tools)
	}
	result, err := session.CallTool(t.Context(), &mcp.CallToolParams{
		Name: "league_workflow", Arguments: map[string]any{
			"command": "draft.queue.set", "mode": "execute", "arguments": map[string]any{"players": []any{}, "week": 3},
		}})
	if err != nil || result.IsError || len(league.workflows) != 1 {
		t.Fatalf("league_workflow = %+v (%v), workflows %+v", result, err, league.workflows)
	}
	got := league.workflows[0]
	var args map[string]json.RawMessage
	if err := json.Unmarshal(got.Arguments, &args); err != nil {
		t.Fatal(err)
	}
	if got.Command != "draft.queue.set" || got.Mode != "execute" ||
		string(args["week"]) != "3" || string(args["league"]) != `"native.test"` {
		t.Fatalf("MCP workflow = %+v arguments %s", got, got.Arguments)
	}
}

// A refused preview keeps its helpful wording even though the server's 409 now carries plain words.
func TestTwrRefusedPreviewStillSaysPreviewRefused(t *testing.T) {
	e, league, _, address := newTestEnv(t)
	if err := e.runTwr(t, testLink, "connect", "--address", address, "--league", "native.test"); err != nil {
		t.Fatal(err)
	}
	change := filepath.Join(t.TempDir(), "change.json")
	if err := os.WriteFile(change, []byte(`{"command":"ownership.correct","reason":"fix"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	league.refuseChanges = "The league could not accept this action."
	if err := e.runTwr(t, "", "preview", change); !errors.Is(err, serverclient.ErrPreviewRefused) {
		t.Fatalf("refused preview = %v, want ErrPreviewRefused", err)
	}
	if err := e.runTwr(t, "", "execute", change, "--confirmation", "123456"); !errors.Is(err, serverclient.ErrRefused) {
		t.Fatalf("refused execute = %v, want ErrRefused", err)
	}
}
