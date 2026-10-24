package leagueservice

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func (a *Service) TargetNativeTrades(league, viewer string) (commissioner.NativeTrades, error) {
	if err := a.host.Ready(); err != nil {
		return commissioner.NativeTrades{}, fmt.Errorf("trade startup: %w", err)
	}
	principal, err := a.host.CommissionerPrincipal()
	if err != nil {
		//nolint:wrapcheck // Preserve the existing refusal text and error identity.
		return commissioner.NativeTrades{}, err
	}
	records, err := a.host.CommissionerRecords(a.host.CommissionerContext(), league)
	if err != nil {
		//nolint:wrapcheck // Preserve the existing refusal text and error identity.
		return commissioner.NativeTrades{}, err
	}
	reading, err := commissioner.NativeTradesReading(league, viewer, principal, records,
		a.host.CommissionerNow(), a.host.CommissionerZone())
	if err != nil {
		return commissioner.NativeTrades{}, fmt.Errorf("read trades: %w", err)
	}
	return reading, nil
}

func (a *Service) TargetNativeWaivers(league, franchise string, at time.Time) (commissioner.NativeWaivers,
	error) {
	if at.IsZero() {
		return commissioner.NativeWaivers{}, fmt.Errorf("choose when to read waivers")
	}
	if err := a.host.Ready(); err != nil {
		return commissioner.NativeWaivers{}, fmt.Errorf("waivers startup: %w", err)
	}
	principal, err := a.host.CommissionerPrincipal()
	if err != nil {
		//nolint:wrapcheck // Preserve the existing refusal text and error identity.
		return commissioner.NativeWaivers{}, err
	}
	reading, err := a.host.NativeWaivers(a.host.CommissionerContext(), league, franchise, principal, at,
		a.host.CommissionerZone(), a.host.NativeFacts())
	if err != nil {
		return commissioner.NativeWaivers{}, fmt.Errorf("native waivers: %w", err)
	}
	return reading, nil
}

func (a *Service) TargetNativeBids(league, franchise string) (commissioner.NativeBids, error) {
	if err := a.host.Ready(); err != nil {
		return commissioner.NativeBids{}, fmt.Errorf("bids startup: %w", err)
	}
	principal, err := a.host.CommissionerPrincipal()
	if err != nil {
		//nolint:wrapcheck // Preserve the existing refusal text and error identity.
		return commissioner.NativeBids{}, err
	}
	reading, err := a.host.NativeBids(a.host.CommissionerContext(), league, franchise, principal,
		a.host.CommissionerNow(), a.host.CommissionerZone(), a.host.NativeFacts())
	if err != nil {
		return commissioner.NativeBids{}, fmt.Errorf("native bids: %w", err)
	}
	return reading, nil
}

func (a *Service) TargetNativeDraft(scope commissioner.RawNativeDraftScope) (commissioner.NativeDraft, error) {
	if scope.At.IsZero() {
		scope.At = a.host.CommissionerNow()
	}
	if err := scope.Validate(); err != nil {
		return commissioner.NativeDraft{}, fmt.Errorf("rookie draft input: %w", err)
	}
	if err := a.host.Ready(); err != nil {
		return commissioner.NativeDraft{}, fmt.Errorf("rookie draft startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.host.CommissionerContext(), refreshTimeout)
	defer cancel()
	reading, err := a.host.NativeDraft(ctx, scope, a.host.CommissionerZone(), a.host.NativeFacts())
	if err != nil {
		return commissioner.NativeDraft{}, fmt.Errorf("rookie draft: %w", err)
	}
	return reading, nil
}

func (a *Service) TargetNativeLineup(league, franchise string, week int,
	at string) (commissioner.NativeLineup, error) {
	if !strings.HasPrefix(league, "native.") || league == "native." || strings.TrimSpace(league) != league ||
		strings.TrimSpace(franchise) == "" || week < 1 || week > 22 {
		return commissioner.NativeLineup{}, fmt.Errorf("choose an app league, a franchise and a week")
	}
	instant := a.host.CommissionerNow()
	if at != "" {
		parsed, err := time.Parse(time.RFC3339, at)
		if err != nil {
			return commissioner.NativeLineup{}, fmt.Errorf("choose when to read the lineup: %w", err)
		}
		instant = parsed
	}
	if err := a.host.Ready(); err != nil {
		return commissioner.NativeLineup{}, fmt.Errorf("native lineup startup: %w", err)
	}
	principal, err := a.host.CommissionerPrincipal()
	if err != nil {
		return commissioner.NativeLineup{}, fmt.Errorf("native lineup identity: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.host.CommissionerContext(), refreshTimeout)
	defer cancel()
	reading, err := a.host.NativeLineup(ctx, league, franchise, principal, week, instant)
	if err != nil {
		return commissioner.NativeLineup{}, fmt.Errorf("native lineup: %w", err)
	}
	return reading, nil
}

func (a *Service) TargetNativeDirectory(leagueID string) (commissioner.NativeDirectory, error) {
	if err := a.host.Ready(); err != nil {
		return commissioner.NativeDirectory{}, fmt.Errorf("native directory startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.host.CommissionerContext(), refreshTimeout)
	defer cancel()
	directory, err := a.host.NativeDirectory(ctx, leagueID)
	if err != nil {
		return commissioner.NativeDirectory{}, fmt.Errorf("native league directory: %w", err)
	}
	return directory, nil
}

func (a *Service) TargetNativeAssets(leagueID string) (commissioner.NativeAssets, error) {
	if err := a.host.Ready(); err != nil {
		return commissioner.NativeAssets{}, fmt.Errorf("native assets startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.host.CommissionerContext(), refreshTimeout)
	defer cancel()
	assets, err := a.host.NativeAssets(ctx, leagueID)
	if err != nil {
		return commissioner.NativeAssets{}, fmt.Errorf("native league assets: %w", err)
	}
	return assets, nil
}
