package history

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/deepclone"
)

// readingFlights joins concurrent identical readings onto the one already in flight. Nothing is kept once
// a reading completes: this is in-flight dedup only, so a later caller always reads afresh.
type readingFlights struct {
	mu      sync.Mutex
	flights map[string]*readingFlight
	started map[string]int // readings actually run, per kind, for tests and diagnostics
}

type readingFlight struct {
	done    chan struct{}
	val     any
	err     error
	waiters int
	cancel  context.CancelFunc
}

// shareReading runs read once for every concurrent caller with the same kind, args and viewer. Joiners
// get the same result, including the clock instant it was read at. The caller that started the flight gets
// its value; each joiner gets its own deep copy, so no caller can change what another caller reads. The flight is detached from any one
// caller's cancellation and is cancelled only when every waiting caller has gone.
func shareReading[T any](ctx context.Context, f *readingFlights, kind, args string,
	read func(context.Context) (T, error)) (T, error) {
	view, ok := commissioner.ReadingViewKey(ctx)
	if !ok {
		return read(ctx)
	}
	key := kind + "\x00" + args + "\x00" + view
	f.mu.Lock()
	if f.flights == nil {
		f.flights = map[string]*readingFlight{}
		f.started = map[string]int{}
	}
	fl, joined := f.flights[key]
	if !joined {
		fctx, cancel := context.WithCancel(context.WithoutCancel(ctx))
		fl = &readingFlight{done: make(chan struct{}), cancel: cancel}
		f.flights[key] = fl
		f.started[kind]++
		go f.fly(fctx, key, kind, fl, func(c context.Context) (any, error) { return read(c) })
	}
	fl.waiters++
	f.mu.Unlock()

	var zero T
	select {
	case <-fl.done:
		if fl.err != nil {
			return zero, fl.err
		}
		val, _ := fl.val.(T)
		if joined {
			return deepclone.Of(val), nil
		}
		return val, nil
	case <-ctx.Done():
		f.mu.Lock()
		fl.waiters--
		if fl.waiters == 0 {
			fl.cancel()
			if f.flights[key] == fl {
				delete(f.flights, key)
			}
		}
		f.mu.Unlock()
		return zero, fmt.Errorf("reading %s: %w", kind, ctx.Err())
	}
}

func (f *readingFlights) fly(ctx context.Context, key, kind string, fl *readingFlight,
	read func(context.Context) (any, error)) {
	defer close(fl.done)
	defer fl.cancel()
	defer func() {
		if r := recover(); r != nil {
			fl.err = fmt.Errorf("reading %s panicked: %v", kind, r)
		}
		f.mu.Lock()
		if f.flights[key] == fl {
			delete(f.flights, key)
		}
		f.mu.Unlock()
	}()
	fl.val, fl.err = read(ctx)
}

// oddsArgs names an odds reading's arguments, including every value it is given.
func oddsArgs(league string, season int, values commissioner.OddsValues) string {
	body, err := json.Marshal(values)
	if err != nil {
		return fmt.Sprintf("%s|%d|unkeyed:%p", league, season, &values) // never shared
	}
	sum := sha256.Sum256(body)
	return fmt.Sprintf("%s|%d|%x", league, season, sum)
}

// recordsMark names a slice of ledger records by its length and its last entry, which is enough because
// the ledger is append-only and the records are read through one instant.
func recordsMark(records []commissioner.Record) string {
	if len(records) == 0 {
		return "0"
	}
	return fmt.Sprintf("%d:%s", len(records), records[len(records)-1].Receipt.Entry)
}

// recordsMarkFor names the league's ledger as it stands, so a reading started before a write is never
// joined by a caller that reads after it.
func (s *Store) recordsMarkFor(ctx context.Context, league string) (string, error) {
	records, err := s.nativeRecords(ctx, league)
	if err != nil {
		return "", err
	}
	return recordsMark(records), nil
}
