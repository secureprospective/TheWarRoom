package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

const testToken = "twr-agent-token-never-printed" //nolint:gosec // a fake token for the fake league

type memoryStore struct{ tokens map[string]string }

type memoryKey struct {
	store *memoryStore
	key   string
}

func (k memoryKey) Get() (string, error) {
	if token, ok := k.store.tokens[k.key]; ok {
		return token, nil
	}
	return "", os.ErrNotExist
}
func (k memoryKey) Set(secret string) error { k.store.tokens[k.key] = secret; return nil }
func (k memoryKey) Delete() error           { delete(k.store.tokens, k.key); return nil }

// fakeLeague answers the two paths twr uses and records what each dispatch asked for.
type fakeLeague struct {
	mu         sync.Mutex
	revoked    bool
	dispatched []serverhttp.RawNativeRequest
	confirms   []string
}

func (f *fakeLeague) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.revoked || r.Header.Get("Authorization") != "Bearer "+testToken {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	var request serverhttp.RawNativeRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	switch r.URL.Path {
	case serverhttp.LedgerPath:
		_, _ = w.Write([]byte(`{"league":"native.test","revision":7}`))
	case serverhttp.DispatchPath:
		f.dispatched = append(f.dispatched, request)
		f.confirms = append(f.confirms, r.Header.Get("X-TWR-Human-Confirmation"))
		_, _ = w.Write([]byte(`{"mode":"` + request.Mode + `"}`))
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func newTestEnv(t *testing.T) (env, *fakeLeague, *bytes.Buffer, string) {
	t.Helper()
	league := &fakeLeague{}
	server := httptest.NewTLSServer(league)
	t.Cleanup(server.Close)
	out := &bytes.Buffer{}
	store := &memoryStore{tokens: map[string]string{}}
	e := env{configDir: t.TempDir(), roots: server.Client().Transport.(*http.Transport).TLSClientConfig.RootCAs, out: out,
		keychain: func(origin, league string) tokenStore { return memoryKey{store: store, key: origin + " " + league} }}
	return e, league, out, server.URL
}

func (e env) runTwr(t *testing.T, stdin string, args ...string) error {
	t.Helper()
	e.in = strings.NewReader(stdin)
	root := e.root()
	root.SetArgs(args)
	return root.ExecuteContext(t.Context())
}

func TestTwrConnectsPreviewsByDefaultAndNeverPrintsTheToken(t *testing.T) {
	e, league, out, address := newTestEnv(t)
	if err := e.runTwr(t, testToken+"\n", "connect", "--address", address, "--league", "native.test"); err != nil {
		t.Fatal(err)
	}
	change := filepath.Join(t.TempDir(), "change.json")
	if err := os.WriteFile(change, []byte(`{"command":"ownership.correct","reason":"fix"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := e.runTwr(t, "", "preview", change); err != nil {
		t.Fatal(err)
	}
	if err := e.runTwr(t, "", "execute", change, "--confirmation", "123456"); err != nil {
		t.Fatal(err)
	}
	if len(league.dispatched) != 2 || league.dispatched[0].Mode != "preview" || league.dispatched[1].Mode != "execute" {
		t.Fatalf("dispatches = %+v", league.dispatched)
	}
	first := league.dispatched[0].Command
	if first == nil || first.League != "native.test" || first.Expected != 7 || league.confirms[0] != "" || league.confirms[1] != "123456" {
		t.Fatalf("first = %+v confirms = %q", first, league.confirms)
	}
	saved, err := os.ReadFile(e.configPath())
	if err != nil || strings.Contains(string(saved), testToken) || strings.Contains(out.String(), testToken) {
		t.Fatalf("token leaked: config %q output %q (%v)", saved, out.String(), err)
	}
	league.revoked = true
	if err := e.runTwr(t, "", "ledger"); err == nil || !strings.Contains(err.Error(), "revoked") {
		t.Fatalf("revoked ledger = %v", err)
	}
	if err := e.runTwr(t, "", "disconnect"); err != nil {
		t.Fatal(err)
	}
	if err := e.runTwr(t, "", "ledger"); err == nil || !strings.Contains(err.Error(), "twr connect") {
		t.Fatalf("after disconnect = %v", err)
	}
}

func TestTwrMCPOffersLedgerPreviewAndExecute(t *testing.T) {
	e, league, _, address := newTestEnv(t)
	if err := e.runTwr(t, testToken+"\n", "connect", "--address", address, "--league", "native.test"); err != nil {
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
	tools, err := session.ListTools(t.Context(), nil)
	if err != nil || len(tools.Tools) != 3 {
		t.Fatalf("tools = %+v (%v)", tools, err)
	}
	result, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: "preview_change",
		Arguments: map[string]any{"command": "ownership.correct", "confirmation": "ignored"}})
	if err != nil || result.IsError || len(league.dispatched) != 1 || league.dispatched[0].Mode != "preview" || league.confirms[0] != "" {
		t.Fatalf("preview = %+v %s dispatched %+v (%v)", result, result.Content[0].(*mcp.TextContent).Text, league.dispatched, err)
	}
	league.revoked = true
	result, err = session.CallTool(t.Context(), &mcp.CallToolParams{Name: "league_ledger", Arguments: map[string]any{}})
	if err != nil || !result.IsError {
		t.Fatalf("revoked ledger = %+v (%v)", result, err)
	}
}
