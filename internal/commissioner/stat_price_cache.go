package commissioner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"slices"
	"sync"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// keepPriceReadings bounds the stat price cache: only the prices the last few readings used or made are kept.
const keepPriceReadings = 4

// StatPriceCache keeps the price of each statistics queue item across readings, so a new ledger revision
// re-prices only the items whose inputs moved. A price is keyed by everything it is read from: the item and
// its held evidence, the league's rules, directory and published schedule, every entry that is not a
// statistics decision, the player's own counted statistics and score, and the results of the item's week as
// they stand (lineups and every starter's score). A statistics decision moves only its own week's results,
// so it re-prices that week's items and no other (TestStatPriceCacheMatchesFullPricing).
// Use NewStatPriceCache. It is safe for concurrent readings.
type StatPriceCache struct {
	mu       sync.Mutex
	readings []map[string]pricedItem // newest last
	hits     int
	misses   int
	// skipWeek leaves the week's results out of the key: tests only, to prove the key needs them.
	skipWeek bool
}

type pricedItem struct {
	points *leaguerules.Decimal
	flip   ResultFlip
}

// NewStatPriceCache is an empty cache.
func NewStatPriceCache() *StatPriceCache { return &StatPriceCache{} }

// Counts reports the cache hits and misses so far.
func (c *StatPriceCache) Counts() (hits, misses int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.hits, c.misses
}

func (p pricedItem) clone() pricedItem {
	if p.points != nil {
		points := *p.points
		p.points = &points
	}
	p.flip.Flips = slices.Clone(p.flip.Flips)
	return p
}

// begin opens a reading: its prices are the newest generation, and the oldest beyond the bound is dropped.
func (c *StatPriceCache) begin() map[string]pricedItem {
	c.mu.Lock()
	defer c.mu.Unlock()
	used := map[string]pricedItem{}
	c.readings = append(c.readings, used)
	if len(c.readings) > keepPriceReadings {
		c.readings = slices.Delete(c.readings, 0, len(c.readings)-keepPriceReadings)
	}
	return used
}

func (c *StatPriceCache) get(used map[string]pricedItem, key string) (pricedItem, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := len(c.readings) - 1; i >= 0; i-- {
		if hit, ok := c.readings[i][key]; ok {
			used[key] = hit
			c.hits++
			return hit.clone(), true
		}
	}
	c.misses++
	return pricedItem{}, false
}

func (c *StatPriceCache) put(used map[string]pricedItem, key string, p pricedItem) {
	c.mu.Lock()
	defer c.mu.Unlock()
	used[key] = p.clone()
}

// priceReading is one reading's share of the cache: the digest of what every item reads alike, and of each
// week's results before any decision.
type priceReading struct {
	cache *StatPriceCache
	used  map[string]pricedItem
	base  string
	weeks map[int]string
}

func digestOf(h hash.Hash, values ...any) {
	for _, v := range values {
		body, err := json.Marshal(v)
		_, _ = fmt.Fprintf(h, "%d:%s|%v\x00", len(body), body, err)
	}
}

// newPriceReading digests the reading's shared inputs. Revision numbers are left out: they move with every
// entry, and no price reads them.
func newPriceReading(cache *StatPriceCache, league string, season int, records []Record,
	rules leaguerules.RuleSet, flips *queueFlips) *priceReading {
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "%s\x00%d\x00", league, season)
	for i := range records {
		if r := &records[i]; !isStatCommand(r.Command.Command) {
			digestOf(h, r.Receipt, r.Canonical, r.Command, r.GeneratedSchedule, r.Before, r.Principal, r.Role)
		}
	}
	digestOf(h, rules, fmt.Sprint(flips.err))
	if flips.base != nil {
		directory := flips.base.directory
		directory.Revision = 0
		digestOf(h, directory, flips.base.published, flips.base.rules)
	}
	return &priceReading{cache: cache, used: cache.begin(), base: hex.EncodeToString(h.Sum(nil)),
		weeks: map[int]string{}}
}

// week digests the week's results before any decision, read once for the reading.
func (r *priceReading) week(ctx context.Context, b queueBuilder, week int) (string, error) {
	if d, ok := r.weeks[week]; ok {
		return d, nil
	}
	held, err := b.flips.base.beforeWeek(ctx, b.league, b.records, b.src, b.season, week)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	if !r.cache.skipWeek {
		results := held.results
		results.Revision = 0
		digestOf(h, results, fmt.Sprint(held.err))
	}
	d := hex.EncodeToString(h.Sum(nil))
	r.weeks[week] = d
	return d, nil
}

// key is the item's price key: the item, its evidence, the player's counted statistics and score before
// the decision, its week's results and the reading's shared inputs.
func (r *priceReading) key(b queueBuilder, item StatQueueItem, week StatWeek, before NativePlayerScore,
	weekDigest string) string {
	k := statKey{week.Player, b.season, week.Week}
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "%s\x00%s\x00%s\x00", r.base, weekDigest, item.ID)
	digestOf(h, week.Evidence, before, b.effects.baselines[k], b.effects.changes[k], b.effects.corrections[k])
	return hex.EncodeToString(h.Sum(nil))
}
