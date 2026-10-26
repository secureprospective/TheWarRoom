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
)

// refreshEngine belongs to the background refresh worker; all inputs finish before the pass.
func (a *App) refreshEngine(parent context.Context) {
	if a.core.Params == nil || !a.core.ModelMu.TryLock() {
		return
	}
	defer a.core.ModelMu.Unlock()
	defer a.sourceHealthChanged(parent)
	ctx, cancel := context.WithTimeout(parent, signalsTimeout)
	defer cancel()
	a.loadSignals(ctx)
	if warning, err := a.loadSeasonPoints(ctx); err != nil {
		log.Printf("engine inputs: %v", err)
		return
	} else if warning != "" {
		log.Printf("engine inputs: %s", warning)
	}
	a.core.RefreshMu.Lock()
	a.refreshFreeAgents(ctx)
	a.core.RefreshMu.Unlock()
	a.checkEngine(ctx)
}

// checkEngine is called with modelMu held.
func (a *App) checkEngine(ctx context.Context) {
	last, exists, err := a.core.History.LatestRun(ctx, a.core.Season, history.RunModel)
	if err != nil {
		log.Printf("engine latest run: %v", err)
		return
	}
	changed, err := a.core.History.ModelInputsChanged(ctx, last.AsOf)
	if err != nil {
		log.Printf("engine input changes: %v", err)
		return
	}
	if !scoringapp.EngineDue(a.timeNow(), last, exists, changed || a.core.ModelDirty.Load()) {
		return
	}
	lk, err := a.directory(ctx)
	if err != nil {
		log.Printf("engine directory: %v", err)
		return
	}
	// Changes arriving during the pass must remain pending for the next refresh.
	dirty := a.core.ModelDirty.Swap(false)
	started := time.Now()
	rep, err := a.runModelLocked(ctx, lk, a.core.Params.Snapshot())
	if err != nil {
		if dirty {
			a.core.ModelDirty.Store(true)
		}
		log.Printf("engine run failed: %v", err)
		return
	}
	a.engineFinished(ctx, started, rep)
}

func (a *App) engineFinished(ctx context.Context, started time.Time, rep modelrun.Report) {
	log.Printf("Engine run %s: %d players valued (%d rostered, %d free agents), %s",
		started.Local().Format(time.RFC3339), rep.Scored, rep.RosteredScored, rep.FreeAgentScored,
		time.Since(started).Round(time.Millisecond))
	if a.core.SeasonChanged != nil {
		a.core.SeasonChanged(ctx)
	}
}

func (a *App) sourceHealthChanged(ctx context.Context) {
	if a.core.SeasonChanged != nil {
		a.core.SeasonChanged(ctx)
	}
}

// runModelLocked runs with modelMu held; the season snapshot remains under seasonMu.
func (a *App) runModelLocked(ctx context.Context, lk normalize.Lookup, set params.Set) (modelrun.Report, error) {
	src := scoringapp.ModelSource{EndWeek: func() (string, bool) { return a.core.Rulebook.GetSetting("endWeek") },
		State: a.core.League.Reader, History: a.core.History, BuildLabel: buildLabel(), Season: a.core.Season,
		FreeAgents: a.modelFreeAgents}
	return src.RunModel(ctx, lk, set) //nolint:wrapcheck // the service preserves the existing app error labels
}

func (a *App) modelFreeAgents() []string {
	a.core.SeasonMu.Lock()
	defer a.core.SeasonMu.Unlock()
	return scoringapp.FreeAgentIDs(a.core.SeasonFreeAgents.Value)
}
