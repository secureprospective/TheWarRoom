// Package league fetches the league's rulebook from two MFL exports, `league` (settings,
// roster limits, starters, cap amount) and `rules` (scoring; MFL does not put scoring in
// `league`, verified 2026-06-26), and returns it as raw strings. It discovers the league host
// first, since league calls route to the league's own server.
package league

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
)

// errEmptyScoring: a live league always has scoring rules, so zero means a failed fetch, and
// storing it would blank the rulebook.
var errEmptyScoring = errors.New("league: rules export contained zero scoring rules")

// Source supplies a RawConfig; the rulebook depends on it so tests can use a canned config.
type Source interface {
	Fetch(ctx context.Context) (RawConfig, error)
}

// APISource is the live Source for one season and league.
type APISource struct {
	Client   *mfl.Client
	Year     string
	LeagueID string
}

// Fetch implements Source against the live MFL API.
func (s APISource) Fetch(ctx context.Context) (RawConfig, error) {
	return Fetch(ctx, s.Client, s.Year, s.LeagueID)
}

// Fetch discovers the host, pulls both exports, checks each for MFL's error envelope and
// returns a shape-validated RawConfig.
func Fetch(ctx context.Context, c *mfl.Client, year, leagueID string) (RawConfig, error) {
	if err := c.DiscoverHost(ctx, year, leagueID); err != nil {
		return RawConfig{}, fmt.Errorf("league: discover host: %w", err)
	}

	leagueBody, err := ingestion.LeagueExport(ctx, c, "league", year, leagueID, nil)
	if err != nil {
		return RawConfig{}, fmt.Errorf("league: %w", err)
	}
	rulesBody, err := ingestion.LeagueExport(ctx, c, "rules", year, leagueID, nil)
	if err != nil {
		return RawConfig{}, fmt.Errorf("league: %w", err)
	}

	cfg, err := assemble(leagueBody, rulesBody)
	if err != nil {
		return RawConfig{}, err
	}
	cfg.Source = fmt.Sprintf("mfl:%s", year)
	return cfg, nil
}

// assemble decodes both exports and rejects a config that cannot be valid: no scoring rules,
// or no cap amount in a salary league.
func assemble(leagueBody, rulesBody []byte) (RawConfig, error) {
	var le leagueEnvelope
	if err := json.Unmarshal(leagueBody, &le); err != nil {
		return RawConfig{}, fmt.Errorf("league: decode league export: %w", err)
	}
	var re rulesEnvelope
	if err := json.Unmarshal(rulesBody, &re); err != nil {
		return RawConfig{}, fmt.Errorf("league: decode rules export: %w", err)
	}

	cfg := mapLeague(le)
	cfg.ScoringRules = mapRules(re)
	if len(cfg.ScoringRules) == 0 {
		return RawConfig{}, errEmptyScoring
	}
	if cfg.UsesSalaries == "1" && cfg.SalaryCapAmount == "" {
		return RawConfig{}, fmt.Errorf("league: salary league missing cap amount")
	}
	return cfg, nil
}

// mapLeague copies the decoded `league` envelope into RawConfig's flat shape.
func mapLeague(le leagueEnvelope) RawConfig {
	l := le.League
	cfg := RawConfig{
		SalaryCapAmount:             l.SalaryCapAmount,
		RosterSize:                  l.RosterSize,
		TaxiSquad:                   l.TaxiSquad,
		InjuredReserve:              l.InjuredReserve,
		KeeperType:                  l.KeeperType,
		UsesSalaries:                l.UsesSalaries,
		UsesContractYear:            l.UsesContractYear,
		StartWeek:                   l.StartWeek,
		EndWeek:                     l.EndWeek,
		LastRegularSeasonWeek:       l.LastRegularSeasonWeek,
		IncludeTaxiWithSalary:       l.IncludeTaxiWithSalary,
		IncludeIRWithSalary:         l.IncludeIRWithSalary,
		IncludeTaxiWithContractYear: l.IncludeTaxiWithContractYear,
		RosterLimits:                mapLimits(l.RosterLimits.Position),
		Starters: Starters{
			Count:       l.Starters.Count,
			IOPStarters: l.Starters.IOPStarters,
			IDPStarters: l.Starters.IDPStarters,
			Positions:   mapLimits(l.Starters.Position),
		},
		Franchises: mapFranchises(l.Franchises.Franchise),
	}
	return cfg
}

// mapFranchises drops entries with an empty id; a blank name is kept and the UI shows the id.
func mapFranchises(in []franchiseEntry) []Franchise {
	out := make([]Franchise, 0, len(in))
	for _, f := range in {
		if f.ID == "" {
			continue
		}
		out = append(out, Franchise(f))
	}
	return out
}

// mapLimits converts decoded position rows into the public PositionLimit slice.
func mapLimits(in []posLimit) []PositionLimit {
	out := make([]PositionLimit, 0, len(in))
	for _, p := range in {
		out = append(out, PositionLimit(p))
	}
	return out
}

// mapRules converts scoring blocks, unwrapping each {"$t":...} leaf.
func mapRules(re rulesEnvelope) []PositionRuleSet {
	out := make([]PositionRuleSet, 0, len(re.Rules.PositionRules))
	for _, b := range re.Rules.PositionRules {
		rules := make([]ScoringRule, 0, len(b.Rule))
		for _, r := range b.Rule {
			rules = append(rules, ScoringRule{
				Event:  r.Event.T,
				Points: r.Points.T,
				Range:  r.Range.T,
			})
		}
		out = append(out, PositionRuleSet{Positions: b.Positions, Rules: rules})
	}
	return out
}
