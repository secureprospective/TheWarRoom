package composition

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/core/coretest"
	"github.com/secureprospective/TheWarRoom/internal/app/season"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/marketfeeds"
	"github.com/secureprospective/TheWarRoom/internal/lab"
)

func TestMarketRefreshSerializesAndReadsContinue(t *testing.T) {
	s := coretest.New(t).Build()
	var calls atomic.Int32
	entered, release := make(chan struct{}, 2), make(chan struct{})
	forced := make(chan bool, 2)
	s.RefreshMarketFeeds = func(ctx context.Context, force bool) {
		if _, ok := ctx.Deadline(); !ok {
			t.Error("refresh has no deadline")
		}
		if s.RefreshMu.TryLock() {
			s.RefreshMu.Unlock()
			t.Error("refresh not serialized")
		}
		calls.Add(1)
		forced <- force
		entered <- struct{}{}
		<-release
	}
	done := make(chan error, 2)
	go func() { done <- (marketHost{State: s}).RefreshMarket(true) }()
	waitRace(t, entered)
	// Held reads complete while the refresh's I/O is blocked.
	read := make(chan struct{})
	go func() {
		_ = s.MarketWeek()
		_ = s.MarketLookup()
		_ = s.CaptureMarket(func() map[string]season.Feed[[]marketfeeds.Feed] { return nil })
		close(read)
	}()
	waitRace(t, read)
	go func() { done <- (marketHost{State: s}).RefreshMarket(false) }()
	time.Sleep(20 * time.Millisecond)
	if calls.Load() != 1 {
		t.Fatal("two refreshes entered together")
	}
	release <- struct{}{}
	waitRace(t, entered)
	release <- struct{}{}
	for range 2 {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
	if <-forced != true || <-forced != false {
		t.Fatal("force not passed through")
	}
	if !s.RefreshMu.TryLock() {
		t.Fatal("refresh retained serializer")
	}
	s.RefreshMu.Unlock()
}

func TestMarketRefreshStartupDenialAndLabLoad(t *testing.T) {
	s := coretest.New(t).Build()
	s.RefreshMarketFeeds = func(context.Context, bool) { t.Fatal("refreshed before ready") }
	s.StartupErr = errors.New("fixture startup failed")
	if err := (marketHost{State: s}).RefreshMarket(true); err == nil || err.Error() != "market refresh: startup: fixture startup failed" {
		t.Fatal(err)
	}
	s.LabData = func() (*lab.Data, error) { return nil, errors.New("fingerprint mismatch") }
	if _, err := s.LabFacts(); err == nil || err.Error() != "fingerprint mismatch" {
		t.Fatal("shared loader error rewritten", err)
	}
}
