package commissioner

import (
	"context"
	"fmt"

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
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return a.reads.TargetCommissionerLedger(league)
}

func (a *API) TargetCommissionerLedgerFiltered(league, body string) (commissioner.Ledger, error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return a.reads.TargetCommissionerLedgerFiltered(league, body)
}

func (a *API) TargetCommissionerPeople(league string) (commissioner.People, error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return a.reads.TargetCommissionerPeople(league)
}

func (a *API) TargetCommissionerLeagues() ([]commissioner.NativeLeague, error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return a.reads.TargetCommissionerLeagues()
}

func (a *API) TargetCommissionerRules(league string) (commissioner.Rules, error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return a.reads.TargetCommissionerRules(league)
}

func (a *API) TargetCommissionerJobs(league string) (commissioner.Jobs, error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return a.reads.TargetCommissionerJobs(league)
}

// TargetCommissionerControls lists every commissioner control, grouped, and whether it is built yet.
func (a *API) TargetCommissionerControls() commissioner.Controls {
	return a.reads.TargetCommissionerControls()
}

// TargetCommissionerNotices reads the posted notices, league notes, release notes and the mail log. Every
// owner may read it.
func (a *API) TargetCommissionerNotices(league string) (commissioner.LeagueNotes, error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return a.reads.TargetCommissionerNotices(league)
}

// TargetCommissionerDigests writes a plain sentence for each change that took a hold or a typed reason.
func (a *API) TargetCommissionerDigests(league string) (commissioner.Digests, error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return a.reads.TargetCommissionerDigests(league)
}

// TargetCorrections reads only this app league's public, active corrections and exceptions.
func (a *API) TargetCorrections(league string) (commissioner.Corrections, error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return a.reads.TargetCorrections(league)
}

// TargetCommissionerMarket reads held targets under the permission of the requested act.
func (a *API) TargetCommissionerMarket(league, command string) (commissioner.Market, error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return a.reads.TargetCommissionerMarket(league, command)
}

// TargetCommissionerActor supplies the session identity for acts that must be performed as oneself.
func (a *API) TargetCommissionerActor() (string, error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return a.reads.TargetCommissionerActor()
}

func (a *API) TargetCommissionerSchedule(league string) (commissioner.ManagedSchedule, error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return a.reads.TargetCommissionerSchedule(league)
}
