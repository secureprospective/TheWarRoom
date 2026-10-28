package composition

import (
	"context"
	"log"
	"time"

	scoringapp "github.com/secureprospective/TheWarRoom/internal/app/scoring"
	"github.com/secureprospective/TheWarRoom/internal/modelrun"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
	recordstore "github.com/secureprospective/TheWarRoom/internal/store/playerrecord"
)

// refreshEngine belongs to the background refresh worker; all inputs finish before the pass.
func (a *App) refreshEngine(parent context.Context) {
	scoringapp.RefreshEngine(parent, scoringapp.EngineRefreshSource{
		Available: func() bool { return a.core.Params != nil }, TryLock: a.core.ModelMu.TryLock, Unlock: a.core.ModelMu.Unlock,
		Changed: a.sourceHealthChanged, Timeout: signalsTimeout,
		Signals: func(ctx context.Context) { a.loadSignals(ctx) }, Points: a.loadSeasonPoints,
		FreeAgents: func(ctx context.Context) {
			a.core.RefreshMu.Lock()
			defer a.core.RefreshMu.Unlock()
			a.refreshFreeAgents(ctx)
		},
		Check: a.checkEngine, Log: func(message string) { log.Print(message) },
	})
}

// checkEngine is called with modelMu held.
func (a *App) checkEngine(ctx context.Context) {
	scoringapp.CheckEngine(ctx, scoringapp.EngineCheckSource{
		Latest: func(ctx context.Context) (history.Run, bool, error) {
			return a.core.History.LatestRun(ctx, a.core.Season, history.RunModel)
		},
		InputsChanged: a.core.History.ModelInputsChanged, Now: a.timeNow, Started: time.Now,
		DirtyLoad: a.core.ModelDirty.Load, DirtySwap: a.core.ModelDirty.Swap, DirtyStore: a.core.ModelDirty.Store,
		Directory: a.directory, Run: func(ctx context.Context, lookup normalize.Lookup) (modelrun.Report, error) {
			return a.runModelLocked(ctx, lookup, a.core.Params.Snapshot())
		},
		Finished: a.engineFinished, Log: func(message string) { log.Print(message) },
	})
}

func (a *App) engineFinished(ctx context.Context, started time.Time, rep modelrun.Report) {
	scoringapp.EngineFinished(ctx, started, rep, time.Since, a.core.SeasonChanged, func(message string) { log.Print(message) })
}

func (a *App) sourceHealthChanged(ctx context.Context) {
	scoringapp.SourceHealthChanged(ctx, a.core.SeasonChanged)
}

// runModelLocked runs with modelMu held; the season snapshot remains under seasonMu.
func (a *App) runModelLocked(ctx context.Context, lk normalize.Lookup, set params.Set) (modelrun.Report, error) {
	src := scoringapp.ModelSource{EndWeek: func() (string, bool) { return a.core.Rulebook.GetSetting("endWeek") },
		State: a.core.League.Reader, History: a.core.History,
		Records: recordstore.New(a.core.HistPools, a.core.History), BuildLabel: buildLabel(), Season: a.core.Season,
		FreeAgents: a.modelFreeAgents}
	return src.RunModel(ctx, lk, set) //nolint:wrapcheck // the service preserves the existing app error labels
}

func (a *App) modelFreeAgents() []string {
	a.core.SeasonMu.Lock()
	defer a.core.SeasonMu.Unlock()
	return scoringapp.FreeAgentIDs(a.core.SeasonFreeAgents.Value)
}
