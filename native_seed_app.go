package main

import (
	"context"
	"fmt"
	"time"

	nativeapp "github.com/secureprospective/TheWarRoom/internal/app/commissioner/native"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func (a *App) nativeSeedCommand(ctx context.Context,
	input RawNativeSeedRequest) (commissioner.RawCommand, error) {
	raw, found, err := a.core.History.NativeSeedRetry(ctx, input.League, input.Request)
	if err != nil {
		return raw, fmt.Errorf("read starting snapshot request: %w", err)
	}
	if found {
		if raw.Command != commissioner.NativeSeed || raw.Expected != input.Expected || raw.Reason != input.Reason {
			return raw, fmt.Errorf("this request was already used for a different change")
		}
		return raw, nil
	}
	seed, err := a.heldNativeSnapshot()
	if err != nil {
		return raw, err
	}
	return commissioner.RawCommand{League: input.League, Command: commissioner.NativeSeed,
		Args: commissioner.RawArgs{Native: &seed}, Expected: input.Expected, Request: input.Request,
		Reason: input.Reason, Effective: "now", Visibility: "league"}, nil
}

func (a *App) heldNativeSnapshot() (commissioner.RawNativeSnapshot, error) {
	a.core.RefreshMu.Lock()
	defer a.core.RefreshMu.Unlock()
	if a.core.League == nil || a.core.Rulebook == nil {
		return commissioner.RawNativeSnapshot{}, fmt.Errorf("hold the league rosters and settings before seeding")
	}
	snap, at, _ := a.core.League.ContractsView()
	if snap.Season != a.core.Season {
		return commissioner.RawNativeSnapshot{},
			fmt.Errorf("the held rosters and draft reports must be from the same season")
	}
	rosterAt, err := time.Parse(time.RFC3339Nano, at)
	if err != nil {
		return commissioner.RawNativeSnapshot{}, fmt.Errorf("read held roster date: %w", err)
	}
	a.core.SeasonMu.Lock()
	results, future := a.core.SeasonDraftResults, a.core.SeasonFuturePicks
	a.core.SeasonMu.Unlock()
	seed := commissioner.RawNativeSnapshot{Season: snap.Season, RosterAt: rosterAt, DraftAt: results.FetchedAt,
		FutureAt: future.FetchedAt, Players: []commissioner.NativePlayer{}, Picks: []commissioner.NativePick{},
		Adjustments: []commissioner.NativeCharge{}}
	config := a.core.Rulebook.ActiveConfig()
	franchises, mapping := nativeapp.Franchises(config.Franchises)
	seed.Franchises, seed.Conferences = franchises, nativeapp.Conferences(config)
	a.core.TargetSnapshotMu.Lock()
	directory := a.core.TargetSnapshot.Players.Value
	a.core.TargetSnapshotMu.Unlock()
	nativeapp.SeedPlayers(&seed, snap, mapping, directory)
	nativeapp.SeedPicks(&seed, results.Value, future.Value, mapping)
	if err := seed.Validate(); err != nil {
		return commissioner.RawNativeSnapshot{}, fmt.Errorf("held starting snapshot: %w", err)
	}
	return seed, nil
}
