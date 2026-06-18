// internal/mfl/client_test.go
package mfl

import (
	"context"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/goleak"
)

// TestMain runs goleak after the suite to catch goroutine leaks — the failure
// class the time.NewTimer+Stop backoff fix exists to prevent (M9).
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

// redirectTransport sends every outbound request to a test server while
// recording the URL the Client actually built — so tests can assert host
// routing (e.g. discovery forces the api host) without leaving the machine.
type redirectTransport struct {
	target *url.URL
	inner  http.RoundTripper

	mu    sync.Mutex
	hosts []string
	query url.Values
}

func (t *redirectTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	t.mu.Lock()
	t.hosts = append(t.hosts, r.URL.Host)
	t.query = r.URL.Query()
	t.mu.Unlock()

	out := r.Clone(r.Context())
	out.URL.Scheme = t.target.Scheme
	out.URL.Host = t.target.Host
	return t.inner.RoundTrip(out)
}

func (t *redirectTransport) builtHosts() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]string(nil), t.hosts...)
}

// newTestClient wires a Client to the given test server with a fast backoff
// schedule and a non-throttling limiter, so tests are deterministic and quick.
func newTestClient(t *testing.T, srv *httptest.Server) (*Client, *redirectTransport) {
	t.Helper()
	c, err := New("api", 1000)
	if err != nil {
		t.Fatalf("New: unexpected error: %v", err)
	}
	target, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("parse test server URL: %v", err)
	}
	rt := &redirectTransport{target: target, inner: http.DefaultTransport}
	c.http = &http.Client{Transport: rt, Timeout: 5 * time.Second}
	c.backoffs = []time.Duration{time.Millisecond, time.Millisecond, time.Millisecond}
	return c, rt
}

func TestNew_RejectsNonPositiveRate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		rps     float64
		wantErr bool
	}{
		{"negative", -1, true},
		{"zero", 0, true},
		{"nan", math.NaN(), true}, // NaN<=0 is false in Go — must be caught explicitly
		{"positive", 5, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c, err := New("api", tc.rps)
			if tc.wantErr {
				if !errors.Is(err, ErrNonPositiveRate) {
					t.Fatalf("rps=%g: want ErrNonPositiveRate, got %v", tc.rps, err)
				}
				if c != nil {
					t.Fatalf("rps=%g: want nil client on error, got %v", tc.rps, c)
				}
				return
			}
			if err != nil {
				t.Fatalf("rps=%g: unexpected error: %v", tc.rps, err)
			}
			if c == nil {
				t.Fatalf("rps=%g: want client, got nil", tc.rps)
			}
		})
	}
}

func TestDo_SetsMandatoryParamsCallerCannotOverride(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c, rt := newTestClient(t, srv)

	// Caller tries to override the transport-mandated TYPE and JSON params.
	resp, err := c.Do(context.Background(), Request{
		Type: "rosters",
		Year: "2026",
		Params: map[string]string{
			"L":    "14432",
			"JSON": "0",          // exact-case override — must be ignored
			"TYPE": "players",    // exact-case override — must be ignored
			"type": "admin_dump", // case-variant smuggle — must be dropped, not added
		},
	})
	if err != nil {
		t.Fatalf("Do: unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: want 200, got %d", resp.StatusCode)
	}

	rt.mu.Lock()
	q := rt.query
	rt.mu.Unlock()
	if got := q.Get("JSON"); got != "1" {
		t.Errorf("JSON param: want forced 1, got %q", got)
	}
	if got := q.Get("TYPE"); got != "rosters" {
		t.Errorf("TYPE param: caller override leaked, want rosters, got %q", got)
	}
	if _, smuggled := q["type"]; smuggled {
		t.Errorf("case-variant key 'type' was smuggled into the query: %v", q["type"])
	}
}

func TestDo_BackoffSucceedsAfter429(t *testing.T) {
	t.Parallel()
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if atomic.AddInt32(&calls, 1) <= 2 { // 429 twice, then succeed
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv)
	resp, err := c.Do(context.Background(), Request{Type: "rosters", Year: "2026", Params: map[string]string{"L": "14432"}})
	if err != nil {
		t.Fatalf("Do: unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status after backoff: want 200, got %d", resp.StatusCode)
	}
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Fatalf("server calls: want 3 (2x429 + 1x200), got %d — bounded backoff, not a storm", got)
	}
}

func TestDo_ExhaustedBackoffReturnsErrorBounded(t *testing.T) {
	t.Parallel()
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusTooManyRequests) // always 429
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv) // 3-entry backoff schedule
	_, err := c.Do(context.Background(), Request{Type: "rosters", Year: "2026", Params: map[string]string{"L": "14432"}})
	if err == nil {
		t.Fatal("Do: want error on persistent 429, got nil")
	}
	// 1 initial attempt + len(backoffs) retries = 4. Proves retries are bounded,
	// not an infinite storm against a rate-limited server.
	if got := atomic.LoadInt32(&calls); got != 4 {
		t.Fatalf("server calls: want 4 (1 + 3 retries), got %d", got)
	}
}

func TestDo_ContextCancelledDuringBackoff(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv)
	c.backoffs = []time.Duration{time.Hour} // long enough that only cancellation can end it

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	_, err := c.Do(ctx, Request{Type: "rosters", Year: "2026", Params: map[string]string{"L": "14432"}})
	if err == nil {
		t.Fatal("Do: want cancellation error, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("Do did not abort the hour-long backoff on cancel (took %s) — timer not stopped", elapsed)
	}
}

func TestDiscoverHost_RecoversFromStaleCachedHost(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"league":{"baseURL":"https://www99.myfantasyleague.com/2026/home/14432"}}`))
	}))
	defer srv.Close()

	c, rt := newTestClient(t, srv)
	c.host = "www47-stale-and-down" // a cached host that must not block re-discovery

	if err := c.DiscoverHost(context.Background(), "2026", "14432"); err != nil {
		t.Fatalf("DiscoverHost: unexpected error: %v", err)
	}

	// The discovery call must target the canonical api host, NOT the stale cache.
	hosts := rt.builtHosts()
	if len(hosts) != 1 {
		t.Fatalf("want exactly 1 discovery request, got %d (%v)", len(hosts), hosts)
	}
	if hosts[0] != "api.myfantasyleague.com" {
		t.Fatalf("discovery routed to stale host: want api.myfantasyleague.com, got %q", hosts[0])
	}

	// And the freshly discovered host must be cached.
	c.mu.RLock()
	got := c.host
	c.mu.RUnlock()
	if got != "www99" {
		t.Fatalf("cached host after discovery: want www99, got %q", got)
	}
}

// TestDo_HostRouting exercises every branch of the host-selection switch in Do.
// Without this, deleting the host=="" fallback (the pre-discovery case) survives
// as a mutant (Gemini review Q5).
func TestDo_HostRouting(t *testing.T) {
	t.Parallel()
	const apiHost = "api.myfantasyleague.com"
	tests := []struct {
		name       string
		cachedHost string
		req        Request
		wantHost   string
	}{
		{
			name:       "league type forces api even with a cached host",
			cachedHost: "www99",
			req:        Request{Type: "league", Year: "2026", Params: map[string]string{"L": "14432"}},
			wantHost:   apiHost,
		},
		{
			name:       "empty L forces api (non-league-specific call)",
			cachedHost: "www99",
			req:        Request{Type: "players", Year: "2026"},
			wantHost:   apiHost,
		},
		{
			name:       "league-specific call before discovery falls back to api",
			cachedHost: "",
			req:        Request{Type: "rosters", Year: "2026", Params: map[string]string{"L": "14432"}},
			wantHost:   apiHost,
		},
		{
			name:       "league-specific call after discovery uses the cached host",
			cachedHost: "www99",
			req:        Request{Type: "rosters", Year: "2026", Params: map[string]string{"L": "14432"}},
			wantHost:   "www99.myfantasyleague.com",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			defer srv.Close()

			c, rt := newTestClient(t, srv)
			c.host = tc.cachedHost

			if _, err := c.Do(context.Background(), tc.req); err != nil {
				t.Fatalf("Do: unexpected error: %v", err)
			}
			hosts := rt.builtHosts()
			if len(hosts) != 1 {
				t.Fatalf("want 1 request, got %d (%v)", len(hosts), hosts)
			}
			if hosts[0] != tc.wantHost {
				t.Fatalf("routed host: want %q, got %q", tc.wantHost, hosts[0])
			}
		})
	}
}

// TestDo_ConcurrentUseIsRaceFree fires many goroutines at one shared Client,
// mixing Do (reads c.host under RLock) and DiscoverHost (writes c.host under
// Lock). Run under -race + goleak, it guards the host-cache mutex and proves the
// shared client is safe for the concurrent fetchers that inherit this template.
func TestDo_ConcurrentUseIsRaceFree(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"league":{"baseURL":"https://www50.myfantasyleague.com/2026/home/14432"}}`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv)

	const goroutines = 50
	var wg sync.WaitGroup
	errs := make(chan error, goroutines)
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				_, err := c.Do(context.Background(), Request{Type: "rosters", Year: "2026", Params: map[string]string{"L": "14432"}})
				if err != nil {
					errs <- err
				}
				return
			}
			if err := c.DiscoverHost(context.Background(), "2026", "14432"); err != nil {
				errs <- err
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("concurrent call failed: %v", err)
	}
}
