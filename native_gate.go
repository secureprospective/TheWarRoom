package main

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// NativeOwnerAction is one thing an owner can do in an app-owned league. Run takes the gate's Permit, so
// the only way to start it is runNativeAction, which asks the calendar first.
type NativeOwnerAction struct {
	Name  string
	Opens leaguerules.Action
	Run   func(ctx context.Context, permit commissioner.Permit) error
}

// nativeOwnerActions is the registry. Each chunk that adds an owner action to the app-owned league adds
// it here; native_gate_test.go holds every entry to the gate.
func nativeOwnerActions() []NativeOwnerAction {
	return []NativeOwnerAction{{Name: draftPickName, Opens: draftPickOpens, Run: runDraftPick}}
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
func (a *App) runNativeAction(registry []NativeOwnerAction, league, franchise, name string) error {
	return a.runNativeActionIn(a.ctx, registry, league, franchise, name)
}

// runNativeActionIn is runNativeAction for an action that takes its details from ctx.
func (a *App) runNativeActionIn(ctx context.Context, registry []NativeOwnerAction, league, franchise,
	name string) error {
	index := slices.IndexFunc(registry, func(r NativeOwnerAction) bool { return r.Name == name })
	if index < 0 {
		return fmt.Errorf("this league action is not available")
	}
	action := registry[index]
	if err := a.ready(); err != nil {
		return fmt.Errorf("league action startup: %w", err)
	}
	records, err := a.leagueRecords(ctx, league)
	if err != nil {
		return err
	}
	gate, err := commissioner.NativeActionGate(league, records, franchise, action.Opens, a.timeNow(), a.ownerZone())
	if err != nil {
		return fmt.Errorf("league action gate: %w", err)
	}
	permit, ok := gate.Permit()
	if !ok {
		return fmt.Errorf("league action closed: %w", commissioner.Refused(errors.New(gate.Sentence)))
	}
	return action.Run(ctx, permit)
}
