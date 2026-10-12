package livegame

import (
	"encoding/json"
	"fmt"
	"slices"
	"time"
)

type RawTeam struct {
	ID           string `json:"id"`
	Abbreviation string `json:"abbreviation"`
}
type RawCompetitor struct {
	ID       string  `json:"id"`
	HomeAway string  `json:"homeAway"`
	Score    string  `json:"score"`
	Team     RawTeam `json:"team"`
}
type RawStatus struct {
	DisplayClock string `json:"displayClock"`
	Period       *int   `json:"period"`
	Type         struct {
		State     string `json:"state"`
		Completed *bool  `json:"completed"`
	} `json:"type"`
}
type RawCompetition struct {
	Date        string          `json:"date"`
	Competitors []RawCompetitor `json:"competitors"`
	Status      RawStatus       `json:"status"`
	Situation   *RawSituation   `json:"situation"`
}
type RawLastPlay struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

func (r RawLastPlay) Validate() error {
	if !digits(r.ID) || r.Text == "" {
		return fmt.Errorf("lastPlay.id/text: missing or invalid")
	}
	return nil
}

type RawSituation struct {
	Possession string       `json:"possession"`
	Down       *int         `json:"down"`
	Distance   *int         `json:"distance"`
	YardLine   *int         `json:"yardLine"`
	LastPlay   *RawLastPlay `json:"lastPlay"`
}
type RawEvent struct {
	ID           string           `json:"id"`
	Competitions []RawCompetition `json:"competitions"`
	Status       *RawStatus       `json:"status"`
}
type RawScoreboard struct {
	Events []RawEvent `json:"events"`
}

func (r RawTeam) Validate() error {
	if !digits(r.ID) || !abbreviation(r.Abbreviation) {
		return fmt.Errorf("team.id/abbreviation: invalid")
	}
	return nil
}
func (r RawStatus) Validate() error {
	if !slices.Contains([]string{"pre", "in", "post"}, r.Type.State) || r.Type.Completed == nil {
		return fmt.Errorf("status.type.state/completed: missing or invalid")
	}
	if *r.Type.Completed != (r.Type.State == "post") {
		return fmt.Errorf("status.type.completed: contradicts state")
	}
	if r.DisplayClock != "" && !clockOK(r.DisplayClock) {
		return fmt.Errorf("status.displayClock: invalid")
	}
	if r.Period != nil && (*r.Period < 0 || *r.Period > 10) {
		return fmt.Errorf("status.period: invalid")
	}
	if r.Type.State == "in" && (r.Period == nil || r.DisplayClock == "") {
		return fmt.Errorf("status.period/displayClock: required in progress")
	}
	return nil
}
func (r RawEvent) Validate() error {
	if !digits(r.ID) || len(r.Competitions) != 1 {
		return fmt.Errorf("event.id/competitions: invalid")
	}
	c := r.Competitions[0]
	if _, err := time.Parse("2006-01-02T15:04Z", c.Date); err != nil {
		if _, e := time.Parse(time.RFC3339, c.Date); e != nil {
			return fmt.Errorf("competition.date: %w", err)
		}
	}
	if len(c.Competitors) != 2 {
		return fmt.Errorf("competitors: requires home and away")
	}
	s := c.Status
	if r.Status != nil {
		s = *r.Status
	}
	if err := validateCompetitors(c.Competitors, s.Type.State); err != nil {
		return err
	}
	if err := s.Validate(); err != nil {
		return err
	}
	if c.Situation != nil {
		return c.Situation.Validate(c.Competitors)
	}
	return nil
}
func validateCompetitors(teams []RawCompetitor, state string) error {
	seen := map[string]bool{}
	for _, t := range teams {
		if err := t.Team.Validate(); err != nil {
			return err
		}
		if t.ID != t.Team.ID || seen[t.HomeAway] || !slices.Contains([]string{"home", "away"}, t.HomeAway) {
			return fmt.Errorf("competitor.id/homeAway: invalid")
		}
		seen[t.HomeAway] = true
		if t.Score == "" && state == "pre" {
			continue
		}
		if _, err := number("competitor.score", t.Score, 200); err != nil {
			return err
		}
	}
	return nil
}
func (r RawSituation) Validate(teams []RawCompetitor) error {
	if !slices.ContainsFunc(teams, func(t RawCompetitor) bool { return t.ID == r.Possession }) {
		return fmt.Errorf("situation.possession: unknown team")
	}
	if r.Down == nil || r.Distance == nil || r.YardLine == nil || *r.Down < 0 || *r.Down > 4 || *r.Distance < 0 || *r.Distance > 100 || *r.YardLine < 0 || *r.YardLine > 100 {
		return fmt.Errorf("situation.down/distance/yardLine: missing or invalid")
	}
	if r.LastPlay != nil {
		return r.LastPlay.Validate()
	}
	return nil
}
func (r RawScoreboard) Validate() error {
	if len(r.Events) == 0 {
		return fmt.Errorf("events: missing")
	}
	seen := map[string]bool{}
	for i, e := range r.Events {
		if seen[e.ID] {
			return fmt.Errorf("events[%d].id: duplicate", i)
		}
		seen[e.ID] = true
		if err := e.Validate(); err != nil {
			return fmt.Errorf("events[%d]: %w", i, err)
		}
	}
	return nil
}
func ParseScoreboard(body []byte) ([]Game, error) {
	var r RawScoreboard
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("ESPN scoreboard decode: %w", err)
	}
	if err := r.Validate(); err != nil {
		return nil, err
	}
	out := make([]Game, 0, len(r.Events))
	for _, e := range r.Events {
		out = append(out, e.game())
	}
	return out, nil
}
func (r RawEvent) game() Game {
	c := r.Competitions[0]
	s := c.Status
	if r.Status != nil {
		s = *r.Status
	}
	kickoff, err := time.Parse(time.RFC3339, c.Date)
	if err != nil {
		kickoff, _ = time.Parse("2006-01-02T15:04Z", c.Date)
	}
	g := Game{ID: r.ID, Kickoff: kickoff, Status: s.Type.State, Clock: s.DisplayClock, Box: []Box{}, Plays: []Play{}}
	if s.Period != nil {
		g.Period = *s.Period
	}
	for _, t := range c.Competitors {
		score, _ := number("score", t.Score, 200)
		v := Team{ID: t.ID, Abbreviation: t.Team.Abbreviation, Score: score}
		if t.HomeAway == "home" {
			g.Home = v
		} else {
			g.Away = v
		}
	}
	if c.Situation != nil {
		v := c.Situation
		g.State = &State{Possession: v.Possession, Down: *v.Down, Distance: *v.Distance, YardLine: *v.YardLine, ToEndzone: -1}
		// The scoreboard's yardLine has no reliable direction; drive state supplies the field.
		if v.LastPlay != nil {
			p := Play{ID: v.LastPlay.ID, Text: v.LastPlay.Text, Clock: g.Clock, Period: g.Period, HomeScore: g.Home.Score, AwayScore: g.Away.Score, Participants: []string{}}
			g.Last = &p
		}
	}
	return g
}
