package leagueservice

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
)

// NFLGame is public NFL detail; Words explains a game the collector has not held yet.
type NFLGame struct {
	League string         `json:"league"`
	Game   *livegame.Game `json:"game"`
	At     time.Time      `json:"at"`
	Stale  bool           `json:"stale"`
	Words  string         `json:"words"`
}

// SetDetails atomically replaces the current slate's details with detached parsed summaries.
func (g *LiveGames) SetDetails(details map[string]NFLGame) {
	if g == nil {
		return
	}
	held := make(map[string]NFLGame, len(details))
	for id, detail := range details {
		detail.Game = cloneLiveGame(detail.Game)
		held[id] = detail
	}
	g.details.Store(&held)
}

func cloneLiveGame(game *livegame.Game) *livegame.Game {
	if game == nil {
		return nil
	}
	out := *game
	if game.State != nil {
		state := *game.State
		out.State = &state
	}
	out.Plays = make([]livegame.Play, len(game.Plays))
	for i, play := range game.Plays {
		out.Plays[i] = cloneLivePlay(play)
	}
	if game.Last != nil {
		last := cloneLivePlay(*game.Last)
		out.Last = &last
	}
	out.Box = make([]livegame.Box, len(game.Box))
	for i, box := range game.Box {
		box.Labels = slices.Clone(box.Labels)
		box.Rows = slices.Clone(box.Rows)
		for j := range box.Rows {
			box.Rows[j].Stats = slices.Clone(box.Rows[j].Stats)
		}
		out.Box[i] = box
	}
	if game.Predictor != nil {
		p := *game.Predictor
		out.Predictor = &p
	}
	if game.Venue != nil {
		v := *game.Venue
		out.Venue = &v
	}
	if game.Weather != nil {
		w := *game.Weather
		if game.Weather.TempF != nil {
			t := *game.Weather.TempF
			w.TempF = &t
		}
		if game.Weather.PrecipPct != nil {
			pct := *game.Weather.PrecipPct
			w.PrecipPct = &pct
		}
		out.Weather = &w
	}
	return &out
}

func cloneLivePlay(play livegame.Play) livegame.Play {
	play.Participants = append([]string{}, play.Participants...)
	if play.Start != nil {
		state := *play.Start
		play.Start = &state
	}
	if play.End != nil {
		state := *play.End
		play.End = &state
	}
	return play
}

func (a *Service) TargetNFLGame(league, game string) (NFLGame, error) {
	if league == "" || game == "" || len(game) > 32 || strings.Trim(game, "0123456789") != "" {
		return NFLGame{}, fmt.Errorf("nfl game: league and numeric game id are required")
	}
	if err := a.admitRead(league, "native.directory.read", ""); err != nil {
		return NFLGame{}, err
	}
	detail := NFLGame{Words: "No live detail is held for this game yet."}
	if a.games != nil {
		if held := a.games.details.Load(); held != nil {
			if found, ok := (*held)[game]; ok {
				detail = found
				detail.Game = cloneLiveGame(found.Game)
				detail.Stale = a.games.now().Sub(detail.At) > liveGamesStale
			}
		}
	}
	detail.League = league
	return detail, nil
}
