// Package playoffs decodes MFL's playoffBrackets export (the brackets a league defines) and its
// playoffBracket export (one bracket's games, seeds and scores so far).
package playoffs

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
)

// ListExport lists the brackets; BracketExport is one bracket, chosen with BRACKET_ID.
const (
	ListExport    = "playoffBrackets"
	BracketExport = "playoffBracket"
)

// RawBracket is one bracket the league defines.
type RawBracket struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Title     string `json:"bracketWinnerTitle"`
	StartWeek string `json:"startWeek"`
	Teams     string `json:"teamsInvolved"`
}

// Validate requires the numbers the bracket report reads.
func (r RawBracket) Validate() error {
	for _, f := range []struct{ name, raw string }{{"id", r.ID}, {"startWeek", r.StartWeek}, {"teamsInvolved", r.Teams}} {
		if _, err := whole(f.raw); err != nil {
			return fmt.Errorf("playoffs: bracket %q %s: %w", r.Name, f.name, err)
		}
	}
	return nil
}

// RawSide is one side of a bracket game; every field is optional until MFL fills it.
type RawSide struct {
	Franchise string `json:"franchise_id"`
	Seed      string `json:"seed"`
	WinnerOf  string `json:"winner_of_game"`
	Bracket   string `json:"bracket"`
	Points    string `json:"points"`
}

// Validate checks every present field.
func (s RawSide) Validate() error {
	if s.Franchise != "" {
		if err := ingestion.FeedFranchise(s.Franchise); err != nil {
			return fmt.Errorf("playoffs: %w", err)
		}
	}
	for _, raw := range []string{s.Seed, s.WinnerOf, s.Bracket} {
		if _, err := whole(raw); raw != "" && err != nil {
			return fmt.Errorf("playoffs: side %s: %w", s.Franchise, err)
		}
	}
	if s.Points != "" {
		v, err := strconv.ParseFloat(s.Points, 64)
		if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
			return fmt.Errorf("playoffs: side %s points %q", s.Franchise, s.Points)
		}
	}
	if s.Seed == "" && s.WinnerOf == "" && s.Franchise == "" {
		return fmt.Errorf("playoffs: a game side names no franchise, seed or earlier game")
	}
	return nil
}

type rawGame struct {
	ID   string  `json:"game_id"`
	Home RawSide `json:"home"`
	Away RawSide `json:"away"`
}

type rawRound struct {
	Week  string                     `json:"week"`
	Games ingestion.MFLList[rawGame] `json:"playoffGame"`
}

func whole(raw string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n < 0 || n > 1000 {
		return 0, fmt.Errorf("%q is not a whole number", raw)
	}
	return n, nil
}

// ParseList decodes the brackets the league defines; a league without playoffs lists none.
func ParseList(body []byte) ([]leaguefeed.Bracket, error) {
	if err := ingestion.CheckAPIError(body); err != nil {
		return nil, fmt.Errorf("playoffs: %w", err)
	}
	var env struct {
		List struct {
			Brackets ingestion.MFLList[RawBracket] `json:"playoffBracket"`
		} `json:"playoffBrackets"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("playoffs: decode brackets: %w", err)
	}
	out := make([]leaguefeed.Bracket, 0, len(env.List.Brackets))
	for _, r := range env.List.Brackets {
		if err := r.Validate(); err != nil {
			return nil, err
		}
		id, _ := whole(r.ID)
		start, _ := whole(r.StartWeek)
		teams, _ := whole(r.Teams)
		out = append(out, leaguefeed.Bracket{ID: id, Name: r.Name, Title: r.Title, StartWeek: start, Teams: teams,
			Games: []leaguefeed.BracketGame{}})
	}
	return out, nil
}

// ParseBracket decodes one bracket's games, checking it is the bracket asked for.
func ParseBracket(id int, body []byte) ([]leaguefeed.BracketGame, error) {
	if err := ingestion.CheckAPIError(body); err != nil {
		return nil, fmt.Errorf("playoffs: %w", err)
	}
	var env struct {
		Bracket struct {
			ID     string                      `json:"bracket_id"`
			Rounds ingestion.MFLList[rawRound] `json:"playoffRound"`
		} `json:"playoffBracket"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("playoffs: decode bracket %d: %w", id, err)
	}
	if got, err := whole(env.Bracket.ID); err != nil || got != id {
		return nil, fmt.Errorf("playoffs: asked for bracket %d, got %q", id, env.Bracket.ID)
	}
	out := []leaguefeed.BracketGame{}
	for _, round := range env.Bracket.Rounds {
		wk, err := whole(round.Week)
		if err != nil {
			return nil, fmt.Errorf("playoffs: bracket %d round week: %w", id, err)
		}
		for _, g := range round.Games {
			gid, err := whole(g.ID)
			if err != nil {
				return nil, fmt.Errorf("playoffs: bracket %d game id: %w", id, err)
			}
			for _, s := range []RawSide{g.Home, g.Away} {
				if err := s.Validate(); err != nil {
					return nil, fmt.Errorf("playoffs: bracket %d game %d: %w", id, gid, err)
				}
			}
			out = append(out, leaguefeed.BracketGame{ID: gid, Week: wk, Home: side(g.Home), Away: side(g.Away)})
		}
	}
	return out, nil
}

// side maps a validated side.
func side(r RawSide) leaguefeed.BracketSide {
	s := leaguefeed.BracketSide{Franchise: r.Franchise, Scored: r.Points != ""}
	s.Seed, _ = whole(r.Seed)
	s.WinnerOf, _ = whole(r.WinnerOf)
	s.FromBracket, _ = whole(r.Bracket)
	if s.Scored {
		v, _ := strconv.ParseFloat(r.Points, 64)
		s.Points = leaguefeed.Points(math.Round(v * 100))
	}
	return s
}

// FetchList returns the brackets the league defines, without their games.
func FetchList(ctx context.Context, c *mfl.Client, year, leagueID string) ([]leaguefeed.Bracket, error) {
	if err := c.DiscoverHost(ctx, year, leagueID); err != nil {
		return nil, fmt.Errorf("playoffs: discover host: %w", err)
	}
	body, err := ingestion.LeagueExport(ctx, c, ListExport, year, leagueID, nil)
	if err != nil {
		return nil, fmt.Errorf("playoffs: %w", err)
	}
	return ParseList(body)
}

// FetchBracket returns one bracket's games.
func FetchBracket(ctx context.Context, c *mfl.Client, year, leagueID string, id int) ([]leaguefeed.BracketGame, error) {
	if err := c.DiscoverHost(ctx, year, leagueID); err != nil {
		return nil, fmt.Errorf("playoffs: discover host: %w", err)
	}
	body, err := ingestion.LeagueExport(ctx, c, BracketExport, year, leagueID,
		map[string]string{"BRACKET_ID": strconv.Itoa(id)})
	if err != nil {
		return nil, fmt.Errorf("playoffs: bracket %d: %w", id, err)
	}
	return ParseBracket(id, body)
}
