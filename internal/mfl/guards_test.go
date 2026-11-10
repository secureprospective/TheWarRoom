package mfl

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"
)

func TestLive_MFLSmoke(t *testing.T) {
	if os.Getenv("TWR_LIVE_MFL") != "1" {
		t.Skip("live MFL smoke test: set TWR_LIVE_MFL=1 to run (makes real network calls)")
	}

	const (
		year     = "2026"
		leagueID = "14432"
	)

	c, err := New("api", 2)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	tr := http.DefaultTransport.(*http.Transport).Clone()
	c.http.Transport = tr
	t.Cleanup(tr.CloseIdleConnections)

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()

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

func TestDiscoverHost_RefusesForeignBaseURL(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"league":{"baseURL":"https://www47.myfantasyleague.com.evil.example/2026/home/14432"}}`))
	}))
	defer srv.Close()
	c, _ := newTestClient(t, srv)
	if err := c.DiscoverHost(context.Background(), "2026", "14432"); !errors.Is(err, ErrDisallowedHost) {
		t.Fatalf("DiscoverHost: want ErrDisallowedHost, got %v", err)
	}
	if c.host != "api" {
		t.Fatalf("host changed to %q", c.host)
	}
}
