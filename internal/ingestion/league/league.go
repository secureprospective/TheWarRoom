// Package league is the Layer 1 fetcher for the MFL league rulebook. It assembles
// the league's configuration from the TWO MFL exports that carry it — `league`
// (settings, roster limits, starters, salary-cap amount) and `rules` (the
// position-additive scoring config) — and returns one RAW RawConfig. It transforms
// nothing: every value stays the verbatim MFL string (B3b's store and the engine
// interpret them). It discovers the league host FIRST (league-specific calls route
// to the league's home server), then issues the two calls through the shared mfl
// transport client, inheriting rate-limiting, 429 backoff, and host routing.
//
// NOTE (corrects the B3b handoff premise): MFL does NOT return scoring config in
// the `league` export. Scoring lives in the separate `rules` export. Verified live
// 2026-06-26 against league 14432. Hence two calls, not one.
package league

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
)

// errEmptyScoring guards the glitch/error shape where the rules export decodes to
// zero scoring rules. A live MFL league always returns a populated rule set, so an
// empty one is a fetch failure (an error payload, maintenance, wrong league), not a
// valid "no scoring" state — surfacing it stops a blank rulebook from being stored.
var errEmptyScoring = errors.New("league: rules export contained zero scoring rules")

// Source supplies a RawConfig. The store depends on this (not on Fetch directly) so
// it is unit-testable with a canned config; APISource is the production adapter.
type Source interface {
	Fetch(ctx context.Context) (RawConfig, error)
}

// APISource is the production Source: it pulls the live config from MFL for a fixed
// season+league through the transport client.
type APISource struct {
	Client   *mfl.Client
	Year     string
	LeagueID string
}

// Fetch implements Source against the live MFL API.
func (s APISource) Fetch(ctx context.Context) (RawConfig, error) {
	return Fetch(ctx, s.Client, s.Year, s.LeagueID)
}

// Fetch retrieves the league rulebook for the given season+league from MFL: it
// discovers the host, pulls the `league` and `rules` exports, guards MFL's HTTP-200
// error envelope on each, and assembles a shape-validated RawConfig. year and
// leagueID are explicit arguments (not globals) so the fetcher is testable and
// multi-season-ready; ingestion.SeasonYear/LeagueID are the canonical values.
func Fetch(ctx context.Context, c *mfl.Client, year, leagueID string) (RawConfig, error) {
	if err := c.DiscoverHost(ctx, year, leagueID); err != nil {
		return RawConfig{}, fmt.Errorf("league: discover host: %w", err)
	}

	leagueBody, err := call(ctx, c, "league", year, leagueID)
	if err != nil {
		return RawConfig{}, err
	}
	rulesBody, err := call(ctx, c, "rules", year, leagueID)
	if err != nil {
		return RawConfig{}, err
	}

	cfg, err := assemble(leagueBody, rulesBody)
	if err != nil {
		return RawConfig{}, err
	}
	cfg.Source = fmt.Sprintf("mfl:%s", year)
	return cfg, nil
}

// call issues one league-specific MFL export and returns its raw body, failing loud
// on a transport error, a non-200 status, or MFL's 200-with-error envelope.
func call(ctx context.Context, c *mfl.Client, endpoint, year, leagueID string) ([]byte, error) {
	resp, err := c.Do(ctx, mfl.Request{
		Type:   endpoint,
		Year:   year,
		Params: map[string]string{"L": leagueID},
	})
	if err != nil {
		return nil, fmt.Errorf("league: fetch %s: %w", endpoint, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("league: %s unexpected status %d", endpoint, resp.StatusCode)
	}
	if err := ingestion.CheckAPIError(resp.Body); err != nil {
		return nil, fmt.Errorf("league: %s: %w", endpoint, err)
	}
	return resp.Body, nil
}

// assemble decodes both exports into one RawConfig and validates the load-bearing
// fields are present. It does not convert types — only rejects a config that cannot
// be valid (no scoring rules, no cap amount when the league uses salaries).
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
		SalaryCapAmount:       l.SalaryCapAmount,
		RosterSize:            l.RosterSize,
		TaxiSquad:             l.TaxiSquad,
		InjuredReserve:        l.InjuredReserve,
		KeeperType:            l.KeeperType,
		UsesSalaries:          l.UsesSalaries,
		UsesContractYear:      l.UsesContractYear,
		StartWeek:             l.StartWeek,
		EndWeek:               l.EndWeek,
		LastRegularSeasonWeek: l.LastRegularSeasonWeek,
		RosterLimits:          mapLimits(l.RosterLimits.Position),
		Starters: Starters{
			Count:       l.Starters.Count,
			IOPStarters: l.Starters.IOPStarters,
			IDPStarters: l.Starters.IDPStarters,
			Positions:   mapLimits(l.Starters.Position),
		},
	}
	return cfg
}

// mapLimits converts decoded position rows into the public PositionLimit slice.
func mapLimits(in []posLimit) []PositionLimit {
	out := make([]PositionLimit, 0, len(in))
	for _, p := range in {
		out = append(out, PositionLimit(p))
	}
	return out
}

// mapRules converts decoded position rule blocks into the public scoring slice,
// unwrapping each leaf's MFL {"$t":...} value to its raw string.
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
