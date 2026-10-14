package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// mflNoDraftYear is the year MFL fills in where it holds no draft year at all.
const mflNoDraftYear = 1970

// nativeFacts reads the football facts the app already holds, with no network: positions and draft years
// from the held players directory, NFL listings from the held injuries feed. A fact that is not held is
// left unmarked as held, so a check says it cannot tell rather than guessing.
func (a *App) nativeFacts() nativecheck.Facts {
	facts := nativecheck.Facts{}
	a.lookupMu.Lock()
	if a.hasLookup {
		for _, id := range a.lookup.IDs() {
			if f, ok := a.lookup.Facts(id); ok {
				facts[id] = nativecheck.PlayerFacts{Name: f.Name, Position: nativePosition(f.Position),
					DraftYear: f.DraftYear, DraftYearHeld: f.HasDraftYear && f.DraftYear > mflNoDraftYear}
			}
		}
	}
	a.lookupMu.Unlock()
	if !a.injuriesHeld() {
		return facts
	}
	listed := a.heldInjuries().Status
	for id, f := range facts {
		f.DesignationHeld, f.Designation = true, nativeDesignation(listed[id])
		facts[id] = f
	}
	return facts
}

// nativePosition maps the engine's position onto the league rules' position codes.
func nativePosition(p domain.Position) leaguerules.Position {
	switch p {
	case domain.PosK:
		return leaguerules.PK
	case domain.PosFlag:
		return ""
	case domain.PosQB, domain.PosRB, domain.PosWR, domain.PosTE, domain.PosDE, domain.PosDT, domain.PosLB,
		domain.PosCB, domain.PosS:
	}
	return leaguerules.Position(p)
}

// nativeDesignation reads MFL's injury vocabulary as the NFL listings a league can accept for IR. A
// listing the menu does not name is kept as MFL words it, so the check can say what it saw.
func nativeDesignation(status string) leaguerules.NFLStatus {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "":
		return ""
	case "ir", "ir-r":
		return leaguerules.InjuredReserve
	case "ir-pup":
		return leaguerules.PUP
	case "ir-nfi":
		return leaguerules.NFI
	case "out":
		return leaguerules.Out
	case "suspended":
		return leaguerules.Suspended
	}
	return leaguerules.NFLStatus(strings.TrimSpace(status))
}

func (a *App) TargetNativeCapCheck(leagueID string) (commissioner.NativeCapCheck, error) {
	if err := a.ready(); err != nil {
		return commissioner.NativeCapCheck{}, fmt.Errorf("native salary cap check startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, refreshTimeout)
	defer cancel()
	reading, err := a.history.NativeCapCheck(ctx, leagueID)
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
