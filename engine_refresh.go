package main

import (
	"context"
	"log"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/modelrun"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// refreshEngine belongs to the background refresh worker; all inputs finish before the pass.
func (a *App) refreshEngine(parent context.Context) {
	if a.params == nil || !a.modelMu.TryLock() {
		return
	}
	defer a.modelMu.Unlock()
	ctx, cancel := context.WithTimeout(parent, signalsTimeout)
	defer cancel()
	a.loadSignals(ctx)
	if warning, err := a.loadSeasonPoints(ctx); err != nil {
		log.Printf("engine inputs: %v", err)
		return
	} else if warning != "" {
		log.Printf("engine inputs: %s", warning)
	}
	a.refreshMu.Lock()
	a.refreshFreeAgents(ctx)
	a.refreshMu.Unlock()
	a.checkEngine(ctx)
}

func engineDue(now time.Time, last history.Run, exists, changed bool) bool {
	return changed || !exists || now.Sub(last.AsOf) >= 24*time.Hour
}

// checkEngine is called with modelMu held, including by the legacy manual scoring path.
func (a *App) checkEngine(ctx context.Context) {
	last, exists, err := a.history.LatestRun(ctx, a.season, history.RunModel)
	if err != nil {
		log.Printf("engine latest run: %v", err)
		return
	}
	changed, err := a.history.ModelInputsChanged(ctx, last.AsOf)
	if err != nil {
		log.Printf("engine input changes: %v", err)
		return
	}
	if !engineDue(time.Now(), last, exists, changed || a.modelDirty.Load()) {
		return
	}
	lk, err := a.directory(ctx)
	if err != nil {
		log.Printf("engine directory: %v", err)
		return
	}
	// Changes arriving during the pass must remain pending for the next refresh.
	dirty := a.modelDirty.Swap(false)
	started := time.Now()
	rep, err := a.runModelLocked(ctx, lk, a.params.Snapshot(), true)
	if err != nil {
		if dirty {
			a.modelDirty.Store(true)
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
	if a.seasonChanged != nil {
		a.seasonChanged(ctx)
	}
}
