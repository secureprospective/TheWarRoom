package commissioner

import (
	"context"
	"fmt"
	"strings"

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
	if !strings.HasPrefix(r.League,
		"native.") || r.League == "native." || strings.TrimSpace(r.League) != r.League ||
		r.Expected < 1 || strings.TrimSpace(r.Request) == "" ||
		(requireReason && strings.TrimSpace(r.Reason) == "") {
		return fmt.Errorf("choose an app league, its version, a request and a reason " +
			"for recording its starting snapshot")
	}
	return nil
}

// TargetNativeSeed captures held data only. It cannot accept client-provided rosters or fetch missing inputs.
func (a *API) TargetNativeSeed(input RawNativeSeedRequest) (commissioner.Receipt, error) {
	if err := input.Validate(); err != nil {
		return commissioner.Receipt{}, err
	}
	if err := a.host.Ready(); err != nil {
		return commissioner.Receipt{}, fmt.Errorf("starting snapshot startup: %w", err)
	}
	principal, err := a.host.CommissionerPrincipal()
	if err != nil {
		//nolint:wrapcheck // Preserve the existing refusal text and error identity.
		return commissioner.Receipt{}, err
	}
	ctx, cancel := context.WithTimeout(a.host.CommissionerContext(), refreshTimeout)
	defer cancel()
	raw, err := a.host.NativeSeedCommand(ctx, input)
	if err != nil {
		//nolint:wrapcheck // Preserve the existing refusal text and error identity.
		return commissioner.Receipt{}, err
	}
	receipt, err := a.host.ExecuteCommissioner(ctx, raw, principal, a.host.CommissionerAuthorization())
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("record starting snapshot: %w", err)
	}
	return receipt, nil
}

func (a *API) TargetNativeDirectory(leagueID string) (commissioner.NativeDirectory, error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return a.reads.TargetNativeDirectory(leagueID)
}

func (a *API) TargetNativeAssets(leagueID string) (commissioner.NativeAssets, error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return a.reads.TargetNativeAssets(leagueID)
}

func BuildSeedCommand(ctx context.Context, input RawNativeSeedRequest,
	retry func(context.Context, string, string) (commissioner.RawCommand, bool, error),
	capture func() (commissioner.RawNativeSnapshot, error)) (commissioner.RawCommand, error) {
	raw, found, err := retry(ctx, input.League, input.Request)
	if err != nil {
		return raw, fmt.Errorf("read starting snapshot request: %w", err)
	}
	if found {
		if raw.Command != commissioner.NativeSeed || raw.Expected != input.Expected || raw.Reason != input.Reason {
			return raw, fmt.Errorf("this request was already used for a different change")
		}
		return raw, nil
	}
	seed, err := capture()
	if err != nil {
		return raw, err
	}
	return commissioner.RawCommand{League: input.League, Command: commissioner.NativeSeed,
		Args: commissioner.RawArgs{Native: &seed}, Expected: input.Expected, Request: input.Request,
		Reason: input.Reason, Effective: "now", Visibility: "league"}, nil
}
