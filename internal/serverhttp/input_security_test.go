package serverhttp_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

func TestBrowserOriginsCannotMutateOrDiscloseSecrets(t *testing.T) {
	f := newHostedFixture(t)
	bearer := f.login(t, "commissioner", "private-password")
	proof := f.privilegedProof(t)
	var logs bytes.Buffer
	f.server.Config.ErrorLog = log.New(&logs, "", 0)
	var ledger commissioner.Ledger
	if err := json.Unmarshal(f.post(t, serverhttp.LedgerPath, `{"protocol":1,"league":"native.hosted"}`, bearer, proof, 200), &ledger); err != nil {
		t.Fatal(err)
	}
	change := `{"protocol":1,"command":{"league":"native.hosted","command":"commish.whatsnew.post","args":{"note":{"text":"sentinel-injected-secret"}},"reason":"Publish note","expected":1,"request":"browser-note","effective":"now","visibility":"league"}}`
	change = strings.Replace(change, `"expected":1`, fmt.Sprintf(`"expected":%d`, ledger.Revision), 1)
	for _, path := range []string{serverhttp.ExecutePath, serverhttp.PreviewPath, serverhttp.LedgerPath, serverhttp.PeoplePath, serverhttp.LoginPath, serverhttp.EnrollmentPath, serverhttp.TOTPBeginPath, serverhttp.TOTPConfirmPath, serverhttp.TOTPStepUpPath, serverhttp.RevokePath} {
		for _, origins := range [][]string{{f.server.URL}, {"https://evil.test"}, {"null"}, {""}, {"", "https://evil.test"}} {
			request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, f.server.URL+path, strings.NewReader(change))
			if err != nil {
				t.Fatal(err)
			}
			request.Header["Origin"] = origins
			request.Header.Set("Authorization", "Bearer "+bearer)
			request.Header.Set("X-TWR-Recent-Auth", proof)
			response, err := f.server.Client().Do(request)
			if err != nil {
				t.Fatal(err)
			}
			body, err := io.ReadAll(response.Body)
			_ = response.Body.Close()
			if err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != 400 || response.Header.Get("Content-Security-Policy") != "default-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'" || response.Header.Get("X-Content-Type-Options") != "nosniff" {
				t.Fatalf("%s origins=%q: status=%d headers=%v", path, origins, response.StatusCode, response.Header)
			}
			for _, secret := range []string{bearer, proof, "sentinel-injected-secret"} {
				if bytes.Contains(body, []byte(secret)) || strings.Contains(logs.String(), secret) {
					t.Fatal("secret echoed or logged")
				}
			}
		}
	}
	f.revision(t, 1)
	// The originless desktop route and the credential still work after refusals.
	f.post(t, serverhttp.PeoplePath, `{"protocol":1,"league":"native.hosted"}`, bearer, proof, 200)
}
