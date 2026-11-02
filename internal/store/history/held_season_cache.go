package history

import (
	"context"
	"crypto/sha256"
	"fmt"
	"maps"
	"slices"
	"sync"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
)

// heldSeasonKey names everything a held season is built from. The archive mark counts every held fetch
// and names the newest; the archive is append-only, so any new body (weekly, plays, snaps, injuries,
// depth charts, byes, or a fetch that blocks zero proofs) moves it. The identity digest covers the
// gsis/pfr bridge the absence proofs are matched through. Live lines, stat corrections, rules and
// positions are applied after the held season is read, per starter, so they are not part of it.
type heldSeasonKey struct {
	fetches, newest int64
	identity        [sha256.Size]byte
}

// heldSeasonEntry is one built season: the files as read, and the same files indexed for readings
// that look up many players.
type heldSeasonEntry struct {
	key     heldSeasonKey
	files   []heldFile
	indexed []heldFile
}

type heldSeasonFlight struct {
	key   heldSeasonKey
	done  chan struct{}
	entry *heldSeasonEntry
	err   error
}

// heldSeasonCache keeps the latest built copy of each season and lets one builder run per season;
// concurrent readers of the same key wait for it instead of reading and hashing the archive again.
type heldSeasonCache struct {
	mu      sync.Mutex
	off     bool // tests: build on every call, as before the cache
	entries map[int]*heldSeasonEntry
	flights map[int]*heldSeasonFlight
	builds  int
}

func (s *Store) heldSeasonKey(ctx context.Context) (heldSeasonKey, error) {
	var key heldSeasonKey
	err := s.pools.Read().QueryRowContext(ctx,
		`SELECT count(*), coalesce(max(fetch_id), 0) FROM fetch_log WHERE status = 200`).Scan(&key.fetches, &key.newest)
	if err != nil {
		return key, fmt.Errorf("history: held season revision: %w", err)
	}
	h := sha256.New()
	for _, idType := range []string{"gsis", "pfr"} {
		ids, err := s.SourceIDs(ctx, idType)
		if err != nil {
			return key, err
		}
		for _, player := range slices.Sorted(maps.Keys(ids)) {
			_, _ = fmt.Fprintf(h, "%s\x00%s\x00%s\n", idType, player, ids[player])
		}
	}
	copy(key.identity[:], h.Sum(nil))
	return key, nil
}

// heldSeason reads every held copy of a season's weekly file once per held revision, newest first. The
// files' contents are shared: callers must not change them.
func (s *Store) heldSeason(ctx context.Context, season int) ([]heldFile, error) {
	entry, err := s.cachedHeldSeason(ctx, season)
	if err != nil {
		return nil, err
	}
	return slices.Clone(entry.files), nil
}

// indexedHeldSeason is heldSeason with every file indexed for many lookups.
func (s *Store) indexedHeldSeason(ctx context.Context, season int) ([]heldFile, error) {
	entry, err := s.cachedHeldSeason(ctx, season)
	if err != nil {
		return nil, err
	}
	return slices.Clone(entry.indexed), nil
}

func (s *Store) cachedHeldSeason(ctx context.Context, season int) (*heldSeasonEntry, error) {
	c := &s.seasons
	c.mu.Lock()
	off := c.off
	c.mu.Unlock()
	if off {
		return s.buildHeldSeason(ctx, season, heldSeasonKey{})
	}
	// The key is read before the build, so a change that lands mid-build leaves a stale key on the entry and
	// the next reader rebuilds; a reader is never served files older than the revision it saw.
	key, err := s.heldSeasonKey(ctx)
	if err != nil {
		return nil, err
	}
	for {
		c.mu.Lock()
		if e := c.entries[season]; e != nil && e.key == key {
			c.mu.Unlock()
			return e, nil
		}
		if f := c.flights[season]; f != nil {
			c.mu.Unlock()
			select {
			case <-f.done:
			case <-ctx.Done():
				return nil, fmt.Errorf("history: wait for held season: %w", ctx.Err())
			}
			if f.key == key {
				return f.entry, f.err
			}
			continue
		}
		f := &heldSeasonFlight{key: key, done: make(chan struct{})}
		if c.flights == nil {
			c.flights = map[int]*heldSeasonFlight{}
		}
		c.flights[season] = f
		c.builds++
		c.mu.Unlock()
		// One reader's cancellation must not fail the others waiting on this build.
		f.entry, f.err = s.buildHeldSeason(context.WithoutCancel(ctx), season, key)
		c.mu.Lock()
		delete(c.flights, season)
		if f.err == nil {
			if c.entries == nil {
				c.entries = map[int]*heldSeasonEntry{}
			}
			c.entries[season] = f.entry
		}
		c.mu.Unlock()
		close(f.done)
		return f.entry, f.err
	}
}

func (s *Store) buildHeldSeason(ctx context.Context, season int, key heldSeasonKey) (*heldSeasonEntry, error) {
	files, err := s.readHeldSeason(ctx, season)
	if err != nil {
		return nil, err
	}
	indexed := slices.Clone(files)
	for i := range indexed {
		indexed[i].index = source.IndexHeldWeeks(indexed[i].weeks)
	}
	return &heldSeasonEntry{key: key, files: files, indexed: indexed}, nil
}
