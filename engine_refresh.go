package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/modelrun"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
)

// refreshEngine belongs to the background refresh worker; all inputs finish before the pass.
func (a *App) refreshEngine(parent context.Context) {
	if a.params == nil || !a.modelMu.TryLock() {
		return
	}
	defer a.modelMu.Unlock()
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
	a.refreshMu.Lock()
	a.refreshFreeAgents(ctx)
	a.refreshMu.Unlock()
	a.checkEngine(ctx)
}

func engineDue(now time.Time, last history.Run, exists, changed bool) bool {
	return changed || !exists || now.Sub(last.AsOf) >= 24*time.Hour
}

// checkEngine is called with modelMu held.
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
	if !engineDue(a.timeNow(), last, exists, changed || a.modelDirty.Load()) {
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
	rep, err := a.runModelLocked(ctx, lk, a.params.Snapshot())
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

func (a *App) sourceHealthChanged(ctx context.Context) {
	if a.seasonChanged != nil {
		a.seasonChanged(ctx)
	}
}

// runModelLocked values the league with the measurables model (plan Stage 7) and writes a model
// run. The caller holds modelMu.
func (a *App) runModelLocked(ctx context.Context, lk normalize.Lookup, set params.Set) (modelrun.Report, error) {
	end, _ := a.rulebook.GetSetting("endWeek")
	last, err := strconv.Atoi(end)
	if err != nil || last < 1 {
		return modelrun.Report{}, fmt.Errorf("app: the league's last scoring week %q is unreadable", end)
	}
	runner, err := modelrun.New(a.league.Reader(), lk, a.history, buildLabel())
	if err != nil {
		return modelrun.Report{}, fmt.Errorf("app: %w", err)
	}
	a.seasonMu.Lock()
	var free []string
	for _, id := range a.seasonFreeAgents.value {
		free = append(free, id.String())
	}
	a.seasonMu.Unlock()
	rep, err := runner.Run(ctx, modelrun.Spec{Season: a.season, AsOf: time.Now(), Params: set, LastWeek: last, FreeAgents: free, Refresh: true})
	if err != nil {
		return modelrun.Report{}, fmt.Errorf("app: %w", err)
	}
	return rep, nil
}
