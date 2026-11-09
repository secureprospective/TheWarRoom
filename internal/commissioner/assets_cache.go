package commissioner

import (
	"slices"
	"sync"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/deepclone"
)

// assetsCache holds native-asset folds for one held revision, shared by every request and owner
// that reads it. Each caller gets its own deep copy, because callers project over the result.
//
// The fold reads the clock in exactly two ways: a record is visible once its effective time is at or
// before the reading instant, and a correction lapses once its expiry is at or before it. Both are
// "instant has reached t", so two instants that have reached the same number of those times give the
// same fold. That count is the key (TestAssetsCacheMatchesFold sweeps every boundary).
type assetsCache struct {
	mu         sync.Mutex
	thresholds []time.Time // sorted; nil until first use
	timed      bool        // thresholds are complete; false when a record's times do not parse
	built      bool
	views      map[assetsKey]*assetsEntry
}

type assetsKey struct {
	league, principals string
	books              *closedBooks
	clocked            bool
	reached            int
}

type assetsEntry struct {
	once   sync.Once
	assets NativeAssets
	book   MarketBook
	err    error
}

// maxAssetsViews bounds the folds held per revision: reading instants move, but rarely across a boundary.
const maxAssetsViews = 8

// nativeAssetsState is the league's assets and market book at the reading instant (the last record
// when at is absent), from the held revision's cache when the records carry one.
func nativeAssetsState(league string, records []Record, at ...time.Time) (NativeAssets, MarketBook, error) {
	memo := memoFor(records)
	if memo == nil {
		return foldNativeAssets(league, records, at...)
	}
	entry, ok := memo.assets.entry(league, records, at)
	if !ok {
		return foldNativeAssets(league, records, at...)
	}
	entry.once.Do(func() { entry.assets, entry.book, entry.err = foldNativeAssets(league, records, at...) })
	return deepclone.Of(entry.assets), deepclone.Of(entry.book), entry.err
}

func (c *assetsCache) entry(league string, records []Record, at []time.Time) (*assetsEntry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := assetsKey{league: league, principals: records[0].principals.Key(), books: records[0].books}
	if len(at) != 0 {
		if !c.built {
			c.thresholds, c.timed = clockThresholds(records)
			c.built = true
		}
		if !c.timed {
			return nil, false
		}
		key.clocked = true
		key.reached, _ = slices.BinarySearchFunc(c.thresholds, at[0], func(t, at time.Time) int {
			if t.After(at) {
				return 1
			}
			return -1
		})
	}
	if hit, ok := c.views[key]; ok {
		return hit, true
	}
	if c.views == nil || len(c.views) >= maxAssetsViews {
		c.views = map[assetsKey]*assetsEntry{}
	}
	hit := &assetsEntry{}
	c.views[key] = hit
	return hit, true
}

// clockThresholds lists every instant at which the fold's answer can change: each record's effective
// time and each correction's expiry. Extra instants only split the cache finer; a time that does not
// parse leaves the fold uncached.
func clockThresholds(records []Record) ([]time.Time, bool) {
	out := make([]time.Time, 0, len(records))
	for _, r := range records {
		effective, err := nativeEntryTime(r)
		if err != nil {
			return nil, false
		}
		out = append(out, effective)
		if c := r.Command.Args.Correction; c != nil && c.Until != "" && correctionCommand(r.Command.Command) {
			end, err := time.Parse(time.RFC3339Nano, c.Until)
			if err != nil {
				return nil, false
			}
			out = append(out, end)
		}
	}
	slices.SortFunc(out, time.Time.Compare)
	return out, true
}
