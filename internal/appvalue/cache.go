package appvalue

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/store/params"
)

// Cache retains one league's completed pass, shared by its franchise headers and inspectors.
// Database commits, changed settings/params and the next as-of observation invalidate it.
// Only the producer assembles evidence and runs the unchanged model for prepared screens.
type Cache struct {
	mu          sync.Mutex
	key         [sha256.Size]byte
	from, until time.Time
	reading     Reading
	ready       bool
	failure     error
	generation  uint64
}

func (c *Cache) get(key [sha256.Size]byte, at time.Time) (Reading, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.ready || c.key != key || at.Before(c.from) || (!c.until.IsZero() && !at.Before(c.until)) {
		return Reading{}, false, nil
	}
	return cloneReading(c.reading), true, c.failure
}

func (src Source) cached(ctx context.Context, revision int64, set params.Set, franchise string, in Inputs) (Reading, error) {
	if err := ctx.Err(); err != nil {
		return Reading{}, fmt.Errorf("league values: %w", err)
	}
	key, err := cacheKey(revision, set, in)
	if err != nil {
		return Reading{}, err
	}
	if out, ok, failure := src.Cache.get(key, in.AsOf); ok && (failure == nil || src.PreparedOnly) {
		if failure != nil {
			return Reading{}, failure
		}
		return scoped(out, in, franchise), nil
	}
	if src.PreparedOnly {
		return preparing(in, franchise), nil
	}
	until, err := src.NextFeatureAt(ctx, in.AsOf)
	if err != nil {
		return Reading{}, fmt.Errorf("league values expiry: %w", err)
	}
	out, valueErr := src.value(ctx, set, "", in)
	after, err := src.revision(ctx)
	if err != nil {
		return Reading{}, err
	}
	if after == revision {
		src.Cache.mu.Lock()
		src.Cache.key, src.Cache.from, src.Cache.until = key, in.AsOf, until
		src.Cache.reading, src.Cache.ready, src.Cache.failure = cloneReading(out), true, valueErr
		src.Cache.generation++
		src.Cache.mu.Unlock()
	}
	if valueErr != nil {
		return Reading{}, valueErr
	}
	return scoped(out, in, franchise), nil
}

func preparing(in Inputs, franchise string) Reading {
	return Reading{League: in.League, Season: in.Season, Franchise: franchise, Preparing: true,
		Players: []Player{}, Age: []Band{}, Notes: []string{},
		Best: Best{Franchise: franchise, Words: "Values are being prepared"}}
}

func cacheKey(revision int64, set params.Set, in Inputs) ([sha256.Size]byte, error) {
	in.AsOf = time.Time{}
	body, err := json.Marshal(struct {
		Revision int64
		Params   map[string]float64
		Inputs   Inputs
	}{revision, set.Values(), in})
	if err != nil {
		return [sha256.Size]byte{}, fmt.Errorf("league values cache key: %w", err)
	}
	return sha256.Sum256(body), nil
}

func cloneReading(in Reading) Reading {
	in.Players, in.Age, in.Notes = slices.Clone(in.Players), slices.Clone(in.Age), slices.Clone(in.Notes)
	return in
}

func scoped(out Reading, in Inputs, franchise string) Reading {
	all := out.Players
	if franchise != "" {
		out.Players = []Player{}
		for _, p := range all {
			if p.Franchise == franchise {
				out.Players = append(out.Players, p)
			}
		}
	}
	out.Franchise, out.Best, out.Age = franchise, best(all, in, franchise), bands(out.Players)
	return out
}
