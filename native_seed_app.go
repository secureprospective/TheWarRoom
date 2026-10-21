package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	nativeapp "github.com/secureprospective/TheWarRoom/internal/app/native"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

type RawNativeSeedRequest struct {
	League   string `json:"league"`
	Expected int64  `json:"expected"`
	Request  string `json:"request"`
	Reason   string `json:"reason"`
}

func (r RawNativeSeedRequest) Validate() error { return r.validate(true) }

// ValidatePreview is Validate without the typed reason, so the tray can show the starting assets before
// the reason is written; the preview then owes the reason as its blocker.
func (r RawNativeSeedRequest) ValidatePreview() error { return r.validate(false) }

func (r RawNativeSeedRequest) validate(requireReason bool) error {
	if !strings.HasPrefix(r.League, "native.") || r.League == "native." || strings.TrimSpace(r.League) != r.League ||
		r.Expected < 1 || strings.TrimSpace(r.Request) == "" || (requireReason && strings.TrimSpace(r.Reason) == "") {
		return fmt.Errorf("choose an app league, its version, a request and a reason for recording its starting snapshot")
	}
	return nil
}

// TargetNativeSeed captures held data only. It cannot accept client-provided rosters or fetch missing inputs.
func (a *App) TargetNativeSeed(input RawNativeSeedRequest) (commissioner.Receipt, error) {
	if err := input.Validate(); err != nil {
		return commissioner.Receipt{}, err
	}
	if err := a.ready(); err != nil {
		return commissioner.Receipt{}, fmt.Errorf("starting snapshot startup: %w", err)
	}
	principal, err := a.commissionerPrincipal()
	if err != nil {
		return commissioner.Receipt{}, err
	}
	ctx, cancel := context.WithTimeout(a.core.Ctx, refreshTimeout)
	defer cancel()
	raw, err := a.nativeSeedCommand(ctx, input)
	if err != nil {
		return commissioner.Receipt{}, err
	}
	receipt, err := a.core.History.ExecuteCommissioner(ctx, raw, principal, a.core.CommissionerAuthorizer)
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("record starting snapshot: %w", err)
	}
	return receipt, nil
}

func (a *App) nativeSeedCommand(ctx context.Context, input RawNativeSeedRequest) (commissioner.RawCommand, error) {
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
		return commissioner.RawNativeSnapshot{}, fmt.Errorf("the held rosters and draft reports must be from the same season")
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

func (a *App) TargetNativeDirectory(leagueID string) (commissioner.NativeDirectory, error) {
	if err := a.ready(); err != nil {
		return commissioner.NativeDirectory{}, fmt.Errorf("native directory startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.core.Ctx, refreshTimeout)
	defer cancel()
	directory, err := a.core.History.NativeDirectory(ctx, leagueID)
	if err != nil {
		return commissioner.NativeDirectory{}, fmt.Errorf("native league directory: %w", err)
	}
	return directory, nil
}

func (a *App) TargetNativeAssets(leagueID string) (commissioner.NativeAssets, error) {
	if err := a.ready(); err != nil {
		return commissioner.NativeAssets{}, fmt.Errorf("native assets startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.core.Ctx, refreshTimeout)
	defer cancel()
	assets, err := a.core.History.NativeAssets(ctx, leagueID)
	if err != nil {
		return commissioner.NativeAssets{}, fmt.Errorf("native league assets: %w", err)
	}
	return assets, nil
}
