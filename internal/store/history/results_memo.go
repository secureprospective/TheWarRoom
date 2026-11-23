package history

import (
	"context"
	"sync"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/deepclone"
)

// resultsMemoTTL bounds how long one week's results are reused even when nothing in their key moved.
const resultsMemoTTL = 30 * time.Second

// resultsMemoLimit bounds the weeks held; past it the memo starts empty.
const resultsMemoLimit = 64

// resultsMemo keeps each week's results reading, viewer-independent, so every phone polling the same week
// between ledger writes reads one answer instead of re-folding rules, the directory and every score. The key
// is everything the reading is made of: the season score revision (exact ledger entries, held archive, rules
// in force and player facts), the ledger mark, the identity view, the reading epoch (each recorded instant the
// clock has passed) and the week's live lines. The per-viewer revision and the evidence freshness are added
// on every read, never kept.
type resultsMemo struct {
	mu      sync.Mutex
	entries map[resultsMemoKey]resultsMemoEntry
	hits    int
	misses  int
}

type resultsMemoKey struct {
	league, revision, mark, view string
	live, overlay                string
	gen                          liveWeekGen
	season, week, epoch          int
}

type resultsMemoEntry struct {
	at  time.Time
	out commissioner.NativeResults
}

// resultsMemoKey names one reading's results, or reports that the reading cannot share (a trial ledger,
// test-supplied evidence or lineups, the score cache off, or an untimed ledger).
func (s *Store) resultsMemoKey(ctx context.Context, src *nativeSources, records []commissioner.Record, mark string,
	season, week int) (resultsMemoKey, bool) {
	revision, ok := src.scoreRevision(ctx, records)
	if !ok {
		return resultsMemoKey{}, false
	}
	epoch, _, ok := commissioner.ReadingEpoch(records, src.now, nil)
	if !ok {
		return resultsMemoKey{}, false
	}
	return resultsMemoKey{league: src.league, revision: revision, mark: mark, view: commissioner.RecordsViewKey(records),
		live: src.liveKey(), overlay: src.weekKey(season, week), gen: s.liveWeekGeneration(season, week),
		season: season, week: week, epoch: epoch}, true
}

func (m *resultsMemo) get(key resultsMemoKey, now time.Time) (commissioner.NativeResults, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	hit, ok := m.entries[key]
	if !ok || now.Sub(hit.at) >= resultsMemoTTL || now.Before(hit.at) {
		m.misses++
		return commissioner.NativeResults{}, false
	}
	m.hits++
	return deepclone.Of(hit.out), true
}

func (m *resultsMemo) put(key resultsMemoKey, now time.Time, out commissioner.NativeResults) {
	kept := deepclone.Of(out)
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.entries == nil || len(m.entries) >= resultsMemoLimit {
		m.entries = map[resultsMemoKey]resultsMemoEntry{}
	}
	m.entries[key] = resultsMemoEntry{at: now, out: kept}
}

func (m *resultsMemo) stats() (hits, misses int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.hits, m.misses
}
