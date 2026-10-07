package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/lineup"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

type rawLineupSelection struct{ Starters []string }

func (r rawLineupSelection) Validate() ([]playerid.PlayerID, error) {
	ids := make([]playerid.PlayerID, 0, len(r.Starters))
	for _, raw := range r.Starters {
		id, err := playerid.New(raw)
		if err != nil {
			return nil, fmt.Errorf("lineup selection: player: %w", err)
		}
		if slices.Contains(ids, id) {
			return nil, fmt.Errorf("lineup selection: duplicate player %s", id)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (a *App) lineupSelection(
	franchiseID string, starters []string,
) (snapshot.Snapshot, []playerid.PlayerID, error) {
	if err := a.ready(); err != nil {
		return snapshot.Snapshot{}, nil, fmt.Errorf("lineup selection: startup: %w", err)
	}
	a.targetSnapshotMu.Lock()
	snap, loaded := a.targetSnapshot, a.hasTargetSnapshot
	a.targetSnapshotMu.Unlock()
	if !loaded {
		return snapshot.Snapshot{}, nil, fmt.Errorf("lineup selection: no snapshot loaded")
	}
	if !slices.ContainsFunc(snap.Franchises.Value, func(f snapshot.Franchise) bool {
		return f.ID == franchiseID
	}) {
		return snapshot.Snapshot{}, nil, fmt.Errorf("lineup selection: unknown franchise %q", franchiseID)
	}
	ids, err := (rawLineupSelection{Starters: starters}).Validate()
	if err != nil {
		return snapshot.Snapshot{}, nil, err
	}
	return snap, ids, nil
}

// TargetCheckLineup judges a lineup being edited in HQ: local only, no network, never refreshMu,
// never saved. Unknown positions are problems in the Result, not errors.
func (a *App) TargetCheckLineup(franchiseID string, starters []string) (lineup.Result, error) {
	snap, ids, err := a.lineupSelection(franchiseID, starters)
	if err != nil {
		return lineup.Result{}, fmt.Errorf("target check lineup: %w", err)
	}
	rules, err := lineup.ParseRules(a.rulebook.ActiveConfig().Starters)
	if err != nil {
		return unreadableLineupRules(err), nil
	}
	return lineup.Check(rules, envelope.LineupPositions(ids, snap)), nil
}

func (a *App) lineupRequest(
	franchiseID string, ids []playerid.PlayerID, now time.Time,
) (envelope.LineupRequest, envelope.LineupCheck) {
	a.seasonMu.Lock()
	held := a.seasonLineups
	a.seasonMu.Unlock()
	a.weekMu.Lock()
	week := a.week
	a.weekMu.Unlock()
	req := envelope.LineupRequest{
		LeagueID: ingestion.LeagueID, FranchiseID: franchiseID, Week: held.value.Week,
		Starters: ids, Target: envelope.Target{Kind: envelope.Unmapped},
	}
	c := envelope.LineupCheck{BaselineKnown: !held.fetchedAt.IsZero() && held.refreshError == ""}
	if c.BaselineKnown {
		for _, saved := range held.value.Franchises {
			if saved.Franchise == franchiseID {
				req.Baseline = saved.Starters
			}
		}
	}
	rules, err := lineup.ParseRules(a.rulebook.ActiveConfig().Starters)
	if err != nil {
		c.Blocks = append(c.Blocks, unreadableLineupRules(err).Problems[0].Message)
	}
	c.Rules = rules
	if c.BaselineKnown && req.Week != week.Number {
		c.Blocks = append(c.Blocks, fmt.Sprintf(
			"MFL's saved lineups are for week %d, the league is in week %d", req.Week, week.Number))
	}
	// With no held feed, scope the blocked plan to the known league week, not an invented week.
	if req.Week == 0 {
		req.Week = week.Number
	}
	// The plan can land until the week's last kickoff; earlier games lock only their own players.
	switch {
	case len(week.Games) == 0:
		c.Blocks = append(c.Blocks, "lineup lock unknown")
	case !leagueweek.LastLock(week).After(now):
		c.Blocks = append(c.Blocks, fmt.Sprintf("week %d lineups are locked", week.Number))
	default:
		last := leagueweek.LastLock(week)
		req.Deadline = &last
	}
	host := ""
	if a.mflClient != nil {
		host = a.mflClient.LeagueHost()
	}
	if host == "" {
		c.Blocks = append(c.Blocks, "MFL host not yet known")
	} else {
		req.Target = envelope.Target{Kind: envelope.Mapped, URL: fmt.Sprintf(
			"https://%s/%d/lineup?L=%s&WEEK=%d&F=%s", host, a.season,
			url.QueryEscape(req.LeagueID), req.Week, url.QueryEscape(franchiseID))}
	}
	return req, c
}

// TargetDraftLineup saves a checked lineup.set plan for the held week, after moving earlier live
// plans for the same franchise and week to Stale.
func (a *App) TargetDraftLineup(franchiseID string, starters []string) (envelope.Receipt, error) {
	snap, ids, err := a.lineupSelection(franchiseID, starters)
	if err != nil {
		return envelope.Receipt{}, fmt.Errorf("target draft lineup: %w", err)
	}
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return envelope.Receipt{}, fmt.Errorf("target draft lineup: correlation ID: %w", err)
	}
	a.movesMu.Lock()
	defer a.movesMu.Unlock()
	now := a.movesNow()
	req, check := a.lineupRequest(franchiseID, ids, now)
	id := "lineup-" + hex.EncodeToString(random[:])
	e, err := envelope.DraftLineup(now, req, snap, check, func() string { return id })
	if err != nil {
		return envelope.Receipt{}, fmt.Errorf("target draft lineup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, m2Timeout)
	defer cancel()
	if err := a.supersedeLineups(ctx, e, now); err != nil {
		return envelope.Receipt{}, err
	}
	if err := a.moves.Save(ctx, e); err != nil {
		return envelope.Receipt{}, fmt.Errorf("target draft lineup: save: %w", err)
	}
	return e.Receipt(), nil
}

func (a *App) supersedeLineups(ctx context.Context, next envelope.Envelope, now time.Time) error {
	return a.supersedeMoves(ctx, next, now)
}

func competingMove(left, right envelope.Spec) bool {
	if left.Intent != right.Intent {
		return false
	}
	switch left.Intent {
	case "lineup.set":
		return left.Expected.Lineup.Week == right.Expected.Lineup.Week
	case "trade.accept":
		return left.Expected.Trade.TradeID == right.Expected.Trade.TradeID
	default:
		return false
	}
}

func (a *App) supersedeMoves(ctx context.Context, next envelope.Envelope, now time.Time) error {
	spec := next.Receipt().Spec
	receipts, err := a.moves.List(ctx, spec.LeagueID, spec.FranchiseID)
	if err != nil {
		return fmt.Errorf("supersede move: list: %w", err)
	}
	// Only a plan that can still be handed off or land competes; drafts and blocked plans are inert.
	live := []envelope.State{envelope.Ready, envelope.HandedOff, envelope.NotYetDone, envelope.NotVerified,
		envelope.DOTReview}
	for _, r := range receipts {
		if !competingMove(r.Spec, spec) || !slices.Contains(live, r.State) {
			continue
		}
		old, err := a.moves.Get(ctx, r.CorrelationID)
		if err != nil {
			return fmt.Errorf("supersede move: load: %w", err)
		}
		stale, err := old.Invalidate(now, "superseded by "+next.ID())
		if err != nil {
			return fmt.Errorf("supersede move: invalidate: %w", err)
		}
		if err := a.moves.Save(ctx, stale); err != nil {
			return fmt.Errorf("supersede move: save: %w", err)
		}
	}
	return nil
}
