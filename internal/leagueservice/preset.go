package leagueservice

import (
	"context"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// RulesBatchHost records several rules changes as one, each checked exactly as a hand edit is.
type RulesBatchHost interface {
	ExecuteRulesBatchWithFacts(ctx context.Context, raws []commissioner.RawCommand, principal string,
		auth commissioner.Authorizer, facts nativecheck.Facts, zone *time.Location) ([]commissioner.Receipt, error)
}

// TargetCommissionerPresetCompare reads every setting where the league differs from a preset. It reads
// under the rules reading's permission: it shows nothing the rules screen does not.
func (a *Service) TargetCommissionerPresetCompare(league, preset string) (commissioner.PresetComparison, error) {
	if err := AppLeague(league); err != nil {
		return commissioner.PresetComparison{}, err
	}
	if err := a.admitRead(league, "commish.rules.preset.read", ""); err != nil {
		return commissioner.PresetComparison{}, err
	}
	if err := a.host.Ready(); err != nil {
		return commissioner.PresetComparison{}, fmt.Errorf("preset comparison startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.viewContext(a.host.CommissionerContext()), refreshTimeout)
	defer cancel()
	return a.presetCompare(ctx, league, preset)
}

func (a *Service) presetCompare(ctx context.Context, league, preset string) (commissioner.PresetComparison, error) {
	records, err := a.host.CommissionerRecords(ctx, league)
	if err != nil {
		return commissioner.PresetComparison{}, fmt.Errorf("preset comparison: %w", err)
	}
	comparison, err := commissioner.PresetCompare(league, records, preset, a.host.CommissionerNow())
	if err != nil {
		return commissioner.PresetComparison{}, ChangeError(err)
	}
	return comparison, nil
}

// PresetChanges are the rules changes a preset apply would record: the hand edits for the chosen settings.
// The rules screen previews each through the ordinary preview before the commissioner confirms.
func (a *Service) PresetChanges(raw commissioner.RawPresetApply) ([]commissioner.RawCommand, error) {
	if err := AppLeague(raw.League); err != nil {
		return nil, err
	}
	if err := a.host.Ready(); err != nil {
		return nil, fmt.Errorf("preset change startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.viewContext(a.host.CommissionerContext()), refreshTimeout)
	defer cancel()
	comparison, err := a.presetCompare(ctx, raw.League, raw.Preset)
	if err != nil {
		return nil, err
	}
	raws, err := commissioner.PresetRuleChanges(raw, comparison)
	return raws, ChangeError(err)
}

// ExecuteRules records rules changes as one league change: every step is validated, admitted, authorized
// and checked exactly as the same change sent alone, and either every change is recorded or none is.
func (a *Service) ExecuteRules(ctx context.Context, raws []commissioner.RawCommand) ([]commissioner.Receipt, error) {
	batch, ok := a.host.(RulesBatchHost)
	if !ok {
		return nil, fmt.Errorf("apply a preset from the desktop app; change each setting here")
	}
	prepared := make([]commissioner.RawCommand, 0, len(raws))
	for _, raw := range raws {
		if raw.Command != commissioner.RulesSet {
			return nil, fmt.Errorf("only rules changes are recorded together")
		}
		raw, err := raw.Prepared()
		if err != nil {
			return nil, fmt.Errorf("prepare league change: %w", err)
		}
		if err := raw.Validate(); err != nil {
			return nil, fmt.Errorf("league change input: %w", err)
		}
		prepared = append(prepared, raw)
	}
	if err := a.host.Ready(); err != nil {
		return nil, fmt.Errorf("league change startup: %w", err)
	}
	principal, err := a.host.CommissionerPrincipal()
	if err != nil {
		//nolint:wrapcheck // Preserve the session refusal and its identity.
		return nil, err
	}
	for _, raw := range prepared {
		if err := commissioner.AdmitCredential(a.viewContext(ctx), commissioner.Access{Principal: principal,
			League: raw.League, Operation: raw.Command, Mode: "execute", Raw: raw}); err != nil {
			return nil, fmt.Errorf("admit league change: %w", err)
		}
	}
	ctx, cancel := context.WithTimeout(a.viewContext(ctx), refreshTimeout)
	defer cancel()
	receipts, err := batch.ExecuteRulesBatchWithFacts(ctx, prepared, principal,
		a.host.CommissionerAuthorization(), a.host.NativeFacts(), a.host.CommissionerZone())
	if err != nil {
		return nil, ChangeError(err)
	}
	return receipts, nil
}
