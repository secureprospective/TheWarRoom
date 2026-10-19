package main

import (
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

// TargetCommissionerMarket reads held targets under the permission of the requested act.
func (a *App) TargetCommissionerMarket(league, command string) (commissioner.Market, error) {
	if err := a.ready(); err != nil {
		return commissioner.Market{}, fmt.Errorf("commissioner market startup: %w", err)
	}
	principal, err := a.commissionerPrincipal()
	if err != nil {
		return commissioner.Market{}, err
	}
	records, err := a.leagueRecords(a.ctx, league)
	if err != nil {
		return commissioner.Market{}, err
	}
	reading, err := commissioner.MarketReading(league, command, principal, records,
		a.timeNow(), a.ownerZone(), a.nativeFacts())
	if err != nil {
		return commissioner.Market{}, fmt.Errorf("commissioner market: %w", err)
	}
	return reading, nil
}

// TargetCommissionerActor supplies the session identity for acts that must be performed as oneself.
func (a *App) TargetCommissionerActor() (string, error) {
	if err := a.ready(); err != nil {
		return "", fmt.Errorf("acting person startup: %w", err)
	}
	return a.commissionerPrincipal()
}

func (a *App) TargetCommissionerSchedule(league string) (commissioner.ManagedSchedule, error) {
	if err := a.ready(); err != nil {
		return commissioner.ManagedSchedule{}, fmt.Errorf("commissioner schedule startup: %w", err)
	}
	principal, err := a.commissionerPrincipal()
	if err != nil {
		return commissioner.ManagedSchedule{}, err
	}
	records, err := a.leagueRecords(a.ctx, league)
	if err != nil {
		return commissioner.ManagedSchedule{}, err
	}
	reading, err := commissioner.ManagedScheduleReading(league, principal, records)
	if err != nil {
		return commissioner.ManagedSchedule{}, fmt.Errorf("commissioner schedule: %w", err)
	}
	return reading, nil
}
