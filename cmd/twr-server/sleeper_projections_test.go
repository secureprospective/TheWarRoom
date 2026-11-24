package main

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

type sleeperProjectionTransport struct {
	body  string
	calls int
}

func (t *sleeperProjectionTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.calls++
	if req.Header.Get("User-Agent") != "TheWarRoom-rehearsal" {
		panic("unexpected user-agent")
	}
	return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(t.body)), Request: req}, nil
}

func TestSleeperProjectionFetchArchivesDeduplicatedBodies(t *testing.T) {
	ctx := t.Context()
	base := newOneFileServer(t)
	pools, err := db.Open(ctx, base.path, history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	registry, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	store := history.New(pools, registry)
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
	stop := startHostedSleeperProjections(t.Context(), nil, false)
	stop()
}
