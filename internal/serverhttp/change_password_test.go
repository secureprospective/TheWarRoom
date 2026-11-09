package serverhttp_test

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

func changePasswordBody(oldPassword, newPassword string) string {
	return fmt.Sprintf(`{"protocol":1,"username":"commissioner","oldPassword":%q,"newPassword":%q}`, oldPassword, newPassword)
}

// The password page is a same-origin form. The endpoint refuses a wrong current password, changes
// the password, and ends every device session issued before that change.
func TestPasswordChangeRefusesWrongOldAndEndsOldSessions(t *testing.T) {
	f := newHostedFixture(t)
	oldBearer := f.login(t, "commissioner", "private-password")

	status, header, body := f.get(t, serverhttp.PasswordPagePath)
	if status != http.StatusOK || !strings.Contains(header.Get("Content-Security-Policy"), "default-src 'none'") ||
		!bytes.Contains(body, []byte(`id="changeForm"`)) || !bytes.Contains(body, []byte(serverhttp.PasswordChangePath)) {
		t.Fatalf("password page: %d %v", status, header)
	}

	f.post(t, serverhttp.PasswordChangePath, changePasswordBody("wrong-password", "new-private-password"), "", "", http.StatusUnauthorized)
	if _, err := accounts.New(f.accounts).Authenticate(t.Context(), accounts.RawCredentials{Username: "commissioner", Password: "private-password"}); err != nil {
		t.Fatalf("wrong current password changed the password: %v", err)
	}

	f.post(t, serverhttp.PasswordChangePath, changePasswordBody("private-password", "new-private-password"), "", "", http.StatusOK)

	f.post(t, serverhttp.PeoplePath, `{"protocol":1,"league":"native.hosted"}`, oldBearer, "", http.StatusUnauthorized)
	f.post(t, serverhttp.LoginPath, `{"protocol":1,"username":"commissioner","password":"private-password","device":"test device"}`, "", "", http.StatusUnauthorized)
	if f.login(t, "commissioner", "new-private-password") == "" {
		t.Fatal("new password cannot sign in")
	}
}

// An account with a confirmed authenticator cannot change its password on the old password alone:
// the current code is required and is checked and consumed through the authenticator service.
func TestPasswordChangeRequiresAuthenticatorCodeWhenEnrolled(t *testing.T) {
	f := newHostedFixture(t)
	credentials := accounts.RawCredentials{Username: "commissioner", Password: "private-password"}
	setup, err := f.totp.Begin(t.Context(), credentials)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.totp.Confirm(t.Context(), credentials, accounts.RawTOTP{Code: testTOTP(t, setup.Seed, f.now)}); err != nil {
		t.Fatal(err)
	}
	f.now = f.now.Add(30 * time.Second)
	code := testTOTP(t, setup.Seed, f.now)
	wrong := "000000"
	if wrong == code {
		wrong = "111111"
	}
	withCode := func(value string) string {
		return fmt.Sprintf(`{"protocol":1,"username":"commissioner","oldPassword":"private-password","newPassword":"new-private-password","code":%q}`, value)
	}
	oldStillWorks := func(t *testing.T) {
		t.Helper()
		if _, err := accounts.New(f.accounts).Authenticate(t.Context(), credentials); err != nil {
			t.Fatalf("the password changed without a valid code: %v", err)
		}
	}

	f.post(t, serverhttp.PasswordChangePath, changePasswordBody("private-password", "new-private-password"), "", "", http.StatusForbidden)
	oldStillWorks(t)
	f.post(t, serverhttp.PasswordChangePath, withCode(wrong), "", "", http.StatusForbidden)
	oldStillWorks(t)
	f.post(t, serverhttp.PasswordChangePath, withCode(code), "", "", http.StatusOK)
	if _, err := accounts.New(f.accounts).Authenticate(t.Context(), credentials); err == nil {
		t.Fatal("the old password still works after the change")
	}
	if _, err := accounts.New(f.accounts).Authenticate(t.Context(), accounts.RawCredentials{Username: "commissioner", Password: "new-private-password"}); err != nil {
		t.Fatalf("the new password does not work: %v", err)
	}
}
