package livegame

import (
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
)

type RawFlexFloat struct {
	Value float64
	Set   bool
}

func (f *RawFlexFloat) UnmarshalJSON(body []byte) error {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" || trimmed == "null" || trimmed == `""` {
		return nil
	}
	var s string
	if err := json.Unmarshal(body, &s); err == nil {
		s = strings.TrimSpace(s)
		if s == "" {
			return nil
		}
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return fmt.Errorf("float string: %w", err)
		}
		f.Value, f.Set = v, true
		return nil
	}
	var v float64
	if err := json.Unmarshal(body, &v); err != nil {
		return fmt.Errorf("float value: %w", err)
	}
	f.Value, f.Set = v, true
	return nil
}

type RawPredictorSide struct {
	ID             string       `json:"id"`
	GameProjection RawFlexFloat `json:"gameProjection"`
}
type RawPredictor struct {
	HomeTeam *RawPredictorSide `json:"homeTeam"`
	AwayTeam *RawPredictorSide `json:"awayTeam"`
}

func (r RawPredictor) Validate() error {
	if r.HomeTeam == nil || r.AwayTeam == nil {
		return fmt.Errorf("predictor: missing homeTeam or awayTeam")
	}
	if !r.HomeTeam.GameProjection.Set || !r.AwayTeam.GameProjection.Set {
		return fmt.Errorf("predictor: missing gameProjection")
	}
	h, a := r.HomeTeam.GameProjection.Value, r.AwayTeam.GameProjection.Value
	if math.IsNaN(h) || math.IsInf(h, 0) || math.IsNaN(a) || math.IsInf(a, 0) {
		return fmt.Errorf("predictor: non-finite projection")
	}
	if h < 0 || h > 100 || a < 0 || a > 100 || h+a <= 0 {
		return fmt.Errorf("predictor: projection out of range")
	}
	return nil
}

func (r *RawPredictor) toPredictor(homeID, awayID string) *Predictor {
	if r == nil || r.Validate() != nil {
		return nil
	}
	if r.HomeTeam.ID != "" && homeID != "" && r.HomeTeam.ID != homeID {
		return nil
	}
	if r.AwayTeam.ID != "" && awayID != "" && r.AwayTeam.ID != awayID {
		return nil
	}
	return &Predictor{
		Home: r.HomeTeam.GameProjection.Value,
		Away: r.AwayTeam.GameProjection.Value,
	}
}

type RawVenueAddress struct {
	City    string `json:"city"`
	State   string `json:"state"`
	Country string `json:"country"`
}
type RawVenue struct {
	FullName string          `json:"fullName"`
	Address  RawVenueAddress `json:"address"`
	Indoor   bool            `json:"indoor"`
}

func (r RawVenue) Validate() error {
	if strings.TrimSpace(r.FullName) == "" && strings.TrimSpace(r.Address.City) == "" {
		return fmt.Errorf("venue: missing name and city")
	}
	return nil
}

func (r *RawVenue) toVenue() *Venue {
	if r == nil || r.Validate() != nil {
		return nil
	}
	return &Venue{
		Name:    strings.TrimSpace(r.FullName),
		City:    strings.TrimSpace(r.Address.City),
		State:   strings.TrimSpace(r.Address.State),
		Country: strings.TrimSpace(r.Address.Country),
		Indoor:  r.Indoor,
	}
}

type RawGameInfo struct {
	Venue *RawVenue `json:"venue"`
}

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
		Previous []RawDrive `json:"previous"`
		Current  *RawDrive  `json:"current"`
	} `json:"drives"`
	Predictor *RawPredictor `json:"predictor"`
	GameInfo  *RawGameInfo  `json:"gameInfo"`
}

func boxCategory(s string) bool {
	return slices.Contains([]string{"passing", "rushing", "receiving", "defensive", "fumbles", "interceptions", "kickReturns", "puntReturns", "kicking", "punting"}, s)
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

// RawDrive is one ESPN drive and its plays.
type RawDrive struct {
	ID    string    `json:"id"`
	Plays []RawPlay `json:"plays"`
}

// rawPlays lists every play once. During a game ESPN also lists the drive in progress as the last
// previous drive, under the same id (seen live on 2026-10-12), so the current drive counts only
// when it is not already among the previous ones.
func (r RawSummary) rawPlays() []RawPlay {
	out := []RawPlay{}
	for _, d := range r.Drives.Previous {
		out = append(out, d.Plays...)
	}
	cur := r.Drives.Current
	if cur != nil && !slices.ContainsFunc(r.Drives.Previous, func(d RawDrive) bool { return cur.ID != "" && d.ID == cur.ID }) {
		out = append(out, cur.Plays...)
	}
	return out
}
func (r RawSummary) Validate() error {
	if err := r.Header.Validate(); err != nil {
		return fmt.Errorf("header: %w", err)
	}
	if g := r.Header.game(); g.Status != "pre" {
		if len(r.Boxscore.Players) == 0 {
			return fmt.Errorf("boxscore.players: missing")
		}
		// ESPN marks a game in progress about a minute before it sends the first play; until then
		// a scoreless first quarter with no drives is the kickoff, not a broken summary.
		kickoff := g.Status == "in" && g.Period == 1 && g.Home.Score == 0 && g.Away.Score == 0
		if len(r.rawPlays()) == 0 && !kickoff {
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
	g.Predictor = r.Predictor.toPredictor(g.Home.ID, g.Away.ID)
	if r.GameInfo != nil {
		g.Venue = r.GameInfo.Venue.toVenue()
		if g.Venue != nil && g.Venue.Indoor {
			g.Weather = &Weather{Text: "Indoors"}
		}
	}
	if g.Status != "pre" {
		for _, b := range r.Boxscore.Players {
			for _, s := range b.Statistics {
				if !boxCategory(s.Name) {
					continue
				}
				box := Box{Team: b.Team.Abbreviation, Category: s.Name, Labels: s.Labels, Rows: []BoxRow{}}
				for _, a := range s.Athletes {
					box.Rows = append(box.Rows, BoxRow{ID: a.Athlete.ID, Name: a.Athlete.DisplayName, Stats: a.Stats})
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
