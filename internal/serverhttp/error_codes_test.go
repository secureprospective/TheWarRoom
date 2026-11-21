package serverhttp_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func (b *lockedBuffer) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf.Reset()
}

func TestNewServerErrorCodeFormatAndValidation(t *testing.T) {
	seen := make(map[string]struct{}, 200)
	for range 200 {
		code := serverhttp.NewServerErrorCode()
		if len(code) != 13 || !strings.HasPrefix(code, "TWR-") {
			t.Fatalf("unexpected format: %q", code)
		}
		if strings.HasPrefix(code, "TWR-C") {
			t.Fatalf("server code collided with client prefix: %q", code)
		}
		if strings.ContainsAny(code[4:], "0O1I") {
			t.Fatalf("ambiguous character in %q", code)
		}
		if !serverhttp.ValidErrorCode(code) {
			t.Fatalf("generated code failed validation: %q", code)
		}
		seen[code] = struct{}{}
	}
	if len(seen) < 190 {
		t.Fatalf("expected unique codes, got %d/200", len(seen))
	}
	for _, valid := range []string{"TWR-C234-5678", "TWR-7K9M-2P4X"} {
		if !serverhttp.ValidErrorCode(valid) {
			t.Fatalf("ValidErrorCode(%q) = false, want true", valid)
		}
	}
	for _, bad := range []string{
		"", "TWR-0000-0000", "TWR-O234-5678", "TWR-1234-5678", "TWR-I234-5678",
		"twr-2345-6789", "TWR-234-5678", "TWR-2345-67890", "ERR-2345-6789",
	} {
		if serverhttp.ValidErrorCode(bad) {
			t.Fatalf("ValidErrorCode(%q) = true, want false", bad)
		}
	}
}

func TestHostedFailuresEmitErrorCodeHeaderAndStructuredLog(t *testing.T) {
	f := newHostedFixture(t, true)
	var logs lockedBuffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	hosted := serverhttp.Hosted{
		Publisher: f.publisher, LiveGames: f.games, Tokens: f.tokens,
		Store: f.history, Leagues: history.NewOneFile(f.history),
		Sessions: f.sessions, AccountPreferences: f.accounts,
		TOTP: f.totp, Enrollment: f.enrollment, Worker: f.worker,
		Join: f.join, ErrorLog: logger, PublicHost: "league.example.test",
	}
	server := httptest.NewTLSServer(hosted.Handler())
	defer server.Close()

	doReq := func(method, path, body, bearer string) (int, string, string) {
		t.Helper()
		req, err := http.NewRequestWithContext(
			t.Context(), method, server.URL+path, strings.NewReader(body),
		)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		if bearer != "" {
			req.Header.Set("Authorization", "Bearer "+bearer)
		}
		resp, err := server.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = resp.Body.Close() }()
		raw, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		return resp.StatusCode, resp.Header.Get(serverhttp.ErrorCodeHeader), string(raw)
	}

	// 1. Failed login: emits code header + structured log without password/username.
	secretPass := "super-secret-password-987"
	loginBody := `{"protocol":1,"username":"owner","password":"` + secretPass +
		`","device":"phone"}`
	status, code, _ := doReq(http.MethodPost, serverhttp.LoginPath, loginBody, "")
	if status != http.StatusUnauthorized || !serverhttp.ValidErrorCode(code) ||
		strings.HasPrefix(code, "TWR-C") {
		t.Fatalf("failed login: status=%d code=%q", status, code)
	}
	line := strings.TrimSpace(logs.String())
	if strings.Count(line, "\n") != 0 || !strings.Contains(line, "time=") ||
		!strings.Contains(line, "code="+code) ||
		!strings.Contains(line, "route="+serverhttp.LoginPath) {
		t.Fatalf("unexpected login log line: %q", line)
	}
	if strings.Contains(line, secretPass) {
		t.Fatalf("password leaked into error log: %q", line)
	}

	// 2. Successful login + authenticated workflow refusal logs command, league, account.
	logs.Reset()
	okLogin := `{"protocol":1,"username":"owner","password":"private-password",` +
		`"device":"phone"}`
	status, okCode, loginRaw := doReq(http.MethodPost, serverhttp.LoginPath, okLogin, "")
	if status != http.StatusOK || okCode != "" || logs.String() != "" {
		t.Fatalf("successful login: status=%d code=%q logs=%q", status, okCode, logs.String())
	}
	var session serverhttp.LoginResponse
	if err := json.Unmarshal([]byte(loginRaw), &session); err != nil {
		t.Fatal(err)
	}
	wf := serverhttp.RawWorkflow{
		Command: commissioner.ProblemReportsRead, Version: "1", Mode: "read",
		Arguments: json.RawMessage(`{"league":"native.hosted"}`),
	}
	wfBody, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &wf})
	if err != nil {
		t.Fatal(err)
	}
	status, wfCode, _ := doReq(
		http.MethodPost, serverhttp.WorkflowPath, string(wfBody), session.Bearer,
	)
	if status != http.StatusConflict || !serverhttp.ValidErrorCode(wfCode) ||
		strings.HasPrefix(wfCode, "TWR-C") {
		t.Fatalf("refused workflow: status=%d code=%q", status, wfCode)
	}
	wfLog := strings.TrimSpace(logs.String())
	for _, want := range []string{
		"time=", "code=" + wfCode, "route=" + serverhttp.WorkflowPath,
		"command=" + commissioner.ProblemReportsRead,
		"league=native.hosted", "account=" + session.Session.Account.ID,
	} {
		if !strings.Contains(wfLog, want) {
			t.Fatalf("workflow log missing %q: %q", want, wfLog)
		}
	}
	if strings.Contains(wfLog, session.Bearer) {
		t.Fatalf("bearer leaked into workflow log: %q", wfLog)
	}

	// 3. Unknown route never logs arbitrary URL text.
	logs.Reset()
	status, nfCode, _ := doReq(http.MethodGet, "/secret-join-code-xyz", "", "")
	if status != http.StatusNotFound || !serverhttp.ValidErrorCode(nfCode) {
		t.Fatalf("not found: status=%d code=%q", status, nfCode)
	}
	nfLog := strings.TrimSpace(logs.String())
	if !strings.Contains(nfLog, "route=/other") ||
		strings.Contains(nfLog, "secret-join-code-xyz") {
		t.Fatalf("unknown route log unsafe: %q", nfLog)
	}

	// 4. Unvalidated league header never enters the server log.
	logs.Reset()
	invReq, err := http.NewRequestWithContext(
		t.Context(), http.MethodGet, server.URL+serverhttp.InvalidationPath, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	invReq.Header.Set("X-TWR-League", "native.bad league!")
	invResp, err := server.Client().Do(invReq)
	if err != nil {
		t.Fatal(err)
	}
	_ = invResp.Body.Close()
	if invResp.StatusCode < http.StatusBadRequest {
		t.Fatalf("invalid invalidation: status=%d", invResp.StatusCode)
	}
	invLog := strings.TrimSpace(logs.String())
	if strings.Contains(invLog, "native.bad") || !strings.Contains(invLog, `league=""`) {
		t.Fatalf("unvalidated league logged: %q", invLog)
	}

	// 5. Failed /v1/auth/agent-token logs validated scope league and account.
	logs.Reset()
	scopeBody := `{"protocol":1,"scope":{"league":"native.hosted","franchise":"franchise.01",` +
		`"commands":["` + commissioner.NotePost + `","commish.ledger.read"],"modes":["preview","read"]}}`
	status, agCode, _ := doReq(
		http.MethodPost, serverhttp.AgentTokenPath, scopeBody, session.Bearer,
	)
	if status != http.StatusForbidden || !serverhttp.ValidErrorCode(agCode) {
		t.Fatalf("agent token refusal: status=%d code=%q", status, agCode)
	}
	agLog := strings.TrimSpace(logs.String())
	if !strings.Contains(agLog, "league=native.hosted") ||
		!strings.Contains(agLog, "account="+session.Session.Account.ID) {
		t.Fatalf("agent token log missing league/account: %q", agLog)
	}

	// 6. Early agent request refusal (400 malformed JSON) logs authenticated account.
	logs.Reset()
	commishBearer, _ := f.directCommissioner(t)
	commishSession := accounts.RawSession{Bearer: commishBearer}
	commishAcct, err := f.sessions.Authenticate(t.Context(), commishSession)
	if err != nil {
		t.Fatal(err)
	}
	issued, err := f.tokens.Issue(
		t.Context(),
		commishSession,
		accounts.RawTokenScope{
			League:    "native.hosted",
			Franchise: "franchise.01",
			Commands:  []string{commissioner.NotePost},
			Modes:     []string{"preview"},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	status, badAgCode, _ := doReq(
		http.MethodPost, serverhttp.WorkflowPath, "{", issued.Bearer,
	)
	if status != http.StatusBadRequest || !serverhttp.ValidErrorCode(badAgCode) {
		t.Fatalf("early agent refusal: status=%d code=%q", status, badAgCode)
	}
	badAgLog := strings.TrimSpace(logs.String())
	if !strings.Contains(badAgLog, "account="+commishAcct.ID) {
		t.Fatalf("early agent log missing account: %q", badAgLog)
	}
}
