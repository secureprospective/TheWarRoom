package leagueservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

// viewContext keeps recipient concurrency metadata request-local, never in the replay cache.
func (a *Service) viewContext(ctx context.Context) context.Context {
	ctx = a.requestContext(ctx)
	if !a.hosted {
		return ctx
	}
	principal, _ := a.host.CommissionerPrincipal()
	return commissioner.WithRecipient(ctx, principal, a.host.CommissionerNow())
}

func (a *Service) TargetCommissionerLedger(league string) (commissioner.Ledger, error) {
	return a.commissionerLedger(league, commissioner.RawLedgerFilter{}, "commish.ledger.read")
}

func (a *Service) TargetCommissionerLedgerFiltered(league, body string) (commissioner.Ledger, error) {
	filter, err := commissioner.DecodeLedgerFilter(body)
	if err != nil {
		return commissioner.Ledger{}, fmt.Errorf("league ledger filters: %w", err)
	}
	return a.commissionerLedger(league, filter, "commish.ledger.filtered.read")
}

func (a *Service) commissionerLedger(league string,
	filter commissioner.RawLedgerFilter, operation string) (commissioner.Ledger, error) {
	if !strings.HasPrefix(league, "native.") {
		return commissioner.Ledger{}, fmt.Errorf("choose an app league")
	}
	if err := a.admitRead(league, operation, ""); err != nil {
		return commissioner.Ledger{}, err
	}
	if err := a.host.Ready(); err != nil {
		return commissioner.Ledger{}, fmt.Errorf("league ledger startup: %w", err)
	}
	principal, err := a.host.CommissionerPrincipal()
	if err != nil {
		//nolint:wrapcheck // Preserve the existing refusal text and error identity.
		return commissioner.Ledger{}, err
	}
	ctx, cancel := context.WithTimeout(a.viewContext(a.host.CommissionerContext()), refreshTimeout)
	defer cancel()
	reading, err := a.host.CommissionerLedgerFiltered(ctx, league, principal,
		a.host.CommissionerAuthorization(), filter)
	if err != nil {
		return commissioner.Ledger{}, fmt.Errorf("league ledger: %w", err)
	}
	return reading, nil
}

func (a *Service) TargetCommissionerPeople(league string) (commissioner.People, error) {
	if !strings.HasPrefix(league, "native.") || league == "native." {
		return commissioner.People{}, fmt.Errorf("choose an app league")
	}
	if err := a.admitRead(league, "commish.people.read", ""); err != nil {
		return commissioner.People{}, err
	}
	if err := a.host.Ready(); err != nil {
		return commissioner.People{}, fmt.Errorf("league people startup: %w", err)
	}
	principal, err := a.host.CommissionerPrincipal()
	if err != nil {
		//nolint:wrapcheck // Preserve the existing refusal text and error identity.
		return commissioner.People{}, err
	}
	ctx, cancel := context.WithTimeout(a.viewContext(a.host.CommissionerContext()), refreshTimeout)
	defer cancel()
	people, err := a.host.CommissionerPeople(ctx, league, principal)
	if err != nil {
		return commissioner.People{}, fmt.Errorf("league people: %w", err)
	}
	return people, nil
}

func (a *Service) TargetCommissionerLeagues() ([]commissioner.NativeLeague, error) {
	if err := a.admitRead("", "commish.leagues.read", ""); err != nil {
		return nil, err
	}
	if err := a.host.Ready(); err != nil {
		return nil, fmt.Errorf("app leagues startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.viewContext(a.host.CommissionerContext()), refreshTimeout)
	defer cancel()
	leagues, err := a.host.NativeLeagues(ctx)
	if err != nil {
		return nil, fmt.Errorf("app leagues: %w", err)
	}
	return leagues, nil
}

func (a *Service) TargetCommissionerRules(league string) (commissioner.Rules, error) {
	if err := AppLeague(league); err != nil {
		return commissioner.Rules{}, err
	}
	if err := a.admitRead(league, "commish.rules.read", ""); err != nil {
		return commissioner.Rules{}, err
	}
	if err := a.host.Ready(); err != nil {
		return commissioner.Rules{}, fmt.Errorf("league rules startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.viewContext(a.host.CommissionerContext()), refreshTimeout)
	defer cancel()
	reading, err := a.host.CommissionerRules(ctx, league)
	if err != nil {
		return commissioner.Rules{}, fmt.Errorf("league rules: %w", err)
	}
	return reading, nil
}

func (a *Service) TargetCommissionerJobs(league string) (commissioner.Jobs, error) {
	if err := AppLeague(league); err != nil {
		return commissioner.Jobs{}, err
	}
	if err := a.admitRead(league, "commish.jobs.read", ""); err != nil {
		return commissioner.Jobs{}, err
	}
	if err := a.host.Ready(); err != nil {
		return commissioner.Jobs{}, fmt.Errorf("league jobs startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.viewContext(a.host.CommissionerContext()), refreshTimeout)
	defer cancel()
	reading, err := a.host.CommissionerJobs(ctx, league)
	if err != nil {
		return commissioner.Jobs{}, fmt.Errorf("league jobs: %w", err)
	}
	return reading, nil
}

func (a *Service) TargetCommissionerControls() (commissioner.Controls, error) {
	if err := a.admitRead("", "commish.controls.read", ""); err != nil {
		return commissioner.Controls{}, err
	}
	return commissioner.ControlsReading(), nil
}

func (a *Service) TargetCommissionerNotices(league string) (commissioner.LeagueNotes, error) {
	if err := AppLeague(league); err != nil {
		return commissioner.LeagueNotes{}, err
	}
	if err := a.admitRead(league, "commish.notices.read", ""); err != nil {
		return commissioner.LeagueNotes{}, err
	}
	if err := a.host.Ready(); err != nil {
		return commissioner.LeagueNotes{}, fmt.Errorf("league notices startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.viewContext(a.host.CommissionerContext()), refreshTimeout)
	defer cancel()
	reading, err := a.host.CommissionerNotes(ctx, league)
	if err != nil {
		return commissioner.LeagueNotes{}, fmt.Errorf("league notices: %w", err)
	}
	return reading, nil
}

func (a *Service) TargetCommissionerDigests(league string) (commissioner.Digests, error) {
	if err := AppLeague(league); err != nil {
		return commissioner.Digests{}, err
	}
	if err := a.admitRead(league, "commish.digests.read", ""); err != nil {
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
	ctx, cancel := context.WithTimeout(a.viewContext(a.host.CommissionerContext()), refreshTimeout)
	defer cancel()
	reading, err := a.host.CommissionerDigests(ctx, league, principal, a.host.CommissionerAuthorization())
	if err != nil {
		return commissioner.Digests{}, fmt.Errorf("league digests: %w", err)
	}
	return reading, nil
}

func (a *Service) TargetCorrections(league string) (commissioner.Corrections, error) {
	if err := a.admitRead(league, "commish.corrections.read", ""); err != nil {
		return commissioner.Corrections{}, err
	}
	if err := a.host.Ready(); err != nil {
		return commissioner.Corrections{}, fmt.Errorf("corrections startup: %w", err)
	}
	out, err := a.host.Corrections(a.viewContext(a.host.CommissionerContext()), league)
	if err != nil {
		return commissioner.Corrections{}, fmt.Errorf("read corrections: %w", err)
	}
	return out, nil
}

func (a *Service) TargetCommissionerMarket(league, command string) (commissioner.Market, error) {
	if err := a.admitReading(commissioner.Access{League: league, Operation: "commish.market.read", Mode: "read", SubjectCommand: command}); err != nil {
		return commissioner.Market{}, err
	}
	if err := a.host.Ready(); err != nil {
		return commissioner.Market{}, fmt.Errorf("commissioner market startup: %w", err)
	}
	principal, err := a.host.CommissionerPrincipal()
	if err != nil {
		//nolint:wrapcheck // Preserve the existing refusal text and error identity.
		return commissioner.Market{}, err
	}
	records, err := a.host.CommissionerRecords(a.viewContext(a.host.CommissionerContext()), league)
	if err != nil {
		//nolint:wrapcheck // Preserve the existing refusal text and error identity.
		return commissioner.Market{}, err
	}
	reading, err := commissioner.MarketReading(league, command, principal, records,
		a.host.CommissionerNow(), a.host.CommissionerZone(), a.host.NativeFacts())
	if err != nil {
		return commissioner.Market{}, fmt.Errorf("commissioner market: %w", err)
	}
	reading.Revision = commissioner.ViewRevision(a.viewContext(a.host.CommissionerContext()), records, reading.Revision)
	return reading, nil
}

func (a *Service) TargetCommissionerActor() (string, error) {
	if err := a.admitRead("", "commish.actor.read", ""); err != nil {
		return "", err
	}
	if err := a.host.Ready(); err != nil {
		return "", fmt.Errorf("acting person startup: %w", err)
	}
	//nolint:wrapcheck // Preserve the existing refusal text and error identity.
	return a.host.CommissionerPrincipal()
}

func (a *Service) TargetCommissionerSchedule(league string) (commissioner.ManagedSchedule, error) {
	if err := a.admitRead(league, "commish.schedule.read", ""); err != nil {
		return commissioner.ManagedSchedule{}, err
	}
	if err := a.host.Ready(); err != nil {
		return commissioner.ManagedSchedule{}, fmt.Errorf("commissioner schedule startup: %w", err)
	}
	principal, err := a.host.CommissionerPrincipal()
	if err != nil {
		//nolint:wrapcheck // Preserve the existing refusal text and error identity.
		return commissioner.ManagedSchedule{}, err
	}
	records, err := a.host.CommissionerRecords(a.viewContext(a.host.CommissionerContext()), league)
	if err != nil {
		//nolint:wrapcheck // Preserve the existing refusal text and error identity.
		return commissioner.ManagedSchedule{}, err
	}
	reading, err := commissioner.ManagedScheduleReading(league, principal, records)
	if err != nil {
		return commissioner.ManagedSchedule{}, fmt.Errorf("commissioner schedule: %w", err)
	}
	reading.Revision = commissioner.ViewRevision(a.viewContext(a.host.CommissionerContext()), records, reading.Revision)
	return reading, nil
}

func (a *Service) Preview(raw commissioner.RawCommand) (commissioner.Preview, error) {
	if err := raw.ValidatePreview(); err != nil {
		return Blocked(err), nil
	}
	if err := a.host.Ready(); err != nil {
		return commissioner.Preview{}, fmt.Errorf("league preview startup: %w", err)
	}
	principal, err := a.host.CommissionerPrincipal()
	if err != nil {
		//nolint:wrapcheck // Preserve the existing refusal text and error identity.
		return commissioner.Preview{}, err
	}
	if err := a.admitReading(commissioner.Access{League: raw.League, Operation: raw.Command, Mode: "preview", Raw: raw}); err != nil {
		return commissioner.Preview{}, err
	}
	catalogue, err := commissioner.ReadPreviewCatalogue(a.host.CommissionerPreviewRegistry())
	if err != nil {
		return commissioner.Preview{}, fmt.Errorf("league preview catalogue: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.viewContext(a.host.CommissionerContext()), refreshTimeout)
	defer cancel()
	preview, err := a.host.PreviewCommissionerWithFacts(ctx, raw, principal, a.host.CommissionerAuthorization(),
		catalogue, a.host.NativeFacts(), a.host.CommissionerZone())
	if err != nil {
		return commissioner.Preview{}, fmt.Errorf("league preview: %w", err)
	}
	return preview, nil
}
