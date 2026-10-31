package history

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/projection"
)

// NativeProjection is one player's projected week under the league's own scoring (D11): floor,
// median and ceiling. Games is how many of his own weeks fed it; 0 means the position's shape.
type NativeProjection struct {
	Player   string  `json:"player"`
	Position string  `json:"position,omitempty"`
	Floor    float64 `json:"floor"`
	Median   float64 `json:"median"`
	Ceiling  float64 `json:"ceiling"`
	Games    int     `json:"games"`
	// Available is false when the player has no NFL statistics identity or no held weeks.
	Available bool   `json:"available"`
	Words     string `json:"words"`
}

type NativeProjections struct {
	League  string             `json:"league"`
	Season  int                `json:"season"`
	Week    int                `json:"week"`
	Players []NativeProjection `json:"players"`
	Notes   []string           `json:"notes"`
}

// projectionPosition maps nflverse's listed position onto the scoring positions the replay used.
func projectionPosition(raw string) (leaguerules.Position, bool) {
	p, ok := map[string]leaguerules.Position{"QB": "QB", "RB": "RB", "FB": "RB", "WR": "WR", "TE": "TE", "K": "PK",
		"DE": "DE", "DT": "DT", "NT": "DT", "LB": "LB", "OLB": "LB", "ILB": "LB", "MLB": "LB",
		"CB": "CB", "S": "S", "SS": "S", "FS": "S", "SAF": "S", "DB": "S"}[strings.ToUpper(raw)]
	return p, ok
}

// projectionBook is every held regular-season week of the projected season and the two before it,
// scored under the league's rules, by NFL statistics id.
type projectionBook struct {
	weeks    map[string][]projection.WeekScore
	position map[string]leaguerules.Position
	byPos    map[leaguerules.Position][]projection.WeekScore
	seasons  []int
}

func scoreProjectionBook(rules leaguerules.ScoringRules, held map[int][]source.PlayerWeek) projectionBook {
	b := projectionBook{weeks: map[string][]projection.WeekScore{}, position: map[string]leaguerules.Position{},
		byPos: map[leaguerules.Position][]projection.WeekScore{}}
	for season, weeks := range held {
		b.seasons = append(b.seasons, season)
		for _, w := range weeks {
			pos, ok := projectionPosition(w.Position)
			if w.SeasonType != "REG" || !ok {
				continue
			}
			points, _, err := fantasyscore.Score(rules, pos, w.Line)
			if err != nil || points.State != leaguerules.IsKnown {
				continue
			}
			ws := projection.WeekScore{Season: w.Season, Week: w.Week, Points: float64(points.Value) / 10000}
			b.weeks[w.GSIS] = append(b.weeks[w.GSIS], ws)
			b.byPos[pos] = append(b.byPos[pos], ws)
			b.position[w.GSIS] = pos
		}
	}
	return b
}

// project reads each player's range for (season, week) from the book. ids maps league player id
// to NFL statistics id.
func (b projectionBook) project(players []string, ids map[string]string, season, week int) []NativeProjection {
	p := projection.DefaultParams()
	shapes := map[leaguerules.Position]projection.Shape{}
	out := make([]NativeProjection, 0, len(players))
	for _, player := range players {
		row := NativeProjection{Player: player}
		gsis := ids[player]
		pos, ok := b.position[gsis]
		switch {
		case gsis == "":
			row.Words = "No NFL statistics identity for this player"
		case !ok:
			row.Words = "No NFL weeks held for this player"
		default:
			shape, seen := shapes[pos]
			if !seen {
				shape = projection.NewShape(b.byPos[pos], season, week, p)
				shapes[pos] = shape
			}
			r, has := projection.Project(projection.NewShape(b.weeks[gsis], season, week, p), shape, p)
			if has {
				row.Position, row.Floor, row.Median, row.Ceiling, row.Games = string(pos), r.Floor, r.Median, r.Ceiling, r.Games
				row.Available, row.Words = true, projectionWords(r.Games)
			} else {
				row.Words = "No weeks held before this week"
			}
		}
		out = append(out, row)
	}
	return out
}

func projectionWords(games int) string {
	switch {
	case games == 0:
		return "Projected off the position's shape — no NFL weeks of his own yet"
	case games < int(projection.DefaultParams().MinScores):
		return fmt.Sprintf("Few weeks of his own (%d); leans on the position's shape", games)
	}
	return fmt.Sprintf("From %d of his own weeks", games)
}

// NativeProjections projects the players' week in an app league under the rules in force now,
// from held NFL weekly statistics for the season and the two before it. No matchup factor: the
// replay showed none helps (docs/ui/data-catalog/D11-REPLAY.md).
func (s *Store) NativeProjections(ctx context.Context, league string, season, week int,
	players []string) (NativeProjections, error) {
	if season < 1920 || season > 9999 || week < 1 || week > 22 {
		return NativeProjections{}, fmt.Errorf("projections: choose an NFL season and week")
	}
	out := NativeProjections{League: league, Season: season, Week: week, Players: []NativeProjection{}, Notes: []string{}}
	records, err := s.nativeRecords(ctx, league)
	if err != nil {
		return NativeProjections{}, err
	}
	rules, err := commissioner.RulesInForce(records, season, s.now())
	if err != nil {
		return NativeProjections{}, fmt.Errorf("projections: rules: %w", err)
	}
	held := map[int][]source.PlayerWeek{}
	for y := season - 2; y <= season; y++ {
		files, err := s.heldSeason(ctx, y)
		if errors.Is(err, source.ErrSeasonNotLoaded) || err == nil && len(files) == 0 {
			out.Notes = append(out.Notes, fmt.Sprintf("No NFL weekly statistics held for %d", y))
			continue
		}
		if err != nil {
			return NativeProjections{}, fmt.Errorf("projections: %d weeks: %w", y, err)
		}
		held[y] = files[0].weeks
	}
	ids, err := s.SourceIDs(ctx, "gsis")
	if err != nil {
		return NativeProjections{}, err
	}
	out.Players = scoreProjectionBook(rules.Scoring, held).project(players, ids, season, week)
	return out, nil
}
