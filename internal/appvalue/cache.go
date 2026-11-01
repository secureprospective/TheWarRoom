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
// Cold reads still assemble the full evidence and run the unchanged model.
type Cache struct {
	mu          sync.Mutex
	key         [sha256.Size]byte
	from, until time.Time
	reading     Reading
	ready       bool
}

func (c *Cache) get(key [sha256.Size]byte, at time.Time) (Reading, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.ready || c.key != key || at.Before(c.from) || (!c.until.IsZero() && !at.Before(c.until)) {
		return Reading{}, false
	}
	return cloneReading(c.reading), true
}

func (src Source) cached(ctx context.Context, revision int64, set params.Set, franchise string, in Inputs) (Reading, error) {
	if err := ctx.Err(); err != nil {
		return Reading{}, fmt.Errorf("league values: %w", err)
	}
	key, err := cacheKey(revision, set, in)
	if err != nil {
		return Reading{}, err
	}
	if out, ok := src.Cache.get(key, in.AsOf); ok {
		return scoped(out, in, franchise), nil
	}
	until, err := src.NextFeatureAt(ctx, in.AsOf)
	if err != nil {
		return Reading{}, fmt.Errorf("league values expiry: %w", err)
	}
	out, err := src.value(ctx, set, "", in)
	if err != nil {
		return Reading{}, err
	}
	after, err := src.revision(ctx)
	if err != nil {
		return Reading{}, err
	}
	if after == revision {
		src.Cache.mu.Lock()
		src.Cache.key, src.Cache.from, src.Cache.until = key, in.AsOf, until
		src.Cache.reading, src.Cache.ready = cloneReading(out), true
		src.Cache.mu.Unlock()
	}
	return scoped(out, in, franchise), nil
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
