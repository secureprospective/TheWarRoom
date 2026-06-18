// Package schedule is the Layer 1 fetcher for the MFL nflSchedule endpoint
// (WF 1B). It clones the reviewed rosters pattern: fetch → schema-validate the
// SHAPE → return RAW records, transforming nothing (isHome stays "0"/"1", scores
// stay raw strings — normalize interprets them at B3).
//
// Unlike rosters, nflSchedule is a NON-league call: it targets the load-balanced
// `api` host and carries no league ID, so there is no DiscoverHost step (the
// transport routes to `api` when no L param is present). It also has no player
// IDs, so the player-ID/team-aggregate boundary helpers do not apply here — the
// only shared inheritance is ingestion.MFLList (the array/object-collapse decoder).
package schedule

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

// errEmptySchedule guards a glitch/empty payload (zero matchups). The NFL schedule
// is never legitimately empty for a season, so we surface it rather than return an
// empty slice a caller might mistake for "no games scheduled".
var errEmptySchedule = errors.New("schedule: response contained zero matchups")

// RawScheduleTeam is one side of a matchup exactly as MFL returns it. IsHome is
// kept as the raw "0"/"1" string; converting it to a home/away semantic is
// normalize's job (the fetcher transforms nothing).
type RawScheduleTeam struct {
	ID     string // NFL team code ("NEP", "SEA"), NOT a player ID
	IsHome string // "0" or "1", raw
	Score  string // raw; empty until games are played
	Spread string // raw point spread
}

// RawScheduleMatchup is one scheduled game: a kickoff time and exactly two teams.
type RawScheduleMatchup struct {
	Kickoff              string // unix timestamp, raw string
	GameSecondsRemaining string // raw
	Teams                []RawScheduleTeam
}

// Validate checks the matchup's SHAPE: a parseable kickoff timestamp and exactly
// two teams, each with a non-empty code and a "0"/"1" isHome flag. It converts
// nothing — kickoff stays a string for normalize to parse at B3.
func (m RawScheduleMatchup) Validate() error {
	if strings.TrimSpace(m.Kickoff) == "" {
		return fmt.Errorf("schedule: matchup missing kickoff")
	}
	if _, err := strconv.ParseInt(strings.TrimSpace(m.Kickoff), 10, 64); err != nil {
		return fmt.Errorf("schedule: kickoff %q is not a unix timestamp: %w", m.Kickoff, err)
	}
	if len(m.Teams) != 2 {
		return fmt.Errorf("schedule: matchup (kickoff %s) has %d teams, want 2", m.Kickoff, len(m.Teams))
	}
	for _, t := range m.Teams {
		if strings.TrimSpace(t.ID) == "" {
			return fmt.Errorf("schedule: matchup (kickoff %s) has a team with empty id", m.Kickoff)
		}
		if h := strings.TrimSpace(t.IsHome); h != "0" && h != "1" {
			return fmt.Errorf("schedule: team %s isHome %q is not 0/1", t.ID, t.IsHome)
		}
	}
	return nil
}

type scheduleEnvelope struct {
	NFLSchedule struct {
		Matchup ingestion.MFLList[matchupBlock] `json:"matchup"`
	} `json:"nflSchedule"`
}

type matchupBlock struct {
	Kickoff              string                       `json:"kickoff"`
	GameSecondsRemaining string                       `json:"gameSecondsRemaining"`
	Team                 ingestion.MFLList[teamBlock] `json:"team"`
}

type teamBlock struct {
	ID     string `json:"id"`
	IsHome string `json:"isHome"`
	Score  string `json:"score"`
	Spread string `json:"spread"`
}

// Fetch retrieves the NFL schedule for the given season from MFL and returns
// shape-validated RawScheduleMatchup records. It is a non-league call (no league
// ID, no host discovery): the transport routes a request with no L param to the
// load-balanced `api` host. Rate-limiting and 429 backoff are inherited.
func Fetch(ctx context.Context, c *mfl.Client, year string) ([]RawScheduleMatchup, error) {
	resp, err := c.Do(ctx, mfl.Request{
		Type: "nflSchedule",
		Year: year,
	})
	if err != nil {
		return nil, fmt.Errorf("schedule: fetch: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("schedule: unexpected status %d", resp.StatusCode)
	}

	var env scheduleEnvelope
	if err := json.Unmarshal(resp.Body, &env); err != nil {
		return nil, fmt.Errorf("schedule: decode: %w", err)
	}
	if len(env.NFLSchedule.Matchup) == 0 {
		return nil, errEmptySchedule
	}

	return flatten(ctx, env)
}

// flatten walks the decoded envelope into RawScheduleMatchup records, validating
// each one's shape. A malformed matchup fails loud (consistent with rosters).
func flatten(ctx context.Context, env scheduleEnvelope) ([]RawScheduleMatchup, error) {
	out := make([]RawScheduleMatchup, 0, len(env.NFLSchedule.Matchup))
	for _, mb := range env.NFLSchedule.Matchup {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("schedule: flatten cancelled: %w", ctx.Err())
		default:
		}

		teams := make([]RawScheduleTeam, 0, len(mb.Team))
		for _, tb := range mb.Team {
			teams = append(teams, RawScheduleTeam(tb))
		}

		m := RawScheduleMatchup{
			Kickoff:              mb.Kickoff,
			GameSecondsRemaining: mb.GameSecondsRemaining,
			Teams:                teams,
		}
		if err := m.Validate(); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}
