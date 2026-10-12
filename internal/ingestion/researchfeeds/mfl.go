// Package researchfeeds validates the display-only research exports.
package researchfeeds

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

type RawPlayer struct {
	ID      string `json:"id"`
	Percent string `json:"percent"`
	Score   string `json:"score"`
	Week    string `json:"week"`
}
type RawPosition struct {
	Name   string `json:"name"`
	Points string `json:"points"`
}
type RawTeam struct {
	ID        string                         `json:"id"`
	Positions ingestion.MFLList[RawPosition] `json:"position"`
}
type RawFeed struct {
	Players ingestion.MFLList[RawPlayer] `json:"player"`
	Scores  ingestion.MFLList[RawPlayer] `json:"playerScore"`
	Teams   ingestion.MFLList[RawTeam]   `json:"team"`
}
type Entry struct {
	ID    playerid.PlayerID
	Value float64
}
type Allowed struct {
	Team, Position string
	Points         float64
}
type Feed struct {
	Players []Entry
	Allowed []Allowed
}

func finite(s string) (float64, error) {
	n, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(n) || math.IsInf(n, 0) {
		return 0, fmt.Errorf("invalid finite number %q", s)
	}
	return n, nil
}
func (r RawPlayer) Validate(kind, week string) error {
	if kind == "playerScores" && r.ID == "" && r.Score == "" && r.Week == week {
		return nil
	}
	if _, err := playerid.New(r.ID); err != nil {
		return fmt.Errorf("research player: %w", err)
	}
	value := r.Percent
	if kind == "playerScores" {
		value = r.Score
		if r.Week != week {
			return fmt.Errorf("playerScores: week %q, want %q", r.Week, week)
		}
	}
	n, err := finite(value)
	if err != nil {
		return err
	}
	if kind != "playerScores" && (n < 0 || n > 100) {
		return fmt.Errorf("research: percentage outside 0–100")
	}
	return nil
}
func (r RawFeed) Validate(kind, week string) error {
	if kind == "pointsAllowed" {
		return r.validateAllowed()
	}
	if kind != "playerScores" && !IsTrend(kind) {
		return fmt.Errorf("research: invalid export %q", kind)
	}
	rows := r.Players
	if kind == "playerScores" {
		rows = r.Scores
	}
	seen := map[string]bool{}
	for _, p := range rows {
		if err := p.Validate(kind, week); err != nil {
			return err
		}
		if seen[p.ID] {
			return fmt.Errorf("research: duplicate player %s", p.ID)
		}
		seen[p.ID] = true
	}
	return nil
}
func (r RawFeed) validateAllowed() error {
	seen := map[string]bool{}
	for _, t := range r.Teams {
		if t.ID == "" || seen[t.ID] {
			return fmt.Errorf("pointsAllowed: missing or duplicate team")
		}
		seen[t.ID] = true
		positions := map[string]bool{}
		for _, p := range t.Positions {
			if p.Name == "" || positions[p.Name] {
				return fmt.Errorf("pointsAllowed: missing or duplicate position")
			}
			positions[p.Name] = true
			if _, err := finite(p.Points); err != nil {
				return err
			}
		}
	}
	return nil
}
func IsTrend(kind string) bool {
	switch kind {
	case "topAdds", "topDrops", "topStarters", "topOwns", "topTrades":
		return true
	}
	return false
}
func Parse(body []byte, kind, week string) (Feed, error) {
	if err := ingestion.CheckAPIError(body); err != nil {
		return Feed{}, fmt.Errorf("%s: %w", kind, err)
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		return Feed{}, fmt.Errorf("research envelope: %w", err)
	}
	section, ok := envelope[kind]
	if !ok {
		return Feed{}, fmt.Errorf("research: missing %s section", kind)
	}
	var raw RawFeed
	if err := json.Unmarshal(section, &raw); err != nil {
		return Feed{}, fmt.Errorf("%s decode: %w", kind, err)
	}
	if err := raw.Validate(kind, week); err != nil {
		return Feed{}, err
	}
	out := Feed{}
	rows := raw.Players
	if kind == "playerScores" {
		rows = raw.Scores
	}
	for _, p := range rows {
		if p.ID == "" {
			continue
		}
		id, err := playerid.New(p.ID)
		if err != nil {
			return Feed{}, fmt.Errorf("research id: %w", err)
		}
		value := p.Percent
		if kind == "playerScores" {
			value = p.Score
		}
		n, err := finite(value)
		if err != nil {
			return Feed{}, err
		}
		out.Players = append(out.Players, Entry{ID: id, Value: n})
	}
	for _, t := range raw.Teams {
		for _, p := range t.Positions {
			n, err := finite(p.Points)
			if err != nil {
				return Feed{}, err
			}
			out.Allowed = append(out.Allowed, Allowed{t.ID, p.Name, n})
		}
	}
	return out, nil
}
func Fetch(ctx context.Context, c *mfl.Client, season int, kind, week string) ([]Feed, error) {
	request := mfl.Request{Type: kind, Year: strconv.Itoa(season)}
	if kind == "playerScores" || kind == "pointsAllowed" {
		if err := c.DiscoverHost(ctx, request.Year, ingestion.LeagueID); err != nil {
			return nil, fmt.Errorf("research host: %w", err)
		}
		request.Params = map[string]string{"L": ingestion.LeagueID}
	}
	if week != "" {
		if request.Params == nil {
			request.Params = map[string]string{}
		}
		request.Params["W"] = week
	}
	response, err := c.Do(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("%s fetch: %w", kind, err)
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: status %d", kind, response.StatusCode)
	}
	feed, err := Parse(response.Body, kind, week)
	if err != nil {
		return nil, err
	}
	return []Feed{feed}, nil
}
