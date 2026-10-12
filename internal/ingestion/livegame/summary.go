package livegame

import (
	"encoding/json"
	"fmt"
	"slices"
)

type RawBox struct {
	Team       RawTeam `json:"team"`
	Statistics []struct {
		Name     string   `json:"name"`
		Labels   []string `json:"labels"`
		Athletes []struct {
			Athlete struct {
				ID          string `json:"id"`
				DisplayName string `json:"displayName"`
			} `json:"athlete"`
			Stats []string `json:"stats"`
		} `json:"athletes"`
	} `json:"statistics"`
}
type RawSummary struct {
	Header   RawEvent `json:"header"`
	Boxscore struct {
		Players []RawBox `json:"players"`
	} `json:"boxscore"`
	Drives struct {
		Previous []struct {
			Plays []RawPlay `json:"plays"`
		} `json:"previous"`
		Current *struct {
			Plays []RawPlay `json:"plays"`
		} `json:"current"`
	} `json:"drives"`
}

func boxCategory(s string) bool {
	return slices.Contains([]string{"passing", "rushing", "receiving", "defensive"}, s)
}
func (r RawBox) Validate() error {
	if err := r.Team.Validate(); err != nil {
		return err
	}
	for i, s := range r.Statistics {
		if s.Name == "" {
			return fmt.Errorf("statistics[%d].name: missing", i)
		}
		if !boxCategory(s.Name) {
			continue
		}
		if len(s.Labels) == 0 {
			return fmt.Errorf("statistics[%d].labels: missing", i)
		}
		for _, label := range s.Labels {
			if label == "" {
				return fmt.Errorf("statistics[%d].labels: empty", i)
			}
		}
		for j, a := range s.Athletes {
			if !digits(a.Athlete.ID) || a.Athlete.DisplayName == "" || len(a.Stats) != len(s.Labels) {
				return fmt.Errorf("statistics[%d].athletes[%d].id/displayName/stats: invalid", i, j)
			}
			for _, v := range a.Stats {
				if v == "" {
					return fmt.Errorf("statistics[%d].athletes[%d].stats: empty", i, j)
				}
			}
		}
	}
	return nil
}
func (r RawSummary) rawPlays() []RawPlay {
	out := []RawPlay{}
	for _, d := range r.Drives.Previous {
		out = append(out, d.Plays...)
	}
	if r.Drives.Current != nil {
		out = append(out, r.Drives.Current.Plays...)
	}
	return out
}
func (r RawSummary) Validate() error {
	if err := r.Header.Validate(); err != nil {
		return fmt.Errorf("header: %w", err)
	}
	if r.Header.game().Status != "pre" {
		if len(r.Boxscore.Players) == 0 {
			return fmt.Errorf("boxscore.players: missing")
		}
		if len(r.rawPlays()) == 0 {
			return fmt.Errorf("drives.plays: missing")
		}
	}
	for i, b := range r.Boxscore.Players {
		if err := b.Validate(); err != nil {
			return fmt.Errorf("boxscore.players[%d]: %w", i, err)
		}
	}
	return validatePlays(r.rawPlays())
}
func ParseSummary(body []byte) (Game, error) {
	var r RawSummary
	if err := json.Unmarshal(body, &r); err != nil {
		return Game{}, fmt.Errorf("ESPN summary decode: %w", err)
	}
	if err := r.Validate(); err != nil {
		return Game{}, err
	}
	g := r.Header.game()
	if g.Status != "pre" {
		for _, b := range r.Boxscore.Players {
			for _, s := range b.Statistics {
				if !boxCategory(s.Name) {
					continue
				}
				box := Box{Team: b.Team.Abbreviation, Category: s.Name, Labels: s.Labels, Rows: []BoxRow{}}
				for _, a := range s.Athletes {
					box.Rows = append(box.Rows, BoxRow{Name: a.Athlete.DisplayName, Stats: a.Stats})
				}
				g.Box = append(g.Box, box)
			}
		}
	}
	for _, p := range r.rawPlays() {
		g.Plays = append(g.Plays, p.play())
	}
	slices.SortFunc(g.Plays, func(a, b Play) int { return a.Sequence - b.Sequence })
	if len(g.Plays) > 0 {
		p := g.Plays[len(g.Plays)-1]
		g.Last = &p
		g.State = p.End
		g.Clock = p.Clock
		g.Period = p.Period
	}
	return g, nil
}
