package serverclient

import (
	"crypto/x509"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
	"github.com/secureprospective/TheWarRoom/internal/serverversion"
)

func TestAgentExposesUpdateRequired(t *testing.T) {
	for _, side := range []serverversion.Side{serverversion.App, serverversion.Server} {
		t.Run(string(side), func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
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
			agent, err := NewAgent(server.URL, "native.test", "connection-token", roots)
			if err != nil {
				t.Fatal(err)
			}
			defer agent.client.Close()
			_, err = agent.Ledger(t.Context())
			assertUpdate(t, err, side)
			if calls.Load() != 1 {
				t.Fatalf("update refusal retried or missing requests: calls %d", calls.Load())
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
