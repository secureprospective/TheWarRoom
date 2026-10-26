package composition

import (
	"context"
	"fmt"
	"time"

	controlapp "github.com/secureprospective/TheWarRoom/internal/app/control"
	platformapp "github.com/secureprospective/TheWarRoom/internal/app/platform"

	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

func (a *App) takeSnapshot() (snapshot.Snapshot, error) {
	if err := a.ready(); err != nil {
		return snapshot.Snapshot{}, fmt.Errorf("target snapshot: startup: %w", err)
	}
	a.core.RefreshMu.Lock()
	defer a.core.RefreshMu.Unlock()
	ctx, cancel := context.WithTimeout(a.core.Ctx, rasFetchTimeout)
	defer cancel()
	return platformapp.ReadSnapshot(ctx, a.targetDirectory, a.targetInjuries, //nolint:wrapcheck // Preserve snapshot build context.
		func(dir snapshot.Directory, inj snapshot.Injuries) (snapshot.Snapshot, error) {
			return snapshot.Build(a.core.Ctx, snapshot.NewSource(a.core.League, a.core.Rulebook), dir, inj)
		},
		func(snap snapshot.Snapshot) {
			a.core.TargetSnapshotMu.Lock()
			defer a.core.TargetSnapshotMu.Unlock()
			a.core.TargetSnapshot, a.core.HasTargetSnapshot = snap, true
		})
}

func (a *App) targetDirectory(ctx context.Context) snapshot.Directory {
	rows, fresh, err := liveOrArchive(ctx, a.fallbackParent(), a.core.League.Season(), archivedFeed[normalize.Lookup]{
		Export: "players",
		Fetch: func(ctx context.Context) ([]normalize.Lookup, error) {
			lk, err := a.directory(ctx)
			if err != nil {
				return nil, fmt.Errorf("target directory: %w", err)
			}
			return []normalize.Lookup{lk}, nil
		},
		Parse: func(body []byte) ([]normalize.Lookup, error) {
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
		Bodies: a.core.History.ArchivedBodies,
	}, a.core.FallbackLog)
	return platformapp.SnapshotDirectory(rows, fresh, err, func() time.Time {
		a.core.LookupMu.Lock()
		defer a.core.LookupMu.Unlock()
		return a.core.LookupAt
	})
}

// targetInjuries returns the held injuries feed, fetching it on first use like the directory,
// so a snapshot taken while the season refresh is still running still carries live rules. The
// caller holds refreshMu; the hold itself is guarded by seasonMu.
func (a *App) targetInjuries(ctx context.Context) snapshot.Injuries {
	return platformapp.SnapshotInjuries(ctx, a.injuriesHeld(), m2Timeout, a.refreshInjuries, a.heldInjuries)
}

func (a *App) injuriesHeld() bool {
	a.core.SeasonMu.Lock()
	defer a.core.SeasonMu.Unlock()
	return !a.core.SeasonInjuries.FetchedAt.IsZero()
}

func (a *App) targetClock(parent context.Context) (snapshot.Sourced[leagueclock.Reading], error) {
	return (controlapp.ClockRead{ //nolint:wrapcheck // Preserve held clock admission/build errors.
		Ready: a.ready, StateError: func() error { return a.core.Whatif.Err() }, Timeout: m2Timeout,
		Now: a.timeNow, Season: func() int { return a.core.League.Season() }, Source: func() snapshot.ClockSource { return a.core.Whatif },
		Lineup: a.weekClock, Windows: a.windowInputs, Merge: a.mergeMFLCalendar,
	}).Read(parent)
}

func (a *App) draftRosterMove(intent, franchiseID, playerID string) (envelope.Receipt, error) {
	return (platformapp.RosterDraft{ //nolint:wrapcheck // Preserve draft admission and persistence errors.
		Ready: a.ready, Own: a.core.RequireOwnFranchise, Snapshot: func() (snapshot.Snapshot, bool) {
			a.core.TargetSnapshotMu.Lock()
			defer a.core.TargetSnapshotMu.Unlock()
			return a.core.TargetSnapshot, a.core.HasTargetSnapshot
		}, Lock: a.core.MovesMu.Lock, Unlock: a.core.MovesMu.Unlock, Target: a.rosterMoveTarget, League: ingestion.LeagueID,
		Parent: func() context.Context { return a.core.Ctx }, Timeout: m2Timeout, Save: func(ctx context.Context, e envelope.Envelope) error { return a.core.Moves.Save(ctx, e) },
	}).Draft(intent, franchiseID, playerID)
}

func (a *App) listMoves(franchiseID string) ([]envelope.Receipt, error) {
	if err := a.ready(); err != nil {
		return nil, fmt.Errorf("target moves: startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.core.Ctx, m2Timeout)
	defer cancel()
	receipts, err := a.core.Moves.List(ctx, ingestion.LeagueID, franchiseID)
	if err != nil {
		return nil, fmt.Errorf("target moves: list: %w", err)
	}
	a.core.TargetSnapshotMu.Lock()
	snap := a.core.TargetSnapshot
	a.core.TargetSnapshotMu.Unlock()
	labelTradeReceipts(receipts, snap)
	return platformapp.RosterRail(receipts), nil
}

func (a *App) rosterMoveTarget(option string) (envelope.Target, []string) {
	host := ""
	if a.core.MFLClient != nil {
		host = a.core.MFLClient.LeagueHost()
	}
	return platformapp.RosterMoveTarget(host, a.core.Season, ingestion.LeagueID, option)
}
