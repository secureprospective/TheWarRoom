// Package playerscores fetches season fantasy-point totals from MFL. Its one consumer is the
// M1 BasePoints placeholder, which must be labeled as a proxy wherever it shows.
//
// The score season is separate from the league year: the league lives at /2026/, but M1 needs
// the last completed season, so it asks for YEAR=2025 through the 2026 host. W=YTD returns the
// season total for every player in one call.
package playerscores

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
)

// errEmptyScores: a completed season is never empty, and empty would flatten the whole board
// to zero.
var errEmptyScores = errors.New("playerscores: response contained zero scores")

// RawScore is one player's season total as MFL sends it. Week echoes the requested window.
type RawScore struct {
	ID    string
	Score string // league scoring, raw ("476.75")
	Week  string // "YTD" for the season total
}

// Validate requires a valid player id and a parseable score. MFL omits unscored players, so
// an empty score is malformed, not zero.
func (s RawScore) Validate() error {
	if _, err := ingestion.ValidatePlayerID(s.ID); err != nil {
		return fmt.Errorf("playerscores: %w", err)
	}
	sc := strings.TrimSpace(s.Score)
	if sc == "" {
		return fmt.Errorf("playerscores: record %s missing score", s.ID)
	}
	if _, err := strconv.ParseFloat(sc, 64); err != nil {
		return fmt.Errorf("playerscores: record %s non-numeric score %q: %w", s.ID, s.Score, err)
	}
	return nil
}

// scoresEnvelope mirrors the MFL playerScores JSON; unknown fields are tolerated.
type scoresEnvelope struct {
	PlayerScores struct {
		PlayerScore ingestion.MFLList[scoreBlock] `json:"playerScore"`
	} `json:"playerScores"`
}

type scoreBlock struct {
	ID    string `json:"id"`
	Score string `json:"score"`
	Week  string `json:"week"`
}

// Fetch returns scoreYear's totals through the year-league host. It checks MFL's error
// envelope before decoding, because an outage read as "no scores" would zero every player's
// BasePoints.
func Fetch(ctx context.Context, c *mfl.Client, year, leagueID, scoreYear string) ([]RawScore, error) {
	if strings.TrimSpace(scoreYear) == "" {
		return nil, fmt.Errorf("playerscores: score year is required")
	}
	env, err := ingestion.FetchLeagueExport[scoresEnvelope](ctx, c, "playerScores", year, leagueID,
		map[string]string{"W": "YTD", "YEAR": scoreYear})
	if err != nil {
		return nil, fmt.Errorf("playerscores: %w", err)
	}
	out, err := flatten(ctx, env)
	if err != nil {
		return nil, err
	}
	return guardNonEmpty(out)
}

// guardNonEmpty runs after the aggregate filter: a payload of only aggregates would pass an
// earlier length check and still leave zero player scores.
func guardNonEmpty(out []RawScore) ([]RawScore, error) {
	if len(out) == 0 {
		return nil, errEmptyScores
	}
	return out, nil
}

// flatten drops team aggregates (Coach, Def, ST and Off score points here but are not
// players) and validates the rest; a malformed real record fails the fetch.
func flatten(ctx context.Context, env scoresEnvelope) ([]RawScore, error) {
	out := make([]RawScore, 0, len(env.PlayerScores.PlayerScore))
	for _, sb := range env.PlayerScores.PlayerScore {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("playerscores: flatten cancelled: %w", ctx.Err())
		default:
		}
		if ingestion.IsTeamAggregateID(sb.ID) {
			continue
		}
		rs := RawScore(sb)
		if err := rs.Validate(); err != nil {
			return nil, err
		}
		// A single-week echo would be about 17× short as a season total. Assert the window.
		if rs.Week != "YTD" {
			return nil, fmt.Errorf("playerscores: record %s echoes week %q, want the requested YTD aggregate", rs.ID, rs.Week)
		}
		out = append(out, rs)
	}
	return out, nil
}
