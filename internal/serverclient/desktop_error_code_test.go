package serverclient

import (
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

func TestDesktopForwardsServerErrorCodeHeader(t *testing.T) {
	const (
		loginCode    = "TWR-8N3Q-4V6W"
		workflowCode = "TWR-7K9M-2P4X"
	)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case serverhttp.LoginPath:
			w.Header().Set(serverhttp.ErrorCodeHeader, loginCode)
			http.Error(w, "login refused", http.StatusUnauthorized)
		case serverhttp.WorkflowPath:
			w.Header().Set(serverhttp.ErrorCodeHeader, workflowCode)
			http.Error(w, "You have sent 10 reports in the past hour.", http.StatusConflict)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	roots := x509.NewCertPool()
	roots.AddCert(server.Certificate())
	desktop, err := NewDesktop(
		server.URL, "native.test", roots, &testCredentials{unavailable: true},
	)
	if err != nil {
		t.Fatal(err)
	}
	defer desktop.Close()

	loginReq := httptest.NewRequest(
		http.MethodPost, DesktopPath,
		strings.NewReader(
			`{"action":"login","username":"owner","password":"private-password"}`,
		),
	)
	loginReq.Header.Set("Origin", "http://wails.localhost")
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	desktop.Handler().ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusBadRequest ||
		loginRec.Header().Get(serverhttp.ErrorCodeHeader) != loginCode {
		t.Fatalf(
			"login header: status=%d code=%q",
			loginRec.Code, loginRec.Header().Get(serverhttp.ErrorCodeHeader),
		)
	}

	desktop.mu.Lock()
	desktop.bearer = "device-bearer-token"
	desktop.mu.Unlock()
	wfBody := `{"command":"native.directory.read","version":"1","mode":"read",` +
		`"arguments":{"league":"native.test"}}`
	wfReq := httptest.NewRequest(
		http.MethodPost, DesktopWorkflowPath, strings.NewReader(wfBody),
	)
	wfReq.Header.Set("Origin", "http://wails.localhost")
	wfReq.Header.Set("Content-Type", "application/json")
	wfRec := httptest.NewRecorder()
	desktop.Handler().ServeHTTP(wfRec, wfReq)
	if wfRec.Code != http.StatusBadRequest ||
		wfRec.Header().Get(serverhttp.ErrorCodeHeader) != workflowCode {
		t.Fatalf(
			"workflow header: status=%d code=%q",
			wfRec.Code, wfRec.Header().Get(serverhttp.ErrorCodeHeader),
		)
	}

	// Local malformed JSON error has no server code so the client mints TWR-C...
	badReq := httptest.NewRequest(
		http.MethodPost, DesktopWorkflowPath, strings.NewReader("{"),
	)
	badReq.Header.Set("Origin", "http://wails.localhost")
	badReq.Header.Set("Content-Type", "application/json")
	badRec := httptest.NewRecorder()
	desktop.Handler().ServeHTTP(badRec, badReq)
	if badRec.Code != http.StatusBadRequest ||
		badRec.Header().Get(serverhttp.ErrorCodeHeader) != "" {
		t.Fatalf(
			"local error should omit server code: status=%d code=%q",
			badRec.Code, badRec.Header().Get(serverhttp.ErrorCodeHeader),
		)
	}
}
