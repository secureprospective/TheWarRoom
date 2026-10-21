package commissioner

import (
	"context"
	"fmt"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func (a *API) TargetCommissionerExecute(body string) (commissioner.Receipt, error) {
	raw, err := commissioner.Decode(body)
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("league change: %w", err)
	}
	if commissioner.OwnerTradeAction(raw.Command) != "" {
		return commissioner.Receipt{},
			fmt.Errorf("use the native trade action to offer, accept, decline or withdraw")
	}
	if commissioner.OwnerWaiverAction(raw.Command) != "" {
		return commissioner.Receipt{}, fmt.Errorf("use the native owner action for cuts and claims")
	}
	if commissioner.OwnerBidAction(raw.Command) != "" {
		return commissioner.Receipt{}, fmt.Errorf("use the native owner action to place, raise or withdraw a bid")
	}
	if raw.Command == commissioner.NativeSeed {
		return commissioner.Receipt{}, fmt.Errorf("record the starting snapshot through the held-data action")
	}
	if raw.Command == commissioner.LineupSet {
		return commissioner.Receipt{},
			fmt.Errorf("set a lineup from the lineup screen, which asks the league calendar first")
	}
	if raw.Command == commissioner.DraftPick {
		return commissioner.Receipt{},
			fmt.Errorf("a rookie pick is made by the franchise on the clock, in the draft room")
	}
	if err := a.host.Ready(); err != nil {
		return commissioner.Receipt{}, fmt.Errorf("league change startup: %w", err)
	}
	principal, err := a.host.CommissionerPrincipal()
	if err != nil {
		//nolint:wrapcheck // Preserve the existing refusal text and error identity.
		return commissioner.Receipt{}, err
	}
	ctx, cancel := context.WithTimeout(a.host.CommissionerContext(), refreshTimeout)
	defer cancel()
	receipt, err := a.host.ExecuteCommissionerWithFacts(ctx, raw, principal,
		a.host.CommissionerAuthorization(), a.host.NativeFacts(), a.host.CommissionerZone())
	if err != nil {
		return commissioner.Receipt{}, ChangeError(err)
	}
	return receipt, nil
}

func (a *API) TargetCommissionerLedger(league string) (commissioner.Ledger, error) {
	return a.commissionerLedger(league, commissioner.RawLedgerFilter{})
}

func (a *API) TargetCommissionerLedgerFiltered(league, body string) (commissioner.Ledger, error) {
	filter, err := commissioner.DecodeLedgerFilter(body)
	if err != nil {
		return commissioner.Ledger{}, fmt.Errorf("league ledger filters: %w", err)
	}
	return a.commissionerLedger(league, filter)
}

func (a *API) commissionerLedger(league string,
	filter commissioner.RawLedgerFilter) (commissioner.Ledger, error) {
	if !strings.HasPrefix(league, "native.") {
		return commissioner.Ledger{}, fmt.Errorf("choose an app league")
	}
	if err := a.host.Ready(); err != nil {
		return commissioner.Ledger{}, fmt.Errorf("league ledger startup: %w", err)
	}
	principal, err := a.host.CommissionerPrincipal()
	if err != nil {
		//nolint:wrapcheck // Preserve the existing refusal text and error identity.
		return commissioner.Ledger{}, err
	}
	ctx, cancel := context.WithTimeout(a.host.CommissionerContext(), refreshTimeout)
	defer cancel()
	reading, err := a.host.CommissionerLedgerFiltered(ctx, league, principal,
		a.host.CommissionerAuthorization(), filter)
	if err != nil {
		return commissioner.Ledger{}, fmt.Errorf("league ledger: %w", err)
	}
	return reading, nil
}

func (a *API) TargetCommissionerPeople(league string) (commissioner.People, error) {
	if !strings.HasPrefix(league, "native.") || league == "native." {
		return commissioner.People{}, fmt.Errorf("choose an app league")
	}
	if err := a.host.Ready(); err != nil {
		return commissioner.People{}, fmt.Errorf("league people startup: %w", err)
	}
	principal, err := a.host.CommissionerPrincipal()
	if err != nil {
		//nolint:wrapcheck // Preserve the existing refusal text and error identity.
		return commissioner.People{}, err
	}
	ctx, cancel := context.WithTimeout(a.host.CommissionerContext(), refreshTimeout)
	defer cancel()
	people, err := a.host.CommissionerPeople(ctx, league, principal)
	if err != nil {
		return commissioner.People{}, fmt.Errorf("league people: %w", err)
	}
	return people, nil
}

func (a *API) TargetCommissionerLeagues() ([]commissioner.NativeLeague, error) {
	if err := a.host.Ready(); err != nil {
		return nil, fmt.Errorf("app leagues startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.host.CommissionerContext(), refreshTimeout)
	defer cancel()
	leagues, err := a.host.NativeLeagues(ctx)
	if err != nil {
		return nil, fmt.Errorf("app leagues: %w", err)
	}
	return leagues, nil
}

func (a *API) TargetCommissionerRules(league string) (commissioner.Rules, error) {
	if err := AppLeague(league); err != nil {
		return commissioner.Rules{}, err
	}
	if err := a.host.Ready(); err != nil {
		return commissioner.Rules{}, fmt.Errorf("league rules startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.host.CommissionerContext(), refreshTimeout)
	defer cancel()
	reading, err := a.host.CommissionerRules(ctx, league)
	if err != nil {
		return commissioner.Rules{}, fmt.Errorf("league rules: %w", err)
	}
	return reading, nil
}

func (a *API) TargetCommissionerJobs(league string) (commissioner.Jobs, error) {
	if err := AppLeague(league); err != nil {
		return commissioner.Jobs{}, err
	}
	if err := a.host.Ready(); err != nil {
		return commissioner.Jobs{}, fmt.Errorf("league jobs startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.host.CommissionerContext(), refreshTimeout)
	defer cancel()
	reading, err := a.host.CommissionerJobs(ctx, league)
	if err != nil {
		return commissioner.Jobs{}, fmt.Errorf("league jobs: %w", err)
	}
	return reading, nil
}

// TargetCommissionerControls lists every commissioner control, grouped, and whether it is built yet.
func (a *API) TargetCommissionerControls() commissioner.Controls {
	return commissioner.ControlsReading()
}

// TargetCommissionerNotices reads the posted notices, league notes, release notes and the mail log. Every
// owner may read it.
func (a *API) TargetCommissionerNotices(league string) (commissioner.LeagueNotes, error) {
	if err := AppLeague(league); err != nil {
		return commissioner.LeagueNotes{}, err
	}
	if err := a.host.Ready(); err != nil {
		return commissioner.LeagueNotes{}, fmt.Errorf("league notices startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.host.CommissionerContext(), refreshTimeout)
	defer cancel()
	reading, err := a.host.CommissionerNotes(ctx, league)
	if err != nil {
		return commissioner.LeagueNotes{}, fmt.Errorf("league notices: %w", err)
	}
	return reading, nil
}

// TargetCommissionerDigests writes a plain sentence for each change that took a hold or a typed reason.
func (a *API) TargetCommissionerDigests(league string) (commissioner.Digests, error) {
	if err := AppLeague(league); err != nil {
		return commissioner.Digests{}, err
	}
	if err := a.host.Ready(); err != nil {
		return commissioner.Digests{}, fmt.Errorf("league digests startup: %w", err)
	}
	principal, err := a.host.CommissionerPrincipal()
	if err != nil {
		//nolint:wrapcheck // Preserve the existing refusal text and error identity.
		return commissioner.Digests{}, err
	}
	ctx, cancel := context.WithTimeout(a.host.CommissionerContext(), refreshTimeout)
	defer cancel()
	reading, err := a.host.CommissionerDigests(ctx, league, principal, a.host.CommissionerAuthorization())
	if err != nil {
		return commissioner.Digests{}, fmt.Errorf("league digests: %w", err)
	}
	return reading, nil
}

// TargetCorrections reads only this app league's public, active corrections and exceptions.
func (a *API) TargetCorrections(league string) (commissioner.Corrections, error) {
	if err := a.host.Ready(); err != nil {
		return commissioner.Corrections{}, fmt.Errorf("corrections startup: %w", err)
	}
	out, err := a.host.Corrections(a.host.CommissionerContext(), league)
	if err != nil {
		return commissioner.Corrections{}, fmt.Errorf("read corrections: %w", err)
	}
	return out, nil
}

// TargetCommissionerMarket reads held targets under the permission of the requested act.
func (a *API) TargetCommissionerMarket(league, command string) (commissioner.Market, error) {
	if err := a.host.Ready(); err != nil {
		return commissioner.Market{}, fmt.Errorf("commissioner market startup: %w", err)
	}
	principal, err := a.host.CommissionerPrincipal()
	if err != nil {
		//nolint:wrapcheck // Preserve the existing refusal text and error identity.
		return commissioner.Market{}, err
	}
	records, err := a.host.CommissionerRecords(a.host.CommissionerContext(), league)
	if err != nil {
		//nolint:wrapcheck // Preserve the existing refusal text and error identity.
		return commissioner.Market{}, err
	}
	reading, err := commissioner.MarketReading(league, command, principal, records,
		a.host.CommissionerNow(), a.host.CommissionerZone(), a.host.NativeFacts())
	if err != nil {
		return commissioner.Market{}, fmt.Errorf("commissioner market: %w", err)
	}
	return reading, nil
}

// TargetCommissionerActor supplies the session identity for acts that must be performed as oneself.
func (a *API) TargetCommissionerActor() (string, error) {
	if err := a.host.Ready(); err != nil {
		return "", fmt.Errorf("acting person startup: %w", err)
	}
	//nolint:wrapcheck // Preserve the existing refusal text and error identity.
	return a.host.CommissionerPrincipal()
}

func (a *API) TargetCommissionerSchedule(league string) (commissioner.ManagedSchedule, error) {
	if err := a.host.Ready(); err != nil {
		return commissioner.ManagedSchedule{}, fmt.Errorf("commissioner schedule startup: %w", err)
	}
	principal, err := a.host.CommissionerPrincipal()
	if err != nil {
		//nolint:wrapcheck // Preserve the existing refusal text and error identity.
		return commissioner.ManagedSchedule{}, err
	}
	records, err := a.host.CommissionerRecords(a.host.CommissionerContext(), league)
	if err != nil {
		//nolint:wrapcheck // Preserve the existing refusal text and error identity.
		return commissioner.ManagedSchedule{}, err
	}
	reading, err := commissioner.ManagedScheduleReading(league, principal, records)
	if err != nil {
		return commissioner.ManagedSchedule{}, fmt.Errorf("commissioner schedule: %w", err)
	}
	return reading, nil
}
