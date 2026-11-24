package commissioner

import (
	"context"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
)

type RawNativeSeedRequest = commissioner.RawNativeSeedRequest

// TargetNativeSeed records the league's own start or held data, never client-provided rosters or live inputs.
func (a *API) TargetNativeSeed(input RawNativeSeedRequest) (commissioner.Receipt, error) {
	if err := input.Validate(); err != nil {
		return commissioner.Receipt{}, err //nolint:wrapcheck // Preserve the existing refusal text and error identity.
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

// TargetNFLGames is the hosted NFL scoreboard read; the desktop holds no shared scoreboard, so it
// answers an empty board.
func (a *API) TargetNFLGames(league string) (leagueservice.NFLGames, error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return a.reads.TargetNFLGames(league)
}

// TargetNFLGame exposes the shared public reading; local mode has no hosted detail holder.
func (a *API) TargetNFLGame(league, game string) (leagueservice.NFLGame, error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return a.reads.TargetNFLGame(league, game)
}

func (a *API) TargetNativeAssets(leagueID string) (commissioner.NativeAssets, error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return a.reads.TargetNativeAssets(leagueID)
}

// seedCommand checks request replay before choosing the league's own start or held capture.
func (a *API) seedCommand(ctx context.Context, input RawNativeSeedRequest) (commissioner.RawCommand, error) {
	return BuildLeagueSeedCommand(ctx, input, a.host.CommissionerRecords, a.host.CommissionerNow,
		a.host.CaptureNativeSeed)
}

func BuildLeagueSeedCommand(ctx context.Context, input RawNativeSeedRequest,
	read func(context.Context, string) ([]commissioner.Record, error), now func() time.Time,
	capture func() (commissioner.RawNativeSnapshot, error)) (commissioner.RawCommand, error) {
	var records []commissioner.Record
	return BuildSeedCommand(ctx, input, func(ctx context.Context, league, request string) (commissioner.RawCommand, bool, error) {
		var err error
		records, err = read(ctx, league)
		if err != nil {
			return commissioner.RawCommand{}, false, err
		}
		raw, found := commissioner.NativeSeedRetry(records, request)
		return raw, found, nil
	}, func() (commissioner.RawNativeSnapshot, error) {
		assets, err := commissioner.StartingAssetsAt(records, now())
		if err != nil {
			return commissioner.RawNativeSnapshot{}, fmt.Errorf("read starting assets: %w", err)
		}
		if assets == leaguerules.EmptyAssets {
			return commissioner.NativeStart(input.League, records, now())
		}
		return capture()
	})
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
