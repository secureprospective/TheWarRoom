package history

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/deepclone"
)

// readingMemoTTL bounds how long one reading is reused even when nothing in its key moved.
const readingMemoTTL = 30 * time.Second

// readingMemoLimit bounds the readings held; past it the memo starts empty.
const readingMemoLimit = 64

// readingMemo keeps a week's results (or a season's standings), viewer-independent, so every phone polling
// the same week between ledger writes reads one answer instead of re-folding rules, the directory and every
// score. Concurrent callers with the same key that find nothing kept wait for the one reading in flight: a
// score change makes every phone re-read at once, and they all need the same answer. The key is everything
// the reading is made of (readingMemoKey). The per-viewer revision and the evidence freshness are added on
// every read, never kept. Every caller gets its own deep copy.
type readingMemo[T any] struct {
	mu      sync.Mutex
	entries map[readingMemoKey]readingMemoEntry[T]
	flights map[readingFlightKey]*readingMemoFlight[T]
	hits    int
	misses  int
	runs    int // readings actually computed, for tests and diagnostics
}

// readingMemoKey: the season score revision (exact ledger entries, held archive, rules in force and player
// facts), the ledger mark, the identity view, the reading epoch (each recorded instant the clock has passed)
// and the live lines the reading scores from. week is 0 for a whole-season reading.
type readingMemoKey struct {
	readingFlightKey
	revision string
}

// readingFlightKey is the memo key without the score revision: cheap to name, so a burst of callers finds the
// one reading in flight before anyone folds the rules. Callers that join a flight get its answer, as
// shareReading's joiners do; only the full key decides what is kept and reused later.
type readingFlightKey struct {
	league, mark, view  string
	live, overlay       string
	gen                 liveWeekGen
	season, week, epoch int
}

type readingMemoEntry[T any] struct {
	at  time.Time
	out T
}

type readingMemoFlight[T any] struct {
	done    chan struct{}
	out     T
	ok      bool
	waiters int // guarded by readingMemo.mu
}

// readingFlightKey names one reading's flight, or reports that the ledger is untimed and never shared. week 0
// names the whole season's live lines.
func (s *Store) readingFlightKey(src *nativeSources, records []commissioner.Record, mark string,
	season, week int) (readingFlightKey, bool) {
	epoch, _, ok := commissioner.ReadingEpoch(records, src.now, nil)
	if !ok {
		return readingFlightKey{}, false
	}
	return readingFlightKey{league: src.league, mark: mark, view: commissioner.RecordsViewKey(records),
		live: src.liveKey(), overlay: src.weekKey(season, week), gen: s.liveGeneration(season, week), season: season,
		week: week, epoch: epoch}, true
}

// liveGeneration names one week's live lines, or the whole season's when week is 0.
func (s *Store) liveGeneration(season, week int) liveWeekGen {
	if week == 0 {
		return s.liveSeasonGeneration(season)
	}
	return s.liveWeekGeneration(season, week)
}

// sharedReading answers a reading through memo, or reads it alone when the ledger is untimed.
func sharedReading[T any](ctx context.Context, s *Store, memo *readingMemo[T], src *nativeSources,
	records []commissioner.Record, mark string, season, week int, read func() (T, error)) (T, error) {
	flight, ok := s.readingFlightKey(src, records, mark, season, week)
	if !ok {
		return read()
	}
	// The full key is worked out inside the flight, after the score revision: a trial ledger, test-supplied
	// evidence or the score cache switched off read afresh and keep nothing. Working out the revision can
	// restore the live lines, so the live generation is named again after it.
	full := func() (readingMemoKey, bool) {
		revision, ok := src.scoreRevision(ctx, records)
		if !ok {
			return readingMemoKey{}, false
		}
		named := flight
		named.gen, named.live, named.overlay = s.liveGeneration(season, week), src.liveKey(), src.weekKey(season, week)
		return readingMemoKey{readingFlightKey: named, revision: revision}, true
	}
	return memo.load(ctx, flight, full, src.now, read)
}

// liveSeasonGeneration names every week's live lines of a season at once: each week's count only grows, so
// any line change moves the sum.
func (s *Store) liveSeasonGeneration(season int) liveWeekGen {
	s.liveMu.Lock()
	defer s.liveMu.Unlock()
	gen := liveWeekGen{epoch: s.liveEpoch}
	for scope, n := range s.liveWeeks {
		if scope[0] == season {
			gen.week += n
		}
	}
	return gen
}

// load joins the reading in flight for the key, or answers from the memo, or runs read and keeps its answer.
// A failed or cancelled reading is never kept; callers that waited on it read again.
func (m *readingMemo[T]) load(ctx context.Context, flight readingFlightKey, full func() (readingMemoKey, bool),
	now time.Time, read func() (T, error)) (T, error) {
	for {
		m.mu.Lock()
		if fl := m.flights[flight]; fl != nil {
			fl.waiters++
			m.mu.Unlock()
			if out, ok, err := m.wait(ctx, fl); ok || err != nil {
				return out, err
			}
			continue
		}
		fl := m.begin(flight)
		m.mu.Unlock()
		return m.run(ctx, flight, fl, func() (T, error) { return m.lookup(ctx, full, now, read) })
	}
}

// lookup runs inside a flight: it names the full key, answers a kept reading, or reads and keeps.
func (m *readingMemo[T]) lookup(ctx context.Context, full func() (readingMemoKey, bool), now time.Time,
	read func() (T, error)) (T, error) {
	key, shared := full()
	if !shared {
		return read()
	}
	m.mu.Lock()
	if hit, ok := m.entries[key]; ok && now.Sub(hit.at) < readingMemoTTL && !now.Before(hit.at) {
		m.hits++
		m.mu.Unlock()
		return share(hit.out), nil
	}
	m.misses++
	m.runs++
	m.mu.Unlock()
	out, err := read()
	if err != nil || ctx.Err() != nil {
		return out, err
	}
	kept := deepclone.Of(out)
	if f, ok := any(&kept).(freezer); ok {
		if err := f.Freeze(); err != nil {
			return out, nil //nolint:nilerr // an answer that cannot be encoded is still answered; it is not kept
		}
		out = kept
	}
	m.mu.Lock()
	if m.entries == nil || len(m.entries) >= readingMemoLimit {
		m.entries = map[readingMemoKey]readingMemoEntry[T]{}
	}
	m.entries[key] = readingMemoEntry[T]{at: now, out: kept}
	m.mu.Unlock()
	return out, nil
}

// begin registers a flight; the caller holds mu.
func (m *readingMemo[T]) begin(key readingFlightKey) *readingMemoFlight[T] {
	if m.flights == nil {
		m.flights = map[readingFlightKey]*readingMemoFlight[T]{}
	}
	fl := &readingMemoFlight[T]{done: make(chan struct{})}
	m.flights[key] = fl
	return fl
}

func (m *readingMemo[T]) wait(ctx context.Context, fl *readingMemoFlight[T]) (T, bool, error) {
	var zero T
	select {
	case <-fl.done:
		if !fl.ok {
			return zero, false, nil
		}
		m.mu.Lock()
		m.hits++
		m.mu.Unlock()
		return share(fl.out), true, nil
	case <-ctx.Done():
		return zero, false, fmt.Errorf("wait for reading: %w", ctx.Err())
	}
}

// run answers one flight. Its waiters are counted under mu while the flight is listed, and it is unlisted under
// the same lock before they are counted here, so a copy is made for them only when someone waits.
func (m *readingMemo[T]) run(ctx context.Context, key readingFlightKey, fl *readingMemoFlight[T],
	read func() (T, error)) (T, error) {
	out, err := read()
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.flights[key] == fl {
		delete(m.flights, key)
	}
	if err == nil && ctx.Err() == nil {
		if fl.waiters > 0 {
			fl.out = share(out)
		}
		fl.ok = true
	}
	close(fl.done)
	return out, err
}

func (m *readingMemo[T]) stats() (hits, misses, runs int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.hits, m.misses, m.runs
}

// freezer is a reading that can encode its viewer-independent body once (commissioner.NativeResults.Freeze).
type freezer interface{ Freeze() error }

// share hands a kept reading to one more caller. A frozen reading is shared read-only (callers set only its
// top-level per-read fields, on their own copy of the struct); anything else is deep-copied.
func share[T any](v T) T {
	if f, ok := any(v).(interface{ Frozen() bool }); ok && f.Frozen() {
		return v
	}
	return deepclone.Of(v)
}
