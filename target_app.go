package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
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
	a.targetSnapshotMu.Lock()
	a.targetSnapshot = snap
	a.hasTargetSnapshot = true
	a.targetSnapshotMu.Unlock()
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

// TargetClock reads the phase log and commissioner calendar from the local store; no network.
func (a *App) TargetClock() (snapshot.Sourced[leagueclock.Reading], error) {
	if err := a.ready(); err != nil {
		return snapshot.Sourced[leagueclock.Reading]{}, fmt.Errorf("target clock: startup: %w", err)
	}
	if err := a.whatif.Err(); err != nil {
		return snapshot.Sourced[leagueclock.Reading]{}, fmt.Errorf("target clock: state: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, m2Timeout)
	defer cancel()
	now := time.Now()
	lineup, fresh := a.weekClock(now)
	clock, err := snapshot.BuildClock(ctx, now.UTC(), a.league.Season(), a.whatif, lineup, snapshot.Provenance{
		Source: "phase-log+commissioner-calendar+nflSchedule", Kind: "live", Freshness: fresh,
	})
	if err != nil {
		return snapshot.Sourced[leagueclock.Reading]{}, fmt.Errorf("target clock: %w", err)
	}
	return clock, nil
}

func (a *App) TargetDraftIR(franchiseID, playerID string) (envelope.Receipt, error) {
	if err := a.ready(); err != nil {
		return envelope.Receipt{}, fmt.Errorf("target draft: startup: %w", err)
	}
	pid, err := playerid.New(playerID)
	if err != nil {
		return envelope.Receipt{}, fmt.Errorf("target draft: player: %w", err)
	}
	a.targetSnapshotMu.Lock()
	snap, loaded := a.targetSnapshot, a.hasTargetSnapshot
	a.targetSnapshotMu.Unlock()
	if !loaded {
		return envelope.Receipt{}, fmt.Errorf("target draft: no snapshot loaded")
	}
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return envelope.Receipt{}, fmt.Errorf("target draft: correlation ID: %w", err)
	}
	id := "ir-" + hex.EncodeToString(random[:])
	e, err := envelope.DraftIR(time.Now().UTC(), envelope.IRRequest{
		LeagueID:    ingestion.LeagueID,
		FranchiseID: franchiseID,
		Player:      pid,
	}, snap, func() string { return id })
	if err != nil {
		return envelope.Receipt{}, fmt.Errorf("target draft: %w", err)
	}
	if err := a.targetMoveLog.Append(e); err != nil {
		return envelope.Receipt{}, fmt.Errorf("target draft: log: %w", err)
	}
	return e.Receipt(), nil
}

func (a *App) TargetMoves(franchiseID string) ([]envelope.Receipt, error) {
	if err := a.ready(); err != nil {
		return nil, fmt.Errorf("target moves: startup: %w", err)
	}
	return a.targetMoveLog.List(envelope.Filter{
		LeagueID:    ingestion.LeagueID,
		FranchiseID: franchiseID,
	}), nil
}
