// Package nflschedule adapts MFL's league-independent dated NFL schedule.
package nflschedule

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
)

const Export = "nflSchedule"

type RawMatchup struct {
	Teams                [2]string
	Kickoff              string
	GameSecondsRemaining string
}

type RawWeek struct {
	Week     string
	Matchups []RawMatchup
}

func (w RawWeek) Validate() error {
	number, err := strconv.Atoi(w.Week)
	if err != nil {
		return fmt.Errorf("nflschedule: parse week %q: %w", w.Week, err)
	}
	if number < 1 || len(w.Matchups) == 0 {
		return fmt.Errorf("nflschedule: week %q must be positive and nonempty", w.Week)
	}
	for i, m := range w.Matchups {
		home, away := strings.TrimSpace(m.Teams[0]), strings.TrimSpace(m.Teams[1])
		if home == "" || away == "" || home == away {
			return fmt.Errorf("nflschedule: week %s matchup %d needs two distinct team IDs", w.Week, i)
		}
		if _, err := strconv.ParseInt(m.Kickoff, 10, 64); err != nil {
			return fmt.Errorf("nflschedule: week %s matchup %d kickoff: %w", w.Week, i, err)
		}
	}
	return nil
}

type scheduleEnvelope struct {
	Schedule struct {
		Week     string                          `json:"week"`
		Matchups ingestion.MFLList[matchupBlock] `json:"matchup"`
	} `json:"nflSchedule"`
}

type matchupBlock struct {
	Kickoff              string                       `json:"kickoff"`
	GameSecondsRemaining string                       `json:"gameSecondsRemaining"`
	Teams                ingestion.MFLList[teamBlock] `json:"team"`
}

type teamBlock struct {
	ID string `json:"id"`
}

// Fetch reads one NFL week; week 0 asks for MFL's default week. The request carries no league,
// so the client sends it to the api host, the only host that serves this export.
func Fetch(ctx context.Context, c *mfl.Client, year string, week int) (RawWeek, error) {
	if week < 0 {
		return RawWeek{}, fmt.Errorf("nflschedule: invalid requested week %d", week)
	}
	params := map[string]string{}
	if week != 0 {
		params["W"] = strconv.Itoa(week)
	}
	resp, err := c.Do(ctx, mfl.Request{Type: Export, Year: year, Params: params})
	if err != nil {
		return RawWeek{}, fmt.Errorf("nflschedule: fetch: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return RawWeek{}, fmt.Errorf("nflschedule: unexpected status %d", resp.StatusCode)
	}
	return Parse(resp.Body)
}

func Parse(body []byte) (RawWeek, error) {
	if err := ingestion.CheckAPIError(body); err != nil {
		return RawWeek{}, fmt.Errorf("nflschedule: %w", err)
	}
	var env scheduleEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return RawWeek{}, fmt.Errorf("nflschedule: decode: %w", err)
	}
	w := RawWeek{Week: env.Schedule.Week, Matchups: make([]RawMatchup, 0, len(env.Schedule.Matchups))}
	for i, m := range env.Schedule.Matchups {
		if len(m.Teams) != 2 {
			return RawWeek{}, fmt.Errorf("nflschedule: matchup %d has %d teams, want 2", i, len(m.Teams))
		}
		w.Matchups = append(w.Matchups, RawMatchup{
			Teams:                [2]string{m.Teams[0].ID, m.Teams[1].ID},
			Kickoff:              m.Kickoff,
			GameSecondsRemaining: m.GameSecondsRemaining,
		})
	}
	if err := w.Validate(); err != nil {
		return RawWeek{}, err
	}
	return w, nil
}

// ToWeek grades finality at the supplied instant, never from a future game's zero counter alone.
func ToWeek(raw RawWeek, at time.Time) (leagueweek.Week, error) {
	if err := raw.Validate(); err != nil {
		return leagueweek.Week{}, err
	}
	number, err := strconv.Atoi(raw.Week)
	if err != nil {
		return leagueweek.Week{}, fmt.Errorf("nflschedule: convert week: %w", err)
	}
	w := leagueweek.Week{Number: number, Games: make([]leagueweek.Game, 0, len(raw.Matchups))}
	for _, m := range raw.Matchups {
		seconds, err := strconv.ParseInt(m.Kickoff, 10, 64)
		if err != nil {
			return leagueweek.Week{}, fmt.Errorf("nflschedule: convert kickoff: %w", err)
		}
		kickoff := time.Unix(seconds, 0).UTC()
		w.Games = append(w.Games, leagueweek.Game{
			Teams:   m.Teams,
			Kickoff: kickoff,
			Final:   !at.Before(kickoff) && m.GameSecondsRemaining == "0",
		})
	}
	return w, nil
}
