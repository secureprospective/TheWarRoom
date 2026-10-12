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
func (a *App) liveGameSpine(ctx context.Context, g *livegame.Game, mine string) ([]SpineEntry, int, error) {
	if g == nil || len(g.Plays) == 0 {
		return []SpineEntry{}, 0, nil
	}
	if a.history == nil {
		return []SpineEntry{}, 0, fmt.Errorf("crosswalk directory unavailable")
	}
	known, err := a.history.KnownIDs(ctx, "espn")
	if err != nil {
		return nil, 0, fmt.Errorf("live game crosswalk: %w", err)
	}
	ids := a.heldMatchupStarters(mine)
	players := map[string]SpinePlayer{}
	a.lookupMu.Lock()
	lookup := a.lookup
	a.lookupMu.Unlock()
	for _, id := range slices.Sorted(maps.Keys(ids)) {
		espn, e := a.history.SourceID(ctx, id, "espn")
		if e != nil {
			return nil, 0, fmt.Errorf("live game player crosswalk: %w", e)
		}
		if espn == "" {
			continue
		}
		facts, ok := lookup.Facts(id)
		if !ok {
			continue
		}
		players[espn] = SpinePlayer{ID: id, Name: proseName(facts.Name), Team: facts.Team, Mine: ids[id], Photo: photoView(id, proseName(facts.Name), a.photosOff.Load())}
	}
	entries, unmatched := matchSpine(g.Plays, known, players)
	return entries, unmatched, nil
}
func matchSpine(plays []livegame.Play, known map[string]bool, players map[string]SpinePlayer) ([]SpineEntry, int) {
	out := []SpineEntry{}
	unmatched := 0
	for _, p := range plays {
		for _, id := range p.Participants {
			if !known[id] {
				unmatched++
				continue
			}
			if player, ok := players[id]; ok {
				out = append(out, SpineEntry{Player: player, Play: p})
			}
		}
	}
	slices.SortStableFunc(out, func(a, b SpineEntry) int { return b.Play.Sequence - a.Play.Sequence })
	return out, unmatched
}
