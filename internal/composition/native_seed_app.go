package composition

import (
	"context"
	"fmt"
	"time"

	commissionerapp "github.com/secureprospective/TheWarRoom/internal/app/commissioner"
	nativeapp "github.com/secureprospective/TheWarRoom/internal/app/mflref/seed"
	"github.com/secureprospective/TheWarRoom/internal/app/season"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/draftreports"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/mflsource"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
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
	return nativeapp.CaptureSeed(a.heldSource())
}

// heldSource reads the held MFL league; callers hold RefreshMu so a refresh cannot change it mid-read.
func (a *App) heldSource() nativeapp.SeedSource {
	return nativeapp.SeedSource{
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
	}
}

// NativeAssets answers the MFL-mirror league from its held rosters in the native shape: the mirror is a
// league setting, so screens read every league through the same command.
func (h commissionerHost) NativeAssets(ctx context.Context, league string) (commissioner.NativeAssets, error) {
	if league != ingestion.LeagueID {
		return h.State.NativeAssets(ctx, league) //nolint:wrapcheck // Preserve the store wording.
	}
	h.app.core.RefreshMu.Lock()
	defer h.app.core.RefreshMu.Unlock()
	return nativeapp.MirrorAssets(h.app.heldSource(), league) //nolint:wrapcheck // Preserve the held-data refusal.
}

// NativeProjections projects the MFL-mirror league under its held settings, translated the same way a
// migration translates them; native leagues keep their rules in force.
func (h commissionerHost) NativeProjections(ctx context.Context, league string, season, week int,
	players []string) (history.NativeProjections, error) {
	if league != ingestion.LeagueID {
		return h.State.NativeProjections(ctx, league, season, week, players) //nolint:wrapcheck // Preserve the store wording.
	}
	if h.app.core.Rulebook == nil {
		return history.NativeProjections{}, fmt.Errorf("projections: hold the league settings first")
	}
	rules, _, _ := mflsource.Settings(h.app.core.Rulebook.ActiveConfig())
	return h.app.core.History.ProjectionsUnder(ctx, league, rules, season, week, players) //nolint:wrapcheck // Preserve the store wording.
}

// CommissionerRules reads the MFL-mirror league's held settings in the native rules shape, at the held
// roster revision so it agrees with the mirror's assets read.
func (h commissionerHost) CommissionerRules(ctx context.Context, league string) (commissioner.Rules, error) {
	if league != ingestion.LeagueID {
		return h.State.CommissionerRules(ctx, league) //nolint:wrapcheck // Preserve the store wording.
	}
	h.app.core.RefreshMu.Lock()
	defer h.app.core.RefreshMu.Unlock()
	source := h.app.heldSource()
	if !source.Available() {
		return commissioner.Rules{}, fmt.Errorf("hold the league rosters and settings first")
	}
	_, at := source.Roster()
	held, err := time.Parse(time.RFC3339Nano, at)
	if err != nil {
		return commissioner.Rules{}, fmt.Errorf("read held roster date: %w", err)
	}
	config := source.Config()
	rules, _, _ := mflsource.Settings(config)
	return commissioner.MirrorRulesReading(league, config.Name, rules, held.Unix()), nil
}
