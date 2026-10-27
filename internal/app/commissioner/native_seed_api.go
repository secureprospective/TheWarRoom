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
	raw, err := a.seedCommand(ctx, input)
	if err != nil {
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

// seedCommand shapes the starting snapshot: a 32-Team Default league starts from its own franchises and
// O-Line cards; every other league captures held data through the host.
func (a *API) seedCommand(ctx context.Context, input RawNativeSeedRequest) (commissioner.RawCommand, error) {
	records, err := a.host.CommissionerRecords(ctx, input.League)
	if err != nil {
		return commissioner.RawCommand{}, fmt.Errorf("read the league before recording its start: %w", err)
	}
	if commissioner.IsDefault32(records) {
		//nolint:wrapcheck // The refusal text is the user-facing reason.
		return commissioner.Default32SeedCommand(records, input.League, input.Request, input.Reason, input.Expected,
			a.host.CommissionerNow())
	}
	//nolint:wrapcheck // Preserve the existing refusal text and error identity.
	return a.host.NativeSeedCommand(ctx, input)
}
