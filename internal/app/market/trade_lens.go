package market

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/appvalue"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/lineupdelta"
	"github.com/secureprospective/TheWarRoom/internal/playerrecord"
	"github.com/secureprospective/TheWarRoom/internal/powerrankings"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
	"github.com/secureprospective/TheWarRoom/internal/tradelens"
)

// TradeLensSource is everything the trade lens reads. All of it is held; nothing is fetched or written.
type TradeLensSource struct {
	Values    func(ctx context.Context, league string, season int) (appvalue.Reading, error)
	Records   func(ctx context.Context, league string) ([]commissioner.Record, error)
	Standings func(ctx context.Context, league string, season int) (commissioner.NativeStandings, error)
	Drafts    func(ctx context.Context, league string) ([]commissioner.NativeDraft, error)
	Players   func(ctx context.Context) ([]playerrecord.Record, error)
	Params    func() params.Set
	Now       func() time.Time
}

// TradeLens gathers one league's values, rosters, picks, lineup rules, records and owner
// directions, and reads the proposed trade through both sides' lenses.
func TradeLens(ctx context.Context, src TradeLensSource, scope tradelens.Scope) (tradelens.Reading, error) {
	out := tradelens.Reading{League: scope.League, Season: scope.Season, Assets: []tradelens.AssetValue{},
		Tune: []tradelens.Suggestion{}, Notes: []string{}}
	values, err := src.Values(ctx, scope.League, scope.Season)
	if err != nil {
		return out, fmt.Errorf("values: %w", err)
	}
	if values.Preparing {
		out.Preparing = true
		return out, nil
	}
	now := src.Now()
	records, snapshot, in, err := lensLeague(ctx, src, scope, now)
	if err != nil {
		return out, err
	}
	in.Assets = lensPlayers(values, snapshot, scope.Season)
	picks, note := lensPicks(ctx, src, scope, values, snapshot, now)
	in.Assets = append(in.Assets, picks...)
	directions, err := commissioner.Directions(records)
	if err != nil {
		return out, fmt.Errorf("owner directions: %w", err)
	}
	standings, err := src.Standings(ctx, scope.League, scope.Season)
	if err != nil {
		return out, fmt.Errorf("standings: %w", err)
	}
	read := ourReads(values, standings, in.N)
	side := func(franchise string) tradelens.Side {
		if d := tradelens.Direction(directions[franchise]); d.Valid() {
			return tradelens.Side{Franchise: franchise, Direction: d, Set: true}
		}
		return tradelens.Side{Franchise: franchise, Direction: read(franchise)}
	}
	in.You = side(scope.Franchise)
	if scope.Partner != "" {
		in.Them = side(scope.Partner)
	}
	reading, err := tradelens.Evaluate(in)
	if err != nil {
		return out, err //nolint:wrapcheck // The lens names its own refusal.
	}
	reading.League, reading.Season = scope.League, scope.Season
	if note != "" {
		reading.Notes = append(reading.Notes, note)
	}
	return reading, nil
}

// lensLeague reads the league's ledger for its rosters and its lineup: the starter count is the N.
func lensLeague(ctx context.Context, src TradeLensSource, scope tradelens.Scope, now time.Time) (
	[]commissioner.Record, *commissioner.RawNativeSnapshot, tradelens.Input, error) {
	in := tradelens.Input{Give: scope.Give, Get: scope.Get}
	records, err := src.Records(ctx, scope.League)
	if err != nil {
		return nil, nil, in, fmt.Errorf("league records: %w", err)
	}
	base, err := commissioner.ReadNativeBase(scope.League, records, now)
	if err != nil {
		return nil, nil, in, fmt.Errorf("league rosters: %w", err)
	}
	if !base.Assets.Seeded || base.Assets.Snapshot == nil {
		return nil, nil, in, fmt.Errorf("this league has no rosters recorded yet")
	}
	rules, err := commissioner.RulesInForce(base.Records, scope.Season, now)
	if err != nil {
		return nil, nil, in, fmt.Errorf("league rules: %w", err)
	}
	lineup, ok := lineupdelta.Rules(rules.Lineup)
	if !ok {
		return nil, nil, in, fmt.Errorf("this league's starter count is not set, so there is no best N to count")
	}
	in.N, in.Teams = lineup.Total, len(base.Assets.Snapshot.Franchises)
	for _, slot := range lineup.Slots {
		in.Slots = append(in.Slots, tradelens.Slot{Position: slot.Position, Min: slot.Min, Max: slot.Max})
	}
	return records, base.Assets.Snapshot, in, nil
}

// lensPlayers is every rostered player with the model's two values; a contract with no known end
// counts as signed, since nothing says he leaves.
func lensPlayers(values appvalue.Reading, snapshot *commissioner.RawNativeSnapshot, season int) []tradelens.Asset {
	valued := make(map[string]appvalue.Player, len(values.Players))
	for _, p := range values.Players {
		valued[p.Player] = p
	}
	out := make([]tradelens.Asset, 0, len(snapshot.Players))
	for _, p := range snapshot.Players {
		if p.Status != domain.RosterActive && p.Status != domain.RosterTaxi && p.Status != domain.RosterIR {
			continue
		}
		v := valued[p.ID]
		words := v.Words
		if !v.Available && words == "" {
			words = "This player has no value in this league yet"
		}
		out = append(out, tradelens.Asset{Token: p.ID, Franchise: p.Franchise, Position: v.Position,
			Now: v.NowPPG, Dynasty: v.DynastyPP, Valued: v.Available, Words: words, Age: v.Age, AgeKnown: v.AgeKnown,
			SignedNext: p.ExpirationYear == 0 || p.ExpirationYear > season})
	}
	return out
}

// lensPicks values every pick still to be made with Pick Value, fitted on this league's own values
// and rookie drafts. When it cannot be fitted the picks are listed without a value and the note says why.
func lensPicks(ctx context.Context, src TradeLensSource, scope tradelens.Scope, values appvalue.Reading,
	snapshot *commissioner.RawNativeSnapshot, now time.Time) ([]tradelens.Asset, string) {
	in := NativeInput{League: scope.League, Season: scope.Season, Teams: len(snapshot.Franchises),
		Values: map[string]float64{}}
	for _, p := range values.Players {
		if p.Available {
			in.Values[p.Player] = p.DynastyPP
		}
	}
	set := src.Params()
	seasons, err := set.GetGlobal(params.KeyDynastySeasons)
	if err == nil {
		in.Seasons = int(math.Round(seasons))
		in.Discount, err = set.GetGlobal(params.KeyPickValueDiscount)
	}
	var drafts []commissioner.NativeDraft
	if err == nil {
		drafts, err = src.Drafts(ctx, scope.League)
	}
	if err == nil {
		in.Records, err = src.Players(ctx)
	}
	if err != nil {
		in.PickWhy = err.Error()
	} else {
		in.History, in.Slots = NativePickHistory(drafts, scope.Season, in.Records)
	}
	model := NewNative(in)
	var out []tradelens.Asset
	note := ""
	for _, p := range snapshot.Picks {
		if p.Player != "" || p.Skipped || p.Season < scope.Season {
			continue
		}
		d := model.Pick(p.Season, p.Round, p.Original, now)
		words := d.ClassLabel
		if !d.Valued {
			words, note = d.Reason, d.Reason
		}
		out = append(out, tradelens.Asset{Token: fmt.Sprintf("pick:%d:%d:%s", p.Season, p.Round, p.Original),
			Franchise: p.Owner, Pick: true, Dynasty: d.Discounted, Valued: d.Valued, Words: words,
			SignedNext: true})
	}
	return out, note
}

// ourReads is our read of each franchise's direction from its record and the value-weighted age
// of its best N, against the league's median age.
func ourReads(values appvalue.Reading, standings commissioner.NativeStandings, n int) func(string) tradelens.Direction {
	counted := map[string][]powerrankings.CountValue{}
	for _, p := range values.Players {
		if !p.Available || p.Franchise == "" {
			continue
		}
		age := math.NaN()
		if p.AgeKnown {
			age = p.Age
		}
		counted[p.Franchise] = append(counted[p.Franchise], powerrankings.CountValue{Position: p.Position,
			Value: p.DynastyPP, Age: age})
	}
	ages := map[string]float64{}
	var known []float64
	for franchise, vals := range counted {
		_, age := powerrankings.Count("topn", powerrankings.LineupRules{Total: n}, vals)
		ages[franchise] = age
		if !math.IsNaN(age) {
			known = append(known, age)
		}
	}
	median := math.NaN()
	if len(known) > 0 {
		sort.Float64s(known)
		median = (known[(len(known)-1)/2] + known[len(known)/2]) / 2
	}
	rows := standings.Computed
	if standings.Override != nil && len(standings.Override.Rows) > 0 {
		rows = standings.Override.Rows
	}
	records := map[string]tradelens.Standing{}
	for _, row := range rows {
		records[row.Franchise] = tradelens.Standing{Wins: row.Wins, Losses: row.Losses, Ties: row.Ties}
	}
	return func(franchise string) tradelens.Direction {
		s := records[franchise]
		s.Age = math.NaN()
		if age, ok := ages[franchise]; ok {
			s.Age = age
		}
		return tradelens.Read(s, median)
	}
}
