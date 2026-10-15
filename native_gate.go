package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// NativeOwnerAction is one thing an owner can do in an app-owned league. Run takes the gate's Permit, so
// the only way to start it is runNativeAction, which asks the calendar first. Input is the owner's request
// as sent; the action decodes it and records its change on the league ledger, returning the receipt.
type NativeOwnerAction struct {
	Name  string
	Opens leaguerules.Action
	Run   func(ctx context.Context, app *App, permit commissioner.Permit, input json.RawMessage) (commissioner.Receipt, error)
}

// errActionClosed marks the gate's no, so a caller can tell a shut window from an action that failed.
var errActionClosed = errors.New("league action closed")

// nativeOwnerActions is the registry. Each chunk that adds an owner action to the app-owned league adds
// it here; native_gate_test.go holds every entry to the gate. The contract actions an owner takes under a
// window are registered from the shared list of priced actions, so a new one cannot be priced without being
// gated.
func nativeOwnerActions() []NativeOwnerAction {
	registry := []NativeOwnerAction{lineupSetAction(), draftPickAction(), waiverAction(commissioner.RosterCut),
		waiverAction(commissioner.WaiverClaim)}
	for _, action := range nativecheck.ContractActions() {
		window, gated := commissioner.OwnerContractWindow(action)
		if !gated {
			continue
		}
		registry = append(registry, NativeOwnerAction{Name: contractActionName(action), Opens: window,
			Run: func(ctx context.Context, a *App, permit commissioner.Permit, _ json.RawMessage) (commissioner.Receipt, error) {
				return commissioner.Receipt{}, a.startContractAction(ctx, permit, action)
			}})
	}
	return registry
}

func contractActionName(action nativecheck.ContractAction) string {
	return "contract." + string(action)
}

// startContractAction is the first thing an owner's contract action does once the gate has said yes:
// confirm the league has that action on. Choosing the contract and paying its price follow it, through the
// same QuoteContract check the preview reads; until that is built the action records nothing.
func (a *App) startContractAction(ctx context.Context, permit commissioner.Permit,
	action nativecheck.ContractAction) error {
	records, err := a.leagueRecords(ctx, permit.League())
	if err != nil {
		return err
	}
	if err := commissioner.ContractActionOn(records, a.timeNow(), action); err != nil {
		return fmt.Errorf("start %s: %w", action.Name(), err)
	}
	return nil
}

// ownerZone is where the owner reads times: the desktop's own zone unless a test pins another.
func (a *App) ownerZone() *time.Location {
	if a.localZone != nil {
		return a.localZone
	}
	return time.Local
}

// runNativeAction is the single door to a native owner action: it names the action in the registry, asks
// the gate at the current time and runs the action only on a yes. A closed gate returns its sentence.
func (a *App) runNativeAction(registry []NativeOwnerAction, league, franchise, name string,
	input json.RawMessage) (commissioner.Receipt, error) {
	ctx := a.ctx
	index := slices.IndexFunc(registry, func(r NativeOwnerAction) bool { return r.Name == name })
	if index < 0 {
		return commissioner.Receipt{}, fmt.Errorf("this league action is not available")
	}
	action := registry[index]
	if err := a.ready(); err != nil {
		return commissioner.Receipt{}, fmt.Errorf("league action startup: %w", err)
	}
	records, err := a.leagueRecords(ctx, league)
	if err != nil {
		return commissioner.Receipt{}, err
	}
	gate, err := commissioner.NativeActionGate(league, records, franchise, action.Opens, a.timeNow(), a.ownerZone())
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("league action gate: %w", err)
	}
	permit, ok := gate.Permit()
	if !ok {
		return commissioner.Receipt{}, fmt.Errorf("%w: %w", errActionClosed, commissioner.Refused(errors.New(gate.Sentence)))
	}
	return action.Run(ctx, a, permit, input)
}
