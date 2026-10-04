// Package players fetches MFL's player database as raw records. It uses the league-scoped feed:
// the global one omits commissioner-created players (live ids 0816, 0820, 0835, 0838), which
// rosters reference. MFL allows this call once a day. Aggregates are filtered by position in
// normalize.
package players

import (
	"context"
	"encoding/json"
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
	ID        string // "0531"
	Name      string // "Last, First"
	Position  string // raw MFL code ("PK", "EDGE", "XX")
	Team      string // 3-letter NFL code, or "FA" for free agent
	Status    string // "R" for a rookie
	Birthdate string // epoch seconds; empty for commissioner-created players
	DraftYear string // §6 experience; "0" and "1970" mean undrafted
	// College feeds SchoolTier; MFL's names differ from CFBD's ("Miami (FL)" vs "Miami").
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
	return records(ctx, env)
}

// Parse reads a players export body as MFL sent it, such as an archived copy, with the checks
// Fetch applies.
func Parse(ctx context.Context, body []byte) ([]RawPlayer, error) {
	var env playersEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("players: decode: %w", err)
	}
	return records(ctx, env)
}

func records(ctx context.Context, env playersEnvelope) ([]RawPlayer, error) {
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
