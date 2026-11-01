package appvalue

import (
	"context"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// Held is the football evidence and identity links the valuation reads: the held NFL weekly stat
type Held interface {
	HeldWeeks(ctx context.Context, season int) ([]source.PlayerWeek, string, error)
	SourceIDs(ctx context.Context, idType string) (map[string]string, error)
}

// Source gathers one league's inputs and reads its values. Everything it reads is a read.
type Source struct {
	Facts   Facts
	Held    Held
	Records func(ctx context.Context, league string) ([]commissioner.Record, error)
	Params  func() params.Set
	Now     func() time.Time
}

// seasonsRead is how many seasons of league points the pass reads: the valued season and the
const seasonsRead = 5

// Read is the one league-scoped value read: the league's own settings and rosters, the record's
func (src Source) Read(ctx context.Context, league, franchise string, season int) (Reading, error) {
	if season < 1920 || season > 9999 {
		return Reading{}, fmt.Errorf("league values: choose an NFL season")
	}
	if src.Held == nil || src.Records == nil || src.Params == nil || src.Now == nil {
		return Reading{}, fmt.Errorf("league values: the read is not wired")
	}
	records, err := src.Records(ctx, league)
	if err != nil {

		return Reading{}, fmt.Errorf("league values records: %w", err)
	}
	base, err := commissioner.ReadNativeBase(league, records, src.Now())
	if err != nil {
		return Reading{}, fmt.Errorf("league values: %w", err)
	}
	in, err := leagueInputs(league, season, franchise, base, src.Now())
	if err != nil {
		return Reading{}, err
	}

	if err := src.weeks(ctx, &in); err != nil {
		return Reading{}, err
	}

	if in.GSIS, err = src.Held.SourceIDs(ctx, "gsis"); err != nil {
		//nolint:wrapcheck // Preserve the directory read's wording.
		return Reading{}, err
	}
	return Value(ctx, src.Facts, src.Params(), franchise, in)
}

// lastScoringWeek is the last week this league scores: its final playoff week, or its regular
func lastScoringWeek(rules leaguerules.RuleSet) int {
	if f := rules.Playoffs.FinalWeek; f.State == leaguerules.IsKnown && f.Value > 0 {
		return f.Value
	}
	if w := rules.Schedule.RegularWeeks; w.State == leaguerules.IsKnown && w.Value > 0 {
		return w.Value
	}
	return 0
}

func (src Source) weeks(ctx context.Context, in *Inputs) error {
	for year := in.Season - seasonsRead + 1; year <= in.Season; year++ {
		weeks, _, err := src.Held.HeldWeeks(ctx, year)
		if err != nil {
			return fmt.Errorf("league values: %d weeks: %w", year, err)
		}
		if len(weeks) == 0 {
			in.Notes = append(in.Notes, fmt.Sprintf("No NFL weekly statistics held for %d", year))
			continue
		}
		in.Weeks[year] = weeks
	}
	return nil
}

func leagueInputs(league string, season int, franchise string, base commissioner.NativeBase, now time.Time) (Inputs, error) {
	in := Inputs{League: league, Season: season, AsOf: now, Owner: map[string]string{}, Contracts: map[string]state.PlayerState{}, Weeks: map[int][]source.PlayerWeek{}, Notes: []string{}}
	rules, err := commissioner.RulesInForce(base.Records, season, now)
	if err != nil {
		return in, fmt.Errorf("league values rules: %w", err)
	}
	in.Scoring, in.Lineup = rules.Scoring, rules.Lineup
	in.LastWeek = lastScoringWeek(rules)
	if in.LastWeek == 0 {
		return in, fmt.Errorf("league values: this league's season length is not set")
	}
	if !base.Assets.Seeded {
		return in, fmt.Errorf("league values: this league has no rosters recorded yet")
	}
	found := franchise == ""
	for _, f := range base.Assets.Snapshot.Franchises {
		in.Franchises = append(in.Franchises, f.ID)
		found = found || f.ID == franchise
	}
	if !found {
		return in, fmt.Errorf("league values: choose a franchise in this league")
	}
	for _, p := range base.Assets.Snapshot.Players {
		if p.Status != domain.RosterActive && p.Status != domain.RosterTaxi && p.Status != domain.RosterIR {
			continue
		}
		if _, exists := in.Owner[p.ID]; exists {
			return in, fmt.Errorf("league values: multiple roster copies of %s need a copy-scoped valuation", p.ID)
		}
		in.Owner[p.ID] = p.Franchise
		in.Contracts[p.ID] = state.PlayerState{MFLID: p.ID, Salary: p.Salary, ExpirationYear: p.ExpirationYear}
	}
	return in, nil
}
