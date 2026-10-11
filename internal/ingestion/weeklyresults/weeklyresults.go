// Package weeklyresults decodes MFL's weeklyResults export for the whole season (W=YTD): every
// week's games, scores, results and each franchise's started and benched players.
package weeklyresults

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// Export is the MFL export this package reads.
const Export = "weeklyResults"

// errNoWeeks: a league's season export always lists its weeks, scored or not.
var errNoWeeks = errors.New("weeklyresults: response contained zero weeks")

// RawPlayer is one rostered player's line. Score is empty until the week is scored.
type RawPlayer struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	ShouldStart string `json:"shouldStart"`
	Score       string `json:"score"`
}

// RawSide is one franchise's week; Result and IsHome are empty outside a game.
type RawSide struct {
	ID      string                       `json:"id"`
	Score   string                       `json:"score"`
	Best    string                       `json:"opt_pts"`
	Result  string                       `json:"result"`
	IsHome  string                       `json:"isHome"`
	Players ingestion.MFLList[RawPlayer] `json:"player"`
}

type rawMatchup struct {
	RegularSeason string                     `json:"regularSeason"`
	Franchise     ingestion.MFLList[RawSide] `json:"franchise"`
}

// RawWeek is one week: its games and the franchises scored without one.
type RawWeek struct {
	Week      string                        `json:"week"`
	Matchup   ingestion.MFLList[rawMatchup] `json:"matchup"`
	Franchise ingestion.MFLList[RawSide]    `json:"franchise"`
}

// Validate checks every field the race reads, so a malformed record fails here and never reads
// as a zero score or a lost game.
func (w RawWeek) Validate() error {
	if _, err := week(w.Week); err != nil {
		return err
	}
	for _, m := range w.Matchup {
		if len(m.Franchise) != 2 {
			return fmt.Errorf("weeklyresults: week %s has a game with %d franchises, want 2", w.Week, len(m.Franchise))
		}
		if m.RegularSeason != "0" && m.RegularSeason != "1" {
			return fmt.Errorf("weeklyresults: week %s regularSeason %q is not 0/1", w.Week, m.RegularSeason)
		}
		if (m.Franchise[0].IsHome == "1") == (m.Franchise[1].IsHome == "1") {
			return fmt.Errorf("weeklyresults: week %s game needs exactly one home side", w.Week)
		}
		if (m.Franchise[0].Score == "") != (m.Franchise[1].Score == "") {
			return fmt.Errorf("weeklyresults: week %s game is half scored", w.Week)
		}
		for _, s := range m.Franchise {
			if err := s.validate(w.Week, true); err != nil {
				return err
			}
		}
	}
	for _, s := range w.Franchise {
		if err := s.validate(w.Week, false); err != nil {
			return err
		}
	}
	return nil
}

func (s RawSide) validate(wk string, inGame bool) error {
	if err := ingestion.FeedFranchise(s.ID); err != nil {
		return fmt.Errorf("weeklyresults: week %s: %w", wk, err)
	}
	if s.Score == "" {
		return s.validatePlayers(wk)
	}
	if inGame && s.Result != "W" && s.Result != "L" && s.Result != "T" {
		return fmt.Errorf("weeklyresults: week %s franchise %s result %q is not W/L/T", wk, s.ID, s.Result)
	}
	for _, raw := range []string{s.Score, s.Best} {
		if _, err := points(raw); raw != "" && err != nil {
			return fmt.Errorf("weeklyresults: week %s franchise %s: %w", wk, s.ID, err)
		}
	}
	return s.validatePlayers(wk)
}

func (s RawSide) validatePlayers(wk string) error {
	for _, p := range s.Players {
		if _, err := playerid.New(p.ID); err != nil {
			return fmt.Errorf("weeklyresults: week %s franchise %s: %w", wk, s.ID, err)
		}
		if p.Status != "starter" && p.Status != "nonstarter" {
			return fmt.Errorf("weeklyresults: week %s player %s status %q", wk, p.ID, p.Status)
		}
		if p.Score != "" {
			if _, err := points(p.Score); err != nil {
				return fmt.Errorf("weeklyresults: week %s player %s: %w", wk, p.ID, err)
			}
		}
	}
	return nil
}

func week(raw string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n < 1 || n > 30 {
		return 0, fmt.Errorf("weeklyresults: week %q is not a week number", raw)
	}
	return n, nil
}

// points reads MFL's decimal score into exact hundredths.
func points(raw string) (leaguefeed.Points, error) {
	v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, fmt.Errorf("score %q is not a number", raw)
	}
	return leaguefeed.Points(math.Round(v * 100)), nil
}

// Fetch discovers the league host, then returns the season's weeks, validated.
func Fetch(ctx context.Context, c *mfl.Client, year, leagueID string) ([]leaguefeed.WeekResult, error) {
	if err := c.DiscoverHost(ctx, year, leagueID); err != nil {
		return nil, fmt.Errorf("weeklyresults: discover host: %w", err)
	}
	body, err := ingestion.LeagueExport(ctx, c, Export, year, leagueID, map[string]string{"W": "YTD"})
	if err != nil {
		return nil, fmt.Errorf("weeklyresults: %w", err)
	}
	return Parse(body)
}

// Parse decodes and validates an export body, live or archived, in week order.
func Parse(body []byte) ([]leaguefeed.WeekResult, error) {
	if err := ingestion.CheckAPIError(body); err != nil {
		return nil, fmt.Errorf("weeklyresults: %w", err)
	}
	var env struct {
		All struct {
			Weeks ingestion.MFLList[RawWeek] `json:"weeklyResults"`
		} `json:"allWeeklyResults"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("weeklyresults: decode: %w", err)
	}
	if len(env.All.Weeks) == 0 {
		return nil, errNoWeeks
	}
	out := make([]leaguefeed.WeekResult, 0, len(env.All.Weeks))
	seen := map[int]bool{}
	for _, raw := range env.All.Weeks {
		if err := raw.Validate(); err != nil {
			return nil, err
		}
		w := convert(raw)
		if seen[w.Week] {
			return nil, fmt.Errorf("weeklyresults: week %d listed twice", w.Week)
		}
		seen[w.Week] = true
		out = append(out, w)
	}
	return out, nil
}

// convert maps a validated week; every parse below already succeeded in Validate.
func convert(raw RawWeek) leaguefeed.WeekResult {
	n, _ := week(raw.Week)
	w := leaguefeed.WeekResult{Week: n, Games: []leaguefeed.Game{}, Others: []leaguefeed.Side{}}
	for _, m := range raw.Matchup {
		home, away := m.Franchise[0], m.Franchise[1]
		if home.IsHome != "1" {
			home, away = away, home
		}
		w.Games = append(w.Games, leaguefeed.Game{Regular: m.RegularSeason == "1", Home: side(home), Away: side(away)})
	}
	for _, s := range raw.Franchise {
		w.Others = append(w.Others, side(s))
	}
	return w
}

func side(raw RawSide) leaguefeed.Side {
	s := leaguefeed.Side{Franchise: raw.ID, Scored: raw.Score != "", Players: []leaguefeed.WeekPlayer{}}
	if s.Scored {
		s.Points, _ = points(raw.Score)
		s.Best, _ = points(raw.Best)
		s.Result = leaguefeed.Result(raw.Result)
	}
	for _, p := range raw.Players {
		id, _ := playerid.New(p.ID)
		pts, _ := points(p.Score)
		s.Players = append(s.Players, leaguefeed.WeekPlayer{
			ID: id, Starter: p.Status == "starter", ShouldStart: p.ShouldStart == "1", Points: pts,
		})
	}
	return s
}
