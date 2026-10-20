package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/playerscores"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// m1Timeout bounds the IPC calls that read history's reports.
const m1Timeout = 120 * time.Second

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

// seasonPointsMeasure is the measure MFL's season fantasy-point totals are stored under.
const seasonPointsMeasure = "outcome.fantasy_points"

// seasonPointsHeld reports whether history already holds MFL's totals for season.
func (a *App) seasonPointsHeld(ctx context.Context, season int) (bool, error) {
	feats, err := a.history.Features(ctx, history.FeatureQuery{AsOf: time.Now(), Season: season,
		Measures: []string{seasonPointsMeasure}})
	if err != nil {
		return false, fmt.Errorf("app: read %d season points: %w", season, err)
	}
	return len(feats) > 0, nil
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
