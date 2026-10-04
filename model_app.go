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
	end, _ := a.rulebook.GetSetting("endWeek")
	last, err := strconv.Atoi(end)
	if err != nil || last < 1 {
		return modelrun.Report{}, fmt.Errorf("app: the league's last scoring week %q is unreadable", end)
	}
	runner, err := modelrun.New(a.league.Reader(), lk, a.history, buildLabel())
	if err != nil {
		return modelrun.Report{}, fmt.Errorf("app: %w", err)
	}
	rep, err := runner.Run(ctx, modelrun.Spec{Season: a.season, AsOf: time.Now(), Params: set, LastWeek: last})
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
