package main

import (
	"context"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

func (a *App) TargetSnapshot() (snapshot.Snapshot, error) {
	if err := a.ready(); err != nil {
		return snapshot.Snapshot{}, fmt.Errorf("target snapshot: startup: %w", err)
	}
	a.refreshMu.Lock()
	defer a.refreshMu.Unlock()
	ctx, cancel := context.WithTimeout(a.ctx, rasFetchTimeout)
	defer cancel()
	dir := a.targetDirectory(ctx)
	snap, err := snapshot.Build(a.ctx, snapshot.NewSource(a.league, a.rulebook), dir)
	if err != nil {
		return snapshot.Snapshot{}, fmt.Errorf("target snapshot: build: %w", err)
	}
	return snap, nil
}

func (a *App) targetDirectory(ctx context.Context) snapshot.Directory {
	rows, fresh, err := liveOrArchive(ctx, a.fallbackParent(), a.league.Season(), archivedFeed[normalize.Lookup]{
		export: "players",
		fetch: func(ctx context.Context) ([]normalize.Lookup, error) {
			lk, err := a.directory(ctx)
			if err != nil {
				return nil, fmt.Errorf("target directory: %w", err)
			}
			return []normalize.Lookup{lk}, nil
		},
		parse: func(body []byte) ([]normalize.Lookup, error) {
			raw, err := players.Parse(context.WithoutCancel(ctx), body)
			if err != nil {
				return nil, fmt.Errorf("target directory: parse archive: %w", err)
			}
			lk, err := normalize.NewLookup(raw)
			if err != nil {
				return nil, fmt.Errorf("target directory: normalize archive: %w", err)
			}
			return []normalize.Lookup{lk}, nil
		},
		bodies: a.history.ArchivedBodies,
	})
	p := snapshot.Provenance{Source: "mfl-players", Kind: "live", Freshness: fresh}
	if err != nil {
		p.Freshness = domain.Freshness{State: domain.FreshFail, Note: err.Error()}
		return snapshot.Directory{Provenance: p}
	}
	if fresh.State == domain.FreshLive {
		a.lookupMu.Lock()
		p.Freshness.FetchedAt = a.lookupAt.UTC().Format(time.RFC3339)
		a.lookupMu.Unlock()
	} else {
		p.Source = "mfl-players-archive"
	}
	return snapshot.Directory{Lookup: rows[0], Provenance: p}
}
