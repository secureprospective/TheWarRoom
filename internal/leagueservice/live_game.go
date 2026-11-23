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

// NFLGamePregame is cached public pre-game win chance, venue and weather for one NFL game.
type NFLGamePregame struct {
	League    string              `json:"league"`
	Game      string              `json:"game"`
	At        time.Time           `json:"at"`
	Predictor *livegame.Predictor `json:"predictor,omitempty"`
	Venue     *livegame.Venue     `json:"venue,omitempty"`
	Weather   *livegame.Weather   `json:"weather,omitempty"`
}

// SetDetails atomically replaces the current slate's details with detached parsed summaries.
func (g *LiveGames) SetDetails(details map[string]NFLGame) {
	if g == nil {
		return
	}
	g.detailMu.Lock()
	defer g.detailMu.Unlock()
	held := make(map[string]NFLGame, len(details))
	for id, detail := range details {
		detail.Game = cloneLiveGame(detail.Game)
		held[id] = detail
	}
	g.details.Store(&held)
}

// MergePregame updates held details for pre-game entries without touching live or final games.
// It reports whether any pre-game entry was updated.
func (g *LiveGames) MergePregame(updates map[string]NFLGame) bool {
	if g == nil || len(updates) == 0 {
		return false
	}
	g.detailMu.Lock()
	defer g.detailMu.Unlock()
	next := make(map[string]NFLGame, len(updates))
	if current := g.details.Load(); current != nil {
		for id, detail := range *current {
			next[id] = detail
		}
	}
	board := g.held.Load()
	changed := false
	for id, detail := range updates {
		if detail.Game == nil || detail.Game.Status != "pre" || !canMergePregame(board, next, id) {
			continue
		}
		detail.Game = cloneLiveGame(detail.Game)
		next[id] = detail
		changed = true
	}
	if changed {
		g.details.Store(&next)
	}
	return changed
}

// PregameSnapshot returns held pre-game predictor, venue and weather sorted by game id for
// change fingerprinting. Timestamps are omitted so only pre-game data changes alter the digest.
func (g *LiveGames) PregameSnapshot() []NFLGamePregame {
	if g == nil {
		return nil
	}
	held := g.details.Load()
	if held == nil || len(*held) == 0 {
		return nil
	}
	out := make([]NFLGamePregame, 0, len(*held))
	for id, detail := range *held {
		if detail.Game == nil || detail.Game.Status != "pre" {
			continue
		}
		if detail.Game.Predictor == nil && detail.Game.Venue == nil && detail.Game.Weather == nil {
			continue
		}
		out = append(out, NFLGamePregame{
			Game:      id,
			Predictor: clonePredictor(detail.Game.Predictor),
			Venue:     cloneVenue(detail.Game.Venue),
			Weather:   cloneWeather(detail.Game.Weather),
		})
	}
	if len(out) == 0 {
		return nil
	}
	slices.SortFunc(out, func(a, b NFLGamePregame) int {
		return strings.Compare(a.Game, b.Game)
	})
	return out
}

func canMergePregame(board *livegame.Held, current map[string]NFLGame, id string) bool {
	if existing, ok := current[id]; ok && existing.Game != nil && existing.Game.Status != "pre" {
		return false
	}
	if board == nil {
		return true
	}
	for _, game := range board.Games {
		if game.ID == id {
			return game.Status == "pre"
		}
	}
	return false
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
	out.Predictor = clonePredictor(game.Predictor)
	out.Venue = cloneVenue(game.Venue)
	out.Weather = cloneWeather(game.Weather)
	return &out
}

func clonePredictor(src *livegame.Predictor) *livegame.Predictor {
	if src == nil {
		return nil
	}
	out := *src
	return &out
}

func cloneVenue(src *livegame.Venue) *livegame.Venue {
	if src == nil {
		return nil
	}
	out := *src
	return &out
}

func cloneWeather(src *livegame.Weather) *livegame.Weather {
	if src == nil {
		return nil
	}
	out := *src
	if src.TempF != nil {
		t := *src.TempF
		out.TempF = &t
	}
	if src.PrecipPct != nil {
		pct := *src.PrecipPct
		out.PrecipPct = &pct
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

func validNFLGameScope(league, game string) bool {
	return league != "" && game != "" && len(game) <= 32 && strings.Trim(game, "0123456789") == ""
}

func (a *Service) TargetNFLGame(league, game string) (NFLGame, error) {
	if !validNFLGameScope(league, game) {
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
				if detail.Game != nil {
					detail.Game.Predictor = nil
					detail.Game.Venue = nil
					detail.Game.Weather = nil
				}
				detail.Stale = a.games.now().Sub(detail.At) > liveGamesStale
			}
		}
	}
	detail.League = league
	return detail, nil
}

// TargetNFLGamePregame reads cached public pre-game preview data without performing network I/O.
func (a *Service) TargetNFLGamePregame(league, game string) (NFLGamePregame, error) {
	if !validNFLGameScope(league, game) {
		return NFLGamePregame{}, fmt.Errorf("nfl game pregame: league and numeric game id are required")
	}
	if err := a.admitRead(league, "native.directory.read", ""); err != nil {
		return NFLGamePregame{}, err
	}
	out := NFLGamePregame{League: league, Game: game}
	if a.games != nil {
		if held := a.games.details.Load(); held != nil {
			if found, ok := (*held)[game]; ok && found.Game != nil && found.Game.Status == "pre" {
				out.At = found.At
				out.Predictor = clonePredictor(found.Game.Predictor)
				out.Venue = cloneVenue(found.Game.Venue)
				out.Weather = cloneWeather(found.Game.Weather)
			}
		}
	}
	return out, nil
}
