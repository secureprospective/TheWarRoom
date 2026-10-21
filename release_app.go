package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"slices"
	"time"

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

func (a *App) rebaseRelease(ctx context.Context, e envelope.Envelope) error {
	if e.State() != envelope.Stale {
		return fmt.Errorf("release rebase: only a stale plan can be repriced")
	}
	r := e.Receipt()
	old, err := a.moves.ReleaseFacts(ctx, e.ID())
	if err != nil {
		return fmt.Errorf("release facts: %w", err)
	}
	m, _, _ := a.league.ContractsView()
	stamps, err := a.league.Stamps(ctx)
	if err != nil {
		return fmt.Errorf("release rebase stamps: %w", err)
	}
	kind := string(transactions.KindWaiver)
	if r.Spec.Intent == envelope.BuyoutIntent {
		kind = string(transactions.KindBuyout)
	}
	s := state.WhatIfSession{Mine: r.Spec.FranchiseID, Snapshot: m, Stamps: stamps, Generation: "release-rebase:" + e.ID() + ":" + fmt.Sprint(time.Now().UnixNano()), TaxiPercent: a.rulebook.TaxiCapPercent(), IRPercent: a.rulebook.IRCapPercent()}
	move := state.WhatIfMove{Kind: kind, Player: old.Player.String(), Name: old.Name}
	f, err := contractsapp.ReleaseFacts(s, move)
	if err != nil {
		return err //nolint:wrapcheck // the service's errors already start "release"
	}
	f.Charge, err = a.releaseCharge(ctx, s, move)
	if err != nil {
		return err
	}
	check, err := a.releaseCheck(ctx, r.Spec.FranchiseID, r.Spec.Intent, e.ID(), f)
	if err != nil {
		return err
	}
	note := snapshot.ContractAmountOf(old.Charge).Text + " → " + snapshot.ContractAmountOf(f.Charge).Text + " dead cap on current mirror"
	check.ChargeText = note
	e, err = e.Rebase(a.movesNow(), note)
	if err == nil {
		e, err = e.Check(a.movesNow(), snapshot.Snapshot{}, check)
	}
	if err != nil {
		return fmt.Errorf("release rebase: %w", err)
	}
	if err := a.moves.SaveRelease(ctx, e, f); err != nil {
		return fmt.Errorf("release rebase save: %w", err)
	}
	a.movesChanged(ctx)
	return nil
}
func (a *App) releaseCharge(ctx context.Context, s state.WhatIfSession, m state.WhatIfMove) (domain.Money, error) {
	store, err := a.loadWhatIf(ctx, s)
	if err != nil {
		return 0, err
	}
	coord, err := transactions.New(store.Writer(), nil, nil)
	if err != nil {
		return 0, fmt.Errorf("release rebase coordinator: %w", err)
	}
	req, err := buildRequest(TransactionRequest{Kind: m.Kind, MFLID: m.Player})
	if err != nil {
		return 0, fmt.Errorf("release rebase request: %w", err)
	}
	receipt, err := coord.Execute(ctx, req)
	if err != nil {
		return 0, fmt.Errorf("release rebase engine: %w", err)
	}
	if len(receipt.CapDeltas) != 1 {
		return 0, fmt.Errorf("release rebase: engine charge missing")
	}
	return receipt.CapDeltas[0].Cents, nil
}

// Caller holds refreshMu and movesMu; this touches only the local plan ledger.
func (a *App) revalidateReleasePlans(ctx context.Context) error {
	receipts, err := a.moves.List(ctx, ingestion.LeagueID, "")
	if err != nil {
		return fmt.Errorf("release validation list: %w", err)
	}
	for _, r := range receipts {
		if !envelope.IsRelease(r.Spec.Intent) {
			continue
		}
		if err := a.revalidateRelease(ctx, r); err != nil {
			return err
		}
	}
	return nil
}
func (a *App) revalidateRelease(ctx context.Context, r envelope.Receipt) error {
	f, err := a.moves.ReleaseFacts(ctx, r.CorrelationID)
	if err != nil {
		return fmt.Errorf("release facts: %w", err)
	}
	m, _, _ := a.league.ContractsView()
	if r.State == envelope.Landed {
		return a.releaseChargeNote(ctx, r, f, m)
	}
	if !contractsapp.NeedsValidation(r.State) {
		return nil
	}
	p, onRoster := contractsapp.Player(m, r.Spec.FranchiseID, f.Player.String())
	stamps, err := a.league.ReleaseStamps(ctx, f.Player.String(), r.Spec.FranchiseID)
	if err != nil {
		return fmt.Errorf("release validation stamps: %w", err)
	}
	adjustments, err := contractsapp.Adjustments(m, r.Spec.FranchiseID)
	if err != nil {
		return err //nolint:wrapcheck // the service's errors already start "release"
	}
	phase, err := a.whatif.CurrentPhase(ctx)
	if err != nil {
		return fmt.Errorf("release validation phase: %w", err)
	}
	contractChanged := contractsapp.ContractChanged(p, f) ||
		(state.MirrorStamps{Contract: stamps.Contract}).NewerThan(state.MirrorStamps{Contract: f.ContractStamp})
	adjustmentChanged := adjustments != f.Adjustments || (state.MirrorStamps{DeadCap: stamps.DeadCap}).NewerThan(state.MirrorStamps{DeadCap: f.DeadCapStamp})
	reason := envelope.ReleaseInvalidation(r.Spec.Intent, r.State != envelope.Ready, onRoster, phase == domain.PhaseOffseason, contractChanged, adjustmentChanged)
	if reason == "" {
		return nil
	}
	e, err := a.moves.Get(ctx, r.CorrelationID)
	if err != nil {
		return fmt.Errorf("release invalidate load: %w", err)
	}
	next, err := e.Invalidate(a.movesNow(), reason)
	if err != nil {
		return fmt.Errorf("release invalidate: %w", err)
	}
	if err := a.moves.Save(ctx, next); err != nil {
		return fmt.Errorf("release invalidate save: %w", err)
	}
	a.movesChanged(ctx)
	return nil
}
func (a *App) releaseChargeNote(ctx context.Context, r envelope.Receipt, f envelope.ReleaseFacts, m state.MirrorSnapshot) error {
	note := contractsapp.ChargeNote(f, r.Spec.FranchiseID, m)
	if len(r.Audit) > 0 && r.Audit[len(r.Audit)-1].Note == note {
		return nil
	}
	e, err := a.moves.Get(ctx, r.CorrelationID)
	if err != nil {
		return fmt.Errorf("release charge note load: %w", err)
	}
	next, err := e.Invalidate(a.movesNow(), note)
	if err != nil {
		return fmt.Errorf("release charge note: %w", err)
	}
	if err := a.moves.Save(ctx, next); err != nil {
		return fmt.Errorf("release charge note save: %w", err)
	}
	a.movesChanged(ctx)
	return nil
}
