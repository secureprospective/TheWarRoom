package serverhttp_test

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

func recoveryCode(t *testing.T, f *hostedFixture) string {
	t.Helper()
	credential, err := f.accounts.Credential(t.Context(), "commissioner")
	if err != nil {
		t.Fatal(err)
	}
	// Install issued-code digests in the real repository; issuance/TOTP is separately exercised.
	code := base64.RawURLEncoding.EncodeToString([]byte(strings.Repeat("r", 32)))
	digest := sha256.Sum256([]byte(code))
	if _, err := f.pools.Write().ExecContext(t.Context(), `INSERT INTO account_recovery_codes(account_id, code_hash) VALUES (?, ?)`, credential.ID, hex.EncodeToString(digest[:])); err != nil {
		t.Fatal(err)
	}
	return code
}

func TestRecoveryHTTPSOnceAndThrottle(t *testing.T) {
	f := newHostedFixture(t)
	code := recoveryCode(t, f)
	bearer := f.login(t, "commissioner", "private-password")
	body := fmt.Sprintf(`{"protocol":1,"username":"commissioner","code":%q,"password":"replacement-password"}`, code)
	f.post(t, serverhttp.RecoveryPath, body, "", "", 200)
	if _, err := f.sessions.Authenticate(t.Context(), accounts.RawSession{Bearer: bearer}); err == nil {
		t.Fatal("old session survived reset")
	}
	f.post(t, serverhttp.RecoveryPath, body, "", "", 401)
	f.login(t, "commissioner", "replacement-password")
	// Login and recovery spend the same ten-attempt peer allowance.
	for range 6 {
		f.post(t, serverhttp.RecoveryPath, body, "", "", 401)
	}
	f.post(t, serverhttp.RecoveryPath, body, "", "", 429)
}
