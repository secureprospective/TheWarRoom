package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/playerscores"
	"github.com/secureprospective/TheWarRoom/internal/modelrun"
	"github.com/secureprospective/TheWarRoom/internal/rankings"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// m1Timeout bounds the M1 IPC calls: ScoreLeague fetches two MFL exports and scores ~1,200
// players.
const m1Timeout = 120 * time.Second

// proxyLabel must accompany every score surface while base points are MFL's YTD totals
// (Christopher's ruling); the UI renders it verbatim.
func (a *App) proxyLabel() string {
	if a.season == 0 { // startup failed before the season was parsed
		return "BasePoints: MFL YTD fantasy points (proxy) — L2 pending"
	}
	return fmt.Sprintf("BasePoints: MFL %d YTD fantasy points (proxy) — L2 pending", a.season-1)
}

// loadSeasonPoints keeps history holding MFL's season totals for every league season: each
// earlier season once, the last one again (so late stat corrections land), and the current one
// every pass, which is empty before its first game. scoringYear says whose rules apply. A failed fetch is recorded against the
// source and returned as a warning: scoring runs from what history holds, and source health
// decides whether the measure counts as lost.
func (a *App) loadSeasonPoints(ctx context.Context) (warning string, err error) {
	var warnings []string
	for y := ingestion.LeagueFirstSeason; y <= a.season; y++ {
		if y < a.season-1 {
			held, err := a.seasonPointsHeld(ctx, y)
			if err != nil {
				return "", err
			}
			if held {
				continue
			}
		}
		batch, ferr := playerscores.Fetch(ctx, a.mflClient, strconv.Itoa(scoringYear(y, a.season)), ingestion.LeagueID, y)
		switch {
		case ferr == nil:
			if _, ierr := a.history.Ingest(ctx, batch); ierr != nil {
				return "", fmt.Errorf("app: store %d season points: %w", y, ierr)
			}
		case y == a.season && errors.Is(ferr, playerscores.ErrEmptyScores):
		default:
			if lerr := a.history.LoadFailed(ctx, playerscores.Source, ferr); lerr != nil {
				return "", errors.Join(ferr, lerr)
			}
			warnings = append(warnings, strings.TrimRight(fmt.Sprintf("%d: %v", y, ferr), ".; "))
		}
	}
	if len(warnings) > 0 {
		return "MFL fantasy points not refreshed (" + strings.Join(warnings, "; ") + "): scored from the last points held", nil
	}
	return "", nil
}

// scoringYear is the league year whose scoring rules MFL applies to season y's totals. The
// current league serves the last season under today's rules, which is what the board's base
// points have always been; older seasons are served only by their own league year, under their
// own rules. The model's percentiles are within a season, so a season-wide rule change barely
// moves them, and its points per game come from last season, under today's rules.
func scoringYear(y, current int) int {
	if y >= current-1 {
		return current
	}
	return y
}

// seasonPointsHeld reports whether history already holds MFL's totals for season.
func (a *App) seasonPointsHeld(ctx context.Context, season int) (bool, error) {
	feats, err := a.history.Features(ctx, history.FeatureQuery{AsOf: time.Now(), Season: season,
		Measures: []string{rankings.BaseMeasure}})
	if err != nil {
		return false, fmt.Errorf("app: read %d season points: %w", season, err)
	}
	return len(feats) > 0, nil
}

// ScoreLeagueResult is the board's scoring report (run, scored, zero-base, exclusions with
// reasons) and the model run's, plus the proxy label. Warning reports a source that failed this
// pass.
type ScoreLeagueResult struct {
	OK      bool            `json:"ok"`
	Error   string          `json:"error"`
	Warning string          `json:"warning"`
	Label   string          `json:"label"`
	Report  rankings.Report `json:"report"`
	Model   modelrun.Report `json:"model"`
}

// ScoreLeague scores all 32 rosters as a board run and a model run in history. A pass that matches the latest
// board exactly writes nothing and reports unchanged.
func (a *App) ScoreLeague() ScoreLeagueResult {
	if err := a.ready(); err != nil {
		return ScoreLeagueResult{Error: err.Error(), Label: a.proxyLabel()}
	}
	ctx, cancel := context.WithTimeout(a.ctx, m1Timeout)
	defer cancel()
	fail := func(err error) ScoreLeagueResult {
		return ScoreLeagueResult{Error: err.Error(), Label: a.proxyLabel()}
	}

	lk, err := a.directory(ctx)
	if err != nil {
		return fail(err)
	}
	warning, err := a.loadSeasonPoints(ctx)
	if err != nil {
		return fail(err)
	}
	// One crosswalk fetch serves the player directory and every scouting signal. A directory
	// write failure is logged, not fatal: the board does not read the directory yet.
	cw, err := a.fetchCrosswalk(ctx)
	if err != nil {
		return fail(err)
	}
	if _, err := a.linkCrosswalk(ctx, cw, lk); err != nil {
		log.Printf("the war room: %v", err)
	}
	// A scouting fetch failure fails the run, so a signal-less league is visible. The exception
	// is an unconfigured CFBD key (in buildScoutingDirectory).
	scout, err := a.buildScoutingDirectory(ctx, lk, cw)
	if err != nil {
		return fail(err)
	}
	runner, err := rankings.New(a.league.Reader(), lk, scout, a.rulebook, a.history, buildLabel())
	if err != nil {
		return fail(err)
	}
	set := a.params.Snapshot() // the board and the model score with the same params
	rep, err := runner.Run(ctx, rankings.RunSpec{
		Kind: history.RunBoard, Season: a.season, AsOf: time.Now(),
		Params: set, Measures: rankings.BoardMeasures(),
	})
	if err != nil {
		return fail(err)
	}
	modelRep, err := a.runModel(ctx, lk, set)
	if err != nil {
		return fail(err)
	}
	names := a.rulebook.FranchiseNames()
	for i := range rep.Excluded {
		rep.Excluded[i].FranchiseName = domain.FranchiseLabel(names, rep.Excluded[i].FranchiseID)
	}
	return ScoreLeagueResult{OK: true, Warning: warning, Label: a.proxyLabel(), Report: rep, Model: modelRep}
}

// describeMeasures pairs each measure name with its dictionary meaning.
func (a *App) describeMeasures(names []string) []MissingMeasure {
	out := make([]MissingMeasure, len(names))
	for i, n := range names {
		m, _ := a.history.Registry().Measure(n)
		out[i] = MissingMeasure{Name: n, Meaning: m.Meaning}
	}
	return out
}

// MissingMeasure is a measure the board's model reads that no active source fed when it ran.
type MissingMeasure struct {
	Name    string `json:"name"`
	Meaning string `json:"meaning"`
}
