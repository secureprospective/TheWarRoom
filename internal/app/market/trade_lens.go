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
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
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
	standings, err := src.Standings(ctx, scope.League, scope.Season)
	if err != nil {
		return out, fmt.Errorf("standings: %w", err)
	}
	picks, note := lensPicks(ctx, src, scope, values, snapshot, standings.Computed)
	in.Assets = append(in.Assets, picks...)
	directions, err := commissioner.Directions(records)
	if err != nil {
		return out, fmt.Errorf("owner directions: %w", err)
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

// lensPosition names a valued player's position the way the lineup rules do: the model says K where
// the rules say PK, and a kicker the lineup cannot place has no replacement level.
func lensPosition(p string) string {
	if p == string(domain.PosK) {
		return string(leaguerules.PK)
	}
	return p
}

// lensPlayers is every rostered player with the model's two values, then every valued free agent; a
// contract with no known end counts as signed, since nothing says he leaves.
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
		out = append(out, tradelens.Asset{Token: p.ID, Franchise: p.Franchise, Position: lensPosition(v.Position),
			Now: v.NowPPG, Dynasty: v.DynastyPP, Valued: v.Available, Words: words, Age: v.Age, AgeKnown: v.AgeKnown,
			SignedNext: p.ExpirationYear == 0 || p.ExpirationYear > season})
		delete(valued, p.ID)
	}
	// Free agents belong to nobody but set each position's replacement level: anyone can sign one.
	for id, v := range valued {
		if v.Available {
			out = append(out, tradelens.Asset{Token: id, Position: lensPosition(v.Position), Now: v.NowPPG, Dynasty: v.DynastyPP,
				Valued: true})
		}
	}
	return out
}

// pickMarket is what this league's owners have paid for a pick against what it adds: the typical
// pick-for-player trade of 2026. It scales every pick alike and bounds none.
const pickMarket = 1.2

// takenPick is one rookie selection the league has made and what that player is worth today.
type takenPick struct {
	round, slot int
	comp        tradelens.Comp
}

// takenPicks reads the league's made selections; a player the model does not value became nothing.
func takenPicks(values appvalue.Reading, picks []commissioner.NativePick, season int) []takenPick {
	valued := make(map[string]appvalue.Player, len(values.Players))
	for _, p := range values.Players {
		valued[p.Player] = p
	}
	var out []takenPick
	for _, p := range picks {
		if p.Player == "" || p.Skipped || p.Season > season {
			continue
		}
		t := takenPick{round: p.Round, slot: p.Slot}
		if v := valued[p.Player]; v.Available {
			t.comp = tradelens.Comp{Position: lensPosition(v.Position), Now: v.NowPPG, Dynasty: v.DynastyPP}
		}
		out = append(out, t)
	}
	return out
}

// pickComps is the players taken where a pick sits, then each later round's. Its own group is its
// round, within a quarter of the league either side of its slot once the slot is known.
func pickComps(taken []takenPick, round, slot, teams int) [][]tradelens.Comp {
	groups := map[int][]tradelens.Comp{}
	last := 0
	for _, t := range taken {
		if t.round < round || (t.round == round && slot > 0 && t.slot > 0 && max(t.slot-slot, slot-t.slot) > teams/4) {
			continue
		}
		groups[t.round], last = append(groups[t.round], t.comp), max(last, t.round)
	}
	if len(groups[round]) == 0 {
		return nil
	}
	out := make([][]tradelens.Comp, 0, len(groups))
	for r := round; r <= last; r++ {
		if len(groups[r]) > 0 {
			out = append(out, groups[r])
		}
	}
	return out
}

// guessSlots places a pick with no draft order where its original team's record points today.
func guessSlots(standings []commissioner.StandingRow, teams int) func(commissioner.NativePick) int {
	slots := map[string]int{}
	for _, row := range standings {
		if row.Rank >= 1 && row.Rank <= teams {
			slots[row.Franchise] = teams - row.Rank + 1
		}
	}
	return func(p commissioner.NativePick) int {
		if p.Original == "" {
			return slots[p.Owner]
		}
		return slots[p.Original]
	}
}

// lensPicks values every pick still to be made by the players this league took where it sits. Until
// a draft order is set, a pick sits where its original team's record points today: the worst
// record picks first, as an owner would guess. The next draft counts in full and each draft after
// it is discounted. With no draft on record the picks are listed without a value and the note says why.
func lensPicks(ctx context.Context, src TradeLensSource, scope tradelens.Scope, values appvalue.Reading,
	snapshot *commissioner.RawNativeSnapshot, standings []commissioner.StandingRow) ([]tradelens.Asset, string) {
	guess := guessSlots(standings, len(snapshot.Franchises))
	discount, err := src.Params().GetGlobal(params.KeyPickValueDiscount)
	var drafts []commissioner.NativeDraft
	if err == nil {
		drafts, err = src.Drafts(ctx, scope.League)
	}
	_, slots := NativePickHistory(drafts, scope.Season, nil)
	taken := takenPicks(values, snapshot.Picks, scope.Season)
	var open []commissioner.NativePick
	next := math.MaxInt
	for _, p := range snapshot.Picks {
		if p.Player == "" && !p.Skipped && p.Season >= scope.Season {
			open, next = append(open, p), min(next, p.Season)
		}
	}
	var out []tradelens.Asset
	note := ""
	for _, p := range open {
		a := tradelens.Asset{Token: fmt.Sprintf("pick:%d:%d:%s", p.Season, p.Round, p.Original), Franchise: p.Owner,
			Pick: true, SignedNext: true}
		slot, where := slots[PickKey{Year: p.Season, Round: p.Round, Original: p.Original}], ""
		if slot == 0 {
			slot, where = guess(p), ", where its team's record points today"
		}
		a.Comps = pickComps(taken, p.Round, slot, len(snapshot.Franchises))
		switch {
		case err != nil:
			a.Words, note = err.Error(), err.Error()
		case len(a.Comps) == 0:
			a.Words = "Picks have no value yet: this league has no rookie draft on record to learn from"
			note = a.Words
		default:
			a.Valued, a.Weight = true, pickMarket*math.Pow(discount, float64(p.Season-next))
			for _, c := range a.Comps[0] {
				a.Dynasty += a.Weight * c.Dynasty / float64(len(a.Comps[0]))
			}
			a.Words = fmt.Sprintf("Valued from the %d players this league took in round %d", len(a.Comps[0]), p.Round)
			if slot > 0 {
				a.Words = fmt.Sprintf("Valued from the %d players this league took near pick %d.%02d%s",
					len(a.Comps[0]), p.Round, slot, where)
			}
		}
		out = append(out, a)
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
