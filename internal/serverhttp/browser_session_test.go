package serverhttp_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

const browserOrigin = "https://league.example.test"

type browserResponse struct {
	StatusCode int
	Header     http.Header
	cookies    []*http.Cookie
}

func (r browserResponse) Cookies() []*http.Cookie { return r.cookies }

func browserRequest(t *testing.T, f *hostedFixture, method, path, body, cookie, authorization string, origins ...string) (browserResponse, []byte) {
	t.Helper()
	request, err := http.NewRequestWithContext(t.Context(), method, f.server.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Host = "league.example.test"
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	for _, origin := range origins {
		request.Header.Add("Origin", origin)
	}
	if cookie != "" {
		request.AddCookie(&http.Cookie{Name: "__Host-twr_session", Value: cookie, Path: "/",
			Secure: true, HttpOnly: true, SameSite: http.SameSiteStrictMode})
	}
	if authorization != "" {
		request.Header.Set("Authorization", authorization)
	}
	response, err := f.server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	result, err := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if response.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("%s %s: cacheable response", method, path)
	}
	return browserResponse{StatusCode: response.StatusCode, Header: response.Header.Clone(), cookies: response.Cookies()}, result
}

func appLogin(t *testing.T, f *hostedFixture, username, password, code string) (serverhttp.BrowserSessionResponse, *http.Cookie) {
	t.Helper()
	//nolint:gosec // The test sends credentials only to the local HTTPS fixture.
	body, err := json.Marshal(serverhttp.RawBrowserLogin{Protocol: 1, Username: username, Password: password, Code: code})
	if err != nil {
		t.Fatal(err)
	}
	response, result := browserRequest(t, f, http.MethodPost, serverhttp.AppSessionPath, string(body), "", "", browserOrigin)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("browser login: status=%d body=%s", response.StatusCode, result)
	}
	var session serverhttp.BrowserSessionResponse
	if err := json.Unmarshal(result, &session); err != nil {
		t.Fatal(err)
	}
	if session.Protocol != 1 || session.Admission.League != "native.hosted" || len(session.Leagues) == 0 {
		t.Fatalf("browser admission = %+v", session)
	}
	if bytes.Contains(result, []byte("bearer")) || bytes.Contains(result, []byte("session")) {
		t.Fatalf("browser response contains credential material: %s", result)
	}
	var cookie *http.Cookie
	for _, candidate := range response.Cookies() {
		if candidate.Name == "__Host-twr_session" {
			if cookie != nil {
				t.Fatal("browser login returned duplicate session cookies")
			}
			cookie = candidate
		}
	}
	if cookie == nil || cookie.Value == "" || !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteStrictMode || cookie.Path != "/" || cookie.Domain != "" || !cookie.Expires.After(time.Now()) {
		t.Fatalf("session cookie flags = %+v", cookie)
	}
	return session, cookie
}

func TestBrowserSessionCookieLifecycleAndNativeRoute(t *testing.T) {
	f := newHostedFixture(t)
	admission, cookie := appLogin(t, f, "commissioner", "private-password", "")
	if admission.Admission.Role != "Commissioner" || admission.Admission.Name != "Hosted test" || len(admission.Admission.Seats) != 2 ||
		len(admission.Leagues) != 1 || admission.Leagues[0].ID != "native.hosted" ||
		admission.Leagues[0].Role != "Commissioner" || admission.Leagues[0].Franchise != admission.Admission.Franchise {
		t.Fatalf("admission = %+v", admission)
	}
	response, body := browserRequest(t, f, http.MethodGet, serverhttp.AppSessionPath, "", cookie.Value, "", browserOrigin)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("GET app session: status=%d body=%s", response.StatusCode, body)
	}
	var current serverhttp.BrowserSessionResponse
	if err := json.Unmarshal(body, &current); err != nil || !reflect.DeepEqual(current, admission) {
		t.Fatalf("GET app session = %+v, err=%v", current, err)
	}
	response, body = browserRequest(t, f, http.MethodPost, serverhttp.LedgerPath,
		`{"protocol":1,"league":"native.hosted"}`, cookie.Value, "", browserOrigin)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("cookie ledger request: status=%d body=%s", response.StatusCode, body)
	}
	response, _ = browserRequest(t, f, http.MethodDelete, serverhttp.AppSessionPath, "", cookie.Value, "", browserOrigin)
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("DELETE app session: status=%d", response.StatusCode)
	}
	cleared := false
	for _, result := range response.Cookies() {
		if result.Name == cookie.Name && result.MaxAge < 0 && result.HttpOnly && result.Secure && result.SameSite == http.SameSiteStrictMode && result.Path == "/" {
			cleared = true
		}
	}
	if !cleared {
		t.Fatalf("sign-out did not clear cookie safely: %v", response.Cookies())
	}
	response, body = browserRequest(t, f, http.MethodGet, serverhttp.AppSessionPath, "", cookie.Value, "", browserOrigin)
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("replayed session cookie: status=%d body=%s", response.StatusCode, body)
	}
}

func TestBrowserSessionGETAllowsMissingOriginButRejectsCrossSite(t *testing.T) {
	f := newHostedFixture(t)
	_, cookie := appLogin(t, f, "commissioner", "private-password", "")

	response, body := browserRequest(t, f, http.MethodGet, serverhttp.AppSessionPath, "", cookie.Value, "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("GET without Origin: status=%d body=%s", response.StatusCode, body)
	}

	for _, tc := range []struct {
		name   string
		origin string
		fetch  string
		method string
		want   int
	}{
		{name: "cross-site fetch", fetch: "cross-site", method: http.MethodGet, want: http.StatusForbidden},
		{name: "wrong origin", origin: "https://elsewhere.example", method: http.MethodGet, want: http.StatusForbidden},
		{name: "login without origin", method: http.MethodPost, want: http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request, err := http.NewRequestWithContext(t.Context(), tc.method, f.server.URL+serverhttp.AppSessionPath, nil)
			if err != nil {
				t.Fatal(err)
			}
			request.Host = "league.example.test"
			if tc.origin != "" {
				request.Header.Set("Origin", tc.origin)
			}
			if tc.fetch != "" {
				request.Header.Set("Sec-Fetch-Site", tc.fetch)
			}
			request.AddCookie(&http.Cookie{Name: "__Host-twr_session", Value: cookie.Value, Path: "/",
				Secure: true, HttpOnly: true, SameSite: http.SameSiteStrictMode})
			result, err := f.server.Client().Do(request)
			if err != nil {
				t.Fatal(err)
			}
			_ = result.Body.Close()
			if result.StatusCode != tc.want {
				t.Fatalf("status=%d, want %d", result.StatusCode, tc.want)
			}
		})
	}
}

func TestBrowserCookieOriginAndCredentialAmbiguityRefused(t *testing.T) {
	f := newHostedFixture(t)
	_, cookie := appLogin(t, f, "commissioner", "private-password", "")
	ledger := `{"protocol":1,"league":"native.hosted"}`
	for _, origins := range [][]string{{"https://elsewhere.example"}, {"null"}, {"http://league.example.test"}, {"https://*.example.test"}, nil, {browserOrigin, browserOrigin}} {
		response, _ := browserRequest(t, f, http.MethodPost, serverhttp.LedgerPath, ledger, cookie.Value, "", origins...)
		if response.StatusCode != http.StatusForbidden {
			t.Errorf("cookie request Origin=%q: status=%d, want 403", origins, response.StatusCode)
		}
	}
	response, _ := browserRequest(t, f, http.MethodPost, serverhttp.LedgerPath, ledger, cookie.Value,
		"Bearer "+cookie.Value, browserOrigin)
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("Authorization plus cookie status=%d, want 403", response.StatusCode)
	}
	f.revision(t, 1)
}

func TestBrowserAgentTokenCannotUseOriginAndCookiePermissionStaysEnforced(t *testing.T) {
	f := newHostedFixture(t, true)
	human, proof := f.directCommissioner(t)
	scope := `{"protocol":1,"scope":{"league":"native.hosted","franchise":"franchise.01","commands":["` + commissioner.NotePost + `","commish.ledger.read"],"modes":["preview","read"]}}`
	body := f.post(t, serverhttp.AgentTokenPath, scope, human, proof, http.StatusOK)
	var token serverhttp.AgentTokenResponse
	if err := json.Unmarshal(body, &token); err != nil || token.Bearer == "" {
		t.Fatalf("agent token response = %+v, err=%v", token, err)
	}
	response, _ := browserRequest(t, f, http.MethodPost, serverhttp.LedgerPath,
		`{"protocol":1,"league":"native.hosted"}`, "", "Bearer "+token.Bearer, browserOrigin)
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("agent bearer with Origin status=%d, want 403", response.StatusCode)
	}
	response, _ = browserRequest(t, f, http.MethodPost, serverhttp.LedgerPath,
		`{"protocol":1,"league":"native.hosted"}`, token.Bearer, "", browserOrigin)
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("agent bearer as browser cookie status=%d, want 401", response.StatusCode)
	}

	_, cookie := appLogin(t, f, "owner", "private-password", "")
	expected, err := f.history.CommittedHead(t.Context(), "native.hosted", commissioner.Stream)
	if err != nil {
		t.Fatal(err)
	}
	command := commissioner.RawCommand{League: "native.hosted", Command: commissioner.NotePost, Request: "browser-owner-refusal",
		Expected: expected, Reason: "test refusal", Effective: "now", Visibility: "league",
		Args: commissioner.RawArgs{Note: &commissioner.RawNote{Text: "must not be written"}}}
	raw := serverhttp.RawNativeRequest{Protocol: 1, Command: &command}
	payload, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	response, body = browserRequest(t, f, http.MethodPost, serverhttp.PreviewPath, string(payload), cookie.Value, "", browserOrigin)
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("owner preview of commissioner command: status=%d body=%s", response.StatusCode, body)
	}
	after, err := f.history.CommittedHead(t.Context(), "native.hosted", commissioner.Stream)
	if err != nil || after != expected {
		t.Fatalf("denied cookie request changed league state: before=%d after=%d err=%v", expected, after, err)
	}
}

func TestBrowserSessionListsOwnMembershipsAndAllowsSwitching(t *testing.T) {
	f := newHostedFixture(t)
	create := commissioner.RawCommand{League: "native.second", Command: commissioner.Create,
		Args:   commissioner.RawArgs{Name: "Second league", Preset: commissioner.PresetEmpty, Season: 2026, FranchiseCount: 2},
		Reason: "Create second test league", Request: "second-league", Effective: "now", Visibility: "league"}
	if _, err := f.history.ExecuteCommissioner(t.Context(), create, f.principal, nil); err != nil {
		t.Fatal(err)
	}

	session, cookie := appLogin(t, f, "commissioner", "private-password", "")
	if len(session.Leagues) != 2 || session.Leagues[0].ID != "native.hosted" || session.Leagues[1].ID != "native.second" ||
		session.Leagues[1].Name != "Second league" || session.Leagues[1].Role != "Commissioner" {
		t.Fatalf("own league memberships = %+v", session.Leagues)
	}
	response, body := browserRequest(t, f, http.MethodGet, serverhttp.AppSessionPath, "", cookie.Value, "", browserOrigin)
	var current serverhttp.BrowserSessionResponse
	if response.StatusCode != http.StatusOK || json.Unmarshal(body, &current) != nil ||
		!reflect.DeepEqual(current.Leagues, session.Leagues) {
		t.Fatalf("GET multi-league session status=%d memberships=%+v body=%s", response.StatusCode, current.Leagues, body)
	}

	response, body = browserRequest(t, f, http.MethodPost, serverhttp.LedgerPath,
		`{"protocol":1,"league":"native.second"}`, cookie.Value, "", browserOrigin)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("read selected second league status=%d body=%s", response.StatusCode, body)
	}
	var active int
	if err := f.pools.Read().QueryRowContext(t.Context(), `SELECT count(*) FROM account_sessions WHERE revoked = 0`).Scan(&active); err != nil {
		t.Fatal(err)
	}
	if active != 1 {
		t.Fatalf("multi-league sign-in left %d active sessions", active)
	}
}

func TestBrowserSessionNeverListsAnotherAccountsLeague(t *testing.T) {
	f := newHostedFixture(t)
	invitation, err := f.enrollment.Issue(t.Context(), "native.hosted", "other-owner")
	if err != nil {
		t.Fatal(err)
	}
	other, err := f.enrollment.Redeem(t.Context(), accounts.RawEnrollment{
		Code: invitation.Code, Username: "other-owner", Password: "private-password",
	})
	if err != nil {
		t.Fatal(err)
	}
	otherPrincipal, err := other.Principal()
	if err != nil {
		t.Fatal(err)
	}
	create := commissioner.RawCommand{League: "native.private", Command: commissioner.Create,
		Args:   commissioner.RawArgs{Name: "Other owner's league", Preset: commissioner.PresetEmpty, Season: 2026, FranchiseCount: 2},
		Reason: "Create another account's league", Request: "other-account-league", Effective: "now", Visibility: "league"}
	if _, err := f.history.ExecuteCommissioner(t.Context(), create, string(otherPrincipal), nil); err != nil {
		t.Fatal(err)
	}

	session, _ := appLogin(t, f, "commissioner", "private-password", "")
	if len(session.Leagues) != 1 || session.Leagues[0].ID != "native.hosted" {
		t.Fatalf("listed a league without this account's membership: %+v", session.Leagues)
	}
}

func TestBrowserPrivilegedAuthenticatorProofStaysServerSide(t *testing.T) {
	f := newHostedFixture(t)
	credentials := accounts.RawCredentials{Username: "commissioner", Password: "private-password"}
	setup, err := f.totp.Begin(t.Context(), credentials)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.totp.Confirm(t.Context(), credentials, accounts.RawTOTP{Code: testTOTP(t, setup.Seed, f.clockTime())}); err != nil {
		t.Fatal(err)
	}
	f.advanceTime(30 * time.Second)
	code := testTOTP(t, setup.Seed, f.clockTime())
	_, cookie := appLogin(t, f, credentials.Username, credentials.Password, code)
	response, body := browserRequest(t, f, http.MethodPost, serverhttp.PeoplePath,
		`{"protocol":1,"league":"native.hosted"}`, cookie.Value, "", browserOrigin)
	if response.StatusCode != http.StatusOK || bytes.Contains(body, []byte("recent_auth")) || bytes.Contains(body, []byte("bearer")) {
		t.Fatalf("privileged cookie read status=%d body=%s", response.StatusCode, body)
	}
}

func TestBrowserAppIsOnlyServedOnConfiguredHost(t *testing.T) {
	f := newHostedFixture(t)
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, f.server.URL+serverhttp.AppPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Host = "other.example.test"
	response, err := f.server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("app request on another Host: status=%d", response.StatusCode)
	}
}

func TestBrowserSessionNeedsExactOriginAndCookie(t *testing.T) {
	f := newHostedFixture(t)
	for _, tc := range []struct {
		name   string
		origin string
		cookie string
		status int
	}{
		{name: "missing origin with invalid cookie", cookie: "some-token", status: http.StatusUnauthorized},
		{name: "other origin", origin: "https://evil.example", status: http.StatusForbidden},
		{name: "http origin", origin: "http://league.example.test", status: http.StatusForbidden},
		{name: "null origin", origin: "null", status: http.StatusForbidden},
		{name: "missing cookie", origin: browserOrigin, status: http.StatusUnauthorized},
		{name: "bad cookie", origin: browserOrigin, cookie: "bad", status: http.StatusUnauthorized},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var origins []string
			if tc.origin != "" {
				origins = []string{tc.origin}
			}
			response, _ := browserRequest(t, f, http.MethodGet, serverhttp.AppSessionPath, "", tc.cookie, "", origins...)
			if response.StatusCode != tc.status {
				t.Fatalf("GET app session status=%d want=%d", response.StatusCode, tc.status)
			}
		})
	}
}

func TestBrowserSessionUsesADeviceSessionRecord(t *testing.T) {
	f := newHostedFixture(t)
	_, _ = appLogin(t, f, "commissioner", "private-password", "")
	var device string
	if err := f.pools.Read().QueryRowContext(t.Context(), `SELECT device FROM account_sessions WHERE revoked = 0 LIMIT 1`).Scan(&device); err != nil {
		t.Fatal(err)
	}
	if device != "The War Room browser" {
		t.Fatalf("session device = %q", device)
	}
}
