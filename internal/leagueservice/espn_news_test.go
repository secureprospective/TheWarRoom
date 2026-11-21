package leagueservice

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/espnnews"
)

type newsRoundTrip func(*http.Request) (*http.Response, error)

func (f newsRoundTrip) Do(req *http.Request) (*http.Response, error) { return f(req) }

type memNewsArchive struct {
	mu   sync.Mutex
	rows map[string]struct {
		body []byte
		at   time.Time
	}
}

func newMemNewsArchive() *memNewsArchive {
	return &memNewsArchive{rows: map[string]struct {
		body []byte
		at   time.Time
	}{}}
}

func (m *memNewsArchive) LoadESPNBody(_ context.Context, key string) ([]byte, time.Time, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.rows[key]
	return slices.Clone(row.body), row.at, ok, nil
}

func (m *memNewsArchive) SaveESPNBody(_ context.Context, key string, body []byte, at time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rows[key] = struct {
		body []byte
		at   time.Time
	}{body: slices.Clone(body), at: at}
	return nil
}

type staticDirectory struct {
	espnByMFL map[string]string
}

func (d staticDirectory) SourceID(_ context.Context, playerID, idType string) (string, error) {
	if idType != "espn" {
		return "", nil
	}
	return d.espnByMFL[playerID], nil
}

func (d staticDirectory) SourceIDs(_ context.Context, idType string) (map[string]string, error) {
	if idType != "espn" {
		return map[string]string{}, nil
	}
	out := make(map[string]string, len(d.espnByMFL))
	for k, v := range d.espnByMFL {
		out[k] = v
	}
	return out, nil
}

func readESPNFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "ingestion", "espnnews", "testdata", name))
	if err != nil {
		t.Fatalf("read espn fixture %s: %v", name, err)
	}
	return b
}

func httpOK(body []byte) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(body)),
	}
}

func TestNewsServiceCacheTTLStaleAndNonBlocking(t *testing.T) {
	leagueBody := readESPNFixture(t, "league_news.json")
	bioBody := readESPNFixture(t, "athlete_bio.json")
	pNewsBody := readESPNFixture(t, "player_news_fantasy.json")

	var fail atomic.Bool
	block := make(chan struct{})
	var blockedCalls atomic.Int32
	transport := newsRoundTrip(func(req *http.Request) (*http.Response, error) {
		if fail.Load() {
			return &http.Response{
				StatusCode: http.StatusBadGateway,
				Body:       io.NopCloser(strings.NewReader("down")),
			}, nil
		}
		if req.URL.Query().Get("playerId") == "999" || strings.HasSuffix(req.URL.Path, "/999") {
			blockedCalls.Add(1)
			select {
			case <-block:
			case <-req.Context().Done():
				return nil, req.Context().Err()
			}
		}
		switch {
		case strings.Contains(req.URL.Path, "/athletes/"):
			return httpOK(bioBody), nil
		case strings.Contains(req.URL.Path, "/fantasy/"):
			return httpOK(pNewsBody), nil
		default:
			return httpOK(leagueBody), nil
		}
	})

	clock := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)
	var clockMu sync.Mutex
	archive := newMemNewsArchive()
	ns := NewNewsService(transport, espnnews.DefaultEndpoints(), archive, staticDirectory{})
	ns.now = func() time.Time {
		clockMu.Lock()
		defer clockMu.Unlock()
		return clock
	}
	defer ns.Close()

	// Non-blocking read: even when ESPN blocks on player 999, ReadPlayer returns immediately.
	done := make(chan struct{})
	go func() {
		_, items, asOf, stale := ns.ReadPlayer(context.Background(), "999")
		if len(items) != 0 || !asOf.IsZero() || !stale {
			t.Errorf("cold ReadPlayer = items:%d asOf:%v stale:%v", len(items), asOf, stale)
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("ReadPlayer blocked on ESPN network call")
	}
	close(block)

	// Synchronous refresh populates memory and persistent archive.
	if err := ns.RefreshLeagueNow(context.Background()); err != nil {
		t.Fatal(err)
	}
	items, asOf, stale := ns.ReadLeagueNews(context.Background())
	if len(items) < 3 || asOf.IsZero() || stale {
		t.Fatalf("fresh league news: len=%d asOf=%v stale=%v", len(items), asOf, stale)
	}

	// Failure after TTL keeps the last cached copy and marks stale=true.
	fail.Store(true)
	clockMu.Lock()
	clock = clock.Add(20 * time.Minute)
	clockMu.Unlock()
	if err := ns.RefreshLeagueNow(context.Background()); err == nil {
		t.Fatal("expected ESPN failure error")
	}
	held, heldAt, heldStale := ns.ReadLeagueNews(context.Background())
	if len(held) != len(items) || !heldAt.Equal(asOf) || !heldStale {
		t.Fatalf("stale fallback failed: len=%d at=%v stale=%v", len(held), heldAt, heldStale)
	}

	// Cold NewsService hydrates from the persistent store without touching ESPN.
	ns2 := NewNewsService(nil, espnnews.DefaultEndpoints(), archive, staticDirectory{})
	ns2.now = func() time.Time { return time.Date(2026, 11, 21, 12, 5, 0, 0, time.UTC) }
	defer ns2.Close()
	restored, restoredAt, restoredStale := ns2.ReadLeagueNews(context.Background())
	if len(restored) != len(items) || !restoredAt.Equal(asOf) || restoredStale {
		t.Fatalf("archive hydrate: len=%d at=%v stale=%v", len(restored), restoredAt, restoredStale)
	}

	// Bio TTL is 24h while player news TTL is 15m.
	fail.Store(false)
	clockMu.Lock()
	clock = time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)
	clockMu.Unlock()
	bio, pItems, err := ns.RefreshPlayerNow(context.Background(), "3139477")
	if err != nil || bio.College != "Texas Tech" || len(pItems) != 2 {
		t.Fatalf("RefreshPlayerNow: bio=%+v items=%d err=%v", bio, len(pItems), err)
	}
	clockMu.Lock()
	clock = clock.Add(20 * time.Minute)
	clockMu.Unlock()
	gotBio, gotItems, _, pStale := ns.ReadPlayer(context.Background(), "3139477")
	if gotBio.College != "Texas Tech" || len(gotItems) != 2 || !pStale {
		t.Fatalf("expired news read: bio=%+v items=%d stale=%v", gotBio, len(gotItems), pStale)
	}
}

func TestTargetPlayerNewsAndNewsFeedCommands(t *testing.T) {
	leagueBody := readESPNFixture(t, "league_news.json")
	bioBody := readESPNFixture(t, "athlete_bio.json")
	pNewsBody := readESPNFixture(t, "player_news_fantasy.json")
	transport := newsRoundTrip(func(req *http.Request) (*http.Response, error) {
		switch {
		case strings.Contains(req.URL.Path, "/athletes/"):
			return httpOK(bioBody), nil
		case strings.Contains(req.URL.Path, "/fantasy/"):
			return httpOK(pNewsBody), nil
		default:
			return httpOK(leagueBody), nil
		}
	})
	dir := staticDirectory{espnByMFL: map[string]string{
		"13116": "3139477",
		"14836": "3919512",
		"15241": "4035687",
	}}
	ns := NewNewsService(transport, espnnews.DefaultEndpoints(), nil, dir)
	defer ns.Close()
	if err := ns.RefreshLeagueNow(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ns.RefreshPlayerNow(context.Background(), "3139477"); err != nil {
		t.Fatal(err)
	}

	var accesses []commissioner.Access
	admit := admissionFunc(func(_ context.Context, a commissioner.Access) error {
		accesses = append(accesses, a)
		return nil
	})
	service := NewHosted(nflStatusTestHost{}, admit).WithNews(ns)

	// Mapped player returns bio + items + espnId.
	pRead, err := service.TargetPlayerNews(RawPlayerReadScope{League: "native.test", Player: "13116"})
	if err != nil {
		t.Fatal(err)
	}
	if pRead.ESPNID != "3139477" || pRead.Bio.Height != `6' 2"` || len(pRead.Items) != 2 || pRead.Stale {
		t.Fatalf("unexpected TargetPlayerNews: %+v", pRead)
	}
	if accesses[0].Operation != PlayerNewsRead || accesses[0].Mode != "read" {
		t.Fatalf("wrong admission for player.news.read: %+v", accesses[0])
	}

	// Unmapped player returns empty items + plain reason, never an error.
	unmapped, err := service.TargetPlayerNews(RawPlayerReadScope{League: "native.test", Player: "99999"})
	if err != nil {
		t.Fatalf("unmapped player errored: %v", err)
	}
	if unmapped.ESPNID != "" || len(unmapped.Items) != 0 || unmapped.Reason == "" {
		t.Fatalf("unexpected unmapped response: %+v", unmapped)
	}

	// League news feed tags items with league MFL player IDs via crosswalk.
	feed, err := service.TargetNewsFeed("native.test")
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) < 3 || feed.Stale || feed.AsOf == "" {
		t.Fatalf("unexpected TargetNewsFeed: %+v", feed)
	}
	var tagged bool
	for _, item := range feed.Items {
		if slices.Contains(item.Players, "14836") || slices.Contains(item.Players, "15241") {
			tagged = true
		}
	}
	if !tagged {
		t.Fatalf("expected league player IDs tagged on feed items: %+v", feed.Items)
	}

	// Denied non-member is refused.
	denied := errors.New("denied")
	outsider := NewHosted(nflStatusTestHost{}, admissionFunc(func(context.Context, commissioner.Access) error {
		return denied
	})).WithNews(ns)
	if _, err := outsider.TargetNewsFeed("native.test"); !errors.Is(err, denied) {
		t.Fatalf("outsider feed not refused: %v", err)
	}
	scope := RawPlayerReadScope{League: "native.test", Player: "13116"}
	if _, err := outsider.TargetPlayerNews(scope); !errors.Is(err, denied) {
		t.Fatalf("outsider player news not refused: %v", err)
	}
}
