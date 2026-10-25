package leagueservice

import (
	"context"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

type draftPickHost interface {
	RecordDraftPick(context.Context, commissioner.Permit, commissioner.DraftPickInput, string, nativecheck.Facts, *time.Location) (commissioner.Receipt, error)
}

// SetLineup uses the lineup screen's calendar gate and transactional lineup checks.
// It deliberately does not enable LineupSet in Execute's generic mutation path.
func (a *Service) SetLineup(ctx context.Context, raw commissioner.RawCommand) (commissioner.Receipt, error) {
	if raw.Command != commissioner.LineupSet {
		return commissioner.Receipt{}, fmt.Errorf("choose a lineup change")
	}
	if err := raw.Validate(); err != nil {
		return commissioner.Receipt{}, fmt.Errorf("lineup input: %w", err)
	}
	if err := a.host.Ready(); err != nil {
		return commissioner.Receipt{}, fmt.Errorf("lineup startup: %w", err)
	}
	principal, err := a.host.CommissionerPrincipal()
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("lineup identity: %w", err)
	}
	ctx = a.viewContext(ctx)
	if err := commissioner.AdmitCredential(ctx, commissioner.Access{Principal: principal, League: raw.League, Operation: raw.Command, Mode: "execute", Raw: raw}); err != nil {
		return commissioner.Receipt{}, fmt.Errorf("admit lineup: %w", err)
	}
	if _, err := a.ownerPermit(ctx, raw.League, raw.Args.Lineup.Franchise, "lineup"); err != nil {
		return commissioner.Receipt{}, err
	}
	ctx = a.ownerContext(ctx, raw.League, raw.Args.Lineup.Franchise, "lineup")
	receipt, err := a.host.ExecuteCommissionerWithFacts(ctx, raw, principal, a.host.CommissionerAuthorization(), a.host.NativeFacts(), a.host.CommissionerZone())
	return receipt, ChangeError(err)
}

// Pick uses the draft room's permit and atomic on-the-clock selection, not Execute.
func (a *Service) Pick(ctx context.Context, in commissioner.DraftPickInput) (commissioner.Receipt, error) {
	if err := in.Validate(); err != nil {
		return commissioner.Receipt{}, fmt.Errorf("rookie pick input: %w", err)
	}
	if err := a.host.Ready(); err != nil {
		return commissioner.Receipt{}, fmt.Errorf("rookie pick startup: %w", err)
	}
	principal, err := a.host.CommissionerPrincipal()
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("rookie pick identity: %w", err)
	}
	ctx = a.viewContext(ctx)
	if err := commissioner.AdmitCredential(ctx, commissioner.Access{Principal: principal, League: in.League, Operation: commissioner.DraftPick, Mode: "execute", Franchise: in.Franchise, DraftPick: &in}); err != nil {
		return commissioner.Receipt{}, fmt.Errorf("admit rookie pick: %w", err)
	}
	permit, err := a.ownerPermit(ctx, in.League, in.Franchise, "draft_pick")
	if err != nil {
		return commissioner.Receipt{}, err
	}
	host, ok := a.host.(draftPickHost)
	if !ok {
		return commissioner.Receipt{}, fmt.Errorf("draft room is unavailable")
	}
	ctx = a.ownerContext(ctx, in.League, in.Franchise, "draft_pick")
	receipt, err := host.RecordDraftPick(ctx, permit, in, principal, a.host.NativeFacts(), a.host.CommissionerZone())
	return receipt, ChangeError(err)
}

func (a *Service) ownerPermit(ctx context.Context, league, franchise string, action leaguerules.Action) (commissioner.Permit, error) {
	records, err := a.host.CommissionerRecords(ctx, league)
	if err != nil {
		return commissioner.Permit{}, fmt.Errorf("owner action records: %w", err)
	}
	gate, err := commissioner.NativeActionGate(league, records, franchise, action, a.host.CommissionerNow(), a.host.CommissionerZone())
	if err != nil {
		return commissioner.Permit{}, fmt.Errorf("owner action calendar: %w", err)
	}
	permit, ok := gate.Permit()
	if !ok {
		return commissioner.Permit{}, fmt.Errorf("%s", gate.Sentence)
	}
	return permit, nil
}

func (h dueHost) RecordDraftPick(ctx context.Context, permit commissioner.Permit, in commissioner.DraftPickInput, principal string, facts nativecheck.Facts, zone *time.Location) (commissioner.Receipt, error) {
	host, ok := h.Host.(draftPickHost)
	if !ok {
		return commissioner.Receipt{}, fmt.Errorf("draft room is unavailable")
	}
	guarded := commissioner.WithCredentialAdmission(ctx, dueAdmission{worker: h.worker, credential: func(access commissioner.Access) error { return commissioner.AdmitCredential(ctx, access) }})
	receipt, err := host.RecordDraftPick(guarded, permit, in, principal, facts, zone)
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("hosted pick: %w", err)
	}
	h.worker.Wake()
	return receipt, nil
}

// ownerContext rechecks the same calendar under the writer, including queued requests.
func (a *Service) ownerContext(ctx context.Context, league, franchise string, action leaguerules.Action) context.Context {
	return commissioner.WithCredentialAdmission(ctx, ownerAdmission{host: a.host, league: league, franchise: franchise, action: action,
		credential: func(access commissioner.Access) error { return commissioner.AdmitCredential(ctx, access) }})
}

type ownerAdmission struct {
	host              Host
	league, franchise string
	action            leaguerules.Action
	credential        func(commissioner.Access) error
}

func (a ownerAdmission) Admit(_ context.Context, access commissioner.Access) error {
	if err := a.credential(access); err != nil {
		return fmt.Errorf("owner credential: %w", err)
	}
	if access.Records == nil {
		return nil
	}
	gate, err := commissioner.NativeActionGate(a.league, access.Records, a.franchise, a.action, a.host.CommissionerNow(), a.host.CommissionerZone())
	if err != nil {
		return fmt.Errorf("owner calendar: %w", err)
	}
	if !gate.Open {
		return fmt.Errorf("%s", gate.Sentence)
	}
	return nil
}
