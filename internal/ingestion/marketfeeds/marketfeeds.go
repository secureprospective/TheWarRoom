// Package marketfeeds reads MFL projections and MFL-wide player market feeds.
package marketfeeds

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

const Projections = "projectedScores"
const Ranks = "playerRanks"
const ADP = "adp"
const AAV = "aav"

type RawPlayer struct {
	ID           string `json:"id"`
	Score        string `json:"score"`
	Rank         string `json:"rank"`
	Last         string `json:"last_week"`
	Change       string `json:"change"`
	AveragePick  string `json:"averagePick"`
	MinPick      string `json:"minPick"`
	MaxPick      string `json:"maxPick"`
	AverageValue string `json:"averageValue"`
	MinValue     string `json:"minValue"`
	MaxValue     string `json:"maxValue"`
}

type RawFeed struct {
	Week          string                       `json:"week"`
	TotalDrafts   string                       `json:"totalDrafts"`
	TotalPicks    string                       `json:"totalPicks"`
	TotalAuctions string                       `json:"totalAuctions"`
	Players       ingestion.MFLList[RawPlayer] `json:"player"`
	Scores        ingestion.MFLList[RawPlayer] `json:"playerScore"`
}

type Player struct {
	New                bool
	ID                 playerid.PlayerID
	Value, Low, High   float64
	Rank, Last, Change int
}

type Feed struct {
	Players       []Player
	Week, Samples int
}

func number(s string) (float64, error) {
	n, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(n) || math.IsInf(n, 0) {
		return 0, fmt.Errorf("invalid finite number %q", s)
	}
	return n, nil
}
func count(s string, minimum int) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil || n < minimum {
		return 0, fmt.Errorf("invalid count %q", s)
	}
	return n, nil
}
func (r RawPlayer) values(kind string) (string, string, string) {
	if kind == ADP {
		return r.AveragePick, r.MinPick, r.MaxPick
	}
	return r.AverageValue, r.MinValue, r.MaxValue
}
func (r RawPlayer) Validate(kind string) error {
	if kind == Projections && r.ID == "" && r.Score == "" {
		return nil
	}
	if _, err := playerid.New(r.ID); err != nil {
		return fmt.Errorf("market player: %w", err)
	}
	switch kind {
	case Projections:
		if _, err := number(r.Score); err != nil {
			return fmt.Errorf("projection: %w", err)
		}
	case Ranks:
		if _, err := count(r.Rank, 1); err != nil {
			return fmt.Errorf("rank: %w", err)
		}
		if _, err := count(r.Last, 0); err != nil {
			return fmt.Errorf("last week's rank: %w", err)
		}
		if r.Change == "NEW" {
			return nil
		}
		if _, err := strconv.Atoi(r.Change); err != nil {
			return fmt.Errorf("rank change: %w", err)
		}
	case ADP, AAV:
		return r.validateRange(kind)
	default:
		return fmt.Errorf("unknown market feed %q", kind)
	}
	return nil
}
func (r RawPlayer) validateRange(kind string) error {
	average, low, high := r.values(kind)
	v, ve := number(average)
	lo, le := number(low)
	hi, he := number(high)
	if ve != nil || le != nil || he != nil || lo < 0 || lo > v || v > hi {
		return fmt.Errorf("%s: invalid average or min–max range for %s", kind, r.ID)
	}
	return nil
}
func (r RawFeed) Validate(kind string) error {
	if err := r.validateScope(kind); err != nil {
		return err
	}
	players := r.Players
	if kind == Projections {
		players = r.Scores
	}
	seen := map[string]bool{}
	for _, p := range players {
		if err := p.Validate(kind); err != nil {
			return err
		}
		if seen[p.ID] {
			return fmt.Errorf("market feed: duplicate player %s", p.ID)
		}
		seen[p.ID] = true
	}
	return nil
}

func (r RawFeed) validateScope(kind string) error {
	switch kind {
	case Projections:
		week, err := count(r.Week, 1)
		if err != nil || week > 18 {
			return fmt.Errorf("projection: invalid NFL week %q", r.Week)
		}
	case Ranks:
	case ADP:
		if _, err := count(r.TotalDrafts, 0); err != nil {
			return fmt.Errorf("draft sample: %w", err)
		}
		picks, err := count(r.TotalPicks, 0)
		if err != nil {
			return fmt.Errorf("draft picks: %w", err)
		}
		if picks == 0 && len(r.Players) > 0 {
			return fmt.Errorf("ADP players with no picks")
		}
	case AAV:
		if _, err := count(r.TotalAuctions, 0); err != nil {
			return fmt.Errorf("auction sample: %w", err)
		}
	default:
		return fmt.Errorf("unknown market feed %q", kind)
	}
	return nil
}

func Parse(kind string, body []byte) (Feed, error) {
	if err := ingestion.CheckAPIError(body); err != nil {
		return Feed{}, fmt.Errorf("market feed: %w", err)
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		return Feed{}, fmt.Errorf("market feed: %w", err)
	}
	key := kind
	if kind == Ranks {
		key = "player_ranks"
	}
	section, ok := envelope[key]
	if !ok || string(section) == "null" {
		return Feed{}, fmt.Errorf("market feed: missing %s", key)
	}
	var raw RawFeed
	if err := json.Unmarshal(section, &raw); err != nil {
		return Feed{}, fmt.Errorf("market feed: %w", err)
	}
	if err := raw.Validate(kind); err != nil {
		return Feed{}, err
	}
	return convert(kind, raw), nil
}
func convert(kind string, raw RawFeed) Feed {
	feed := Feed{Players: []Player{}}
	players := raw.Players
	switch kind {
	case Projections:
		feed.Week, _ = strconv.Atoi(raw.Week)
		players = raw.Scores
	case ADP:
		feed.Samples, _ = strconv.Atoi(raw.TotalDrafts)
	case AAV:
		feed.Samples, _ = strconv.Atoi(raw.TotalAuctions)
	}
	for _, r := range players {
		if kind == Projections && r.ID == "" {
			continue
		}
		id, _ := playerid.New(r.ID)
		p := Player{ID: id}
		switch kind {
		case Projections:
			p.Value, _ = strconv.ParseFloat(r.Score, 64)
		case Ranks:
			p.Rank, _ = strconv.Atoi(r.Rank)
			p.Last, _ = strconv.Atoi(r.Last)
			p.New = r.Change == "NEW"
			if !p.New {
				p.Change, _ = strconv.Atoi(r.Change)
			}
		default:
			average, low, high := r.values(kind)
			p.Value, _ = strconv.ParseFloat(average, 64)
			p.Low, _ = strconv.ParseFloat(low, 64)
			p.High, _ = strconv.ParseFloat(high, 64)
		}
		feed.Players = append(feed.Players, p)
	}
	return feed
}

func Fetch(ctx context.Context, c *mfl.Client, kind, year, league string, week int) (Feed, error) {
	params := map[string]string{}
	if kind == Projections {
		params["L"], params["W"] = league, strconv.Itoa(week)
		if week < 1 || week > 18 {
			return Feed{}, fmt.Errorf("projection: NFL week not known")
		}
		if err := c.DiscoverHost(ctx, year, league); err != nil {
			return Feed{}, fmt.Errorf("projection host: %w", err)
		}
	}
	if kind == ADP {
		params["PERIOD"] = "ALL"
	}
	response, err := c.Do(ctx, mfl.Request{Type: kind, Year: year, Params: params})
	if err != nil {
		return Feed{}, fmt.Errorf("market fetch: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		return Feed{}, fmt.Errorf("market fetch: HTTP %d", response.StatusCode)
	}
	feed, err := Parse(kind, response.Body)
	if err != nil {
		return Feed{}, fmt.Errorf("market fetch %s: %w", kind, err)
	}
	if kind == Projections && feed.Week != week {
		return Feed{}, fmt.Errorf("projection: requested week %d, received %d", week, feed.Week)
	}
	return feed, nil
}
