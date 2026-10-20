package main

import (
	"context"
	"fmt"
	"time"

	nativeapp "github.com/secureprospective/TheWarRoom/internal/app/native"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// nativeFacts reads the football facts the app already holds, with no network: positions and draft years
// from the held players directory under its lock, NFL listings from the held injuries feed.
func (a *App) nativeFacts() nativecheck.Facts {
	facts := nativecheck.Facts{}
	a.lookupMu.Lock()
	if a.hasLookup {
		facts = nativeapp.PlayerFacts(a.lookup)
	}
	a.lookupMu.Unlock()
	if !a.injuriesHeld() {
		return facts
	}
	nativeapp.Designate(facts, a.heldInjuries().Status)
	return facts
}

func (a *App) TargetNativeCapCheck(leagueID, viewerZone string) (commissioner.NativeCapCheck, error) {
	scope := commissioner.RawNativeCapScope{League: leagueID, ViewerZone: viewerZone}
	if err := scope.Validate(); err != nil {
		return commissioner.NativeCapCheck{}, fmt.Errorf("native salary cap input: %w", err)
	}
	if err := a.ready(); err != nil {
		return commissioner.NativeCapCheck{}, fmt.Errorf("native salary cap check startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, refreshTimeout)
	defer cancel()
	reading, err := a.history.NativeCapCheck(ctx, scope)
	if err != nil {
		return commissioner.NativeCapCheck{}, fmt.Errorf("native salary cap check: %w", err)
	}
	return reading, nil
}

func (a *App) TargetNativeRosterCheck(leagueID string) (commissioner.NativeRosterCheck, error) {
	if err := a.ready(); err != nil {
		return commissioner.NativeRosterCheck{}, fmt.Errorf("native roster check startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, refreshTimeout)
	defer cancel()
	reading, err := a.history.NativeRosterCheck(ctx, leagueID, a.nativeFacts())
	if err != nil {
		return commissioner.NativeRosterCheck{}, fmt.Errorf("native roster check: %w", err)
	}
	return reading, nil
}

func (a *App) TargetNativeReserveCheck(leagueID string) (commissioner.NativeReserveCheck, error) {
	if err := a.ready(); err != nil {
		return commissioner.NativeReserveCheck{}, fmt.Errorf("native reserve check startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, refreshTimeout)
	defer cancel()
	reading, err := a.history.NativeReserveCheck(ctx, leagueID, a.nativeFacts())
	if err != nil {
		return commissioner.NativeReserveCheck{}, fmt.Errorf("native reserve check: %w", err)
	}
	return reading, nil
}

// TargetNativeLineupCheck judges a lineup of one native franchise's active players; it records nothing.
func (a *App) TargetNativeLineupCheck(input commissioner.RawNativeLineup) (commissioner.NativeLineupCheck, error) {
	if err := input.Validate(); err != nil {
		return commissioner.NativeLineupCheck{}, fmt.Errorf("native lineup input: %w", err)
	}
	if err := a.ready(); err != nil {
		return commissioner.NativeLineupCheck{}, fmt.Errorf("native lineup check startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, refreshTimeout)
	defer cancel()
	reading, err := a.history.NativeLineupCheck(ctx, input, a.nativeFacts())
	if err != nil {
		return commissioner.NativeLineupCheck{}, fmt.Errorf("native lineup check: %w", err)
	}
	return reading, nil
}

func (a *App) TargetNativeHouseRules(leagueID string) (commissioner.NativeHouseRules, error) {
	if err := a.ready(); err != nil {
		return commissioner.NativeHouseRules{}, fmt.Errorf("native house rules startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, refreshTimeout)
	defer cancel()
	reading, err := a.history.NativeHouseRules(ctx, leagueID)
	if err != nil {
		return commissioner.NativeHouseRules{}, fmt.Errorf("native house rules: %w", err)
	}
	return reading, nil
}

// TargetNativePlayerScore reads only held NFL evidence and the app league's recorded decisions.
func (a *App) TargetNativePlayerScore(league, player string, season, week int) (commissioner.NativePlayerScore, error) {
	scope := source.RawNativeScope{Player: player, Season: season, Week: week}
	if err := scope.Validate(); err != nil {
		return commissioner.NativePlayerScore{}, fmt.Errorf("player score input: %w", err)
	}
	if err := a.ready(); err != nil {
		return commissioner.NativePlayerScore{}, fmt.Errorf("player score startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, refreshTimeout)
	defer cancel()
	reading, err := a.history.NativePlayerScore(ctx, league, scope)
	if err != nil {
		return commissioner.NativePlayerScore{}, fmt.Errorf("player score: %w", err)
	}
	return reading, nil
}

// TargetNativeContractQuote prices one contract action on one recorded native contract: allowed or
// refused with the plain reason, the new salary and years, the dead money by season and the cap effect.
// It reads the app league's ledger only and records nothing; the owner action that follows runs the same
// check.
func (a *App) TargetNativeContractQuote(raw commissioner.RawContractQuote) (commissioner.ContractQuote, error) {
	if err := raw.Validate(); err != nil {
		return commissioner.ContractQuote{}, fmt.Errorf("native contract quote input: %w", err)
	}
	if err := a.ready(); err != nil {
		return commissioner.ContractQuote{}, fmt.Errorf("native contract quote startup: %w", err)
	}
	zone, err := time.LoadLocation(raw.ViewerZone)
	if err != nil {
		return commissioner.ContractQuote{}, fmt.Errorf("read viewer timezone: %w", err)
	}
	records, err := a.leagueRecords(a.ctx, raw.League)
	if err != nil {
		return commissioner.ContractQuote{}, err
	}
	quote, err := commissioner.QuoteContract(raw, records, a.timeNow(), zone, a.nativeFacts())
	if err != nil {
		return commissioner.ContractQuote{}, fmt.Errorf("native contract quote: %w", err)
	}
	return quote, nil
}
