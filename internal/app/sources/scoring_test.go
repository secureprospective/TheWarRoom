package sources

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/archive"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

type scoringTransport struct {
	reads int
	fail  bool
	bad   bool
}

func (f *scoringTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	f.reads++
	if f.fail {
		return nil, errors.New("offline")
	}
	path := "../../fantasyscore/source/testdata/weeks.csv"
	if r.URL.String() == source.PlaysURL(2025) {
		path = "../../fantasyscore/source/testdata/plays.csv"
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if r.URL.String() == source.PlaysURL(2025) {
		var out bytes.Buffer
		w := gzip.NewWriter(&out)
		if _, err := w.Write(body); err != nil {
			return nil, err
		}
		if err := w.Close(); err != nil {
			return nil, err
		}
		body = out.Bytes()
	}
	if f.bad {
		body = []byte("damaged")
	}
	return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(bytes.NewReader(body))}, nil
}

func TestSyncScoringAcquiresAndCachesSeason(t *testing.T) {
	t.Parallel()
	pools, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "history.db"), history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := pools.Close(); err != nil {
			t.Error(err)
		}
	})
	reg, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	store := history.New(pools, reg)
	if err := store.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	feed := &scoringTransport{}
	transport := &archive.Transport{Base: feed, Sink: store}
	scope := source.RawStatScope{GSIS: "00-0023459", Season: 2025, Week: 1}
	if err := SyncScoring(t.Context(), store, transport, 2025); err != nil {
		t.Fatal(err)
	}
	weeks, err := store.PlayerWeekStats(t.Context(), scope)
	if err != nil || len(weeks) < 2 || weeks[0].Week == nil || len(weeks[0].Missing) != 0 {
		t.Fatalf("complete held evidence: %+v %v", weeks, err)
	}
	if weeks[0].SHA256 == weeks[1].SHA256 || len(weeks[1].Missing) == 0 {
		t.Fatal("completion overwrote the weekly-only revision")
	}
	if err := SyncScoring(t.Context(), store, transport, 2025); err != nil || feed.reads != 2 {
		t.Fatalf("cache: %d %v", feed.reads, err)
	}
	feed.bad = true
	if _, err := ingestion.Get(t.Context(), &http.Client{Transport: transport}, source.PlaysURL(2025), 64<<20); err != nil {
		t.Fatal(err)
	}
	preserved, err := store.PlayerWeekStats(t.Context(), scope)
	if err != nil || preserved[0].SHA256 != weeks[0].SHA256 {
		t.Fatal("damaged download hid the last validated plays", err)
	}
	feed.bad = false
	feed.fail = true
	if err := SyncScoring(t.Context(), store, transport, 2026); err == nil {
		t.Fatal("offline acquisition passed")
	}
	again, err := store.PlayerWeekStats(t.Context(), scope)
	if err != nil || again[0].SHA256 != weeks[0].SHA256 {
		t.Fatal("failure damaged held season", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := SyncScoring(ctx, store, transport, 2025); err == nil {
		t.Fatal("cancelled acquisition passed")
	}
}

func TestSyncScoringRejectsMalformedPlays(t *testing.T) {
	t.Parallel()
	pools, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "history.db"), history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := pools.Close(); err != nil {
			t.Error(err)
		}
	})
	reg, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	store := history.New(pools, reg)
	if err := store.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	feed := &scoringTransport{bad: true}
	transport := &archive.Transport{Base: feed, Sink: store}
	if err := SyncScoring(t.Context(), store, transport, 2025); err == nil {
		t.Fatal("damaged release accepted")
	}
	if _, ok, err := store.LastLoadOf(t.Context(), source.PlaysURL(2025)); err != nil || ok {
		t.Fatal("invalid plays marked fresh", err)
	}
	feed.bad = false
	if err := SyncScoring(t.Context(), store, transport, 2025); err != nil {
		t.Fatal("failed load did not retry", err)
	}
}
