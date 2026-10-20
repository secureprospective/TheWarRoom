package main

import (
	"context"
	"fmt"
	"time"

	nativeapp "github.com/secureprospective/TheWarRoom/internal/app/native"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// nativeFacts reads the football facts the app already holds, with no network: positions and draft years
// from the held players directory under its lock, NFL listings from the held injuries feed.
func (a *App) nativeFacts() nativecheck.Facts {
	facts := nativecheck.Facts{}
	a.lookupMu.Lock()
	if a.hasLookup {
		facts = nativeapp.PlayerFacts(a.lookup)
	}
	a.lookupMu.Unlock()
	if !a.injuriesHeld() {
		return facts
	}
	nativeapp.Designate(facts, a.heldInjuries().Status)
	return facts
}

func (a *App) TargetNativeCapCheck(leagueID, viewerZone string) (commissioner.NativeCapCheck, error) {
	scope := commissioner.RawNativeCapScope{League: leagueID, ViewerZone: viewerZone}
	if err := scope.Validate(); err != nil {
		return commissioner.NativeCapCheck{}, fmt.Errorf("native salary cap input: %w", err)
	}
	if err := a.ready(); err != nil {
		return commissioner.NativeCapCheck{}, fmt.Errorf("native salary cap check startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, refreshTimeout)
	defer cancel()
	reading, err := a.history.NativeCapCheck(ctx, scope)
	if err != nil {
		return commissioner.NativeCapCheck{}, fmt.Errorf("native salary cap check: %w", err)
	}
	return reading, nil
}

func (a *App) TargetNativeRosterCheck(leagueID string) (commissioner.NativeRosterCheck, error) {
	if err := a.ready(); err != nil {
		return commissioner.NativeRosterCheck{}, fmt.Errorf("native roster check startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, refreshTimeout)
	defer cancel()
	reading, err := a.history.NativeRosterCheck(ctx, leagueID, a.nativeFacts())
	if err != nil {
		return commissioner.NativeRosterCheck{}, fmt.Errorf("native roster check: %w", err)
	}
	return reading, nil
}

func (a *App) TargetNativeReserveCheck(leagueID string) (commissioner.NativeReserveCheck, error) {
	if err := a.ready(); err != nil {
		return commissioner.NativeReserveCheck{}, fmt.Errorf("native reserve check startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, refreshTimeout)
	defer cancel()
	reading, err := a.history.NativeReserveCheck(ctx, leagueID, a.nativeFacts())
	if err != nil {
		return commissioner.NativeReserveCheck{}, fmt.Errorf("native reserve check: %w", err)
	}
	return reading, nil
}

// TargetNativeLineupCheck judges a lineup of one native franchise's active players; it records nothing.
func (a *App) TargetNativeLineupCheck(input commissioner.RawNativeLineup) (commissioner.NativeLineupCheck, error) {
	if err := input.Validate(); err != nil {
		return commissioner.NativeLineupCheck{}, fmt.Errorf("native lineup input: %w", err)
	}
	if err := a.ready(); err != nil {
		return commissioner.NativeLineupCheck{}, fmt.Errorf("native lineup check startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, refreshTimeout)
	defer cancel()
	reading, err := a.history.NativeLineupCheck(ctx, input, a.nativeFacts())
	if err != nil {
		return commissioner.NativeLineupCheck{}, fmt.Errorf("native lineup check: %w", err)
	}
	return reading, nil
}

func (a *App) TargetNativeHouseRules(leagueID string) (commissioner.NativeHouseRules, error) {
	if err := a.ready(); err != nil {
		return commissioner.NativeHouseRules{}, fmt.Errorf("native house rules startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, refreshTimeout)
	defer cancel()
	reading, err := a.history.NativeHouseRules(ctx, leagueID)
	if err != nil {
		return commissioner.NativeHouseRules{}, fmt.Errorf("native house rules: %w", err)
	}
	return reading, nil
}

// TargetNativePlayerScore reads only held NFL evidence and the app league's recorded decisions.
func (a *App) TargetNativePlayerScore(league, player string, season, week int) (commissioner.NativePlayerScore, error) {
	scope := source.RawNativeScope{Player: player, Season: season, Week: week}
	if err := scope.Validate(); err != nil {
		return commissioner.NativePlayerScore{}, fmt.Errorf("player score input: %w", err)
	}
	if err := a.ready(); err != nil {
		return commissioner.NativePlayerScore{}, fmt.Errorf("player score startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, refreshTimeout)
	defer cancel()
	reading, err := a.history.NativePlayerScore(ctx, league, scope)
	if err != nil {
		return commissioner.NativePlayerScore{}, fmt.Errorf("player score: %w", err)
	}
	return reading, nil
}

// TargetNativeContractQuote prices one contract action on one recorded native contract: allowed or
// refused with the plain reason, the new salary and years, the dead money by season and the cap effect.
// It reads the app league's ledger only and records nothing; the owner action that follows runs the same
// check.
func (a *App) TargetNativeContractQuote(raw commissioner.RawContractQuote) (commissioner.ContractQuote, error) {
	if err := raw.Validate(); err != nil {
		return commissioner.ContractQuote{}, fmt.Errorf("native contract quote input: %w", err)
	}
	if err := a.ready(); err != nil {
		return commissioner.ContractQuote{}, fmt.Errorf("native contract quote startup: %w", err)
	}
	zone, err := time.LoadLocation(raw.ViewerZone)
	if err != nil {
		return commissioner.ContractQuote{}, fmt.Errorf("read viewer timezone: %w", err)
	}
	records, err := a.leagueRecords(a.ctx, raw.League)
	if err != nil {
		return commissioner.ContractQuote{}, err
	}
	quote, err := commissioner.QuoteContract(raw, records, a.timeNow(), zone, a.nativeFacts())
	if err != nil {
		return commissioner.ContractQuote{}, fmt.Errorf("native contract quote: %w", err)
	}
	return quote, nil
}

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

// TargetNativeBracket reads a season's playoffs: the seeds, each round's games, anything the commissioner set
// by hand, and the champion with the season's title once the final is decided.
func (a *App) TargetNativeBracket(league string, season int) (commissioner.NativeBracket, error) {
	if err := (RawNativeResultsScope{League: league, Season: season, Week: 1}).Validate(); err != nil {
		return commissioner.NativeBracket{}, err
	}
	if err := a.ready(); err != nil {
		return commissioner.NativeBracket{}, fmt.Errorf("native bracket startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, refreshTimeout)
	defer cancel()
	reading, err := a.history.NativeBracket(ctx, league, season)
	if err != nil {
		return commissioner.NativeBracket{}, fmt.Errorf("native bracket: %w", err)
	}
	return reading, nil
}

// TargetStatQueue lists the statistics the NFL changed after the league counted them, newest first, with the
// decisions already made. Nothing in the queue moves a score until it is approved.
func (a *App) TargetStatQueue(league string, season int) (commissioner.StatQueue, error) {
	if err := (RawNativeResultsScope{League: league, Season: season, Week: 1}).Validate(); err != nil {
		return commissioner.StatQueue{}, err
	}
	if err := a.ready(); err != nil {
		return commissioner.StatQueue{}, fmt.Errorf("statistics queue startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, refreshTimeout)
	defer cancel()
	reading, err := a.history.StatQueue(ctx, league, season)
	if err != nil {
		return commissioner.StatQueue{}, fmt.Errorf("statistics queue: %w", err)
	}
	return reading, nil
}
