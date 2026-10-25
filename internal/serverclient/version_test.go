package serverclient

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
	"github.com/secureprospective/TheWarRoom/internal/serverversion"
)

func TestDesktopAndAgentExposeUpdateRequiredWithoutStream(t *testing.T) {
	for _, side := range []serverversion.Side{serverversion.App, serverversion.Server} {
		t.Run(string(side), func(t *testing.T) {
			calls := 0
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Header.Get(serverhttp.ProtocolHeader) != serverversion.ClientProtocol() || r.Header.Get(serverhttp.CommandsHeader) != serverversion.ClientCommands() {
					t.Error("client omitted its versions")
				}
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set(serverhttp.UpdateHeader, string(side))
				w.WriteHeader(http.StatusUpgradeRequired)
				_ = json.NewEncoder(w).Encode(serverhttp.RawUpdateRequiredResponse{Protocol: serverversion.ProtocolMajor, Update: string(side), Message: "Install the latest update."})
			}))
			defer server.Close()
			roots := x509.NewCertPool()
			roots.AddCert(server.Certificate())
			desktop, err := NewDesktop(server.URL, "native.test", roots, &testCredentials{unavailable: true})
			if err != nil {
				t.Fatal(err)
			}
			defer desktop.Close()
			_, err = desktop.Act(t.Context(), RawDesktop{Action: "login", Username: "owner", Password: "private-password"})
			assertUpdate(t, err, side)
			agent, err := NewAgent(server.URL, "native.test", "connection-token", roots)
			if err != nil {
				t.Fatal(err)
			}
			defer agent.client.Close()
			_, err = agent.Ledger(t.Context())
			assertUpdate(t, err, side)
			desktop.bearer = "session"
			retry, err := desktop.watchInvalidations(t.Context(), func(_ context.Context) error { t.Fatal("refused stream reread native data"); return nil })
			assertUpdate(t, err, side)
			request := httptest.NewRequest(http.MethodPost, DesktopPath, strings.NewReader(`{"action":"login","username":"owner","password":"private-password"}`))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			desktop.Handler().ServeHTTP(response, request)
			if response.Code != http.StatusUpgradeRequired || response.Header().Get(serverhttp.UpdateHeader) != string(side) {
				t.Fatalf("webview lost update signal: %d %s", response.Code, response.Body.String())
			}
			expectedCalls := 4
			if side == serverversion.App {
				expectedCalls = 3
			} // Once refused, native actions stay blocked locally.
			if retry || calls != expectedCalls {
				t.Fatalf("update refusal retried or missing requests: retry %v, calls %d", retry, calls)
			}
		})
	}
}

func assertUpdate(t *testing.T, err error, side serverversion.Side) {
	t.Helper()
	var update *UpdateRequired
	if !errors.Is(err, ErrUpdateRequired) || !errors.As(err, &update) || update == nil || update.Update != side || update.Error() != "Install the latest update." {
		t.Fatalf("lost update-required signal: %v", err)
	}
}
