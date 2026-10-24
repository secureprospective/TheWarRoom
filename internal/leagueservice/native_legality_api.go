package leagueservice

import (
	"context"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
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

func (a *Service) TargetNativeCapCheck(leagueID, viewerZone string) (commissioner.NativeCapCheck, error) {
	scope := commissioner.RawNativeCapScope{League: leagueID, ViewerZone: viewerZone}
	if err := scope.Validate(); err != nil {
		return commissioner.NativeCapCheck{}, fmt.Errorf("native salary cap input: %w", err)
	}
	if err := a.host.Ready(); err != nil {
		return commissioner.NativeCapCheck{}, fmt.Errorf("native salary cap check startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.host.CommissionerContext(), refreshTimeout)
	defer cancel()
	reading, err := a.host.NativeCapCheck(ctx, scope)
	if err != nil {
		return commissioner.NativeCapCheck{}, fmt.Errorf("native salary cap check: %w", err)
	}
	return reading, nil
}

func (a *Service) TargetNativeRosterCheck(leagueID string) (commissioner.NativeRosterCheck, error) {
	if err := a.host.Ready(); err != nil {
		return commissioner.NativeRosterCheck{}, fmt.Errorf("native roster check startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.host.CommissionerContext(), refreshTimeout)
	defer cancel()
	reading, err := a.host.NativeRosterCheck(ctx, leagueID, a.host.NativeFacts())
	if err != nil {
		return commissioner.NativeRosterCheck{}, fmt.Errorf("native roster check: %w", err)
	}
	return reading, nil
}

func (a *Service) TargetNativeReserveCheck(leagueID string) (commissioner.NativeReserveCheck, error) {
	if err := a.host.Ready(); err != nil {
		return commissioner.NativeReserveCheck{}, fmt.Errorf("native reserve check startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.host.CommissionerContext(), refreshTimeout)
	defer cancel()
	reading, err := a.host.NativeReserveCheck(ctx, leagueID, a.host.NativeFacts())
	if err != nil {
		return commissioner.NativeReserveCheck{}, fmt.Errorf("native reserve check: %w", err)
	}
	return reading, nil
}

func (a *Service) TargetNativeLineupCheck(
	input commissioner.RawNativeLineup,
) (commissioner.NativeLineupCheck, error) {
	if err := input.Validate(); err != nil {
		return commissioner.NativeLineupCheck{}, fmt.Errorf("native lineup input: %w", err)
	}
	if err := a.host.Ready(); err != nil {
		return commissioner.NativeLineupCheck{}, fmt.Errorf("native lineup check startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.host.CommissionerContext(), refreshTimeout)
	defer cancel()
	reading, err := a.host.NativeLineupCheck(ctx, input, a.host.NativeFacts())
	if err != nil {
		return commissioner.NativeLineupCheck{}, fmt.Errorf("native lineup check: %w", err)
	}
	return reading, nil
}

func (a *Service) TargetNativeHouseRules(leagueID string) (commissioner.NativeHouseRules, error) {
	if err := a.host.Ready(); err != nil {
		return commissioner.NativeHouseRules{}, fmt.Errorf("native house rules startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.host.CommissionerContext(), refreshTimeout)
	defer cancel()
	reading, err := a.host.NativeHouseRules(ctx, leagueID)
	if err != nil {
		return commissioner.NativeHouseRules{}, fmt.Errorf("native house rules: %w", err)
	}
	return reading, nil
}

func (a *Service) TargetNativePlayerScore(league, player string, season,
	week int) (commissioner.NativePlayerScore, error) {
	scope := source.RawNativeScope{Player: player, Season: season, Week: week}
	if err := scope.Validate(); err != nil {
		return commissioner.NativePlayerScore{}, fmt.Errorf("player score input: %w", err)
	}
	if err := a.host.Ready(); err != nil {
		return commissioner.NativePlayerScore{}, fmt.Errorf("player score startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.host.CommissionerContext(), refreshTimeout)
	defer cancel()
	reading, err := a.host.NativePlayerScore(ctx, league, scope)
	if err != nil {
		return commissioner.NativePlayerScore{}, fmt.Errorf("player score: %w", err)
	}
	return reading, nil
}

func (a *Service) TargetNativeContractQuote(raw commissioner.RawContractQuote) (commissioner.ContractQuote,
	error) {
	if err := raw.Validate(); err != nil {
		return commissioner.ContractQuote{}, fmt.Errorf("native contract quote input: %w", err)
	}
	if err := a.host.Ready(); err != nil {
		return commissioner.ContractQuote{}, fmt.Errorf("native contract quote startup: %w", err)
	}
	zone, err := time.LoadLocation(raw.ViewerZone)
	if err != nil {
		return commissioner.ContractQuote{}, fmt.Errorf("read viewer timezone: %w", err)
	}
	records, err := a.host.CommissionerRecords(a.host.CommissionerContext(), raw.League)
	if err != nil {
		//nolint:wrapcheck // Preserve the existing refusal text and error identity.
		return commissioner.ContractQuote{}, err
	}
	quote, err := commissioner.QuoteContract(raw, records, a.host.CommissionerNow(), zone, a.host.NativeFacts())
	if err != nil {
		return commissioner.ContractQuote{}, fmt.Errorf("native contract quote: %w", err)
	}
	return quote, nil
}

func (a *Service) TargetNativeResults(league string, season, week int) (commissioner.NativeResults, error) {
	if err := (RawNativeResultsScope{League: league, Season: season, Week: week}).Validate(); err != nil {
		return commissioner.NativeResults{}, err
	}
	if err := a.host.Ready(); err != nil {
		return commissioner.NativeResults{}, fmt.Errorf("native results startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.host.CommissionerContext(), refreshTimeout)
	defer cancel()
	reading, err := a.host.NativeResults(ctx, league, season, week)
	if err != nil {
		return commissioner.NativeResults{}, fmt.Errorf("native results: %w", err)
	}
	return reading, nil
}

func (a *Service) TargetNativeStandings(league string, season int) (commissioner.NativeStandings, error) {
	if err := (RawNativeResultsScope{League: league, Season: season, Week: 1}).Validate(); err != nil {
		return commissioner.NativeStandings{}, err
	}
	if err := a.host.Ready(); err != nil {
		return commissioner.NativeStandings{}, fmt.Errorf("native standings startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.host.CommissionerContext(), refreshTimeout)
	defer cancel()
	reading, err := a.host.NativeStandings(ctx, league, season)
	if err != nil {
		return commissioner.NativeStandings{}, fmt.Errorf("native standings: %w", err)
	}
	return reading, nil
}

func (a *Service) TargetNativeScorePreview(body string) (commissioner.ScorePreview, error) {
	raw, err := commissioner.Decode(body)
	if err != nil {
		return commissioner.ScorePreview{}, fmt.Errorf("score preview input: %w", err)
	}
	if err := a.host.Ready(); err != nil {
		return commissioner.ScorePreview{}, fmt.Errorf("score preview startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.host.CommissionerContext(), refreshTimeout)
	defer cancel()
	preview, err := a.host.NativeScorePreview(ctx, raw)
	if err != nil {
		return commissioner.ScorePreview{}, fmt.Errorf("score preview: %w", err)
	}
	return preview, nil
}

func (a *Service) TargetNativeBracket(league string, season int) (commissioner.NativeBracket, error) {
	if err := (RawNativeResultsScope{League: league, Season: season, Week: 1}).Validate(); err != nil {
		return commissioner.NativeBracket{}, err
	}
	if err := a.host.Ready(); err != nil {
		return commissioner.NativeBracket{}, fmt.Errorf("native bracket startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.host.CommissionerContext(), refreshTimeout)
	defer cancel()
	reading, err := a.host.NativeBracket(ctx, league, season)
	if err != nil {
		return commissioner.NativeBracket{}, fmt.Errorf("native bracket: %w", err)
	}
	return reading, nil
}

func (a *Service) TargetStatQueue(league string, season int) (commissioner.StatQueue, error) {
	if err := (RawNativeResultsScope{League: league, Season: season, Week: 1}).Validate(); err != nil {
		return commissioner.StatQueue{}, err
	}
	if err := a.host.Ready(); err != nil {
		return commissioner.StatQueue{}, fmt.Errorf("statistics queue startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.host.CommissionerContext(), refreshTimeout)
	defer cancel()
	reading, err := a.host.StatQueue(ctx, league, season)
	if err != nil {
		return commissioner.StatQueue{}, fmt.Errorf("statistics queue: %w", err)
	}
	return reading, nil
}
