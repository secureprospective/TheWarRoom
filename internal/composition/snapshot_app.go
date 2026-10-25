package composition

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
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
	dir := a.targetDirectory(ctx)
	inj := a.targetInjuries(ctx)
	snap, err := snapshot.Build(a.core.Ctx, snapshot.NewSource(a.core.League, a.core.Rulebook), dir, inj)
	if err != nil {
		return snapshot.Snapshot{}, fmt.Errorf("target snapshot: build: %w", err)
	}
	a.core.TargetSnapshotMu.Lock()
	a.core.TargetSnapshot = snap
	a.core.HasTargetSnapshot = true
	a.core.TargetSnapshotMu.Unlock()
	return snap, nil
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
	p := snapshot.Provenance{Source: "mfl-players", Kind: "live", Freshness: fresh}
	if err != nil {
		p.Freshness = domain.Freshness{State: domain.FreshFail, Note: report.FailureText(err)}
		return snapshot.Directory{Provenance: p}
	}
	if fresh.State == domain.FreshLive {
		a.core.LookupMu.Lock()
		p.Freshness.FetchedAt = a.core.LookupAt.UTC().Format(time.RFC3339)
		a.core.LookupMu.Unlock()
	} else {
		p.Source = "mfl-players-archive"
	}
	return snapshot.Directory{Lookup: rows[0], Provenance: p}
}

// targetInjuries returns the held injuries feed, fetching it on first use like the directory,
// so a snapshot taken while the season refresh is still running still carries live rules. The
// caller holds refreshMu; the hold itself is guarded by seasonMu.
func (a *App) targetInjuries(ctx context.Context) snapshot.Injuries {
	if !a.injuriesHeld() {
		inner, cancel := context.WithTimeout(ctx, m2Timeout)
		a.refreshInjuries(inner)
		cancel()
	}
	return a.heldInjuries()
}

func (a *App) injuriesHeld() bool {
	a.core.SeasonMu.Lock()
	defer a.core.SeasonMu.Unlock()
	return !a.core.SeasonInjuries.FetchedAt.IsZero()
}

func (a *App) targetClock(parent context.Context) (snapshot.Sourced[leagueclock.Reading], error) {
	if err := a.ready(); err != nil {
		return snapshot.Sourced[leagueclock.Reading]{}, fmt.Errorf("target clock: startup: %w", err)
	}
	if err := a.core.Whatif.Err(); err != nil {
		return snapshot.Sourced[leagueclock.Reading]{}, fmt.Errorf("target clock: state: %w", err)
	}
	ctx, cancel := context.WithTimeout(parent, m2Timeout)
	defer cancel()
	now := a.timeNow()
	lineup, fresh := a.weekClock(now)
	clock, err := snapshot.BuildClock(ctx, now.UTC(), a.core.League.Season(), a.core.Whatif, lineup, a.windowInputs(now), snapshot.Provenance{
		Source: "phase-log+commissioner-calendar+nflSchedule", Kind: "live", Freshness: fresh,
	})
	if err != nil {
		return snapshot.Sourced[leagueclock.Reading]{}, fmt.Errorf("target clock: %w", err)
	}
	a.mergeMFLCalendar(ctx, now, &clock)
	return clock, nil
}

func (a *App) rosterMoveTarget(option string) (envelope.Target, []string) {
	host := ""
	if a.core.MFLClient != nil {
		host = a.core.MFLClient.LeagueHost()
	}
	if host == "" {
		return envelope.Target{Kind: envelope.Unmapped}, []string{"MFL host not yet known"}
	}
	return envelope.Target{Kind: envelope.Mapped, URL: fmt.Sprintf(
		"https://%s/%d/options?L=%s&O=%s", host, a.core.Season, url.QueryEscape(ingestion.LeagueID), option)}, nil
}

func (a *App) draftRosterMove(intent, franchiseID, playerID string) (envelope.Receipt, error) {
	if err := a.ready(); err != nil {
		return envelope.Receipt{}, fmt.Errorf("target draft: startup: %w", err)
	}
	if err := a.core.RequireOwnFranchise(franchiseID); err != nil {
		return envelope.Receipt{}, err //nolint:wrapcheck // Preserve the shared authorization refusal.
	}
	pid, err := playerid.New(playerID)
	if err != nil {
		return envelope.Receipt{}, fmt.Errorf("target draft: player: %w", err)
	}
	a.core.TargetSnapshotMu.Lock()
	snap, loaded := a.core.TargetSnapshot, a.core.HasTargetSnapshot
	a.core.TargetSnapshotMu.Unlock()
	if !loaded {
		return envelope.Receipt{}, fmt.Errorf("target draft: no snapshot loaded")
	}
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return envelope.Receipt{}, fmt.Errorf("target draft: correlation ID: %w", err)
	}
	a.core.MovesMu.Lock()
	defer a.core.MovesMu.Unlock()
	option, prefix := "18", "ir-"
	if intent == "roster.taxi" {
		option, prefix = "98", "taxi-"
	}
	target, blocks := a.rosterMoveTarget(option)
	req := envelope.IRRequest{
		LeagueID:    ingestion.LeagueID,
		FranchiseID: franchiseID,
		Player:      pid,
		Target:      target,
		Blocks:      blocks,
	}
	id := prefix + hex.EncodeToString(random[:])
	var e envelope.Envelope
	if intent == "roster.taxi" {
		e, err = envelope.DraftTaxi(time.Now().UTC(), envelope.TaxiRequest(req), snap, func() string { return id })
	} else {
		e, err = envelope.DraftIR(time.Now().UTC(), req, snap, func() string { return id })
	}
	if err != nil {
		return envelope.Receipt{}, fmt.Errorf("target draft: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.core.Ctx, m2Timeout)
	defer cancel()
	if err := a.core.Moves.Save(ctx, e); err != nil {
		return envelope.Receipt{}, fmt.Errorf("target draft: log: %w", err)
	}
	return e.Receipt(), nil
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
	// Bid plans carry facts the move rail cannot show; TargetBidPlans lists them.
	return slices.DeleteFunc(receipts, func(r envelope.Receipt) bool {
		return r.Spec.Intent == envelope.BidIntent || envelope.IsRelease(r.Spec.Intent) || envelope.IsContract(r.Spec.Intent)
	}), nil
}
