package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

// waiverAction is the registered owner action for a native cut or waiver claim; native_gate_test.go holds
// both to the gate with the rest.
func waiverAction(command string) NativeOwnerAction {
	return NativeOwnerAction{Name: command, Opens: commissioner.OwnerWaiverAction(command), Run: runWaiverAction}
}

// runWaiverAction records a cut or claim cleared by the gate. The store re-asks the gate inside its own
// transaction and makes the waiver decision there.
func runWaiverAction(ctx context.Context, a *App, permit commissioner.Permit,
	input json.RawMessage) (commissioner.Receipt, error) {
	var raw commissioner.RawCommand
	if err := json.Unmarshal(input, &raw); err != nil || raw.Args.Waiver == nil {
		return commissioner.Receipt{}, fmt.Errorf("choose a native cut or waiver claim")
	}
	if permit.League() != raw.League || permit.Franchise() != raw.Args.Waiver.Franchise ||
		permit.Action() != commissioner.OwnerWaiverAction(raw.Command) {
		return commissioner.Receipt{}, fmt.Errorf("ask the league calendar before acting")
	}
	principal, err := a.commissionerPrincipal()
	if err != nil {
		return commissioner.Receipt{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, refreshTimeout)
	defer cancel()
	receipt, err := a.history.ExecuteCommissionerWithFacts(ctx, raw, principal, nil, a.nativeFacts(), a.ownerZone())
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("record native waiver action: %w", err)
	}
	return receipt, nil
}

// TargetNativeWaiverAction is the owner door; identity comes from the local session, not the request.
func (a *App) TargetNativeWaiverAction(raw commissioner.RawCommand) (commissioner.Receipt, error) {
	if commissioner.OwnerWaiverAction(raw.Command) == "" || raw.Args.WaiverOutcome != nil {
		return commissioner.Receipt{}, fmt.Errorf("choose a native cut or waiver claim")
	}
	if err := raw.Validate(); err != nil {
		return commissioner.Receipt{}, fmt.Errorf("native action input: %w", err)
	}
	if err := a.ready(); err != nil {
		return commissioner.Receipt{}, fmt.Errorf("native action startup: %w", err)
	}
	principal, err := a.commissionerPrincipal()
	if err != nil {
		return commissioner.Receipt{}, err
	}
	if receipt, retried, err := a.nativeWaiverRetry(raw, principal); retried || err != nil {
		return receipt, err
	}
	body, err := json.Marshal(raw)
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("native action input: %w", err)
	}
	receipt, err := a.runNativeAction(nativeOwnerActions(), raw.League, raw.Args.Waiver.Franchise, raw.Command, body)
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("native waiver action: %w", err)
	}
	return receipt, nil
}

func (a *App) TargetNativeWaivers(league, franchise string, at time.Time) (commissioner.NativeWaivers, error) {
	if at.IsZero() {
		return commissioner.NativeWaivers{}, fmt.Errorf("choose when to read waivers")
	}
	if err := a.ready(); err != nil {
		return commissioner.NativeWaivers{}, fmt.Errorf("waivers startup: %w", err)
	}
	principal, err := a.commissionerPrincipal()
	if err != nil {
		return commissioner.NativeWaivers{}, err
	}
	reading, err := a.history.NativeWaivers(a.ctx, league, franchise, principal, at, a.ownerZone(), a.nativeFacts())
	if err != nil {
		return commissioner.NativeWaivers{}, fmt.Errorf("native waivers: %w", err)
	}
	return reading, nil
}

// nativeWaiverClock resumes persisted windows on startup and checks them without any network feed.
func (a *App) nativeWaiverClock(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		a.processNativeWaiversDue(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (a *App) processNativeWaiversDue(ctx context.Context) {
	leagues, err := a.history.NativeLeagues(ctx)
	if err != nil {
		log.Printf("the war room: read native waiver leagues: %v", err)
		return
	}
	facts := a.nativeFacts()
	for _, league := range leagues {
		if err := a.history.ProcessDueWaivers(ctx, league.ID, facts); err != nil {
			log.Printf("the war room: process native waivers for %s: %v", league.ID, err)
		}
	}
}

// A replay does not mutate a roster, so a closed calendar does not discard a recorded receipt.
func (a *App) nativeWaiverRetry(raw commissioner.RawCommand, principal string) (commissioner.Receipt, bool, error) {
	_, found, err := a.history.NativeSeedRetry(a.ctx, raw.League, raw.Request)
	if err != nil {
		return commissioner.Receipt{}, false, fmt.Errorf("read native action retry: %w", err)
	}
	if !found {
		return commissioner.Receipt{}, false, nil
	}
	receipt, err := a.history.ExecuteCommissionerWithFacts(a.ctx, raw, principal, nil, a.nativeFacts(), a.ownerZone())
	if err != nil {
		return commissioner.Receipt{}, true, fmt.Errorf("retry native action: %w", err)
	}
	return receipt, true, nil
}
