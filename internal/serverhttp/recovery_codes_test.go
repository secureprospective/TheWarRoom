package serverhttp_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

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
	f.advanceTime(6 * time.Minute)
	f.post(t, serverhttp.RecoveryCodesPath, body, "", proof, 403)
}
