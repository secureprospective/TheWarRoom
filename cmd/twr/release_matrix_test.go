package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/hostedfixture"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

type releaseCLIRow struct {
	Args       commissioner.RawArgs `json:"args"`
	Effective  string               `json:"effective"`
	Visibility string               `json:"visibility"`
}

type releaseCLIPolicy struct {
	state     governance.State
	principal governance.Principal
}

func (p releaseCLIPolicy) EnrollmentPolicy(_ context.Context, _ string) (governance.State, governance.Principal, error) {
	return p.state, p.principal, nil
}

type releaseCLIClock struct{}

func (releaseCLIClock) Now() time.Time { return time.Now().UTC() }
func (releaseCLIClock) Trusted() error { return nil }

type releaseCLIFixture struct {
	store     *history.Store
	tokens    *accounts.TokenService
	human     accounts.RawSession
	principal string
	env       env
	memory    *memoryStore
	address   string
}

func newReleaseCLI(t *testing.T) *releaseCLIFixture {
	t.Helper()
	pools, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "server.db"), history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := pools.Close(); err != nil {
			t.Error(err)
		}
	})
	ledger := history.New(pools, &measures.Registry{})
	if err := ledger.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	credentials := accountstore.New(pools)
	raw := accounts.RawCredentials{Username: "commissioner", Password: "release-matrix-password"}
	principal, err := hostedfixture.Bootstrap(t.Context(), credentials, raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := hostedfixture.CreateLeague(t.Context(), ledger, principal); err != nil {
		t.Fatal(err)
	}
	if _, err := ledger.ExecuteCommissioner(t.Context(), commissioner.RawCommand{League: "native.hosted", Command: "commish.seat.replace", Args: commissioner.RawArgs{People: &commissioner.RawPeople{Franchise: "franchise.01", Principal: string(principal)}}, Expected: 1, Request: "seat", Reason: "Seat disposable commissioner", Effective: "now", Visibility: "league"}, string(principal), nil); err != nil {
		t.Fatal(err)
	}
	if _, err := ledger.ExecuteCommissioner(t.Context(), commissioner.RawCommand{League: "native.other", Command: commissioner.Create, Args: commissioner.RawArgs{Name: "Other league", Preset: commissioner.PresetEmpty, Season: 2026, FranchiseCount: 2}, Request: "create", Reason: "Create a second disposable league", Effective: "now", Visibility: "league"}, string(principal), nil); err != nil {
		t.Fatal(err)
	}
	passwords := accounts.New(credentials)
	sessions := accounts.NewSessions(passwords, credentials)
	human, err := sessions.Login(t.Context(), raw, "matrix CLI")
	if err != nil {
		t.Fatal(err)
	}
	totp, err := accounts.NewTOTP(passwords, credentials, bytes.Repeat([]byte{19}, 32))
	if err != nil {
		t.Fatal(err)
	}
	events, err := ledger.Events(t.Context(), "native.hosted", commissioner.Stream, 0)
	if err != nil {
		t.Fatal(err)
	}
	records := make([]commissioner.Record, len(events))
	for i, event := range events {
		records[i] = *event.Payload.Commissioner
	}
	state, err := commissioner.GovernanceState(records)
	if err != nil {
		t.Fatal(err)
	}
	tokens := accounts.NewTokens(credentials, serverhttp.TokenPolicy{Store: ledger})
	confirmations, err := serverhttp.NewConfirmations()
	if err != nil {
		t.Fatal(err)
	}
	handler := serverhttp.Hosted{Store: ledger, Sessions: sessions, Tokens: tokens, TOTP: totp, Enrollment: accounts.NewEnrollment(credentials, releaseCLIPolicy{state, principal}), Worker: leagueservice.NewDueWorker(ledger, nil, releaseCLIClock{}), Confirmations: confirmations, Registry: "id,name,gm_question,governs\n"}
	server := httptest.NewTLSServer(handler.Handler())
	t.Cleanup(server.Close)
	memory := &memoryStore{tokens: map[string]string{}}
	e := env{configDir: t.TempDir(), roots: server.Client().Transport.(*http.Transport).TLSClientConfig.RootCAs, out: &bytes.Buffer{}, keychain: func(origin, league string) tokenStore { return memoryKey{store: memory, key: origin + " " + league} }}
	return &releaseCLIFixture{ledger, tokens, human.RawSession, string(principal), e, memory, server.URL}
}

func TestReleaseMatrixCLIAndMCPTokenModes(t *testing.T) {
	t.Parallel()
	body, err := os.ReadFile("../../internal/app/commissioner/testdata/release-mutations.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows map[string]releaseCLIRow
	if err := json.Unmarshal(body, &rows); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"preview", "execute"} {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()
			f := newReleaseCLI(t)
			before, err := f.store.Events(t.Context(), "native.hosted", commissioner.Stream, 0)
			if err != nil {
				t.Fatal(err)
			}
			otherBefore, err := f.store.Events(t.Context(), "native.other", commissioner.Stream, 0)
			if err != nil {
				t.Fatal(err)
			}
			serverSide, clientSide := mcp.NewInMemoryTransports()
			serverSession, err := f.env.mcpServer().Connect(t.Context(), serverSide, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = serverSession.Close() }()
			client, err := mcp.NewClient(&mcp.Implementation{Name: "release-matrix", Version: "1"}, nil).Connect(t.Context(), clientSide, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = client.Close() }()
			for command, row := range rows {
				t.Run(command, func(t *testing.T) {
					scope := accounts.RawTokenScope{League: "native.hosted", Franchise: "franchise.01", Commands: []string{command, "commish.ledger.read"}, Modes: []string{mode, "read"}}
					token, err := f.tokens.Issue(t.Context(), f.human, scope)
					undelegable := !accounts.DelegableCommand(command)
					if undelegable {
						if err == nil {
							t.Fatal("undelegable command received authority")
						}
						scope.Commands = []string{commissioner.NotePost, "commish.ledger.read"}
						token, err = f.tokens.Issue(t.Context(), f.human, scope)
					}
					if err != nil {
						t.Fatal(err)
					}
					opposite := "execute"
					if mode == "execute" {
						opposite = "preview"
					}
					raw := commissioner.RawCommand{League: "native.hosted", Command: command, Args: row.Args, Expected: 2, Request: "matrix." + command, Reason: "Disposable release matrix change", Effective: row.Effective, Visibility: row.Visibility}
					raw = releaseCLIActor(t, raw, f.principal)
					for _, league := range []string{"native.hosted", "native.other"} {
						f.selectToken(t, token, league)
						raw.League = league
						f.refused(t, client, raw, opposite)
						if league != "native.hosted" || undelegable {
							f.refused(t, client, raw, mode)
						}
					}
					if err := f.tokens.Revoke(t.Context(), f.human, token.ID); err != nil {
						t.Fatal(err)
					}
					f.selectToken(t, token, "native.hosted")
					raw.League = "native.hosted"
					f.refused(t, client, raw, mode)
				})
			}
			after, err := f.store.Events(t.Context(), "native.hosted", commissioner.Stream, 0)
			if err != nil || !reflect.DeepEqual(before, after) {
				t.Fatalf("CLI/MCP changed primary league: %v", err)
			}
			otherAfter, err := f.store.Events(t.Context(), "native.other", commissioner.Stream, 0)
			if err != nil || !reflect.DeepEqual(otherBefore, otherAfter) {
				t.Fatalf("CLI/MCP changed other league: %v", err)
			}
		})
	}
}

func releaseCLIActor(t *testing.T, raw commissioner.RawCommand, principal string) commissioner.RawCommand {
	t.Helper()
	if raw.Args.People != nil {
		people := *raw.Args.People
		if people.Principal == "$actor" {
			people.Principal = principal
		}
		raw.Args.People = &people
	}
	if err := raw.Validate(); err != nil {
		t.Fatal(err)
	}
	return raw
}

func (f *releaseCLIFixture) selectToken(t *testing.T, token accounts.AgentToken, league string) {
	t.Helper()
	f.memory.tokens[f.address+" "+league] = token.Bearer
	if err := f.env.save(connection{Address: f.address, League: league, Helper: token.ID}); err != nil {
		t.Fatal(err)
	}
}

func (f *releaseCLIFixture) refused(t *testing.T, client *mcp.ClientSession, raw commissioner.RawCommand, mode string) {
	t.Helper()
	body, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.env.runTwr(t, string(body), mode, "-"); err == nil {
		t.Fatalf("CLI %s admitted %s", mode, raw.Command)
	}
	args := changeInput{Command: raw.Command, Args: raw.Args, Reason: raw.Reason, Request: raw.Request, Effective: raw.Effective, Visibility: raw.Visibility}
	result, err := client.CallTool(t.Context(), &mcp.CallToolParams{Name: mode + "_change", Arguments: args})
	if err != nil || !result.IsError {
		t.Fatalf("MCP %s admitted %s: result=%+v err=%v", mode, raw.Command, result, err)
	}
	f.env.out.(*bytes.Buffer).Reset()
}
