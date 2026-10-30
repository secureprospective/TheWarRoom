package serverhttp_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/serverclient"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

func TestRecoveryCodesNeedFreshProofAndReplaceOldCodes(t *testing.T) {
	f := newHostedFixture(t)
	old := recoveryCode(t, f)
	body := `{"protocol":1,"username":"commissioner","password":"private-password"}`
	f.post(t, serverhttp.RecoveryCodesPath, body, "", "", 403)
	proof := f.privilegedProof(t)
	var printed serverhttp.RecoveryCodesResponse
	if err := json.Unmarshal(f.post(t, serverhttp.RecoveryCodesPath, body, "", proof, 200), &printed); err != nil {
		t.Fatal(err)
	}
	if len(printed.Codes) != 8 || printed.Codes[0] == "" || printed.Codes[0] == printed.Codes[1] {
		t.Fatalf("codes: %d", len(printed.Codes))
	}
	reset := `{"protocol":1,"username":"commissioner","code":%q,"password":"replacement-password"}`
	f.post(t, serverhttp.RecoveryPath, fmt.Sprintf(reset, old), "", "", 401)
	f.post(t, serverhttp.RecoveryPath, fmt.Sprintf(reset, printed.Codes[3]), "", "", 200)
}

func TestRecoveryCodesRefuseWrongPasswordForgedOrExpiredProof(t *testing.T) {
	f := newHostedFixture(t)
	body := `{"protocol":1,"username":"commissioner","password":"private-password"}`
	proof := f.privilegedProof(t)
	f.post(t, serverhttp.RecoveryCodesPath, `{"protocol":1,"username":"commissioner","password":"wrong-password-x"}`, "", proof, 403)
	f.post(t, serverhttp.RecoveryCodesPath, body, "", "not-a-proof", 403)
	f.now = f.now.Add(6 * time.Minute)
	f.post(t, serverhttp.RecoveryCodesPath, body, "", proof, 403)
}

func TestDesktopPrintsRecoveryCodesAfterAuthenticatorCheck(t *testing.T) {
	f := newHostedFixture(t)
	desktop := desktopFor(t, f)
	credentials := serverclient.RawDesktop{Action: "login", Username: "commissioner", Password: "private-password"}
	if _, err := desktop.Act(t.Context(), credentials); err != nil {
		t.Fatal(err)
	}
	credentials.Action = "recovery-codes"
	if _, err := desktop.Act(t.Context(), credentials); err == nil || !strings.Contains(err.Error(), "authenticator") {
		t.Fatalf("codes without a check: %v", err)
	}
	credentials.Action = "totp-begin"
	setup, err := desktop.Act(t.Context(), credentials)
	if err != nil {
		t.Fatal(err)
	}
	credentials.Action, credentials.Code = "totp-confirm", testTOTP(t, setup.Seed, f.now)
	if _, err := desktop.Act(t.Context(), credentials); err != nil {
		t.Fatal(err)
	}
	f.now = f.now.Add(30 * time.Second)
	credentials.Action, credentials.Code = "totp-step-up", testTOTP(t, setup.Seed, f.now)
	if _, err := desktop.Act(t.Context(), credentials); err != nil {
		t.Fatal(err)
	}
	credentials.Action, credentials.Code = "recovery-codes", ""
	view, err := desktop.Act(t.Context(), credentials)
	if err != nil || len(view.RecoveryCodes) != 8 || !strings.Contains(view.Message, "will not be shown again") {
		t.Fatalf("codes: %d %v", len(view.RecoveryCodes), err)
	}
	reset := serverclient.RawDesktop{Action: "recover", Username: "commissioner", Code: view.RecoveryCodes[0], Password: "replacement-password"}
	if _, err := desktop.Act(t.Context(), reset); err != nil {
		t.Fatalf("printed code did not recover: %v", err)
	}
}
