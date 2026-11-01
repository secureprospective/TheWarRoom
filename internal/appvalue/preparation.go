package appvalue

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

// Scope identifies a prepared league pass; franchises share it.
type Scope struct {
	League string
	Season int
}

// Preparation owns completed passes and one background producer. Reads only assemble the
// freshness key. The producer polls for commits, params and as-of changes, including changes
// made outside this process; no stale pass is returned while it catches up.
type Preparation struct {
	mu     sync.Mutex
	scopes map[Scope]*Cache
}

func (p *Preparation) cache(scope Scope) *Cache {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.scopes == nil {
		p.scopes = make(map[Scope]*Cache)
	}
	if p.scopes[scope] == nil {
		p.scopes[scope] = &Cache{}
	}
	return p.scopes[scope]
}

// Read never performs valuation. Unknown scopes join the producer's next sweep.
func (p *Preparation) Read(ctx context.Context, src Source, league, franchise string, season int) (Reading, error) {
	src.Cache = p.cache(Scope{league, season})
	src.PreparedOnly = true
	return src.Read(ctx, league, franchise, season)
}

// Run prepares discovered leagues at startup, then refreshes every tracked scope. Its caller
// owns cancellation and joins it before closing stores. A single producer bounds expensive work.
func (p *Preparation) Run(ctx context.Context, src Source, discover func(context.Context) ([]Scope, error), changed func(), report func(error)) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		scopes, err := discover(ctx)
		if err != nil {
			report(err)
		}
		for _, scope := range scopes {
			p.cache(scope)
		}
		p.sweep(ctx, src, changed, report)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (p *Preparation) sweep(ctx context.Context, src Source, changed func(), report func(error)) {
	p.mu.Lock()
	scopes := make(map[Scope]*Cache, len(p.scopes))
	for scope, cache := range p.scopes {
		scopes[scope] = cache
	}
	p.mu.Unlock()
	src.PreparedOnly = false
	for scope, cache := range scopes {
		if ctx.Err() != nil {
			return
		}
		src.Cache = cache
		before := cache.version()
		_, err := src.Read(ctx, scope.League, "", scope.Season)
		if err != nil {
			report(err)
		}
		if cache.version() != before {
			changed()
		}
	}
}

func (c *Cache) version() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.generation
}

// Scopes discovers current roster seasons; unseeded leagues have no valuation to prepare.
func (src Source) Scopes(ctx context.Context, list func(context.Context) ([]commissioner.NativeLeague, error)) ([]Scope, error) {
	leagues, err := list(ctx)
	if err != nil {
		return nil, fmt.Errorf("prepare values leagues: %w", err)
	}
	scopes := make([]Scope, 0, len(leagues))
	for _, league := range leagues {
		records, err := src.Records(ctx, league.ID)
		if err != nil {
			return nil, err
		}
		base, err := commissioner.ReadNativeBase(league.ID, records, src.Now())
		if err != nil {
			return nil, fmt.Errorf("prepare values %s: %w", league.ID, err)
		}
		if base.Assets.Seeded {
			scopes = append(scopes, Scope{League: league.ID, Season: base.Assets.Snapshot.Season})
		}
	}
	return scopes, nil
}
