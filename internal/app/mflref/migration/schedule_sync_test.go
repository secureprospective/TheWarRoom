package migration

import (
	"context"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/store/history"

	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
)

type unavailableFeed struct{}

func (unavailableFeed) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("offline fixture")
}

func TestScheduleSyncWithoutHistory(t *testing.T) {
	(ScheduleSync{}).Sync(t.Context())
}

func TestFollowedLineupsWorkerLifetime(t *testing.T) {
	client, err := mfl.New("api", 10000, mfl.WithTransport(unavailableFeed{}))
	if err != nil {
		t.Fatal(err)
	}
	work, cancel := context.WithCancel(t.Context())
	defer cancel()
	var running, started atomic.Bool
	var queued []func()
	s := ScheduleSync{Client: client, WorkContext: func() context.Context { return work }, Running: &running, Started: &started,
		StartWorker: func(f func()) { queued = append(queued, f) }}
	ledger := leagueservice.MigrationLedger{}
	s.syncFollowedLineups(t.Context(), ledger, []followedLeague{{"league", "owner", 2026, "reference"}, {"other", "owner", 2026, "reference"}})
	s.syncFollowedLineups(t.Context(), ledger, []followedLeague{{"league", "owner", 2026, "reference"}, {"other", "owner", 2026, "reference"}})
	if len(queued) != 1 || !running.Load() || started.Load() {
		t.Fatalf("before worker: queued=%d running=%t started=%t", len(queued), running.Load(), started.Load())
	}
	cancel()
	queued[0]()
	if running.Load() || !started.Load() {
		t.Fatal("cancelled first-launch worker did not release its guard")
	}
	if playedLineupsTimeout != 10*time.Minute || playedLineupsLaunchDelay != 5*time.Minute {
		t.Fatal("reference import timing changed")
	}
}

// Every eligible league must reach the same single history worker, even when an earlier feed fails.
func TestHistoryWorkerProcessesEveryLeague(t *testing.T) {
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
	var leagues []followedLeague
	for _, id := range []string{"native.one", "native.two"} {
		raw := commissioner.RawCommand{League: id, Command: commissioner.Create, Args: commissioner.RawArgs{Name: id, Preset: commissioner.PresetEmpty, Season: 2026, FranchiseCount: 2}, Reason: "start", Request: "create", Effective: "now", Visibility: "league"}
		if _, err := store.ExecuteCommissioner(t.Context(), raw, "creator", nil); err != nil {
			t.Fatal(err)
		}
		leagues = append(leagues, followedLeague{id, "creator", 2026, id})
	}
	feed := &countedFeed{reads: map[string]int{}}
	client, err := mfl.New("api", 10000, mfl.WithTransport(feed))
	if err != nil {
		t.Fatal(err)
	}
	var running, started atomic.Bool
	started.Store(true)
	var queued []func()
	s := ScheduleSync{History: store, Client: client, Now: time.Now, WorkContext: func() context.Context { return t.Context() }, Running: &running, Started: &started, StartWorker: func(f func()) { queued = append(queued, f) }}
	s.syncFollowedLineups(t.Context(), leagueservice.MigrationLedger{Store: store}, leagues)
	if len(queued) != 1 {
		t.Fatalf("workers %d", len(queued))
	}
	queued[0]()
	for _, league := range leagues {
		if feed.reads[league.source] == 0 {
			t.Fatalf("dropped %s: %v", league.league, feed.reads)
		}
	}
	if running.Load() {
		t.Fatal("worker guard retained")
	}
}

type countedFeed struct{ reads map[string]int }

func (f *countedFeed) RoundTrip(r *http.Request) (*http.Response, error) {
	f.reads[r.URL.Query().Get("L")]++
	return &http.Response{StatusCode: http.StatusBadGateway, Body: io.NopCloser(strings.NewReader("offline")), Header: make(http.Header)}, nil
}
