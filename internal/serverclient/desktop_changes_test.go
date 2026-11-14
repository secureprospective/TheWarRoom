package serverclient

import (
	"context"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

func TestDesktopChangesDeliveryTimeoutAndBoundary(t *testing.T) {
	d := &Desktop{}
	done := make(chan DesktopChanges, 1)
	go func() { done <- d.changesAfter(t.Context(), 0, time.Second) }()
	d.mu.Lock()
	d.changeLocked("connected", RawInvalidation{Protocol: 2, ServerEpoch: "boot", League: "native.test", Topics: []serverhttp.TopicVersion{{Kind: "league", Version: "1"}}})
	d.mu.Unlock()
	got := <-done
	if got.Seq != 1 || len(got.Topics) != 1 || got.State != "connected" {
		t.Fatalf("delivery: %+v", got)
	}
	start := time.Now()
	got = d.changesAfter(t.Context(), got.Seq, 10*time.Millisecond)
	if len(got.Topics) != 0 || got.Topics == nil || time.Since(start) < 10*time.Millisecond {
		t.Fatalf("timeout: %+v", got)
	}
	for _, origin := range []string{"wails://wails", "https://evil.invalid"} {
		req := httptest.NewRequest(http.MethodPost, DesktopChangesPath, strings.NewReader(`{"after":0}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", origin)
		response := httptest.NewRecorder()
		d.Handler().ServeHTTP(response, req)
		want := 200
		if origin == "https://evil.invalid" {
			want = 403
		}
		if response.Code != want || response.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("origin %s: %d", origin, response.Code)
		}
	}
}

func TestDesktopWatcherStartsAfterLoginStopsOnForget(t *testing.T) {
	opened, closed := make(chan struct{}, 2), make(chan struct{}, 2)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == serverhttp.LoginPath {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{"protocol":1,"bearer":%q,"session":{"account":{"id":"11111111-1111-4111-8111-111111111111","username":"owner"},"expires_at":"2030-01-01T00:00:00Z"}}`, strings.Repeat("a", 42)+"A")
			return
		}
		if r.Header.Get("X-TWR-Invalidation") != "2" {
			t.Error("missing negotiation")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "event: invalidate\ndata: {\"protocol\":2,\"serverEpoch\":\"boot\",\"league\":\"native.test\",\"topics\":[{\"kind\":\"league\",\"version\":\"1\"}]}\n\n")
		w.(http.Flusher).Flush()
		opened <- struct{}{}
		<-r.Context().Done()
		closed <- struct{}{}
	}))
	defer server.Close()
	roots := x509.NewCertPool()
	roots.AddCert(server.Certificate())
	d, err := NewDesktop(server.URL, "native.test", roots, &testCredentials{})
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if _, err := d.Act(t.Context(), RawDesktop{Action: "login", Username: "owner", Password: "private-password"}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-opened:
	case <-time.After(time.Second):
		t.Fatal("watcher did not start")
	}
	if _, err := d.Act(t.Context(), RawDesktop{Action: "forget"}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("forget left stream alive")
	}
	got := d.changesAfter(t.Context(), 0, time.Millisecond)
	if got.State != "signed-out" {
		t.Fatalf("forget: %+v", got)
	}
}

func TestDesktopWatcherUnauthorizedEndsSession(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusUnauthorized) }))
	defer server.Close()
	roots := x509.NewCertPool()
	roots.AddCert(server.Certificate())
	d, err := NewDesktop(server.URL, "native.test", roots, &testCredentials{})
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	d.mu.Lock()
	d.bearer = "session"
	d.loaded = true
	d.startWatchLocked()
	d.mu.Unlock()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		d.mu.Lock()
		state, bearer := d.changes.State, d.bearer
		d.mu.Unlock()
		if state == "signed-out" && bearer == "" {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("401 did not end session")
}

func TestV2InvalidationTopicsAndV1Fallback(t *testing.T) {
	for _, data := range []string{`{"protocol":1}`, `{"protocol":2,"serverEpoch":"boot","league":"native.test","topics":[{"kind":"scores","version":"2","season":2026,"week":10}]}`} {
		response := &http.Response{Body: http.NoBody}
		response.Body = io.NopCloser(strings.NewReader("event: invalidate\ndata: " + data + "\n\n"))
		calls := 0
		_, _ = consumeTopicInvalidations(context.Background(), response, func(_ context.Context, message RawInvalidation) error {
			calls++
			if message.Protocol == 2 && (len(message.Topics) != 1 || message.Topics[0].Week != 10) {
				t.Fatalf("topics lost: %+v", message)
			}
			return nil
		})
		if calls != 1 {
			t.Fatalf("calls=%d", calls)
		}
	}
}

func TestDesktopWatcherReloginFencesOldUnauthorized(t *testing.T) {
	oldOpened := make(chan struct{}, 1)
	newOpened := make(chan struct{}, 1)
	releaseOld := make(chan struct{})
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "Bearer old" {
			oldOpened <- struct{}{}
			<-releaseOld
			w.WriteHeader(401)
			return
		}
		newOpened <- struct{}{}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "event: invalidate\ndata: {\"protocol\":1}\n\n")
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer server.Close()
	roots := x509.NewCertPool()
	roots.AddCert(server.Certificate())
	d, err := NewDesktop(server.URL, "native.test", roots, &testCredentials{})
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	d.mu.Lock()
	d.loaded = true
	d.bearer = "old"
	d.startWatchLocked()
	d.mu.Unlock()
	select {
	case <-oldOpened:
	case <-time.After(time.Second):
		t.Fatal("old watch missing")
	}
	d.mu.Lock()
	d.bearer = "new"
	d.startWatchLocked()
	d.mu.Unlock()
	close(releaseOld)
	select {
	case <-newOpened:
	case <-time.After(time.Second):
		t.Fatal("new watch missing")
	}
	got := d.changesAfter(t.Context(), 0, time.Second)
	deadline := time.Now().Add(time.Second)
	for got.State != "connected" && time.Now().Before(deadline) {
		got = d.changesAfter(t.Context(), got.Seq, time.Second)
	}
	d.mu.Lock()
	bearer := d.bearer
	d.mu.Unlock()
	if bearer != "new" || got.State != "connected" {
		t.Fatalf("old 401 changed newer session: %q %+v", bearer, got)
	}
}

func TestChangesRetainsCrossingTopicsAndResyncsOnOverflow(t *testing.T) {
	d := &Desktop{}
	d.mu.Lock()
	for i, kind := range []string{"league", "scores"} {
		d.changeLocked("connected", RawInvalidation{Topics: []serverhttp.TopicVersion{{Kind: kind, Version: fmt.Sprint(i + 1)}}})
	}
	d.mu.Unlock()
	got := d.changesAfter(t.Context(), 0, time.Millisecond)
	if len(got.Topics) != 2 {
		t.Fatalf("lost crossing topics: %+v", got)
	}
	d.mu.Lock()
	for range 65 {
		d.changeLocked("connected", RawInvalidation{Topics: []serverhttp.TopicVersion{{Kind: "league", Version: "3"}}})
	}
	d.mu.Unlock()
	if got := d.changesAfter(t.Context(), 0, time.Millisecond); len(got.Topics) != 0 {
		t.Fatalf("overflow should resync: %+v", got)
	}
}
