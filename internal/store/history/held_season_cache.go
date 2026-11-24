package history

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"maps"
	"slices"
	"sync"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
)

// heldSeasonKey covers weekly statistics, plays, absence proofs and their identity bridge.
// Secondary statistics archives also invalidate it: their presence can block proven zeros.
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
	// gsis is the directory's gsis id per league player at key's identity revision, as SourceID reads it.
	gsis map[string]string
}

type heldSeasonFlight struct {
	key   heldSeasonKey
	done  chan struct{}
	entry *heldSeasonEntry
	err   error
}

type heldIdentity struct {
	revision int64
	digest   [sha256.Size]byte
	gsis     map[string]string
}

type heldIdentityFlight struct {
	revision int64
	done     chan struct{}
	entry    *heldIdentity
	err      error
}

// heldIdentityCache keeps the latest gsis/pfr digest and shared read-only gsis map, keyed by the
// reference database's SQLite data_version so writes from other Stores or processes invalidate it.
type heldIdentityCache struct {
	mu     sync.Mutex
	entry  *heldIdentity
	flight *heldIdentityFlight
	builds int
}

// heldSeasonCache keeps the latest built copy of each season and lets one builder run per season;
// concurrent readers of the same key wait for it instead of reading and hashing the archive again.
type heldSeasonCache struct {
	mu       sync.Mutex
	off      bool // tests: build on every call, as before the cache
	entries  map[int]*heldSeasonEntry
	flights  map[int]*heldSeasonFlight
	builds   int
	identity heldIdentityCache
}

// heldSeasonKey names the held archive and outside-id directory revision for season. The returned
// gsis map is shared read-only: callers must not mutate it.
func (s *Store) heldSeasonKey(ctx context.Context, season int) (heldSeasonKey, map[string]string, error) {
	var key heldSeasonKey
	var urls [3]string
	for i, name := range []string{"stats_player_week", "snap_counts", "injuries"} {
		if feed, ok := s.reg.Feed(name); ok {
			urls[i] = feed.URLFor(season)
		}
	}
	err := s.pools.Read().QueryRowContext(ctx,
		`SELECT count(*), coalesce(max(fetch_id), 0) FROM fetch_log WHERE status = 200 AND
   (url IN (?, ?, ?, ?, ?) OR instr(url, ?) > 0 OR instr(url, ?) > 0 OR instr(url, ?) > 0)`,
		urls[0], urls[1], urls[2], source.PlaysURL(season), source.DepthHistoryURL(season),
		"TYPE=nflByeWeeks", "espn", "sleeper").Scan(&key.fetches, &key.newest)
	if err != nil {
		return key, nil, fmt.Errorf("history: held season revision: %w", err)
	}
	digest, gsis, err := s.heldIdentity(ctx)
	if err != nil {
		return key, nil, err
	}
	key.identity = digest
	return key, gsis, nil
}

// heldIdentity returns the gsis/pfr digest and shared read-only gsis map, recomputing only when the
// reference database's SQLite data_version advances. Callers must not mutate the map.
func (s *Store) heldIdentity(ctx context.Context) ([sha256.Size]byte, map[string]string, error) {
	c := &s.seasons.identity
	for {
		rev, err := s.pools.ReferencePools().Revision(ctx)
		if err != nil {
			return [sha256.Size]byte{}, nil, fmt.Errorf("history: identity revision: %w", err)
		}
		c.mu.Lock()
		if e := c.entry; e != nil && e.revision == rev {
			c.mu.Unlock()
			return e.digest, e.gsis, nil
		}
		if f := c.flight; f != nil {
			c.mu.Unlock()
			if e, done, err := waitHeldIdentity(ctx, f, rev); done {
				return e.digest, e.gsis, err
			}
			continue
		}
		f := &heldIdentityFlight{revision: rev, done: make(chan struct{})}
		c.flight = f
		c.builds++
		c.mu.Unlock()
		return s.runHeldIdentity(ctx, c, f)
	}
}

func waitHeldIdentity(ctx context.Context, f *heldIdentityFlight, rev int64) (heldIdentity, bool, error) {
	select {
	case <-f.done:
	case <-ctx.Done():
		return heldIdentity{}, true, fmt.Errorf("history: wait for held identities: %w", ctx.Err())
	}
	if f.revision != rev {
		return heldIdentity{}, false, nil
	}
	if f.err != nil {
		return heldIdentity{}, true, f.err
	}
	return *f.entry, true, nil
}

func (s *Store) runHeldIdentity(ctx context.Context, c *heldIdentityCache, f *heldIdentityFlight) ([sha256.Size]byte, map[string]string, error) {
	f.entry, f.err = s.buildHeldIdentity(context.WithoutCancel(ctx), f.revision)
	c.mu.Lock()
	c.flight = nil
	if f.err == nil {
		c.entry = f.entry
	}
	c.mu.Unlock()
	close(f.done)
	if f.err != nil {
		return [sha256.Size]byte{}, nil, f.err
	}
	return f.entry.digest, f.entry.gsis, nil
}

func (s *Store) buildHeldIdentity(ctx context.Context, rev int64) (*heldIdentity, error) {
	tx, err := s.pools.Read().BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("history: begin identity read: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	h := sha256.New()
	var gsis map[string]string
	for _, idType := range []string{"gsis", "pfr"} {
		ids, err := querySourceIDs(ctx, tx, idType)
		if err != nil {
			return nil, err
		}
		if idType == "gsis" {
			gsis = ids
		}
		for _, player := range slices.Sorted(maps.Keys(ids)) {
			_, _ = fmt.Fprintf(h, "%s\x00%s\x00%s\n", idType, player, ids[player])
		}
	}
	var digest [sha256.Size]byte
	copy(digest[:], h.Sum(nil))
	return &heldIdentity{revision: rev, digest: digest, gsis: gsis}, nil
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
	key, gsis, err := s.heldSeasonKey(ctx, season)
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
		if f.err == nil {
			f.entry.gsis = gsis
		}
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

// SourceID answers a reading's gsis lookups from the directory read with its held season, so scoring every
// starter of every week does not query the directory once per starter. Other id types, and readings
// without a cached season, ask the store.
func (n *nativeSources) SourceID(ctx context.Context, playerID, idType string) (string, error) {
	if idType == "gsis" {
		if n.gsis == nil {
			if entry, err := n.store.cachedHeldSeason(ctx, n.season); err == nil && entry.gsis != nil {
				n.gsis = entry.gsis
			}
		}
		if n.gsis != nil {
			return n.gsis[playerID], nil
		}
	}
	return n.store.SourceID(ctx, playerID, idType)
}
