package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

type sleeperProjectionTransport struct {
	body  string
	calls int
	seen  atomic.Int64
}

func (t *sleeperProjectionTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.calls++
	t.seen.Add(1)
	if req.Header.Get("User-Agent") != "TheWarRoom-rehearsal" {
		panic("unexpected user-agent")
	}
	return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(t.body)), Request: req}, nil
}

func TestSleeperProjectionFetchArchivesDeduplicatedBodies(t *testing.T) {
	ctx := t.Context()
	store, pools := sleeperProjectionTestStore(t)
	transport := &sleeperProjectionTransport{body: "same projection body"}
	fetcher := newSleeperProjectionFetcher(store, transport)
	for range 2 {
		if err := fetcher.Fetch(ctx, 2026, 12); err != nil {
			t.Fatal(err)
		}
	}
	const target = "https://api.sleeper.app/projections/nfl/2026/12?season_type=regular"
	assertSleeperArchive(t, pools, store, target, 2, 1, "same projection body")
	transport.body = "changed projection body"
	if err := fetcher.Fetch(ctx, 2026, 12); err != nil {
		t.Fatal(err)
	}
	assertSleeperArchive(t, pools, store, target, 3, 2, "changed projection body")
}

func assertSleeperArchive(t *testing.T, pools *db.Pools, store *history.Store, target string, logs, bodies int, latest string) {
	t.Helper()
	count, _, err := store.ArchiveMark(t.Context(), "/projections/nfl/2026/12")
	if err != nil || count != int64(logs) {
		t.Fatalf("fetch log count=%d err=%v, want %d", count, err, logs)
	}
	var stored int
	if err := pools.Read().QueryRowContext(t.Context(), `SELECT count(*) FROM raw_archive`).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored != bodies {
		t.Fatalf("unique archived bodies=%d, want %d", stored, bodies)
	}
	found := map[string]bool{}
	_, err = store.ArchivedBodies(t.Context(), target, func(_ string, body []byte, _ time.Time) bool { found[string(body)] = true; return false })
	if err != nil {
		t.Fatal(err)
	}
	if !found[latest] {
		t.Fatalf("latest body %q not retained", latest)
	}
	if logs > 1 && !found["same projection body"] {
		t.Fatal("old snapshot was not retained")
	}
}

func TestSleeperProjectionFlagOffMakesNoRequest(t *testing.T) {
	week := func(context.Context) (int, int, error) { return 2026, 12, nil }
	store, _ := sleeperProjectionTestStore(t)
	run := func(enabled bool) int64 {
		transport := &sleeperProjectionTransport{body: "[]"}
		stop := sleeperProjectionWorker{base: transport, week: week}.start(t.Context(), store, enabled)
		deadline := time.Now().Add(2 * time.Second)
		for enabled && transport.seen.Load() < 2 && time.Now().Before(deadline) {
			time.Sleep(10 * time.Millisecond)
		}
		if !enabled {
			time.Sleep(200 * time.Millisecond)
		}
		stop()
		return transport.seen.Load()
	}
	if got := run(true); got < 2 {
		t.Fatalf("flag on: %d requests, want the current and next week", got)
	}
	if got := run(false); got != 0 {
		t.Fatalf("flag off: %d requests, want none", got)
	}
}

func sleeperProjectionTestStore(t *testing.T) (*history.Store, *db.Pools) {
	t.Helper()
	base := newOneFileServer(t)
	pools, err := db.Open(t.Context(), base.path, history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	registry, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	return history.New(pools, registry), pools
}

// The offline report runs on a server database; a league with no held Sleeper snapshot reports zero players
// rather than failing, and a missing week is refused in words.
func TestSleeperAccuracyCommandPrintsReport(t *testing.T) {
	host, _, _ := scheduleRefreshFixture(t)
	if err := host.files.Close(); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(host.dir, "server.db")
	var out bytes.Buffer
	if err := runSleeperAccuracy(t.Context(), []string{"-db", dbPath, "-league", "native.hosted", "-season", "2026"}, &out); err == nil ||
		!strings.Contains(err.Error(), "-week") {
		t.Fatalf("no week: err = %v, want the usage naming -week", err)
	}
	args := []string{"-db", dbPath, "-league", "native.hosted", "-season", "2026", "-week", "12"}
	if err := runSleeperAccuracy(t.Context(), args, &out); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Players: 0", "Mean miss: 0.00", "Bias: 0.00"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output %q lacks %q", out.String(), want)
		}
	}
}
