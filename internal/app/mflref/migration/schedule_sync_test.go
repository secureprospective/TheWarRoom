package migration

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

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
	s.syncFollowedLineups(t.Context(), ledger, "league", "owner", 2026, "reference")
	s.syncFollowedLineups(t.Context(), ledger, "league", "owner", 2026, "reference")
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
