package serverhttp_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
)

type enrollmentAuthority struct {
	policy    governance.State
	principal governance.Principal
}

func (a enrollmentAuthority) EnrollmentPolicy(context.Context, string) (governance.State, governance.Principal, error) {
	return a.policy, a.principal, nil
}

func TestEnrollmentHTTPSRoute(t *testing.T) {
	pools, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "server.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = pools.Close() }()
	store := accountstore.New(pools)
	if err := store.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	issuer, err := accounts.New(store).LocalBootstrap(t.Context(), accounts.RawCredentials{Username: "commissioner", Password: "issuer-password"})
	if err != nil {
		t.Fatal(err)
	}
	principal, err := issuer.Principal()
	if err != nil {
		t.Fatal(err)
	}
	policy, err := governance.Apply(governance.State{}, governance.LeagueCreated{Creator: principal, Franchises: []governance.FranchiseID{"001"}})
	if err != nil {
		t.Fatal(err)
	}
	service := accounts.NewEnrollment(store, enrollmentAuthority{policy: policy, principal: principal})
	code, err := service.Issue(t.Context(), "native.test", "new.owner")
	if err != nil {
		t.Fatal(err)
	}
	handler := serverhttp.EnrollmentHandler(service)
	server := httptest.NewTLSServer(handler)
	defer server.Close()
	client := server.Client()
	payload := `{"protocol":1,"enrollment":{"code":"` + code.Code + `","username":"new.owner","password":"sentinel-new-password"}}`
	for _, tc := range []struct {
		name, method, path, body, origin string
		status                           int
	}{
		{name: "method", method: "GET", path: serverhttp.EnrollmentPath, status: 405},
		{name: "no issuance route", method: "POST", path: "/v1/auth/issue", body: payload, status: 404},
		{name: "protocol", method: "POST", path: serverhttp.EnrollmentPath, body: strings.Replace(payload, `"protocol":1`, `"protocol":2`, 1), status: 400},
		{name: "spoofed principal", method: "POST", path: serverhttp.EnrollmentPath, body: strings.Replace(payload, `"protocol":1`, `"principal":"creator","protocol":1`, 1), status: 400},
		{name: "seat grant", method: "POST", path: serverhttp.EnrollmentPath, body: strings.Replace(payload, `"username"`, `"franchise":"001","username"`, 1), status: 400},
		{name: "trailing JSON", method: "POST", path: serverhttp.EnrollmentPath, body: payload + `{}`, status: 400},
		{name: "oversize", method: "POST", path: serverhttp.EnrollmentPath, body: strings.Repeat(" ", 8192) + payload, status: 400},
		{name: "origin", method: "POST", path: serverhttp.EnrollmentPath, body: payload, origin: "https://untrusted.example", status: 400},
		{name: "query", method: "POST", path: serverhttp.EnrollmentPath + "?code=secret", body: payload, status: 400},
		{name: "wrong identity", method: "POST", path: serverhttp.EnrollmentPath, body: strings.Replace(payload, "new.owner", "wrong.owner", 1), status: 403},
		{name: "redeem", method: "POST", path: serverhttp.EnrollmentPath, body: payload, status: 201},
		{name: "replay", method: "POST", path: serverhttp.EnrollmentPath, body: payload, status: 403},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request, err := http.NewRequestWithContext(t.Context(), tc.method, server.URL+tc.path, strings.NewReader(tc.body))
			if err != nil {
				t.Fatal(err)
			}
			request.Header.Set("Origin", tc.origin)
			response, err := client.Do(request)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = response.Body.Close() }()
			body, err := io.ReadAll(response.Body)
			if err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != tc.status || response.Header.Get("Cache-Control") != "no-store" {
				t.Fatalf("status=%d body=%s", response.StatusCode, body)
			}
			if strings.Contains(string(body), code.Code) || strings.Contains(string(body), "sentinel-new-password") {
				t.Fatal("response leaks secrets")
			}
			if tc.status == 201 {
				var result struct {
					Protocol int
					Account  accounts.Account
				}
				if err := json.Unmarshal(body, &result); err != nil || result.Protocol != 1 || result.Account.Username != "new.owner" {
					t.Fatalf("bad account response: %v", err)
				}
				got, err := accounts.New(store).Authenticate(t.Context(), accounts.RawCredentials{Username: "new.owner", Password: "sentinel-new-password"})
				if err != nil || got != result.Account {
					t.Fatalf("enrolled account cannot log in: %v", err)
				}
			}
		})
	}
	raceCode, err := service.Issue(t.Context(), "native.test", "race.owner")
	if err != nil {
		t.Fatal(err)
	}
	racePayload := strings.ReplaceAll(strings.ReplaceAll(payload, code.Code, raceCode.Code), "new.owner", "race.owner")
	results := make(chan int, 3)
	var wait sync.WaitGroup
	for range 3 {
		wait.Go(func() {
			request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, server.URL+serverhttp.EnrollmentPath, strings.NewReader(racePayload))
			if err != nil {
				t.Error(err)
				results <- 0
				return
			}
			response, err := client.Do(request)
			if err != nil {
				t.Error(err)
				results <- 0
				return
			}
			defer func() { _ = response.Body.Close() }()
			results <- response.StatusCode
		})
	}
	wait.Wait()
	close(results)
	winners := 0
	for status := range results {
		if status == http.StatusCreated {
			winners++
		} else if status != http.StatusForbidden {
			t.Fatalf("concurrent status: %d", status)
		}
	}
	if winners != 1 {
		t.Fatalf("concurrent HTTPS redemption winners: %d", winners)
	}
	plain := httptest.NewRecorder()
	handler.ServeHTTP(plain, httptest.NewRequest(http.MethodPost, "http://localhost"+serverhttp.EnrollmentPath, strings.NewReader(payload)))
	if plain.Code != http.StatusBadRequest {
		t.Fatal("accepted plaintext HTTP")
	}
}
