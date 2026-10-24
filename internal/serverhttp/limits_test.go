package serverhttp

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestProxySpoofingAndAuthThrottle(t *testing.T) {
	calls := 0
	handler := resourceLimits(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		deadline, ok := r.Context().Deadline()
		if !ok || time.Until(deadline) > 10*time.Second {
			t.Error("missing auth deadline")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	for _, header := range []string{"Forwarded", "X-Forwarded-For", "X-Forwarded-Proto", "X-Real-IP", "CF-Connecting-IP", "True-Client-IP"} {
		r := httptest.NewRequest(http.MethodPost, LoginPath, nil)
		r.Header.Set(header, "sentinel-secret")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != 400 || strings.Contains(w.Body.String(), "sentinel-secret") {
			t.Fatalf("spoofed %s: %d %s", header, w.Code, w.Body.String())
		}
	}
	if calls != 0 {
		t.Fatal("spoof reached downstream")
	}
	for i := range 100 {
		r := httptest.NewRequest(http.MethodPost, LoginPath, nil)
		r.RemoteAddr = fmt.Sprintf("192.0.2.1:%d", 1000+i)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		want := 204
		if i >= 10 {
			want = 429
		}
		if w.Code != want {
			t.Fatalf("attempt %d: %d", i, w.Code)
		}
	}
	if calls != 10 {
		t.Fatalf("throttle admitted %d", calls)
	}
}

func TestAuthThrottleStorageAndExpiry(t *testing.T) {
	throttle := &authThrottle{peers: make(map[string]authWindow)}
	now := time.Now()
	for i := range 4096 {
		peer := fmt.Sprintf("[2001:db8::%x]:1", i)
		if !throttle.allow(peer, now) {
			t.Fatalf("peer %d refused", i)
		}
	}
	if throttle.allow("192.0.2.1:1", now) || len(throttle.peers) != 4096 {
		t.Fatal("unbounded peer storage")
	}
	if !throttle.allow("192.0.2.1:1", now.Add(time.Minute)) || len(throttle.peers) != 1 {
		t.Fatal("expired entries not reclaimed")
	}
	for range 9 {
		if !throttle.allow("192.0.2.1:2", now.Add(time.Minute)) {
			t.Fatal("unexpected refusal")
		}
	}
	if throttle.allow("192.0.2.1:3", now.Add(time.Minute)) {
		t.Fatal("port evaded throttle")
	}
	if !throttle.allow("192.0.2.1:3", now.Add(2*time.Minute)) {
		t.Fatal("window did not reset")
	}
}

func TestHTTPInFlightAndCommandDeadline(t *testing.T) {
	entered := make(chan struct{}, 64)
	done := make(chan struct{}, 64)
	unblock := make(chan struct{})
	handler := resourceLimits(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deadline, ok := r.Context().Deadline()
		if !ok || time.Until(deadline) > 5*time.Second {
			t.Error("missing command deadline")
		}
		entered <- struct{}{}
		<-unblock
		w.WriteHeader(http.StatusNoContent)
	}))
	for range 64 {
		go func() {
			handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, ExecutePath, nil))
			done <- struct{}{}
		}()
	}
	for range 64 {
		<-entered
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodPost, ExecutePath, nil))
	close(unblock)
	for range 64 {
		<-done
	}
	if w.Code != 503 {
		t.Fatalf("request cap: %d", w.Code)
	}
}

func TestHTTPAuthFloodReservesCommandSlots(t *testing.T) {
	entered := make(chan struct{}, 6)
	done := make(chan struct{}, 6)
	unblock := make(chan struct{})
	handler := resourceLimits(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == LoginPath {
			entered <- struct{}{}
			<-unblock
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	for range 6 {
		go func() {
			handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, LoginPath, nil))
			done <- struct{}{}
		}()
	}
	for range 6 {
		<-entered
	}
	login := httptest.NewRecorder()
	handler.ServeHTTP(login, httptest.NewRequest(http.MethodPost, LoginPath, nil))
	command := httptest.NewRecorder()
	handler.ServeHTTP(command, httptest.NewRequest(http.MethodPost, ExecutePath, nil))
	close(unblock)
	for range 6 {
		<-done
	}
	if login.Code != 429 || command.Code != 204 {
		t.Fatalf("auth=%d command=%d", login.Code, command.Code)
	}
}
