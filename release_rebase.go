package main

import (
	"context"
	"fmt"
	"time"

	contractsapp "github.com/secureprospective/TheWarRoom/internal/app/contracts"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
	"github.com/secureprospective/TheWarRoom/internal/transactions"
)

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
