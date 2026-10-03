// Package players fetches MFL's player database (id → name, position, team, rookie flag) as
// raw records; normalize maps the codes.
//
// It uses the league-scoped feed, not the global api one, because the global feed omits
// commissioner-created players: owners can bid on players MFL does not have yet, and the
// commissioner creates them locally (live ids 0816, 0820, 0835, 0838). Rosters reference those
// ids, so only the league feed resolves every rostered player. MFL allows this call once a day;
// caching is the caller's job.
//
// Aggregates are not filtered here: this feed carries position, so normalize filters them by
// position in one place.
package players

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
)

// errEmptyPlayers: the player database is never empty, and an empty lookup would make every
// roster join silently "unknown".
var errEmptyPlayers = errors.New("players: response contained zero players")

// RawPlayer is one player row as MFL sends it.
type RawPlayer struct {
	ID       string // "0531"
	Name     string // "Last, First"
	Position string // raw MFL code ("PK", "EDGE", "XX")
	Team     string // 3-letter NFL code, or "FA" for free agent
	Status   string // "R" for a rookie
	// Birthdate is epoch seconds as a string, or empty (commissioner-created and some deep
	// rows lack it). The age consumer decides what absent means.
	Birthdate string
	// DraftYear feeds the §6 minimum-salary experience count. MFL uses "0" and "1970" for
	// undrafted, so present is not necessarily real; the consumer judges.
	DraftYear string
	// College feeds SchoolTier. MFL omits it for team-defense and coach rows and about 15% of
	// players, and its names differ from CFBD's ("Miami (FL)" vs "Miami"); the scouting join
	// reconciles them.
	College string
}

// Validate checks shape only: a valid player id and a present position. Name and team are
// checked in normalize, after aggregates (which legitimately lack names) are filtered.
func (p RawPlayer) Validate() error {
	if _, err := ingestion.ValidatePlayerID(p.ID); err != nil {
		return fmt.Errorf("players: %w", err)
	}
	if strings.TrimSpace(p.Position) == "" {
		return fmt.Errorf("players: record %s (%q) missing position", p.ID, p.Name)
	}
	// A present birthdate must parse as epoch seconds.
	if bd := strings.TrimSpace(p.Birthdate); bd != "" {
		if _, err := strconv.ParseInt(bd, 10, 64); err != nil {
			return fmt.Errorf("players: record %s (%q) non-numeric birthdate %q: %w", p.ID, p.Name, p.Birthdate, err)
		}
	}
	// A present draft year must parse; the sentinels are numeric and pass.
	if dy := strings.TrimSpace(p.DraftYear); dy != "" {
		if _, err := strconv.Atoi(dy); err != nil {
			return fmt.Errorf("players: record %s (%q) non-numeric draft year %q: %w", p.ID, p.Name, p.DraftYear, err)
		}
	}
	return nil
}

// playersEnvelope mirrors the MFL players JSON; unknown fields are tolerated.
type playersEnvelope struct {
	Players struct {
		Player ingestion.MFLList[playerBlock] `json:"player"`
	} `json:"players"`
}

type playerBlock struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Position  string `json:"position"`
	Team      string `json:"team"`
	Status    string `json:"status"`
	Birthdate string `json:"birthdate"`
	DraftYear string `json:"draft_year"`
	College   string `json:"college"`
}

// Fetch discovers the league host, then fetches the league-scoped player database and returns
// shape-validated records.
func Fetch(ctx context.Context, c *mfl.Client, year, leagueID string) ([]RawPlayer, error) {
	env, err := ingestion.FetchLeagueExport[playersEnvelope](ctx, c, "players", year, leagueID, map[string]string{"DETAILS": "1"})
	if err != nil {
		return nil, fmt.Errorf("players: %w", err)
	}
	if len(env.Players.Player) == 0 {
		return nil, errEmptyPlayers
	}
	return flatten(ctx, env)
}

// flatten validates every record; a malformed one fails the fetch rather than being dropped.
// It honors ctx so a shutdown stops promptly.
func flatten(ctx context.Context, env playersEnvelope) ([]RawPlayer, error) {
	out := make([]RawPlayer, 0, len(env.Players.Player))
	for _, pb := range env.Players.Player {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("players: flatten cancelled: %w", ctx.Err())
		default:
		}

		rp := RawPlayer(pb)
		if err := rp.Validate(); err != nil {
			return nil, err
		}
		out = append(out, rp)
	}
	return out, nil
}
