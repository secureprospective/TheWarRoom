package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/now"
	"github.com/secureprospective/TheWarRoom/internal/archive"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

const (
	sleeperProjectionCadence = 6 * time.Hour
	sleeperProjectionTimeout = 2 * time.Minute
	sleeperProjectionURL     = "https://api.sleeper.app/projections/nfl/%d/%d?season_type=regular"
)

type sleeperProjectionFetcher struct {
	store  *history.Store
	client *http.Client
}

func newSleeperProjectionFetcher(store *history.Store, base http.RoundTripper) sleeperProjectionFetcher {
	return sleeperProjectionFetcher{store: store, client: &http.Client{
		Transport: &archive.Transport{Base: base, Sink: store}, Timeout: sleeperProjectionTimeout,
	}}
}

func (f sleeperProjectionFetcher) Fetch(ctx context.Context, season, week int) error {
	address := fmt.Sprintf(sleeperProjectionURL, season, week)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return fmt.Errorf("build Sleeper projections request: %w", err)
	}
	req.Header.Set("User-Agent", "TheWarRoom-rehearsal")
	resp, err := f.client.Do(req)
	if err != nil {
		return fmt.Errorf("fetch Sleeper projections %d/%d: %w", season, week, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		return fmt.Errorf("read Sleeper projections %d/%d: %w", season, week, err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch Sleeper projections %d/%d: HTTP %d", season, week, resp.StatusCode)
	}
	return nil
}

// sleeperProjectionWorker is what the worker reaches outside itself; tests swap the transport, the first wait
// and the week lookup.
type sleeperProjectionWorker struct {
	base  http.RoundTripper
	first time.Duration
	week  func(context.Context) (int, int, error)
}

func startHostedSleeperProjections(ctx context.Context, store *history.Store, enabled bool) func() {
	return sleeperProjectionWorker{first: 2 * time.Minute, week: currentNFLWeek}.start(ctx, store, enabled)
}

func (w sleeperProjectionWorker) start(ctx context.Context, store *history.Store, enabled bool) func() {
	if !enabled {
		return func() {}
	}
	workerCtx, stop := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		fetcher := newSleeperProjectionFetcher(store, w.base)
		refresh := func() error {
			season, week, err := w.week(workerCtx)
			if err != nil {
				return err
			}
			for _, target := range []int{week, week + 1} {
				attempt, cancel := context.WithTimeout(workerCtx, sleeperProjectionTimeout)
				err := fetcher.Fetch(attempt, season, target)
				cancel()
				if err != nil {
					return err
				}
				if target == week {
					select {
					case <-workerCtx.Done():
						return workerCtx.Err()
					case <-time.After(time.Second):
					}
				}
			}
			return nil
		}
		wait := w.first
		for {
			select {
			case <-workerCtx.Done():
				return
			case <-time.After(wait):
			}
			if err := refresh(); err != nil {
				log.Printf("hosted Sleeper projections: %v", err)
				wait *= 2
				if wait > sleeperProjectionCadence {
					wait = sleeperProjectionCadence
				}
			} else {
				log.Print("hosted Sleeper projections: refreshed current and next NFL week")
				wait = sleeperProjectionCadence
			}
		}
	}()
	return func() { stop(); <-done }
}

// currentNFLWeek uses the same ESPN scoreboard parsing and scope selection as liveCollector.
func currentNFLWeek(ctx context.Context) (int, int, error) {
	endpoints, err := now.LiveEndpoints()
	if err != nil {
		return 0, 0, fmt.Errorf("sleeper projections: get live endpoints: %w", err)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	body, err := livegame.HTTPFetch(client, endpoints)(ctx, "espn", "")
	if err != nil {
		return 0, 0, fmt.Errorf("sleeper projections: read current scoreboard: %w", err)
	}
	games, err := livegame.ParseScoreboard(body)
	if err != nil {
		return 0, 0, fmt.Errorf("sleeper projections: parse current scoreboard: %w", err)
	}
	season, week, _, ok := boardScope(games, false)
	if !ok {
		return 0, 0, fmt.Errorf("sleeper projections: scoreboard has no current regular-season week")
	}
	return season, week, nil
}
