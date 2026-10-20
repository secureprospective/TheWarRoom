package main

import (
	"context"
	"fmt"
	"maps"
	"slices"

	nowapp "github.com/secureprospective/TheWarRoom/internal/app/now"

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

func (a *App) heldMatchupStarters(mine string) map[string]bool {
	week := a.marketWeek()
	a.seasonMu.Lock()
	defer a.seasonMu.Unlock()
	if a.seasonLineups.value.Week == week {
		if ids := nowapp.MatchupStarterIDs(a.seasonLineups.value, mine); len(ids) > 0 {
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
	held := nowapp.HeldWeekGames(nowapp.LiveSnapshot{Details: maps.Clone(adapter.Details)})
	v.WeekHeld = len(held)
	if v.Followed == nil && len(held) == 0 {
		return nil
	}
	if a.history == nil {
		return fmt.Errorf("crosswalk directory unavailable")
	}
	known, err := a.history.KnownIDs(ctx, "espn")
	if err != nil {
		return fmt.Errorf("live game crosswalk: %w", err)
	}
	if v.Followed != nil {
		if err := a.inspectLiveBox(ctx, v); err != nil {
			return err
		}
	}
	players, err := a.spinePlayers(ctx, mine)
	if err != nil {
		return err
	}
	if v.Followed != nil {
		entries, unmatched := nowapp.MatchSpine([]livegame.Game{*v.Followed}, known, serviceSpinePlayers(players))
		v.Spine, v.Unmatched = wireSpines(entries), unmatched
	}
	entries, _ := nowapp.MatchSpine(held, known, serviceSpinePlayers(players))
	v.WeekSpine = wireSpines(entries)
	return nil
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

func (a *App) inspectLiveBox(ctx context.Context, v *LiveGameView) error {
	ids, err := a.history.SourceIDs(ctx, "espn")
	if err != nil {
		return fmt.Errorf("box score crosswalk: %w", err)
	}
	bySource := map[string]string{}
	for id, source := range ids {
		bySource[source] = id
	}
	entities := map[string]ReportEntity{}
	for _, box := range v.Followed.Box {
		for _, p := range box.Rows {
			if id := bySource[p.ID]; id != "" {
				entities[id] = ReportEntity{Text: p.Name, Kind: "player", ID: id}
			}
		}
	}
	attachReportEntities(&v.Box, slices.Collect(maps.Values(entities)))
	return nil
}
func wireSpines(entries []nowapp.SpineEntry) []SpineEntry {
	if entries == nil {
		return nil
	}
	out := make([]SpineEntry, len(entries))
	for i, e := range entries {
		out[i] = SpineEntry{Player: SpinePlayer{ID: e.Player.ID, Name: e.Player.Name, Team: e.Player.Team, Mine: e.Player.Mine, Photo: PhotoView(e.Player.Photo)}, Play: e.Play, Game: e.Game}
	}
	return out
}
func serviceSpinePlayers(players map[string]SpinePlayer) map[string]nowapp.SpinePlayer {
	out := make(map[string]nowapp.SpinePlayer, len(players))
	for k, p := range players {
		out[k] = nowapp.SpinePlayer{ID: p.ID, Name: p.Name, Team: p.Team, Mine: p.Mine, Photo: nowapp.PhotoView(p.Photo)}
	}
	return out
}
func wireLive(v nowapp.LiveGameView) LiveGameView {
	return LiveGameView{Games: v.Games, Followed: v.Followed, Source: v.Source, AsOf: v.AsOf, Failure: v.Failure, DetailAsOf: v.DetailAsOf, DetailFailure: v.DetailFailure, Scoring: v.Scoring, Spine: wireSpines(v.Spine), Unmatched: v.Unmatched, WeekSpine: wireSpines(v.WeekSpine), WeekHeld: v.WeekHeld, WeekGames: v.WeekGames, Box: wireReport(v.Box), Active: v.Active}
}
