// Package playerscores is the Layer 1 fetcher for the MFL playerScores endpoint.
// It clones the reviewed rosters template (build Request → c.Do → schema-validate
// the SHAPE → return raw) and exists for ONE consumer: the M1 BasePoints
// placeholder (Christopher's 2026-06-28 decision, option (b)) — league-scoring
// fantasy points fed to the engine as `PlayerInput.BasePoints` until the real L2
// base-scoring block ships. Everywhere a score derived from this data is shown it
// must be labeled as the proxy ("MFL YTD fantasy points — L2 pending"); the
// fetcher itself transforms nothing (score stays a raw string, parsed at the
// orchestrator boundary).
//
// The score season is an explicit argument SEPARATE from the league year: the
// league lives at /2026/ but 2026 has no completed weeks, so M1 pulls the last
// COMPLETED season's YTD totals (YEAR=2025) through the 2026 league host. W=YTD
// asks MFL for the season aggregate, so one call covers every player.
package playerscores

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
)

// errEmptyScores guards a glitch/empty payload. A completed NFL season's YTD
// export is never legitimately empty; returning an empty slice would silently
// zero every player's BasePoints downstream (the whole board would render as a
// meaningless flat ranking) instead of failing loud.
var errEmptyScores = errors.New("playerscores: response contained zero scores")

// RawScore is one player's YTD fantasy-point total exactly as MFL returns it.
// Score stays a raw string (MFL encodes numbers as strings; a fetcher transforms
// nothing — the orchestrator parses at its boundary). Week is retained so a
// consumer can assert it got the aggregate it asked for ("YTD"), not a single week.
type RawScore struct {
	ID    string // MFL player ID, string, may carry leading zeros
	Score string // YTD fantasy points in the league's own scoring, raw ("476.75")
	Week  string // echo of the requested window; "YTD" for the season aggregate
}

// Validate checks the raw record's SHAPE before anything downstream runs: the
// player ID must pass the ingestion boundary (RISK-003 site #1) and a present
// score must be parseable (it is kept as a string regardless). An empty score is
// rejected — MFL omits unscored players from the export rather than sending
// blanks, so a blank here is a malformed record, not a legitimate zero.
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

// scoresEnvelope mirrors the MFL playerScores JSON. Unknown fields are tolerated
// (external API, the internal/schema unknown-field policy); correctness comes from
// Validate asserting the fields we depend on. playerScore[] uses MFLList so the
// single-element collapse cannot crash the decode.
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

// Fetch retrieves the YTD fantasy-point totals for scoreYear through the league's
// host and returns shape-validated RawScore records. year is the league/path year
// (canonical ingestion.SeasonYear), scoreYear the season whose completed totals
// are wanted — they differ on purpose (see the package comment). It discovers the
// league host FIRST (the ingestion layer owns "discover before any league-specific
// call") and checks the MFL HTTP-200 error envelope BEFORE decoding: this is a
// money-path-adjacent feed (it becomes every player's BasePoints), so an outage
// payload must fail loud, never read as "no scores".
func Fetch(ctx context.Context, c *mfl.Client, year, leagueID, scoreYear string) ([]RawScore, error) {
	if strings.TrimSpace(scoreYear) == "" {
		return nil, fmt.Errorf("playerscores: score year is required")
	}
	if err := c.DiscoverHost(ctx, year, leagueID); err != nil {
		return nil, fmt.Errorf("playerscores: discover host: %w", err)
	}

	resp, err := c.Do(ctx, mfl.Request{
		Type: "playerScores",
		Year: year,
		Params: map[string]string{
			"L":    leagueID,
			"W":    "YTD",
			"YEAR": scoreYear,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("playerscores: fetch: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("playerscores: unexpected status %d", resp.StatusCode)
	}
	if err := ingestion.CheckAPIError(resp.Body); err != nil {
		return nil, fmt.Errorf("playerscores: %w", err)
	}

	var env scoresEnvelope
	if err := json.Unmarshal(resp.Body, &env); err != nil {
		return nil, fmt.Errorf("playerscores: decode: %w", err)
	}

	out, err := flatten(ctx, env)
	if err != nil {
		return nil, err
	}
	return guardNonEmpty(out)
}

// guardNonEmpty is the empty-payload gate, run AFTER the aggregate filter (GLM
// M1 review): a payload of nothing-but-aggregates would pass a pre-filter length
// check and still hand the caller zero player scores — exactly the silent
// zero-BasePoints board errEmptyScores exists to prevent.
func guardNonEmpty(out []RawScore) ([]RawScore, error) {
	if len(out) == 0 {
		return nil, errEmptyScores
	}
	return out, nil
}

// flatten walks the decoded envelope into RawScore records, dropping team-
// aggregate IDs (Coach/Def/ST/Off blocks score points in this league, but they are
// not players and must not enter the player ranking) and validating every
// surviving record's shape. A malformed real record fails LOUD rather than being
// silently dropped — only the known aggregate block is filtered. It honors ctx
// cancellation so a shutdown mid-parse returns promptly.
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
		// The fetcher is the only party that knows it asked for the YTD aggregate
		// (GLM M1 review): a single-week echo here would be ~17× off as a season
		// total and every consumer would silently rank on it. Assert the window.
		if rs.Week != "YTD" {
			return nil, fmt.Errorf("playerscores: record %s echoes week %q, want the requested YTD aggregate", rs.ID, rs.Week)
		}
		out = append(out, rs)
	}
	return out, nil
}
