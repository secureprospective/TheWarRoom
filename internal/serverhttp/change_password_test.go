package serverhttp_test

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"testing"

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
