package serverhttp

import (
	"bytes"
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/db"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
)

// Password work that ran out of request time while queued is the server being busy, not a wrong
// password, code or phone clock: every password door answers 503 with plain wording.
func TestPasswordWorkTimeoutIsBusyNotRefused(t *testing.T) {
	pools, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "accounts.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	store := accountstore.New(pools)
	if err := store.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	passwords := accounts.New(store)
	if _, err := passwords.LocalBootstrap(t.Context(), accounts.RawCredentials{Username: "commissioner", Password: "private-password"}); err != nil {
		t.Fatal(err)
	}
	totp, err := accounts.NewTOTP(passwords, store, bytes.Repeat([]byte{7}, 32))
	if err != nil {
		t.Fatal(err)
	}
	h := Hosted{Sessions: accounts.NewSessions(passwords, store), TOTP: totp,
		Recovery: accounts.NewRecovery(passwords, store, [32]byte{}), Enrollment: accounts.NewEnrollment(store, nil)}
	code := strings.Repeat("A", 43)
	credentials := `"username":"commissioner","password":"private-password"`
	for path, body := range map[string]string{
		TOTPBeginPath:   `{"protocol":1,` + credentials + `}`,
		TOTPConfirmPath: `{"protocol":1,` + credentials + `,"code":"123456"}`,
		TOTPStepUpPath:  `{"protocol":1,` + credentials + `,"code":"123456"}`,
		LoginPath:       `{"protocol":1,` + credentials + `,"device":"laptop"}`,
		RecoveryPath:    `{"protocol":1,"username":"commissioner","code":"` + code + `","password":"new-password"}`,
		EnrollmentPath:  `{"protocol":1,"enrollment":{"code":"` + code + `","username":"newowner","password":"new-password"}}`,
	} {
		// The deadline the queued password work was waiting on has already passed.
		ctx, cancel := context.WithDeadline(t.Context(), time.Now().Add(-time.Second))
		r := httptest.NewRequestWithContext(ctx, http.MethodPost, path, strings.NewReader(body))
		r.TLS = &tls.ConnectionState{}
		w := httptest.NewRecorder()
		switch path {
		case LoginPath:
			h.login(w, r)
		case RecoveryPath:
			h.recoverAccount(w, r)
		case EnrollmentPath:
			EnrollmentHandler(h.Enrollment).ServeHTTP(w, r)
		default:
			h.authenticator(w, r)
		}
		cancel()
		if w.Code != http.StatusServiceUnavailable || strings.TrimSpace(w.Body.String()) != BusyMessage || w.Header().Get("Retry-After") == "" {
			t.Errorf("%s: %d %q, want 503 %q", path, w.Code, w.Body.String(), BusyMessage)
		}
	}
}
