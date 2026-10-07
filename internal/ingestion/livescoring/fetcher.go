// Package livescoring adapts MFL's saved weekly lineups and live franchise totals.
package livescoring

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

const Export = "liveScoring"

type RawPlayer struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type RawFranchise struct {
	ID               string `json:"id"`
	Score            string `json:"score"`
	SecondsRemaining string `json:"gameSecondsRemaining"`
	Players          struct {
		Player ingestion.MFLList[RawPlayer] `json:"player"`
	} `json:"players"`
}

type RawMatchup struct {
	Franchises ingestion.MFLList[RawFranchise] `json:"franchise"`
}

type RawLineups struct {
	Week     string                        `json:"week"`
	Matchups ingestion.MFLList[RawMatchup] `json:"matchup"`
}

func (r RawLineups) Validate() error {
	week, err := strconv.Atoi(r.Week)
	if err != nil {
		return fmt.Errorf("livescoring: week: %w", err)
	}
	if week < 1 || len(r.Matchups) == 0 {
		return fmt.Errorf("livescoring: needs a positive week and matchups")
	}
	seen := make(map[string]bool)
	for i, m := range r.Matchups {
		if len(m.Franchises) != 2 {
			return fmt.Errorf("livescoring: matchup %d needs two franchises", i)
		}
		for _, f := range m.Franchises {
			if seen[f.ID] {
				return fmt.Errorf("livescoring: duplicate franchise %q", f.ID)
			}
			seen[f.ID] = true
			if err := f.Validate(); err != nil {
				return fmt.Errorf("livescoring: matchup %d franchise %q: %w", i, f.ID, err)
			}
		}
	}
	return nil
}

func (r RawFranchise) Validate() error {
	_, err := convert(r)
	return err
}

func Fetch(ctx context.Context, c *mfl.Client, year, league string, week int) (RawLineups, error) {
	if week < 0 {
		return RawLineups{}, fmt.Errorf("livescoring: invalid requested week %d", week)
	}
	if err := c.DiscoverHost(ctx, year, league); err != nil {
		return RawLineups{}, fmt.Errorf("livescoring: discover: %w", err)
	}
	params := map[string]string{}
	if week != 0 {
		params["W"] = strconv.Itoa(week)
	}
	body, err := ingestion.LeagueExport(ctx, c, Export, year, league, params)
	if err != nil {
		return RawLineups{}, fmt.Errorf("livescoring: fetch: %w", err)
	}
	return Parse(body)
}

func Parse(body []byte) (RawLineups, error) {
	if err := ingestion.CheckAPIError(body); err != nil {
		return RawLineups{}, fmt.Errorf("livescoring: %w", err)
	}
	var env struct {
		Lineups RawLineups `json:"liveScoring"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return RawLineups{}, fmt.Errorf("livescoring: decode: %w", err)
	}
	if err := env.Lineups.Validate(); err != nil {
		return RawLineups{}, err
	}
	return env.Lineups, nil
}

func ToLineups(raw RawLineups) (leaguefeed.Lineups, error) {
	if err := raw.Validate(); err != nil {
		return leaguefeed.Lineups{}, err
	}
	week, err := strconv.Atoi(raw.Week)
	if err != nil {
		return leaguefeed.Lineups{}, fmt.Errorf("livescoring: convert week: %w", err)
	}
	rows := leaguefeed.Lineups{Week: week, Franchises: make([]leaguefeed.Lineup, 0)}
	for _, m := range raw.Matchups {
		for _, f := range m.Franchises {
			row, err := convert(f)
			if err != nil {
				return leaguefeed.Lineups{}, fmt.Errorf("livescoring: convert franchise %q: %w", f.ID, err)
			}
			rows.Franchises = append(rows.Franchises, row)
		}
	}
	return rows, nil
}

func convert(r RawFranchise) (leaguefeed.Lineup, error) {
	if err := ingestion.FeedFranchise(r.ID); err != nil {
		return leaguefeed.Lineup{}, fmt.Errorf("franchise: %w", err)
	}
	score, err := strconv.ParseFloat(r.Score, 64)
	if err != nil {
		return leaguefeed.Lineup{}, fmt.Errorf("score: %w", err)
	}
	if math.IsNaN(score) || math.IsInf(score, 0) {
		return leaguefeed.Lineup{}, fmt.Errorf("non-finite score %q", r.Score)
	}
	seconds, err := strconv.Atoi(r.SecondsRemaining)
	if err != nil {
		return leaguefeed.Lineup{}, fmt.Errorf("seconds remaining: %w", err)
	}
	if seconds < 0 {
		return leaguefeed.Lineup{}, fmt.Errorf("negative seconds remaining")
	}
	row := leaguefeed.Lineup{
		Franchise: r.ID, Score: score, SecondsRemaining: seconds,
		Starters: make([]playerid.PlayerID, 0), NonStarters: make([]playerid.PlayerID, 0),
	}
	if err := assignPlayers(&row, r.Players.Player); err != nil {
		return leaguefeed.Lineup{}, err
	}
	return row, nil
}

func assignPlayers(row *leaguefeed.Lineup, players []RawPlayer) error {
	seen := make(map[playerid.PlayerID]bool)
	for i, p := range players {
		id, err := playerid.New(p.ID)
		if err != nil {
			return fmt.Errorf("player %d: %w", i, err)
		}
		if seen[id] {
			return fmt.Errorf("duplicate player %q", p.ID)
		}
		seen[id] = true
		switch p.Status {
		case "starter":
			row.Starters = append(row.Starters, id)
		case "nonstarter":
			row.NonStarters = append(row.NonStarters, id)
		default:
			return fmt.Errorf("player %d unknown status %q", i, p.Status)
		}
	}
	return nil
}
