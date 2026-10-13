package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/modelrun"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
)

// runModel values the league with the measurables model (plan Stage 7) and writes a model run.
func (a *App) runModel(ctx context.Context, lk normalize.Lookup, set params.Set) (modelrun.Report, error) {
	a.modelMu.Lock()
	defer a.modelMu.Unlock()
	started := time.Now()
	rep, err := a.runModelLocked(ctx, lk, set, false)
	if err == nil {
		a.engineFinished(ctx, started, rep)
	}
	return rep, err
}

func (a *App) runModelLocked(ctx context.Context, lk normalize.Lookup, set params.Set, refresh bool) (modelrun.Report, error) {
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
	rep, err := runner.Run(ctx, modelrun.Spec{Season: a.season, AsOf: time.Now(), Params: set, LastWeek: last, FreeAgents: free, Refresh: refresh})
	if err != nil {
		return modelrun.Report{}, fmt.Errorf("app: %w", err)
	}
	return rep, nil
}

// joinModel fills each row's measurables from the season's latest model run and returns its id,
// 0 when there is none yet.
func (a *App) joinModel(ctx context.Context, rows []RankRow) (int64, error) {
	run, ok, err := a.history.LatestRun(ctx, a.season, history.RunModel)
	if err != nil || !ok {
		return 0, err //nolint:wrapcheck // display-only; the caller reports it as a warning
	}
	scores, err := a.history.ModelScores(ctx, run.ID)
	if err != nil {
		return 0, fmt.Errorf("measurables unavailable: %w", err)
	}
	byID := make(map[string]history.ModelScore, len(scores))
	for _, s := range scores {
		byID[s.MFLID] = s
	}
	for i := range rows {
		if s, ok := byID[rows[i].MFLID]; ok {
			rows[i].NowPPG, rows[i].DynastyPPG, rows[i].ModelOK = s.NowPPG, s.DynastyPPG, true
		}
	}
	return run.ID, nil
}
