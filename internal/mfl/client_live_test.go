// internal/mfl/client_live_test.go
package mfl

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"
)

// TestLive_MFLSmoke exercises the client against the REAL MyFantasyLeague API.
// It is opt-in: set TWR_LIVE_MFL=1 to run it. It makes outbound network calls and
// depends on MFL being reachable, so it stays out of the default suite, CI, and
// the pre-commit hook — but it still compiles and is linted/vetted with everything
// else (no build tag), so it cannot rot.
//
//	TWR_LIVE_MFL=1 go test -run TestLive_MFLSmoke -v ./internal/mfl/...
//
// This is the live half of the B1 close gate: DiscoverHost resolves the real
// league host while ignoring a stale cached one (T3 finding #2), and Do returns
// 200 against the discovered host.
func TestLive_MFLSmoke(t *testing.T) {
	if os.Getenv("TWR_LIVE_MFL") != "1" {
		t.Skip("live MFL smoke test: set TWR_LIVE_MFL=1 to run (makes real network calls)")
	}

	const (
		year     = "2026"
		leagueID = "14432"
	)

	c, err := New("api", 2) // 2 req/s — comfortably under MFL's limits
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Give this client its OWN transport (cloned from the default so it keeps proxy
	// and timeout settings) and tear down only it. Never CloseIdleConnections on
	// the shared http.DefaultTransport — under `go test ./...` that would sever
	// other tests' pooled connections (Gemini review Q3). Closing this transport's
	// idle conns also drains its goroutines before goleak runs in TestMain.
	tr := http.DefaultTransport.(*http.Transport).Clone()
	c.http.Transport = tr
	t.Cleanup(tr.CloseIdleConnections)

	// Must exceed the client's full 429 backoff cycle (1+2+…+60 ≈ 123s) plus request
	// time, so a single transient 429 self-heals instead of masquerading as a
	// failure (Gemini review Q2). The happy path returns in <1s; a hard network
	// failure still fails fast via the client's own 15s per-request timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()

	// 1) Seed a stale, bogus cached host. DiscoverHost must ignore it (force the
	//    api host) and resolve the real league host — the live proof of T3 #2.
	c.mu.Lock()
	c.host = "www00-does-not-exist"
	c.mu.Unlock()

	if err := c.DiscoverHost(ctx, year, leagueID); err != nil {
		t.Fatalf("DiscoverHost against live MFL: %v", err)
	}

	c.mu.RLock()
	discovered := c.host
	c.mu.RUnlock()
	if discovered == "" || discovered == "www00-does-not-exist" {
		t.Fatalf("DiscoverHost did not resolve a real host, got %q", discovered)
	}
	t.Logf("discovered live league host: %q", discovered)

	// 2) A league-specific call routed through the freshly discovered host.
	resp, err := c.Do(ctx, Request{
		Type:   "rosters",
		Year:   year,
		Params: map[string]string{"L": leagueID},
	})
	if err != nil {
		t.Fatalf("Do(rosters) against live MFL: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Do(rosters): want 200, got %d (body: %.200s)", resp.StatusCode, resp.Body)
	}
	// MFL's PHP backend can return 200 with an HTML maintenance/error page, so a
	// 200 alone is not proof. Require the body to parse as JSON AND carry the
	// rosters payload (Gemini review Q1).
	var probe struct {
		Rosters json.RawMessage `json:"rosters"`
	}
	if err := json.Unmarshal(resp.Body, &probe); err != nil {
		t.Fatalf("Do(rosters): 200 but body is not JSON: %v (body: %.200s)", err, resp.Body)
	}
	if len(probe.Rosters) == 0 {
		t.Fatalf("Do(rosters): 200 JSON but no \"rosters\" key (body: %.200s)", resp.Body)
	}
	t.Logf("rosters 200, %d bytes, rosters payload present", len(resp.Body))
}
