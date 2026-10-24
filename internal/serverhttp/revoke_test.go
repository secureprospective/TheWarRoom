package serverhttp_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

func TestRevokeBoundaryPreservesOtherSessions(t *testing.T) {
	f := newHostedFixture(t)
	bearer := f.login(t, "commissioner", "private-password")
	for _, tc := range []struct {
		body, bearer, content string
		status                int
	}{
		{`{"protocol":1,"allDevices":true}`, "", "application/json", 401},
		{`{"protocol":1,"allDevices":true}`, "invalid", "application/json", 401},
		{`{"protocol":2,"allDevices":true}`, bearer, "application/json", 400},
		{`{"protocol":1,"allDevices":true,"account":"somebody-else"}`, bearer, "application/json", 400},
		{`{"protocol":1} {}`, bearer, "application/json", 400},
		{`{"protocol":1}`, bearer, "text/plain", 400},
	} {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, f.server.URL+serverhttp.RevokePath, strings.NewReader(tc.body))
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("Content-Type", tc.content)
		if tc.bearer != "" {
			request.Header.Set("Authorization", "Bearer "+tc.bearer)
		}
		response, err := f.server.Client().Do(request)
		if err != nil {
			t.Fatal(err)
		}
		if err := response.Body.Close(); err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != tc.status {
			t.Fatalf("revoke status %d want %d", response.StatusCode, tc.status)
		}
		if _, err := f.sessions.Authenticate(t.Context(), accounts.RawSession{Bearer: bearer}); err != nil {
			t.Fatal("denial revoked session", err)
		}
	}
	f.revision(t, 1)
}
