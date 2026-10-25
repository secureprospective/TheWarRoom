package accounts_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// This trusted policy fixture resolves references and current grants independently
// of token metadata. Production transport composition belongs to the next slice.
type tokenPolicy struct {
	state     governance.State
	principal governance.Principal
}

func (p *tokenPolicy) CheckTokenScope(_ context.Context, account accounts.Account, scope accounts.RawTokenScope) error {
	principal, err := account.Principal()
	if err != nil || principal != p.principal || scope.League != "native.tokens" || scope.Franchise != "franchise.01" {
		return accounts.ErrTokenRefused
	}
	for _, command := range scope.Commands {
		if command != commissioner.NotePost && command != "commish.people.read" {
			return accounts.ErrTokenRefused
		}
	}
	for _, mode := range scope.Modes {
		if mode == "read" && p.state.HasRole(principal, governance.DOT) {
			continue
		}
		if p.state.HasRole(principal, governance.CoCommissioner) {
			continue
		}
		return accounts.ErrTokenRefused
	}
	return nil
}
func (p *tokenPolicy) CheckTokenAccess(_ context.Context, _ accounts.Account, franchise string, access commissioner.Access) error {
	for _, reference := range access.Raw.Args.Franchises {
		if reference != franchise {
			return accounts.ErrTokenRefused
		}
	}
	if access.Player == "other-owner-player" {
		return accounts.ErrTokenRefused
	}
	return nil
}

func tokenFixture(t *testing.T) (*totpFixture, *accounts.TokenService, accounts.DeviceSession, *tokenPolicy) {
	t.Helper()
	f := newTOTPFixture(t)
	sessions := accounts.NewSessions(accounts.New(f.store), f.store)
	session := loginDevice(t, f, sessions, "token issuer")
	principal, err := f.account.Principal()
	if err != nil {
		t.Fatal(err)
	}
	policy := &tokenPolicy{principal: principal}
	policy.state, err = governance.Apply(governance.State{}, governance.LeagueCreated{Creator: "other-commissioner", Franchises: []governance.FranchiseID{"franchise.01"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, role := range []governance.Role{governance.CoCommissioner, governance.DOT} {
		policy.state, err = governance.Apply(policy.state, governance.RoleGranted{Principal: principal, Role: role})
		if err != nil {
			t.Fatal(err)
		}
	}
	return f, accounts.NewTokens(f.store, policy), session, policy
}
func tokenScope(modes ...string) accounts.RawTokenScope {
	return accounts.RawTokenScope{League: "native.tokens", Franchise: "franchise.01", Commands: []string{commissioner.NotePost, "commish.people.read"}, Modes: modes}
}
func issueToken(t *testing.T, service *accounts.TokenService, session accounts.DeviceSession, scope accounts.RawTokenScope) accounts.AgentToken {
	t.Helper()
	token, err := service.Issue(t.Context(), session.RawSession, scope)
	if err != nil {
		t.Fatal(err)
	}
	return token
}
func tokenAccess(token accounts.AgentToken, mode string) commissioner.Access {
	principal, _ := token.Account.Principal()
	command := commissioner.NotePost
	if mode == "read" {
		command = "commish.people.read"
	}
	return commissioner.Access{Principal: string(principal), League: token.Scope.League, Operation: command, Mode: mode, Franchise: token.Scope.Franchise}
}

func TestAgentTokenLifetimesScopeAndCurrentIssuerRights(t *testing.T) {
	f, service, session, policy := tokenFixture(t)
	issued := f.now
	for _, modes := range [][]string{{"read"}, {"read", "preview"}, {"execute"}, {"read", "preview", "execute"}} {
		f.now = issued
		token := issueToken(t, service, session, tokenScope(modes...))
		lifetime := accounts.TokenReadLifetime
		if strings.Contains(strings.Join(modes, ","), "execute") {
			lifetime = accounts.TokenExecuteLifetime
		}
		if !token.ExpiresAt.Equal(issued.Add(lifetime)) {
			t.Fatal("wrong token lifetime")
		}
		for _, delta := range []time.Duration{-time.Nanosecond, 0, time.Nanosecond} {
			f.now = issued.Add(lifetime + delta)
			_, err := service.Authenticate(t.Context(), token.RawSession)
			if (err == nil) != (delta < 0) {
				t.Fatalf("expiry %v %v: %v", modes, delta, err)
			}
		}
	}
	f.now = issued
	token := issueToken(t, service, session, tokenScope("read", "preview", "execute"))
	gate := service.Admission(token.RawSession)
	for _, mode := range []string{"read", "preview", "execute"} {
		if err := gate.Admit(t.Context(), tokenAccess(token, mode)); err != nil {
			t.Fatal(err)
		}
	}
	for _, mutate := range []func(*commissioner.Access){
		func(a *commissioner.Access) { a.League = "native.other" },
		func(a *commissioner.Access) { a.Franchise = "franchise.02" },
		func(a *commissioner.Access) { a.Operation = commissioner.NoticePublish },
		func(a *commissioner.Access) { a.Principal = "other-account" },
		func(a *commissioner.Access) { a.Raw.Args.Franchises = []string{"franchise.02"} },
		func(a *commissioner.Access) { a.Player = "other-owner-player" },
		func(a *commissioner.Access) { a.Raw.League = "native.other" },
		func(a *commissioner.Access) { a.Raw.Command = "commish.role.grant" },
	} {
		access := tokenAccess(token, "execute")
		mutate(&access)
		if err := gate.Admit(t.Context(), access); err == nil {
			t.Fatalf("scope accepted: %+v", access)
		}
	}
	var err error
	policy.state, err = governance.Apply(policy.state, governance.RoleRevoked{Principal: policy.principal, Role: governance.CoCommissioner})
	if err != nil {
		t.Fatal(err)
	}
	if err := gate.Admit(t.Context(), tokenAccess(token, "execute")); err == nil {
		t.Fatal("lost role retained execution")
	}
	if err := gate.Admit(t.Context(), tokenAccess(token, "read")); err != nil {
		t.Fatalf("remaining read right lost: %v", err)
	}
	readOnly := issueToken(t, service, session, tokenScope("read"))
	policy.state, err = governance.Apply(policy.state, governance.RoleGranted{Principal: policy.principal, Role: governance.CoCommissioner})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Admission(readOnly.RawSession).Admit(t.Context(), tokenAccess(readOnly, "execute")); err == nil {
		t.Fatal("new role exceeded issuance ceiling")
	}
}

func TestAgentTokenHashOnlyCeilingAndCredentialIsolation(t *testing.T) {
	f, service, session, _ := tokenFixture(t)
	token := issueToken(t, service, session, tokenScope("read", "execute"))
	var stored string
	if err := f.pools.Read().QueryRowContext(t.Context(), `SELECT token_hash FROM account_agent_tokens WHERE id = ?`, token.ID).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored != digestToken(token.Bearer) {
		t.Fatal("bearer not hashed")
	}
	body, err := json.Marshal(token)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), token.Bearer) || strings.Contains(fmt.Sprintf("%v %+v %#v", token, token.RawSession, token), token.Bearer) {
		t.Fatal("token leaked")
	}
	if _, err := f.pools.Write().ExecContext(t.Context(), `UPDATE account_agent_tokens SET scope = ? WHERE id = ?`, `{}`, token.ID); err == nil {
		t.Fatal("issuance ceiling mutated")
	}
	for _, command := range []string{"auth.password.reset", "account.session.issue", "token.issue", "commish.role.grant", "commish.seat.replace", "commish.succession.takeover", "commish.vote.close", commissioner.Revert, "unknown.command"} {
		scope := tokenScope("execute")
		scope.Commands = []string{command}
		if _, err := service.Issue(t.Context(), session.RawSession, scope); !errors.Is(err, accounts.ErrTokenRefused) {
			t.Fatalf("credential/role operation %s: %v", command, err)
		}
	}
	if _, err := service.Issue(t.Context(), token.RawSession, tokenScope("read")); err == nil {
		t.Fatal("agent issued credentials")
	}
	sessions := accounts.NewSessions(accounts.New(f.store), f.store)
	if _, err := sessions.Authenticate(t.Context(), token.RawSession); err == nil {
		t.Fatal("agent used as device session")
	}
	if _, err := service.Authenticate(t.Context(), session.RawSession); err == nil {
		t.Fatal("device used as agent")
	}
	if err := service.Revoke(t.Context(), token.RawSession, token.ID); err == nil {
		t.Fatal("agent administered token revocation")
	}
	other, err := f.store.Credential(t.Context(), f.account.Username)
	if err != nil {
		t.Fatal(err)
	}
	other.ID, other.Username = "12345678-1234-4234-8234-123456789abc", "other-owner"
	if err := f.store.Insert(t.Context(), other, false); err != nil {
		t.Fatal(err)
	}
	otherSession, err := sessions.Login(t.Context(), accounts.RawCredentials{Username: other.Username, Password: f.credentials.Password}, "other device")
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Revoke(t.Context(), otherSession.RawSession, token.ID); !errors.Is(err, accounts.ErrTokenRefused) {
		t.Fatalf("another account revoked delegation: %v", err)
	}
	if _, err := service.Authenticate(t.Context(), token.RawSession); err != nil {
		t.Fatalf("denied revoke changed token: %v", err)
	}
	if err := service.Revoke(t.Context(), session.RawSession, token.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Authenticate(t.Context(), token.RawSession); !errors.Is(err, accounts.ErrTokenRefused) {
		t.Fatalf("revoked token: %v", err)
	}
	fresh := issueToken(t, service, session, tokenScope("read"))
	if _, err := f.store.ResetAfterRestore(t.Context(), "restore drill"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Authenticate(t.Context(), fresh.RawSession); !errors.Is(err, accounts.ErrTokenRefused) {
		t.Fatalf("restored epoch: %v", err)
	}
}

func TestAgentTokenRevokedQueuedAppendAndRetry(t *testing.T) {
	f, service, session, _ := tokenFixture(t)
	registry, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	ledger := history.New(f.pools, registry)
	if err := ledger.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	principal, _ := f.account.Principal()
	create := commissioner.RawCommand{League: "native.tokens", Command: commissioner.Create, Args: commissioner.RawArgs{Name: "Tokens", Preset: commissioner.PresetEmpty, Season: 2026, FranchiseCount: 2}, Reason: "Create token test league", Request: "create", Effective: "now", Visibility: "league"}
	receipt, err := ledger.ExecuteCommissioner(t.Context(), create, string(principal), nil)
	if err != nil {
		t.Fatal(err)
	}
	token := issueToken(t, service, session, tokenScope("execute"))
	raw := commissioner.RawCommand{League: create.League, Command: commissioner.NotePost, Args: commissioner.RawArgs{Note: &commissioner.RawNote{Text: "Token note"}}, Reason: "Post test note", Request: "note", Expected: receipt.Revision, Effective: "now", Visibility: "league"}
	ctx := commissioner.WithCredentialAdmission(t.Context(), service.Admission(token.RawSession))
	receipt, err = ledger.ExecuteCommissioner(ctx, raw, string(principal), nil)
	if err != nil {
		t.Fatal(err)
	}
	retry, err := ledger.ExecuteCommissioner(ctx, raw, string(principal), nil)
	if err != nil || retry != receipt {
		t.Fatalf("retry: %+v %v", retry, err)
	}
	tx, err := f.pools.Write().BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(t.Context(), `UPDATE account_agent_tokens SET revoked = 1 WHERE id = ?`, token.ID); err != nil {
		t.Fatal(err)
	}
	waits := f.pools.Write().Stats().WaitCount
	done := make(chan error, 1)
	go func() { _, err := ledger.ExecuteCommissioner(ctx, raw, string(principal), nil); done <- err }()
	deadline := time.Now().Add(5 * time.Second)
	for f.pools.Write().Stats().WaitCount == waits {
		if time.Now().After(deadline) {
			t.Fatal("token write did not queue")
		}
		time.Sleep(time.Millisecond)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if !errors.Is(err, accounts.ErrTokenRefused) {
			t.Fatalf("revoked retry: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("token admission deadlocked")
	}
	raw.Request, raw.Expected = "queued-new-note", receipt.Revision
	if _, err := ledger.ExecuteCommissioner(ctx, raw, string(principal), nil); !errors.Is(err, accounts.ErrTokenRefused) {
		t.Fatalf("revoked append: %v", err)
	}
	events, err := ledger.Events(t.Context(), raw.League, commissioner.Stream, 0)
	if err != nil || int64(len(events)) != receipt.Revision {
		t.Fatalf("denial changed ledger: %d %v", len(events), err)
	}
}

func TestAgentTokenValidationAndFailClosedPolicy(t *testing.T) {
	f, service, session, _ := tokenFixture(t)
	for _, scope := range []accounts.RawTokenScope{
		{},
		{League: "native.tokens", Franchise: "*", Commands: []string{commissioner.NotePost}, Modes: []string{"execute"}},
		tokenScope("admin"), tokenScope("read", "read"), tokenScope(),
		{League: "native.tokens", Franchise: "franchise.01", Commands: []string{commissioner.NotePost, commissioner.NotePost}, Modes: []string{"execute"}},
	} {
		if _, err := service.Issue(t.Context(), session.RawSession, scope); !errors.Is(err, accounts.ErrTokenRefused) {
			t.Fatalf("invalid scope: %+v %v", scope, err)
		}
	}
	noPolicy := accounts.NewTokens(f.store, nil)
	if _, err := noPolicy.Issue(t.Context(), session.RawSession, tokenScope("read")); !errors.Is(err, accounts.ErrTokenRefused) {
		t.Fatal("nil issuance policy allowed")
	}
	token := issueToken(t, service, session, tokenScope("read", "preview"))
	if err := noPolicy.Admission(token.RawSession).Admit(t.Context(), tokenAccess(token, "read")); !errors.Is(err, accounts.ErrTokenRefused) {
		t.Fatalf("nil admission policy: %v", err)
	}
	if err := service.Admission(token.RawSession).Admit(t.Context(), tokenAccess(token, "execute")); !errors.Is(err, accounts.ErrTokenRefused) {
		t.Fatalf("preview token executed: %v", err)
	}
	if _, err := service.Authenticate(t.Context(), accounts.RawSession{Bearer: "malformed"}); !errors.Is(err, accounts.ErrTokenRefused) {
		t.Fatal("malformed token accepted")
	}
	sessions := accounts.NewSessions(accounts.New(f.store), f.store)
	if err := sessions.Revoke(t.Context(), session.RawSession, false); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Issue(t.Context(), session.RawSession, tokenScope("read")); !errors.Is(err, accounts.ErrSessionRefused) {
		t.Fatalf("revoked device issued token: %v", err)
	}
}
