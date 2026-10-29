package platform

import (
	"context"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

func ReadSnapshot(ctx context.Context, directory func(context.Context) snapshot.Directory, injuries func(context.Context) snapshot.Injuries,
	build func(snapshot.Directory, snapshot.Injuries) (snapshot.Snapshot, error), publish func(snapshot.Snapshot)) (snapshot.Snapshot, error) {
	dir := directory(ctx)
	inj := injuries(ctx)
	snap, err := build(dir, inj)
	if err != nil {
		return snapshot.Snapshot{}, fmt.Errorf("target snapshot: build: %w", err)
	}
	publish(snap)
	return snap, nil
}

func SnapshotDirectory(rows []normalize.Lookup, fresh domain.Freshness, err error, lookupAt func() time.Time) snapshot.Directory {
	provenance := snapshot.Provenance{Source: "mfl-players", Kind: "live", Freshness: fresh}
	if err != nil {
		provenance.Freshness = domain.Freshness{State: domain.FreshFail, Note: report.FailureText(err)}
		return snapshot.Directory{Provenance: provenance}
	}
	if fresh.State == domain.FreshLive {
		provenance.Freshness.FetchedAt = lookupAt().UTC().Format(time.RFC3339)
	} else {
		provenance.Source = "mfl-players-archive"
	}
	return snapshot.Directory{Lookup: rows[0], Provenance: provenance}
}

func SnapshotInjuries(ctx context.Context, held bool, timeout time.Duration, refresh func(context.Context), read func() snapshot.Injuries) snapshot.Injuries {
	if !held {
		inner, cancel := context.WithTimeout(ctx, timeout)
		refresh(inner)
		cancel()
	}
	return read()
}
