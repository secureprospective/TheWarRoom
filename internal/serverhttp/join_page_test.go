package serverhttp_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

func (f *hostedFixture) get(t *testing.T, path string) (int, http.Header, []byte) {
	t.Helper()
	response, err := f.server.Client().Get(f.server.URL + path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return response.StatusCode, response.Header.Clone(), body
}

func (f *hostedFixture) sendOrigin(t *testing.T, path, body string, origins []string) (int, []byte) {
	t.Helper()
	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, f.server.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header["Origin"] = origins
	response, err := f.server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	result, err := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	return response.StatusCode, result
}

// The join page is served from the server itself with a strict nonce CSP, no external asset and no
// team name baked into it: every name comes from the join API.
func TestJoinPageServedWithStrictCSPAndNoHardcodedTeams(t *testing.T) {
	f := newHostedFixture(t, true)
	status, header, body := f.get(t, serverhttp.JoinPagePath)
	if status != http.StatusOK {
		t.Fatalf("status=%d body=%s", status, body)
	}
	if header.Get("Content-Type") != "text/html; charset=utf-8" ||
		header.Get("X-Content-Type-Options") != "nosniff" ||
		header.Get("Cache-Control") != "no-store" {
		t.Fatalf("page headers: %v", header)
	}
	csp := header.Get("Content-Security-Policy")
	nonce := regexp.MustCompile(`'nonce-([^']+)'`).FindStringSubmatch(csp)
	if !strings.Contains(csp, "default-src 'none'") || !strings.Contains(csp, "connect-src 'self'") || nonce == nil {
		t.Fatalf("csp = %q", csp)
	}
	if !bytes.Contains(body, []byte(`nonce="`+nonce[1]+`"`)) || bytes.Contains(body, []byte("__CSP_NONCE__")) {
		t.Fatal("page has no matching nonce or kept the placeholder")
	}
	for _, placeholder := range []string{"Franchise ", "Franchise 05", "Franchise NN"} {
		if bytes.Contains(body, []byte(placeholder)) {
			t.Fatalf("page hard-codes a team name %q", placeholder)
		}
	}
	for _, external := range []string{"http://", "https://", "src=\"//", "href=\"//"} {
		if bytes.Contains(body, []byte(external)) {
			t.Fatalf("page loads an external asset %q", external)
		}
	}
	for _, route := range []string{serverhttp.JoinFranchisesPath, serverhttp.JoinClaimPath} {
		if !bytes.Contains(body, []byte(route)) {
			t.Fatalf("page does not call %s", route)
		}
	}
	if bytes.Contains(body, []byte("innerHTML")) {
		t.Fatal("page script uses innerHTML")
	}
	if !bytes.Contains(body, []byte("Taken by ")) {
		t.Fatal("page does not show who claimed a team")
	}
	if status, _, _ := f.get(t, serverhttp.JoinPagePath+"?x=1"); status != http.StatusBadRequest {
		t.Fatalf("join page accepted a query: %d", status)
	}
}

// A browser POST to the join doors is allowed only when it comes from this server's own origin.
func TestJoinDoorsAllowOnlySameOriginBrowser(t *testing.T) {
	f := newHostedFixture(t, true)
	code, err := f.join.Rotate(t.Context(), "native.hosted", governance.Principal(f.principal))
	if err != nil {
		t.Fatal(err)
	}
	body := joinBody(code.Code, "", "")
	for _, tc := range []struct {
		name    string
		origins []string
		status  int
	}{
		{"same origin", []string{f.server.URL}, http.StatusOK},
		{"no origin", nil, http.StatusOK},
		{"other origin", []string{"https://evil.test"}, http.StatusBadRequest},
		{"null origin", []string{"null"}, http.StatusBadRequest},
		{"two origins", []string{f.server.URL, "https://evil.test"}, http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, result := f.sendOrigin(t, serverhttp.JoinFranchisesPath, body, tc.origins)
			if status != tc.status {
				t.Fatalf("status=%d want=%d body=%s", status, tc.status, result)
			}
		})
	}
	// The same exception must not open any other account route to a browser.
	if status, _ := f.sendOrigin(t, serverhttp.LoginPath, `{"protocol":1,"username":"commissioner","password":"private-password","device":"browser"}`, []string{f.server.URL}); status != http.StatusBadRequest {
		t.Fatalf("browser login status=%d", status)
	}
}

func TestJoinClaimIssuesRecoveryCodesOnce(t *testing.T) {
	f := newHostedFixture(t, true)
	code, err := f.join.Rotate(t.Context(), "native.hosted", governance.Principal(f.principal))
	if err != nil {
		t.Fatal(err)
	}
	var claim serverhttp.JoinClaimResponse
	if err := json.Unmarshal(f.post(t, serverhttp.JoinClaimPath, joinBody(code.Code, "franchise.04", "codes.owner"), "", "", http.StatusCreated), &claim); err != nil {
		t.Fatal(err)
	}
	if len(claim.RecoveryCodes) != accounts.RecoveryCodeCount {
		t.Fatalf("codes=%d want=%d", len(claim.RecoveryCodes), accounts.RecoveryCodeCount)
	}
	seen := map[string]bool{}
	for _, value := range claim.RecoveryCodes {
		if len(value) != 43 || seen[value] {
			t.Fatalf("bad recovery code %q", value)
		}
		seen[value] = true
	}
	var stored int
	if err := f.pools.Read().QueryRowContext(t.Context(), `SELECT count(*) FROM account_recovery_codes WHERE account_id = ?`, claim.Account.ID).Scan(&stored); err != nil || stored != accounts.RecoveryCodeCount {
		t.Fatalf("stored codes=%d: %v", stored, err)
	}
	f.post(t, serverhttp.JoinClaimPath, joinBody(code.Code, "franchise.04", "codes.owner"), "", "", http.StatusConflict)
}
