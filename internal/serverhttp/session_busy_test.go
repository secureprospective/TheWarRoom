package serverhttp

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
)

type sessionStoreStub struct{ err error }

func (s sessionStoreStub) CreateSession(context.Context, accounts.Credential, string, string) (time.Time, error) {
	return time.Time{}, s.err
}

func (s sessionStoreStub) SessionAccount(context.Context, string, bool) (accounts.Account, error) {
	return accounts.Account{}, s.err
}

func (s sessionStoreStub) RevokeSession(context.Context, string, bool) error { return s.err }

// A busy store is the server being busy, not a refused owner: only an unknown, expired or
// revoked session is 401.
func TestSessionAuthenticationMapsServerTroubleToBusy(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		status int
		body   string
	}{
		{"locked store", errors.New("database is locked"), http.StatusServiceUnavailable, BusyMessage},
		{"expired request", context.DeadlineExceeded, http.StatusServiceUnavailable, BusyMessage},
		{"refused session", accounts.ErrSessionRefused, http.StatusUnauthorized, "session refused"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := Hosted{Sessions: accounts.NewSessions(nil, sessionStoreStub{err: tc.err})}
			r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, LedgerPath, strings.NewReader(`{"protocol":1,"league":"native.test"}`))
			r.TLS = &tls.ConnectionState{}
			r.Header.Set("Authorization", "Bearer "+strings.Repeat("A", 43))
			w := httptest.NewRecorder()
			h.native(w, r)
			if w.Code != tc.status || strings.TrimSpace(w.Body.String()) != tc.body {
				t.Fatalf("%s: %d %q, want %d %q", tc.name, w.Code, w.Body.String(), tc.status, tc.body)
			}
			if tc.status == http.StatusServiceUnavailable && w.Header().Get("Retry-After") == "" {
				t.Fatalf("%s: missing Retry-After", tc.name)
			}
		})
	}
}
