package composition

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	controlapp "github.com/secureprospective/TheWarRoom/internal/app/control"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/rosters"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/salaryadjustments"
	"github.com/secureprospective/TheWarRoom/internal/store/rulebook"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// refreshTimeout bounds one MFL refresh: the league, rules, rosters and salary adjustments.
const refreshTimeout = 2 * time.Minute

// RefreshResult reports one pull of MFL truth. Changed is false when MFL held nothing new.
type RefreshResult = controlapp.LeagueRefreshResult

// discoverSource fetches the league config for MFL's current season; the rulebook seeds from it
// on a fresh database.
type discoverSource struct{ app *App }

func (s discoverSource) Fetch(ctx context.Context) (league.RawConfig, error) {
	cfg, err := league.Discover(ctx, s.app.core.MFLClient, ingestion.LeagueID, s.app.seasonGuess())
	if err != nil {
		return league.RawConfig{}, fmt.Errorf("app: %w", err)
	}
	return cfg, nil
}

// seasonGuess is where season discovery starts: the season held, else the calendar year.
func (a *App) seasonGuess() int {
	return controlapp.SeasonGuess(a.core.Season, time.Now())
}

// refreshLeague stages the rules and mirror from MFL's rosters and salary
// adjustments for the current season. Every fetch is archived by the transport.
func (a *App) refreshLeague(
	ctx context.Context, rb *rulebook.Store, mirror *state.Mirror,
) (RefreshResult, error) {
	a.core.RefreshMu.Lock()
	defer a.sourceHealthChanged(ctx)
	defer a.core.RefreshMu.Unlock()
	return a.leagueRefresh(rb, mirror).Refresh(ctx) //nolint:wrapcheck // Preserve refresh refusals.

}

// The caller holds refreshMu, including when the watcher refreshes several feeds.
func (a *App) refreshLeagueLocked(ctx context.Context, rb *rulebook.Store, mirror *state.Mirror) (RefreshResult, error) {
	return a.leagueRefresh(rb, mirror).Run(ctx) //nolint:wrapcheck // Preserve refresh refusals.
}

func (a *App) fetchLeague(ctx context.Context, season int) (state.MirrorSnapshot, error) {
	//nolint:wrapcheck // Preserve roster and salary conversion errors.
	return controlapp.PullLeague(ctx, season,
		func(ctx context.Context, season int) ([]rosters.RawRoster, error) {
			return rosters.Fetch(ctx, a.core.MFLClient, strconv.Itoa(season), ingestion.LeagueID)
		},
		func(ctx context.Context, season int) ([]salaryadjustments.RawSalaryAdjustment, error) {
			return salaryadjustments.Fetch(ctx, a.core.MFLClient, strconv.Itoa(season), ingestion.LeagueID)
		})
}

func (a *App) leagueRefresh(rb *rulebook.Store, mirror *state.Mirror) controlapp.LeagueRefresh {
	return controlapp.LeagueRefresh{
		Discover: func(ctx context.Context) (league.RawConfig, error) {
			return league.Discover(ctx, a.core.MFLClient, ingestion.LeagueID, a.seasonGuess())
		},
		Fetch: a.fetchLeague, StageRules: rb.StageSync, StageMirror: mirror.StageReplace, Commit: a.core.Pools.CommitPrepared,
		RulesChecked: a.core.RulesCheckedAt.Store, Dirty: a.core.ModelDirty.Store, RostersChecked: func(at time.Time) { a.core.RostersCheckedAt = at }, Now: time.Now,
		DraftReports: a.core.RefreshDraftReportsFeeds, Revalidate: a.revalidateReadyRosterMoves,
	}
}

// refreshInBackground runs the launch refresh off the startup path, so an MFL outage never
// delays or blocks opening the app. It waits for startup without holding up the caller, Wails'
// message loop, and skips the refresh when startup failed or already refreshed. The result goes
// to the log.
func (a *App) refreshInBackground(parent context.Context) {
	a.core.WeekStart.Do(func() {
		a.core.WorkerMu.Lock()
		if a.core.Stopping {
			a.core.WorkerMu.Unlock()
			return
		}
		parent, a.core.WeekCancel = context.WithCancel(parent) //nolint:gosec // WorkerGroup.Stop cancels the stored function under WorkerMu and joins before closing resources.
		a.core.WorkerMu.Unlock()
		(controlapp.BackgroundRefresh{Ready: a.ready, Start: a.startWorker,
			Due: func() bool { return a.core.LaunchRefreshDue }, Clock: a.nativeWaiverClock, Watch: a.watchMoves,
			Launch: a.launchRefresh, Weeks: a.refreshWeeksInBackground}).StartWorkers(parent)
	})
}

func (a *App) launchRefresh(parent context.Context) {
	controlapp.LaunchLeagueRefresh(parent, refreshTimeout, func(ctx context.Context) (RefreshResult, error) {
		return a.refreshLeague(ctx, a.core.Rulebook, a.core.League)
	}, a.core.ModelDirty.Store, func(line string) { log.Print(line) })
}
