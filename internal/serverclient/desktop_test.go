package serverclient

import (
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

func TestDesktopBoundaryAndTLS(t *testing.T) {
	for _, address := range []string{"http://localhost", "https://user:secret@localhost", "https://localhost/path", "https://localhost?x=1", "https://localhost#x"} {
		if _, err := NewDesktop(address, "native.test", nil, &testCredentials{unavailable: true}); err == nil {
			t.Fatalf("unsafe origin accepted: %s", address)
		}
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "secret-sentinel", http.StatusForbidden) }))
	defer server.Close()
	desktop, err := NewDesktop(server.URL, "native.test", nil, &testCredentials{unavailable: true})
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
	trusted, err := NewDesktop(server.URL, "native.test", roots, &testCredentials{unavailable: true})
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
	// The shell's startup probe: native mode answers GET /desktop/native with a JSON view, never 404.
	r := httptest.NewRequest(http.MethodGet, DesktopPath, nil)
	r.Header.Set("Origin", "http://wails.localhost")
	w := httptest.NewRecorder()
	trusted.Handler().ServeHTTP(w, r)
	if w.Code != http.StatusOK || !strings.HasPrefix(w.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("native startup probe: %d %q %s", w.Code, w.Header().Get("Content-Type"), w.Body.String())
	}
}

func TestDesktopRejectsMalformedSuccessfulRepliesAndRedirects(t *testing.T) {
	for _, body := range []string{`{}`, `{"protocol":2}`, `{"protocol":1,"bearer":"invalid"}`, strings.Repeat("x", desktopReplyLimit+1)} {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if _, err := w.Write([]byte(body)); err != nil {
				t.Error(err)
			}
		}))
		roots := x509.NewCertPool()
		roots.AddCert(server.Certificate())
		desktop, err := NewDesktop(server.URL, "native.test", roots, &testCredentials{unavailable: true})
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
	desktop, err := NewDesktop(server.URL, "native.test", roots, &testCredentials{unavailable: true})
	if err != nil {
		t.Fatal(err)
	}
	defer desktop.Close()
	if _, err := desktop.Act(t.Context(), RawDesktop{Action: "login", Username: "owner", Password: "owner-password"}); err == nil {
		t.Fatal("redirect followed")
	}
}

// A busy server reads as busy on every account screen, never as a password or phone-clock problem.
func TestDesktopBusyServerWording(t *testing.T) {
	for _, path := range []string{serverhttp.LoginPath, serverhttp.TOTPStepUpPath, serverhttp.RecoveryPath, serverhttp.EnrollmentPath, serverhttp.PeoplePath} {
		if err := desktopRefusal(path, 503); err == nil || !strings.EqualFold(err.Error(), serverhttp.BusyMessage) {
			t.Errorf("%s: %v", path, err)
		}
	}
	if err := desktopRefusal(serverhttp.TOTPStepUpPath, 403); err == nil || !strings.Contains(err.Error(), "phone clock") {
		t.Errorf("a refused code lost its advice: %v", err)
	}
}

func TestWorkflowParallelRequests(t *testing.T) {
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		started <- struct{}{}
		<-release
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"label":"league","data":{}}`))
	}))
	defer server.Close()
	released := false
	defer func() {
		if !released {
			close(release)
		}
	}()
	roots := x509.NewCertPool()
	roots.AddCert(server.Certificate())
	d, err := NewDesktop(server.URL, "native.test", roots, &testCredentials{secret: strings.Repeat("a", 42) + "A"})
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	raw := serverhttp.RawWorkflow{Command: "native.assets.read", Version: "1", Mode: "read", Arguments: json.RawMessage(`{"league":"native.test"}`)}
	results := make(chan error, 2)
	go func() { _, err := d.Workflow(t.Context(), raw); results <- err }()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("first request did not start")
	}
	go func() { _, err := d.Workflow(t.Context(), raw); results <- err }()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("second request blocked behind first")
	}
	close(release)
	released = true
	for range 2 {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
}

func TestWorkflowReadHonorsRetryAfterOnce(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == serverhttp.InvalidationPath {
			<-r.Context().Done()
			return
		}
		if calls.Add(1) == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"label":"league","data":{}}`))
	}))
	defer server.Close()
	roots := x509.NewCertPool()
	roots.AddCert(server.Certificate())
	d, err := NewDesktop(server.URL, "native.test", roots,
		&testCredentials{secret: strings.Repeat("a", 42) + "A"})
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	started := time.Now()
	raw := serverhttp.RawWorkflow{Command: "native.assets.read", Version: "1", Mode: "read", Arguments: json.RawMessage(`{"league":"native.test"}`)}
	if _, err := d.Workflow(t.Context(), raw); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 || time.Since(started) < 950*time.Millisecond {
		t.Fatalf("calls=%d elapsed=%s; want one delayed retry", calls.Load(), time.Since(started))
	}
}

func TestWorkflowUnauthorizedSessionGeneration(t *testing.T) {
	for _, relogin := range []bool{false, true} {
		t.Run(fmt.Sprintf("relogin=%t", relogin), func(t *testing.T) {
			started, release := make(chan struct{}), make(chan struct{})
			old := strings.Repeat("a", 42) + "A"
			fresh := strings.Repeat("b", 42) + "A"
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == serverhttp.InvalidationPath {
					<-r.Context().Done()
					return
				}
				if r.URL.Path == serverhttp.LoginPath {
					w.Header().Set("Content-Type", "application/json")
					_, _ = fmt.Fprintf(w, `{"protocol":1,"bearer":%q,"session":{"account":{"id":"12345678-1234-4234-8234-123456789abc","username":"owner"},"expires_at":"2026-11-01T00:00:00Z"}}`, fresh)
					return
				}
				close(started)
				<-release
				w.WriteHeader(http.StatusUnauthorized)
			}))
			defer server.Close()
			roots := x509.NewCertPool()
			roots.AddCert(server.Certificate())
			store := &testCredentials{secret: old}
			d, err := NewDesktop(server.URL, "native.test", roots, store)
			if err != nil {
				t.Fatal(err)
			}
			defer d.Close()
			raw := serverhttp.RawWorkflow{Command: "native.assets.read", Version: "1", Mode: "read", Arguments: json.RawMessage(`{"league":"native.test"}`)}
			done := make(chan error, 1)
			go func() { _, err := d.Workflow(t.Context(), raw); done <- err }()
			select {
			case <-started:
			case <-time.After(3 * time.Second):
				close(release)
				t.Fatal("workflow did not start")
			}
			before, err := d.Act(t.Context(), RawDesktop{Action: "status"})
			if err != nil {
				t.Fatal(err)
			}
			if relogin {
				if before, err = d.Act(t.Context(), RawDesktop{Action: "login", Username: "owner", Password: "owner-password"}); err != nil {
					close(release)
					t.Fatal(err)
				}
			}
			close(release)
			if err := <-done; err == nil {
				t.Fatal("unauthorized workflow succeeded")
			}
			after, err := d.Act(t.Context(), RawDesktop{Action: "status"})
			if err != nil {
				t.Fatal(err)
			}
			if (after.AuthGeneration == before.AuthGeneration) != relogin {
				t.Fatalf("generation before=%s after=%s relogin=%t", before.AuthGeneration, after.AuthGeneration, relogin)
			}
			want := ""
			if relogin {
				want = fresh
			}
			if d.bearer != want || store.secret != want {
				t.Fatal("wrong session retained after unauthorized response")
			}
		})
	}
}

func TestWorkflowForbiddenPreservesSession(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()
	roots := x509.NewCertPool()
	roots.AddCert(server.Certificate())
	store := &testCredentials{secret: strings.Repeat("a", 42) + "A"}
	d, err := NewDesktop(server.URL, "native.test", roots, store)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	before, err := d.Act(t.Context(), RawDesktop{Action: "status"})
	if err != nil {
		t.Fatal(err)
	}
	bearer := d.bearer
	raw := serverhttp.RawWorkflow{Command: "native.assets.read", Version: "1", Mode: "read", Arguments: json.RawMessage(`{"league":"native.test"}`)}
	if _, err := d.Workflow(t.Context(), raw); err == nil {
		t.Fatal("forbidden workflow succeeded")
	}
	after, err := d.Act(t.Context(), RawDesktop{Action: "status"})
	if err != nil {
		t.Fatal(err)
	}
	if after.AuthGeneration != before.AuthGeneration || d.bearer != bearer || store.secret != bearer {
		t.Fatal("refused read changed the session or saved credential")
	}
}

// A real league's league-wide read is near 2 MB; the desktop must accept it, not call it unreadable.
func TestDesktopAcceptsALeagueSizedReply(t *testing.T) {
	reply := `{"pad":"` + strings.Repeat("x", 2<<20) + `"}`
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(reply)); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	session := desktopSnapshot{client: server.Client(), address: server.URL, league: "native.test"}
	body, _, err := postDesktopSnapshot(t.Context(), session, DesktopPath, serverhttp.RawNativeRequest{Protocol: 1})
	if err != nil || len(body) != len(reply) {
		t.Fatalf("league-sized reply: %d bytes, %v", len(body), err)
	}
}
