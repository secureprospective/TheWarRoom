package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"slices"

	contractsapp "github.com/secureprospective/TheWarRoom/internal/app/contracts"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
	"github.com/secureprospective/TheWarRoom/internal/transactions"
)

type RawReleaseRequest struct {
	Mine      string `json:"mine"`
	Action    string `json:"action"`
	Move      string `json:"move"`
	ID        string `json:"id"`
	Confirmed bool   `json:"confirmed"`
}

func (r RawReleaseRequest) Validate() error {
	return contractsapp.Release(r).Validate() //nolint:wrapcheck // the service's errors already start "release"
}

type ReleasePlanView struct {
	ID            string           `json:"id"`
	Player        string           `json:"player"`
	Label         string           `json:"label"`
	Charge        string           `json:"charge"`
	Confirmation  string           `json:"confirmation"`
	Post          string           `json:"post"`
	Channel       string           `json:"channel"`
	Receipt       envelope.Receipt `json:"receipt"`
	CanHandOff    bool             `json:"canHandOff"`
	CanRebase     bool             `json:"canRebase"`
	RebaseLabel   string           `json:"rebaseLabel"`
	WithdrawLabel string           `json:"withdrawLabel"`
	Cut           bool             `json:"cut"`
}

func (a *App) TargetReleasePlans(raw RawReleaseRequest) Reading[[]ReleasePlanView] {
	if err := raw.Validate(); err != nil {
		return releaseReading(nil, err)
	}
	if err := a.ready(); err != nil {
		return releaseReading(nil, err)
	}
	a.whatifMu.Lock()
	defer a.whatifMu.Unlock()
	a.refreshMu.Lock()
	defer a.refreshMu.Unlock()
	a.movesMu.Lock()
	defer a.movesMu.Unlock()
	ctx, cancel := context.WithTimeout(a.ctx, m2Timeout)
	defer cancel()
	if err := a.releaseAction(ctx, raw); err != nil {
		return releaseReading(nil, err)
	}
	if err := a.revalidateReleasePlans(ctx); err != nil {
		return releaseReading(nil, err)
	}
	receipts, err := a.moves.List(ctx, ingestion.LeagueID, raw.Mine)
	if err != nil {
		return releaseReading(nil, fmt.Errorf("release list: %w", err))
	}
	out := []ReleasePlanView{}
	for _, r := range receipts {
		if !envelope.IsRelease(r.Spec.Intent) {
			continue
		}
		f, err := a.moves.ReleaseFacts(ctx, r.CorrelationID)
		if err != nil {
			return releaseReading(nil, err)
		}
		out = append(out, releasePlanView(r, f))
	}
	return releaseReading(out, nil)
}
func releaseReading(v []ReleasePlanView, err error) Reading[[]ReleasePlanView] {
	return contractReading(v, "", "local envelopes; MFL mirror", "Plans are drafted from a pinned what-if move, with the engine charge in cents and its baseline stamps. Ready is not accepted. Buyouts require offseason, 2–4 years left and fewer than two other landed/open plans this season. MFL is never submitted to or written. Only explicitly player-labelled dead-cap adjustments can verify a charge; franchise totals cannot.", len(v), err)
}

// releasePlanView is a release plan as the owner reads it, as its wire type.
func releasePlanView(r envelope.Receipt, f envelope.ReleaseFacts) ReleasePlanView {
	return ReleasePlanView(contractsapp.Plan(r, f))
}

func (a *App) releaseAction(ctx context.Context, r RawReleaseRequest) error {
	if r.Action == "read" {
		return nil
	}
	if r.Action == "draft" {
		return a.draftRelease(ctx, r)
	}
	e, err := a.moves.Get(ctx, r.ID)
	if err != nil {
		return fmt.Errorf("release plan: %w", err)
	}
	spec := e.Receipt().Spec
	if !envelope.IsRelease(spec.Intent) || spec.FranchiseID != r.Mine {
		return fmt.Errorf("release: plan outside scope")
	}
	if err := a.revalidateReleasePlans(ctx); err != nil {
		return err
	}
	e, err = a.moves.Get(ctx, r.ID)
	if err != nil {
		return fmt.Errorf("release reload: %w", err)
	}
	if r.Action == "rebase" {
		return a.rebaseRelease(ctx, e)
	}
	if r.Action == "withdraw" {
		return a.withdrawRelease(ctx, e)
	}
	next, err := e.HandOffRelease(a.movesNow(), r.Confirmed)
	if err != nil {
		return fmt.Errorf("release hand-off: %w", err)
	}
	if spec.Intent == envelope.CutIntent {
		if err := validateMoveURL(spec.Target.URL); err != nil {
			return err
		}
	}
	if err := a.moves.Save(ctx, next); err != nil {
		return fmt.Errorf("release hand-off save: %w", err)
	}
	if spec.Intent == envelope.CutIntent {
		a.openURL(ctx, spec.Target.URL)
	}
	a.movesChanged(ctx)
	a.wakeMoves()
	return nil
}
func (a *App) draftRelease(ctx context.Context, r RawReleaseRequest) error {
	s, found, err := state.NewWhatIfSessions(a.whatifPools).Load(ctx, r.Mine)
	if err != nil {
		return fmt.Errorf("release baseline: %w", err)
	}
	index := slices.IndexFunc(s.Moves, func(m state.WhatIfMove) bool { return m.ID == r.Move })
	if !found || index < 0 {
		return fmt.Errorf("release: what-if move no longer exists")
	}
	m := s.Moves[index]
	f, err := contractsapp.ReleaseFacts(s, m)
	if err != nil {
		return err //nolint:wrapcheck // the service's errors already start "release"
	}
	intent, target := envelope.CutIntent, envelope.Target{Kind: envelope.Manual}
	if m.Kind == string(transactions.KindBuyout) {
		intent = envelope.BuyoutIntent
	} else {
		target, _ = a.rosterMoveTarget("257")
	}
	check, err := a.releaseCheck(ctx, r.Mine, intent, "", f)
	if err != nil {
		return err
	}
	spec := envelope.Spec{Intent: intent, LeagueID: ingestion.LeagueID, FranchiseID: r.Mine,
		Subject: envelope.Subject{Players: []playerid.PlayerID{f.Player}}, Expected: envelope.ExpectedChange{Player: f.Player}, Gravity: envelope.G3, Undo: envelope.Irreversible, Target: target}
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return fmt.Errorf("release id: %w", err)
	}
	id := "release-" + hex.EncodeToString(random[:])
	e, err := envelope.DraftRelease(a.movesNow(), spec, check, func() string { return id })
	if err != nil {
		return fmt.Errorf("release draft: %w", err)
	}
	if err := a.moves.SaveRelease(ctx, e, f); err != nil {
		return fmt.Errorf("release save: %w", err)
	}
	a.movesChanged(ctx)
	return nil
}
func (a *App) releaseCheck(ctx context.Context, mine, _, id string, f envelope.ReleaseFacts) (envelope.ReleaseCheck, error) {
	m, _, _ := a.league.ContractsView()
	p, found := contractsapp.Player(m, mine, f.Player.String())
	phase, err := a.whatif.CurrentPhase(ctx)
	if err != nil {
		return envelope.ReleaseCheck{}, fmt.Errorf("release phase: %w", err)
	}
	count, err := a.releaseBuyoutCount(ctx, mine, id, m.Season)
	if err != nil {
		return envelope.ReleaseCheck{}, err
	}
	return envelope.ReleaseCheck{OnRoster: found, Offseason: phase == domain.PhaseOffseason, OtherBuyouts: count, Remaining: p.ContractYear - m.Season, ChargeText: snapshot.ContractAmountOf(f.Charge).Text}, nil
}
func (a *App) releaseBuyoutCount(ctx context.Context, mine, except string, season int) (int, error) {
	receipts, err := a.moves.List(ctx, ingestion.LeagueID, mine)
	if err != nil {
		return 0, fmt.Errorf("release buyout count: %w", err)
	}
	n := 0
	for _, r := range receipts {
		if r.CorrelationID == except || r.Spec.Intent != envelope.BuyoutIntent || (!envelope.ReleaseOpen(r.State) && r.State != envelope.Landed) {
			continue
		}
		f, err := a.moves.ReleaseFacts(ctx, r.CorrelationID)
		if err != nil {
			return 0, fmt.Errorf("release count facts: %w", err)
		}
		if f.Season == season {
			n++
		}
	}
	return n, nil
}

func (a *App) withdrawRelease(ctx context.Context, e envelope.Envelope) error {
	note := contractsapp.WithdrawNote(e.Receipt())
	next, err := e.WithdrawRelease(a.movesNow(), note)
	if err != nil {
		return fmt.Errorf("release withdraw: %w", err)
	}
	if err := a.moves.Save(ctx, next); err != nil {
		return fmt.Errorf("release withdraw save: %w", err)
	}
	a.movesChanged(ctx)
	return nil
}
