package serverclient

import (
	"crypto/x509"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

func TestHelperPreviewFailureNeverOffersCode(t *testing.T) {
	for _, tc := range []struct{ name, body, want string }{
		{"blocked", `{"blockers":["This change is no longer allowed."]}`, "This change is no longer allowed."},
		{"malformed", `{}`, "unreadable change preview"},
		{"wrong league", `{"target":{"league":"native.other","version":1},"fields":[{"label":"Name"}]}`,
			"different league version"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			confirmations := 0
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == serverhttp.ConfirmPath {
					confirmations++
				}
				w.Header().Set("Content-Type", "application/json")
				if _, err := fmt.Fprint(w, tc.body); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()
			roots := x509.NewCertPool()
			roots.AddCert(server.Certificate())
			desktop, err := NewDesktop(server.URL, "native.test", roots, &testCredentials{unavailable: true})
			if err != nil {
				t.Fatal(err)
			}
			defer desktop.Close()
			desktop.bearer, desktop.proof = "session", "proof"
			request, err := EncodeConfirmRequest(strings.Repeat("A", 43), commissioner.RawCommand{
				League: "native.test", Command: "commish.franchise.edit", Expected: 1,
				Args: commissioner.RawArgs{People: &commissioner.RawPeople{
					Franchise: "franchise.01", Name: "Helper team", Abbreviation: "HT"}},
				Request: "rename", Effective: "now", Visibility: "league",
			})
			if err != nil {
				t.Fatal(err)
			}
			for _, action := range []string{"helper-check", "helper-confirm"} {
				view, err := desktop.Act(t.Context(), RawDesktop{Action: action, Change: request})
				if err == nil || !strings.Contains(err.Error(), tc.want) || view.Confirmation != "" {
					t.Fatalf("%s: %+v %v", action, view, err)
				}
			}
			if confirmations != 0 {
				t.Fatal("failed preview reached confirmation")
			}
		})
	}
}
