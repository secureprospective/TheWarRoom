package composition

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/core"
	"github.com/secureprospective/TheWarRoom/internal/app/core/coretest"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/draftreports"
)

func policyDraftState(t *testing.T) (*core.State, *coretest.Transport) {
	t.Helper()
	bodies := map[string]string{}
	for export, name := range map[string]string{"draftResults": "results-2026.json", "futureDraftPicks": "future-2026.json"} {
		body, err := os.ReadFile(filepath.Join("internal", "ingestion", "draftreports", "testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		bodies[export] = string(body)
	}
	s := coretest.New(t).Build()
	transport, client := coretest.NewTransport(t, bodies)
	s.MFLClient = client
	return s, transport
}
func TestDraftRefreshReadWhileBlockedAndSerializer(t *testing.T) {
	s, transport := policyDraftState(t)
	entered, release := make(chan struct{}), make(chan struct{})
	var pauses, events atomic.Int32
	s.SeasonChanged = func(context.Context) { events.Add(1) }
	s.SeasonPause = func(ctx context.Context, _ time.Duration) error {
		if pauses.Add(1) == 1 {
			close(entered)
			select {
			case <-release:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	}
	first, second := make(chan error, 1), make(chan error, 1)
	go func() { first <- (draftHost{State: s, app: &App{core: s}}).RefreshDraftReports(true) }()
	<-entered
	if s.RefreshMu.TryLock() {
		s.RefreshMu.Unlock()
		t.Fatal("refresh serializer released during pause")
	}
	read := make(chan struct{})
	go func() { s.DraftFeeds(); close(read) }()
	select {
	case <-read:
	case <-time.After(2 * time.Second):
		close(release)
		t.Fatal("read blocked on refresh I/O")
	}
	go func() { second <- (draftHost{State: s, app: &App{core: s}}).RefreshDraftReports(true) }()
	requests, _ := transport.Requests()
	if !slices.Equal(requests, []string{"draftResults"}) {
		t.Fatal(requests)
	}
	close(release)
	if err := <-first; err != nil {
		t.Fatal(err)
	}
	if err := <-second; err != nil {
		t.Fatal(err)
	}
	requests, _ = transport.Requests()
	if !slices.Equal(requests, []string{"draftResults", "futureDraftPicks", "draftResults", "futureDraftPicks"}) || events.Load() != 0 {
		t.Fatal("refresh order/notification changed", requests, events.Load())
	}
	if err := (draftHost{State: s, app: &App{core: s}}).RefreshDraftReports(false); err != nil {
		t.Fatal(err)
	}
	requests, _ = transport.Requests()
	if len(requests) != 4 {
		t.Fatal("fresh feeds refetched")
	}
	transport.Fail("draftResults")
	if err := (draftHost{State: s, app: &App{core: s}}).RefreshDraftReports(true); err != nil {
		t.Fatal(err)
	}
	results, future := s.DraftFeeds()
	if len(results.Value) != 160 || results.RefreshError == "" || len(future.Value) != 320 || future.RefreshError != "" {
		t.Fatal("last good copies not independent")
	}
}

func TestDraftPublicationRaceAndJointCapture(t *testing.T) {
	s, _ := policyDraftState(t)
	(&App{core: s}).refreshDraftReportsFeeds(s.Ctx)
	var workers sync.WaitGroup
	workers.Go(func() {
		for range 30 {
			s.RefreshMu.Lock()
			(&App{core: s}).refreshDraftReportsFeeds(s.Ctx)
			s.RefreshMu.Unlock()
		}
	})
	for range 3 {
		workers.Go(func() {
			for range 100 {
				results, future := s.DraftFeeds()
				results.Value[0].Franchise = "local"
				future.Value[0].Owner = "local"
			}
		})
	}
	workers.Wait()
	// Metadata and shared feeds retain the original single SeasonMu capture.
	workers.Go(func() {
		for i := range 200 {
			s.SeasonMu.Lock()
			fresh := domain.Freshness{FetchedAt: time.Unix(int64(i), 0).UTC().Format(time.RFC3339)}
			core.HoldFeed(&s.SeasonDraftResults, []draftreports.Pick{{Round: i}}, fresh, nil)
			core.HoldFeed(&s.SeasonFuturePicks, []draftreports.FuturePick{{Round: i}}, fresh, nil)
			s.SeasonMu.Unlock()
		}
	})
	workers.Go(func() {
		for range 400 {
			results, future := s.DraftFeeds()
			// Before the first synthetic publication, fixture rounds need not match.
			if len(results.Value) == 1 && (results.Value[0].Round != future.Value[0].Round || results.FetchedAt != future.FetchedAt) {
				t.Error("torn joint capture")
			}
		}
	})
	workers.Wait()
}
