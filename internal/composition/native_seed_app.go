package composition

import (
	"context"

	commissionerapp "github.com/secureprospective/TheWarRoom/internal/app/commissioner"
	nativeapp "github.com/secureprospective/TheWarRoom/internal/app/mflref/seed"
	"github.com/secureprospective/TheWarRoom/internal/app/season"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/draftreports"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

func (a *App) nativeSeedCommand(ctx context.Context,
	input RawNativeSeedRequest) (commissioner.RawCommand, error) {
	return commissionerapp.BuildLeagueSeedCommand(ctx, input, a.leagueRecords, a.timeNow, a.heldNativeSnapshot) //nolint:wrapcheck // Preserve the seed request refusal.
}

func (a *App) heldNativeSnapshot() (commissioner.RawNativeSnapshot, error) {
	a.core.RefreshMu.Lock()
	defer a.core.RefreshMu.Unlock()
	//nolint:wrapcheck // Preserve held starting-snapshot refusal.
	return nativeapp.CaptureSeed(nativeapp.SeedSource{
		Available: func() bool { return a.core.League != nil && a.core.Rulebook != nil },
		Roster:    func() (state.MirrorSnapshot, string) { snap, at, _ := a.core.League.ContractsView(); return snap, at },
		Season:    func() int { return a.core.Season },
		Draft: func() (season.Feed[[]draftreports.Pick], season.Feed[[]draftreports.FuturePick]) {
			a.core.SeasonMu.Lock()
			defer a.core.SeasonMu.Unlock()
			return season.Feed[[]draftreports.Pick](a.core.SeasonDraftResults), season.Feed[[]draftreports.FuturePick](a.core.SeasonFuturePicks)
		},
		Config: func() league.RawConfig { return a.core.Rulebook.ActiveConfig() },
		Directory: func() []snapshot.Player {
			a.core.TargetSnapshotMu.Lock()
			defer a.core.TargetSnapshotMu.Unlock()
			return a.core.TargetSnapshot.Players.Value
		},
	})
}
