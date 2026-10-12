package main

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
)

type SpinePlayer struct {
	ID    string    `json:"id"`
	Name  string    `json:"name"`
	Team  string    `json:"team"`
	Mine  bool      `json:"mine"`
	Photo PhotoView `json:"photo"`
}
type SpineEntry struct {
	Player SpinePlayer   `json:"player"`
	Play   livegame.Play `json:"play"`
	Game   string        `json:"game"` // "AWAY at HOME", so a week-wide entry names its game
}

func matchupStarterIDs(l leaguefeed.Lineups, mine string) map[string]bool {
	opponent := ""
	for _, m := range l.Matchups {
		if m.Home == mine {
			opponent = m.Away
		}
		if m.Away == mine {
			opponent = m.Home
		}
	}
	out := map[string]bool{}
	if opponent == "" || mine == "" {
		return out
	}
	for _, f := range l.Franchises {
		if f.Franchise != mine && f.Franchise != opponent {
			continue
		}
		for _, id := range f.Starters {
			out[id.String()] = f.Franchise == mine
		}
	}
	return out
}
func (a *App) heldMatchupStarters(mine string) map[string]bool {
	week := a.marketWeek()
	a.seasonMu.Lock()
	defer a.seasonMu.Unlock()
	if a.seasonLineups.value.Week == week {
		if ids := matchupStarterIDs(a.seasonLineups.value, mine); len(ids) > 0 {
			return ids
		}
	}
	out := map[string]bool{}
	for _, w := range a.race.weeks.value {
		if w.Week != week {
			continue
		}
		if g, ok := findGame(w, mine); ok {
			for _, side := range []leaguefeed.Side{g.Home, g.Away} {
				for _, p := range side.Players {
					if p.Starter {
						out[p.ID.String()] = side.Franchise == mine
					}
				}
			}
		}
	}
	return out
}

// liveGameSpines fills the followed game's spine and the week's: every held play (followed
// games and finals already read) with a starter from my matchup. Nothing is fetched here.
func (a *App) liveGameSpines(ctx context.Context, v *LiveGameView, adapter livegame.Adapter, mine string) error {
	held := heldWeekGames(adapter)
	v.WeekHeld = len(held)
	if (v.Followed == nil || len(v.Followed.Plays) == 0) && len(held) == 0 {
		return nil
	}
	if a.history == nil {
		return fmt.Errorf("crosswalk directory unavailable")
	}
	known, err := a.history.KnownIDs(ctx, "espn")
	if err != nil {
		return fmt.Errorf("live game crosswalk: %w", err)
	}
	players, err := a.spinePlayers(ctx, mine)
	if err != nil {
		return err
	}
	if v.Followed != nil {
		v.Spine, v.Unmatched = matchSpine([]livegame.Game{*v.Followed}, known, players)
	}
	v.WeekSpine, _ = matchSpine(held, known, players)
	return nil
}

// heldWeekGames returns each game whose plays are held, once (a final is held under two ids).
func heldWeekGames(adapter livegame.Adapter) []livegame.Game {
	out := []livegame.Game{}
	seen := map[string]bool{}
	for _, key := range slices.Sorted(maps.Keys(adapter.Details)) {
		d := adapter.Details[key]
		if d.At.IsZero() || len(d.Game.Plays) == 0 || seen[d.Game.ID] {
			continue
		}
		seen[d.Game.ID] = true
		out = append(out, d.Game)
	}
	return out
}

func (a *App) spinePlayers(ctx context.Context, mine string) (map[string]SpinePlayer, error) {
	ids := a.heldMatchupStarters(mine)
	players := map[string]SpinePlayer{}
	espnIDs, err := a.history.SourceIDs(ctx, "espn")
	if err != nil {
		return nil, fmt.Errorf("live game player crosswalk: %w", err)
	}
	a.lookupMu.Lock()
	lookup := a.lookup
	a.lookupMu.Unlock()
	for _, id := range slices.Sorted(maps.Keys(ids)) {
		espn := espnIDs[id]
		if espn == "" {
			continue
		}
		facts, ok := lookup.Facts(id)
		if !ok {
			continue
		}
		players[espn] = SpinePlayer{ID: id, Name: proseName(facts.Name), Team: facts.Team, Mine: ids[id], Photo: photoView(id, proseName(facts.Name), a.photosOff.Load())}
	}
	return players, nil
}

// matchSpine joins plays to players by ESPN id only, newest first: by the play's wall clock when
// both plays carry one, otherwise by the game's own play order.
func matchSpine(games []livegame.Game, known map[string]bool, players map[string]SpinePlayer) ([]SpineEntry, int) {
	out := []SpineEntry{}
	unmatched := 0
	for _, g := range games {
		label := g.Away.Abbreviation + " at " + g.Home.Abbreviation
		for _, p := range g.Plays {
			for _, id := range p.Participants {
				if !known[id] {
					unmatched++
					continue
				}
				if player, ok := players[id]; ok {
					out = append(out, SpineEntry{Player: player, Play: p, Game: label})
				}
			}
		}
	}
	slices.SortStableFunc(out, func(a, b SpineEntry) int {
		if !a.Play.At.IsZero() && !b.Play.At.IsZero() && !a.Play.At.Equal(b.Play.At) {
			return b.Play.At.Compare(a.Play.At)
		}
		if a.Game != b.Game {
			return 0
		}
		return b.Play.Sequence - a.Play.Sequence
	})
	return out, unmatched
}
