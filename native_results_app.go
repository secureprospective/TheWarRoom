package main

import (
	"context"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

type RawNativeResultsScope struct {
	League string `json:"league"`
	Season int    `json:"season"`
	Week   int    `json:"week"`
}

func (r RawNativeResultsScope) Validate() error {
	if err := (RawLeagueScheduleScope{League: r.League}).Validate(); err != nil {
		return fmt.Errorf("native results scope: %w", err)
	}
	if r.Season < 1920 || r.Season > 9999 || r.Week < 1 || r.Week > 22 {
		return fmt.Errorf("choose a season and week")
	}
	return nil
}

// TargetNativeResults scores one week of the app league's published schedule from its lineups and held
// NFL evidence. A game missing a score has no result yet.
func (a *App) TargetNativeResults(league string, season, week int) (commissioner.NativeResults, error) {
	if err := (RawNativeResultsScope{League: league, Season: season, Week: week}).Validate(); err != nil {
		return commissioner.NativeResults{}, err
	}
	if err := a.ready(); err != nil {
		return commissioner.NativeResults{}, fmt.Errorf("native results startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, refreshTimeout)
	defer cancel()
	reading, err := a.history.NativeResults(ctx, league, season, week)
	if err != nil {
		return commissioner.NativeResults{}, fmt.Errorf("native results: %w", err)
	}
	return reading, nil
}

// TargetNativeStandings ranks a season's regular-season results under the league's tie-breakers, with any
// commissioner override beside the computed standings.
func (a *App) TargetNativeStandings(league string, season int) (commissioner.NativeStandings, error) {
	if err := (RawNativeResultsScope{League: league, Season: season, Week: 1}).Validate(); err != nil {
		return commissioner.NativeStandings{}, err
	}
	if err := a.ready(); err != nil {
		return commissioner.NativeStandings{}, fmt.Errorf("native standings startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, refreshTimeout)
	defer cancel()
	reading, err := a.history.NativeStandings(ctx, league, season)
	if err != nil {
		return commissioner.NativeStandings{}, fmt.Errorf("native standings: %w", err)
	}
	return reading, nil
}

// TargetNativeScorePreview reads a score adjustment, in the form it would be sent, and warns when it
// would turn over a game's result. It records nothing.
func (a *App) TargetNativeScorePreview(body string) (commissioner.ScorePreview, error) {
	raw, err := commissioner.Decode(body)
	if err != nil {
		return commissioner.ScorePreview{}, fmt.Errorf("score preview input: %w", err)
	}
	if err := a.ready(); err != nil {
		return commissioner.ScorePreview{}, fmt.Errorf("score preview startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, refreshTimeout)
	defer cancel()
	preview, err := a.history.NativeScorePreview(ctx, raw)
	if err != nil {
		return commissioner.ScorePreview{}, fmt.Errorf("score preview: %w", err)
	}
	return preview, nil
}
