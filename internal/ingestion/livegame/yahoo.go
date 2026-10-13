package livegame

import (
	"encoding/json"
	"fmt"
	"slices"
	"time"
)

type RawYahooGame struct {
	ID        string  `json:"gameid"`
	Start     string  `json:"start_time"`
	Status    string  `json:"status_type"`
	HomeID    string  `json:"home_team_id"`
	AwayID    string  `json:"away_team_id"`
	HomeScore string  `json:"total_home_points"`
	AwayScore string  `json:"total_away_points"`
	Period    string  `json:"current_period_id"`
	Clock     *string `json:"game_time_elapsed_display"`
}
type RawYahooPlay struct {
	ID     string `json:"play_id"`
	Text   string `json:"play_text"`
	Period string `json:"period"`
	Clock  string `json:"clock"`
}
type RawYahooBoard struct {
	Games map[string]RawYahooGame `json:"games"`
	Teams map[string]struct {
		Abbreviation string `json:"abbr"`
	} `json:"teams"`
	Last map[string]*RawYahooPlay `json:"gamelast_play"`
}
type RawYahoo struct {
	Service struct {
		Scoreboard RawYahooBoard `json:"scoreboard"`
	} `json:"service"`
}

func yahooState(s string) string {
	switch s {
	case "status.type.final":
		return "post"
	case "status.type.scheduled", "status.type.pregame":
		return "pre"
	// Yahoo sent status.type.in_progress for BUF at LAR on 2026-10-12; no live game was captured before.
	case "status.type.in_progress", "status.type.inprogress":
		return "in"
	default:
		return ""
	}
}
func (r RawYahooGame) Validate() error {
	if r.ID == "" || r.HomeID == "" || r.AwayID == "" || r.HomeID == r.AwayID || yahooState(r.Status) == "" {
		return fmt.Errorf("gameid/home_team_id/away_team_id/status_type: invalid")
	}
	if _, err := time.Parse(time.RFC1123Z, r.Start); err != nil {
		return fmt.Errorf("start_time: %w", err)
	}
	for _, v := range []struct {
		field, raw string
		upper      int
	}{{"total_home_points", r.HomeScore, 200}, {"total_away_points", r.AwayScore, 200}, {"current_period_id", r.Period, 10}} {
		if v.raw == "" && yahooState(r.Status) == "pre" {
			continue
		}
		if _, err := number(v.field, v.raw, v.upper); err != nil {
			return err
		}
	}
	if r.Clock != nil && *r.Clock != "" && !clockOK(*r.Clock) {
		return fmt.Errorf("game_time_elapsed_display: invalid")
	}
	return nil
}
func (r RawYahooPlay) Validate() error {
	if !digits(r.ID) || r.Text == "" || !clockOK(r.Clock) {
		return fmt.Errorf("play_id/play_text/clock: invalid")
	}
	if _, err := number("period", r.Period, 10); err != nil {
		return err
	}
	return nil
}
func (r RawYahoo) Validate() error {
	b := r.Service.Scoreboard
	if len(b.Games) == 0 || b.Teams == nil || b.Last == nil {
		return fmt.Errorf("service.scoreboard.games/teams/gamelast_play: missing")
	}
	for id, g := range b.Games {
		if id != g.ID {
			return fmt.Errorf("games[%s].gameid: key mismatch", id)
		}
		if err := g.Validate(); err != nil {
			return fmt.Errorf("games[%s]: %w", id, err)
		}
		for _, t := range []string{g.HomeID, g.AwayID} {
			if !abbreviation(b.Teams[t].Abbreviation) {
				return fmt.Errorf("teams[%s].abbr: missing or invalid", t)
			}
		}
		if p := b.Last[id]; p != nil {
			if err := p.Validate(); err != nil {
				return fmt.Errorf("gamelast_play[%s]: %w", id, err)
			}
		}
	}
	return nil
}
func ParseYahoo(body []byte) ([]Game, error) {
	var r RawYahoo
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("yahoo decode: %w", err)
	}
	if err := r.Validate(); err != nil {
		return nil, err
	}
	b := r.Service.Scoreboard
	out := make([]Game, 0, len(b.Games))
	for _, v := range b.Games {
		at, _ := time.Parse(time.RFC1123Z, v.Start)
		home, _ := number("home", v.HomeScore, 200)
		away, _ := number("away", v.AwayScore, 200)
		period, _ := number("period", v.Period, 10)
		g := Game{ID: v.ID, Kickoff: at, Home: Team{ID: v.HomeID, Abbreviation: b.Teams[v.HomeID].Abbreviation, Score: home}, Away: Team{ID: v.AwayID, Abbreviation: b.Teams[v.AwayID].Abbreviation, Score: away}, Status: yahooState(v.Status), Period: period, Plays: []Play{}, Box: []Box{}}
		if v.Clock != nil {
			g.Clock = *v.Clock
		}
		if p := b.Last[v.ID]; p != nil {
			n, _ := number("period", p.Period, 10)
			g.Last = &Play{ID: p.ID, Text: p.Text, Period: n, Clock: p.Clock, HomeScore: home, AwayScore: away, Participants: []string{}}
		}
		out = append(out, g)
	}
	slices.SortFunc(out, func(a, b Game) int { return a.Kickoff.Compare(b.Kickoff) })
	return out, nil
}
