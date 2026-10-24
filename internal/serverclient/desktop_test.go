package serverclient

import (
	"crypto/x509"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDesktopBoundaryAndTLS(t *testing.T) {
	for _, address := range []string{"http://localhost", "https://user:secret@localhost", "https://localhost/path", "https://localhost?x=1", "https://localhost#x"} {
		if _, err := NewDesktop(address, "native.test", nil); err == nil {
			t.Fatalf("unsafe origin accepted: %s", address)
		}
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "secret-sentinel", http.StatusForbidden) }))
	defer server.Close()
	desktop, err := NewDesktop(server.URL, "native.test", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer desktop.Close()
	raw := RawDesktop{Action: "login", Username: "owner", Password: "private-password"}
	if _, err := desktop.Act(t.Context(), raw); err == nil || strings.Contains(err.Error(), "private-password") {
		t.Fatalf("untrusted certificate: %v", err)
	}
	roots := x509.NewCertPool()
	roots.AddCert(server.Certificate())
	trusted, err := NewDesktop(server.URL, "native.test", roots)
	if err != nil {
		t.Fatal(err)
	}
	defer trusted.Close()
	if _, err := trusted.Act(t.Context(), raw); err == nil || strings.Contains(err.Error(), "secret-sentinel") {
		t.Fatalf("upstream error disclosed: %v", err)
	}
	for _, tc := range []struct {
		body, origin, content string
		status                int
	}{
		{`{"action":"status"}`, "http://evil.example", "application/json", 403},
		{`{"action":"status"}`, "http://wails.localhost", "text/plain", 400},
		{`{"action":"status","principal":"creator"}`, "http://wails.localhost", "application/json", 400},
		{`{"action":"status"} {}`, "http://wails.localhost", "application/json", 400},
		{`{"action":"execute"}`, "http://wails.localhost", "application/json", 400},
		{`{"action":"status"}`, "http://wails.localhost", "application/json", 200},
	} {
		r := httptest.NewRequest(http.MethodPost, DesktopPath, strings.NewReader(tc.body))
		r.Header.Set("Origin", tc.origin)
		r.Header.Set("Content-Type", tc.content)
		w := httptest.NewRecorder()
		trusted.Handler().ServeHTTP(w, r)
		if w.Code != tc.status || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("boundary %s: %d %s", tc.body, w.Code, w.Body.String())
		}
	}
}

func TestDesktopRejectsMalformedSuccessfulRepliesAndRedirects(t *testing.T) {
	for _, body := range []string{`{}`, `{"protocol":2}`, `{"protocol":1,"bearer":"invalid"}`, strings.Repeat("x", (256<<10)+1)} {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if _, err := w.Write([]byte(body)); err != nil {
				t.Error(err)
			}
		}))
		roots := x509.NewCertPool()
		roots.AddCert(server.Certificate())
		desktop, err := NewDesktop(server.URL, "native.test", roots)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := desktop.Act(t.Context(), RawDesktop{Action: "login", Username: "owner", Password: "owner-password"}); err == nil {
			t.Fatal("malformed successful response accepted")
		}
		if desktop.bearer != "" {
			t.Fatal("bad reply installed credential")
		}
		desktop.Close()
		server.Close()
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://elsewhere.invalid", http.StatusTemporaryRedirect)
	}))
	defer server.Close()
	roots := x509.NewCertPool()
	roots.AddCert(server.Certificate())
	desktop, err := NewDesktop(server.URL, "native.test", roots)
	if err != nil {
		t.Fatal(err)
	}
	defer desktop.Close()
	if _, err := desktop.Act(t.Context(), RawDesktop{Action: "login", Username: "owner", Password: "owner-password"}); err == nil {
		t.Fatal("redirect followed")
	}
}

func TestDesktopViewNeverSerializesCredentials(t *testing.T) {
	desktop, err := NewDesktop("https://example.invalid", "native.test", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer desktop.Close()
	desktop.bearer, desktop.proof = "bearer-sentinel", "proof-sentinel"
	view, err := desktop.Act(t.Context(), RawDesktop{Action: "status"})
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(view)
	if err != nil || strings.Contains(string(body), "sentinel") {
		t.Fatalf("secret crossed webview: %s %v", body, err)
	}
}
