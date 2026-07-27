// Package leagueschedule is the Layer 1 fetcher for MFL's `schedule` export — the
// FANTASY LEAGUE'S OWN weekly matchups (who plays whom, real franchise ids), distinct
// from the already-built `internal/ingestion/schedule` package (MFL's `nflSchedule`,
// the real-NFL kickoff/game schedule). Like `league`/`rules`, this is a league-scoped
// call: it discovers the host first, then issues one request carrying the league id.
// Omitting the optional week param (`W`) returns the whole season's schedule in one
// call. It transforms nothing — every value stays MFL's raw string; the App seam
// resolves franchise ids to display names and formats for the frontend.
package leagueschedule

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
)

// errEmptySchedule guards the glitch/error shape where the schedule export decodes to
// zero weeks. A live league always has a full-season schedule once one has been
// entered in MFL, so an empty response is treated as a fetch failure, not "no games."
var errEmptySchedule = errors.New("leagueschedule: response contained zero weeks")

// RawMatchupSide is one franchise's side of a matchup exactly as MFL returns it.
// IsHome stays the raw "0"/"1" string (converting it is the App seam's job, mirroring
// how the nflSchedule fetcher keeps IsHome raw); Score is empty until the week plays.
type RawMatchupSide struct {
	FranchiseID string
	IsHome      string
	Score       string
}

// RawMatchup is one scheduled game: exactly two franchise sides.
type RawMatchup struct {
	Franchises [2]RawMatchupSide
}

// RawScheduleWeek is one week's full slate of matchups, raw week number as a string
// (MFL convention — every fetcher in this codebase keeps numeric fields as strings
// until normalize/the App seam parses them).
type RawScheduleWeek struct {
	Week     string
	Matchups []RawMatchup
}

// Validate checks a week's SHAPE: a non-empty week number, and every matchup (if any) carrying
// exactly two franchise sides with non-empty ids and EXACTLY ONE side marked isHome="1" (the
// other "0"). The one-home-one-away check matters beyond a bare "0"/"1" format check: the App
// seam (GetLeagueSchedule) trusts this invariant to decide which side is home with a single
// `if isHome == "0" { swap }` — a matchup where MFL returned both sides "1" (or both "0") would
// silently mislabel a team's home/away status rather than fail loud (DeepSeek review finding,
// blind pass). It converts nothing.
//
// A week with ZERO matchups is NOT an error — confirmed live (2026-07-27): a future playoff
// week whose bracket hasn't been seeded yet (pending final regular-season standings) legitimately
// has no matchups. Rejecting it would fail the ENTIRE season's fetch over one unseeded week,
// exactly the outage-shaped failure this fetcher exists to avoid triggering needlessly.
func (w RawScheduleWeek) Validate() error {
	if strings.TrimSpace(w.Week) == "" {
		return fmt.Errorf("leagueschedule: week missing its number")
	}
	for _, m := range w.Matchups {
		homeCount := 0
		for _, side := range m.Franchises {
			if strings.TrimSpace(side.FranchiseID) == "" {
				return fmt.Errorf("leagueschedule: week %s has a matchup with an empty franchise id", w.Week)
			}
			h := strings.TrimSpace(side.IsHome)
			if h != "0" && h != "1" {
				return fmt.Errorf("leagueschedule: week %s franchise %s isHome %q is not 0/1",
					w.Week, side.FranchiseID, side.IsHome)
			}
			if h == "1" {
				homeCount++
			}
		}
		if homeCount != 1 {
			return fmt.Errorf(
				"leagueschedule: week %s has a matchup where %d of 2 sides are marked home (want exactly 1)",
				w.Week, homeCount)
		}
	}
	return nil
}

type scheduleEnvelope struct {
	Schedule struct {
		WeeklySchedule ingestion.MFLList[weekBlock] `json:"weeklySchedule"`
	} `json:"schedule"`
}

type weekBlock struct {
	Week    string                          `json:"week"`
	Matchup ingestion.MFLList[matchupBlock] `json:"matchup"`
}

type matchupBlock struct {
	Franchise ingestion.MFLList[franchiseBlock] `json:"franchise"`
}

type franchiseBlock struct {
	ID     string `json:"id"`
	IsHome string `json:"isHome"`
	Score  string `json:"score"`
}

// Fetch retrieves the league's full-season matchup schedule from MFL: it discovers
// the host (league-specific calls route to the league's home server), issues one
// `schedule` request scoped by league id (no `W`, so MFL returns every week), guards
// MFL's HTTP-200 error envelope, and returns shape-validated RawScheduleWeek records.
func Fetch(ctx context.Context, c *mfl.Client, year, leagueID string) ([]RawScheduleWeek, error) {
	if err := c.DiscoverHost(ctx, year, leagueID); err != nil {
		return nil, fmt.Errorf("leagueschedule: discover host: %w", err)
	}

	resp, err := c.Do(ctx, mfl.Request{
		Type:   "schedule",
		Year:   year,
		Params: map[string]string{"L": leagueID},
	})
	if err != nil {
		return nil, fmt.Errorf("leagueschedule: fetch: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("leagueschedule: unexpected status %d", resp.StatusCode)
	}
	if err := ingestion.CheckAPIError(resp.Body); err != nil {
		return nil, fmt.Errorf("leagueschedule: %w", err)
	}

	var env scheduleEnvelope
	if err := json.Unmarshal(resp.Body, &env); err != nil {
		return nil, fmt.Errorf("leagueschedule: decode: %w", err)
	}
	if len(env.Schedule.WeeklySchedule) == 0 {
		return nil, errEmptySchedule
	}

	return flatten(ctx, env)
}

// flatten walks the decoded envelope into RawScheduleWeek records, validating each
// one's shape. A malformed week fails loud, consistent with every sibling fetcher.
func flatten(ctx context.Context, env scheduleEnvelope) ([]RawScheduleWeek, error) {
	out := make([]RawScheduleWeek, 0, len(env.Schedule.WeeklySchedule))
	for _, wb := range env.Schedule.WeeklySchedule {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("leagueschedule: flatten cancelled: %w", ctx.Err())
		default:
		}

		matchups := make([]RawMatchup, 0, len(wb.Matchup))
		for _, mb := range wb.Matchup {
			if len(mb.Franchise) != 2 {
				return nil, fmt.Errorf("leagueschedule: week %s has a matchup with %d franchises, want 2",
					wb.Week, len(mb.Franchise))
			}
			matchups = append(matchups, RawMatchup{
				Franchises: [2]RawMatchupSide{
					{FranchiseID: mb.Franchise[0].ID, IsHome: mb.Franchise[0].IsHome, Score: mb.Franchise[0].Score},
					{FranchiseID: mb.Franchise[1].ID, IsHome: mb.Franchise[1].IsHome, Score: mb.Franchise[1].Score},
				},
			})
		}

		w := RawScheduleWeek{Week: wb.Week, Matchups: matchups}
		if err := w.Validate(); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, nil
}
