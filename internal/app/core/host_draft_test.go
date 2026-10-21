package core_test

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/core"
	"github.com/secureprospective/TheWarRoom/internal/app/core/coretest"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/draftreports"
	"github.com/secureprospective/TheWarRoom/internal/lab"
)

func draftState(t *testing.T) (*core.State, *coretest.Transport) {
	t.Helper()
	bodies := map[string]string{}
	for export, name := range map[string]string{"draftResults": "results-2026.json", "futureDraftPicks": "future-2026.json"} {
		body, err := os.ReadFile(filepath.Join("..", "..", "ingestion", "draftreports", "testdata", name))
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

func TestDraftFeedsDetachedAndNilDistinct(t *testing.T) {
	s, _ := draftState(t)
	s.RefreshDraftReportsFeeds(s.Ctx)
	results, future := s.DraftFeeds()
	assertCounterpartiesDetached(t, reflect.ValueOf(results.Value), reflect.ValueOf(s.SeasonDraftResults.Value), "results")
	assertCounterpartiesDetached(t, reflect.ValueOf(future.Value), reflect.ValueOf(s.SeasonFuturePicks.Value), "future")
	results.Value[0].Franchise = "changed"
	*results.Value[0].Player = *results.Value[1].Player
	future.Value[0].Owner = "changed"
	again, next := s.DraftFeeds()
	if again.Value[0].Franchise == "changed" || next.Value[0].Owner == "changed" {
		t.Fatal("snapshot aliases shared state")
	}
	s.SeasonDraftResults.Value, s.SeasonFuturePicks.Value = nil, []draftreports.FuturePick{}
	results, future = s.DraftFeeds()
	if results.Value != nil || future.Value == nil {
		t.Fatal("nil and empty collapsed")
	}
}

func TestDraftFactsDetached(t *testing.T) {
	s := coretest.New(t).Build()
	d, err := lab.Load()
	if err != nil {
		t.Fatal(err)
	}
	s.LabData = func() (*lab.Data, error) { return d, nil }
	got, _, err := s.DraftFacts()
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"Draft", "Trades", "Delivered"} {
		left, right := reflect.ValueOf(got).Elem().FieldByName(field), reflect.ValueOf(d).Elem().FieldByName(field)
		if !reflect.DeepEqual(left.Interface(), right.Interface()) {
			t.Fatal("facts lost", field)
		}
		assertCounterpartiesDetached(t, left, right, field)
	}
	*got.Draft[0].Wins[0] = -9
	got.Draft[0].SlotWins[0] = -9
	if *d.Draft[0].Wins[0] == -9 || d.Draft[0].SlotWins[0] == -9 {
		t.Fatal("nested draft storage shared")
	}
}

func TestDraftRefreshReadWhileBlockedAndSerializer(t *testing.T) {
	s, transport := draftState(t)
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
	go func() { first <- s.RefreshDraftReports(true) }()
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
	go func() { second <- s.RefreshDraftReports(true) }()
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
	if err := s.RefreshDraftReports(false); err != nil {
		t.Fatal(err)
	}
	requests, _ = transport.Requests()
	if len(requests) != 4 {
		t.Fatal("fresh feeds refetched")
	}
	transport.Fail("draftResults")
	if err := s.RefreshDraftReports(true); err != nil {
		t.Fatal(err)
	}
	results, future := s.DraftFeeds()
	if len(results.Value) != 160 || results.RefreshError == "" || len(future.Value) != 320 || future.RefreshError != "" {
		t.Fatal("last good copies not independent")
	}
}

func TestDraftPublicationRaceAndJointCapture(t *testing.T) {
	s, _ := draftState(t)
	s.RefreshDraftReportsFeeds(s.Ctx)
	var workers sync.WaitGroup
	workers.Go(func() {
		for range 30 {
			s.RefreshMu.Lock()
			s.RefreshDraftReportsFeeds(s.Ctx)
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
