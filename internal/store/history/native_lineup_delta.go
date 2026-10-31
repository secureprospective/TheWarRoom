package history

import (
	"context"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/lineupdelta"
)

// NativeLineupDelta is what a proposed trade does to both parties' best lineups (D33).
type NativeLineupDelta struct {
	League string           `json:"league"`
	Season int              `json:"season"`
	Week   int              `json:"week"`
	You    lineupdelta.Side `json:"you"`
	Them   lineupdelta.Side `json:"them"`
	Notes  []string         `json:"notes"`
}

// TradeSides names a proposed trade: what the franchise gives and what it gets from the partner.
type TradeSides struct {
	Franchise, Partner string
	Give, Get          []string
}

// NativeLineupDelta reads each party's best lineup before and after a proposed trade, valued at
// the D11 median projection under the rules in force. It is computed per request and never cached.
func (s *Store) NativeLineupDelta(ctx context.Context, league string, season, week int,
	trade TradeSides) (NativeLineupDelta, error) {
	out := NativeLineupDelta{League: league, Season: season, Week: week, Notes: []string{}}
	records, err := s.nativeRecords(ctx, league)
	if err != nil {
		return out, err
	}
	base, err := commissioner.ReadNativeBase(league, records, s.now())
	if err != nil {
		return out, fmt.Errorf("lineup delta: %w", err)
	}
	rules, ok := lineupdelta.Rules(base.Rules.Lineup)
	if !ok {
		return out, fmt.Errorf("lineup delta: this league's starting limits are not all set")
	}
	mine, theirs, err := tradeRosters(base, trade)
	if err != nil {
		return out, err
	}
	ids := append(append([]string{}, mine...), theirs...)
	proj, err := s.NativeProjections(ctx, league, season, week, ids)
	if err != nil {
		return out, err
	}
	out.Notes = append(out.Notes, proj.Notes...)
	players := s.deltaPlayers(ctx, proj, season, week, &out.Notes)
	pick := func(list []string) []lineupdelta.Player {
		got := make([]lineupdelta.Player, 0, len(list))
		for _, id := range list {
			got = append(got, players[id])
		}
		return got
	}
	out.You = lineupdelta.Compute(trade.Franchise, pick(mine), trade.Give, pick(trade.Get), rules)
	out.Them = lineupdelta.Compute(trade.Partner, pick(theirs), trade.Get, pick(trade.Give), rules)
	return out, nil
}

// tradeRosters checks the trade against the recorded rosters and returns each side's active players.
func tradeRosters(base commissioner.NativeBase, trade TradeSides) (mine, theirs []string, err error) {
	if !base.Assets.Seeded {
		return nil, nil, fmt.Errorf("lineup delta: this league has no rosters recorded yet")
	}
	if trade.Franchise == trade.Partner || len(trade.Give)+len(trade.Get) == 0 {
		return nil, nil, fmt.Errorf("lineup delta: name two franchises and at least one player")
	}
	owner := map[string]string{}
	for _, p := range base.Assets.Snapshot.Players {
		owner[p.ID] = p.Franchise
		if p.Status != domain.RosterActive {
			continue
		}
		switch p.Franchise {
		case trade.Franchise:
			mine = append(mine, p.ID)
		case trade.Partner:
			theirs = append(theirs, p.ID)
		}
	}
	for _, id := range trade.Give {
		if owner[id] != trade.Franchise {
			return nil, nil, fmt.Errorf("lineup delta: player %s is not on franchise %s", id, trade.Franchise)
		}
	}
	for _, id := range trade.Get {
		if owner[id] != trade.Partner {
			return nil, nil, fmt.Errorf("lineup delta: player %s is not on franchise %s", id, trade.Partner)
		}
	}
	return withReserves(mine, trade.Give), withReserves(theirs, trade.Get), nil
}

// withReserves adds traded taxi or IR players, who still change hands, to a side's candidates.
func withReserves(active, traded []string) []string {
	have := map[string]bool{}
	for _, id := range active {
		have[id] = true
	}
	for _, id := range traded {
		if !have[id] {
			active = append(active, id)
		}
	}
	return active
}

// deltaPlayers turns projections into lineup candidates: position from the held player facts
// (the projection's when none), and whether each plays in the week from the held schedule.
func (s *Store) deltaPlayers(ctx context.Context, proj NativeProjections, season, week int,
	notes *[]string) map[string]lineupdelta.Player {
	var facts commissioner.LineupFacts
	held := false
	if s.lineupEnv != nil {
		if f, err := s.lineupEnv.LineupFacts(ctx, season, week); err == nil {
			facts, held = f, f.ScheduleHeld
		}
	}
	if !held {
		*notes = append(*notes, "NFL schedule for the week not held: this week's change counts every player as playing")
	}
	out := map[string]lineupdelta.Player{}
	for _, p := range proj.Players {
		pos := p.Position
		if f, ok := facts.Facts[p.Player]; ok && f.Position != "" {
			pos = string(f.Position)
		}
		out[p.Player] = lineupdelta.Player{ID: p.Player, Position: pos, PPG: p.Median, Projected: p.Available,
			Plays: !held || plays(facts, p.Player)}
	}
	return out
}

func plays(facts commissioner.LineupFacts, player string) bool {
	f := facts.Facts[player]
	if f.Team == "" || f.Team == "FA" || f.Designation == leaguerules.Out {
		return false
	}
	_, ok := facts.Locks[f.Team]
	return ok
}
