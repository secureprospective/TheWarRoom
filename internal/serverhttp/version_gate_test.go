package serverhttp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/serverversion"
)

func TestHostedRefusesVersionsBeforeNativeAccessWithoutSSE(t *testing.T) {
	// No services exist. Any fallthrough would return 503, not the compatibility refusal.
	handler := (Hosted{}).Handler()
	for _, path := range []string{LoginPath, LedgerPath, DispatchPath, ExecutePath, InvalidationPath} {
		for _, versions := range [][3]string{{"0.9", "1", "app"}, {"2.0", "1", "server"}, {"1.9999", "1", "server"}, {"1.1", "0", "app"}, {"1.1", "2", "server"}} {
			t.Run(path+versions[0]+versions[1], func(t *testing.T) {
				request := httptest.NewRequest(http.MethodPost, "https://localhost"+path, nil)
				request.RemoteAddr = "127.0.0.1:1234"
				request.Header.Set(ProtocolHeader, versions[0])
				request.Header.Set(CommandsHeader, versions[1])
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, request)
				var raw RawUpdateRequiredResponse
				if response.Code != http.StatusUpgradeRequired || json.Unmarshal(response.Body.Bytes(), &raw) != nil || raw.Validate() != nil || raw.Update != versions[2] || response.Header().Get(UpdateHeader) != versions[2] {
					t.Fatalf("refusal = %d %s", response.Code, response.Body.String())
				}
			})
		}
	}
}

func TestVersionGateAdmitsCurrentPreviousAndHeaderless(t *testing.T) {
	calls := 0
	handler := versionGate(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { calls++; w.WriteHeader(http.StatusNoContent) }))
	for _, protocol := range []string{serverversion.ClientProtocol(), "1.0", ""} {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		if protocol != "" {
			request.Header.Set(ProtocolHeader, protocol)
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusNoContent {
			t.Fatalf("supported %q = %d", protocol, response.Code)
		}
	}
	for _, headers := range [][]string{{"1"}, {"1.1", "2.0"}, {""}, {"1.1, 2.0"}} {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		for _, value := range headers {
			request.Header.Add(ProtocolHeader, value)
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("ambiguous %v = %d", headers, response.Code)
		}
	}
	if calls != 3 {
		t.Fatalf("rejected request reached services: %d", calls)
	}
}
