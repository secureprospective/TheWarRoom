// Package freeagents fetches MFL's free-agent list: the players on no roster, which is the pool
// the bid plans are made from. It is a league export, so the request carries L.
package freeagents

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// Export is the MFL export name; the archived fallback keys on it.
const Export = "freeAgents"

// RawFreeAgent is one free agent as MFL sends it: only the player id; names, positions and
// ages come from the players directory.
type RawFreeAgent struct {
	ID string // MFL player id, leading zeros kept
}

// Validate checks the id, so a malformed one fails the fetch rather than joining the pool.
func (r RawFreeAgent) Validate() error {
	if _, err := ingestion.ValidatePlayerID(r.ID); err != nil {
		return fmt.Errorf("freeagents: %w", err)
	}
	return nil
}

// freeAgentsEnvelope mirrors the MFL JSON. A league with divisions lists one unit each; a
// single unit and a single player arrive as bare objects, which MFLList reads.
type freeAgentsEnvelope struct {
	FreeAgents struct {
		LeagueUnit ingestion.MFLList[struct {
			Player ingestion.MFLList[struct {
				ID string `json:"id"`
			}] `json:"player"`
		}] `json:"leagueUnit"`
	} `json:"freeAgents"`
}

// Fetch discovers the league host, then returns the free agents, validated. An empty pool is
// a valid answer: a league can have no free agents.
func Fetch(ctx context.Context, c *mfl.Client, year, leagueID string) ([]RawFreeAgent, error) {
	if err := c.DiscoverHost(ctx, year, leagueID); err != nil {
		return nil, fmt.Errorf("freeagents: discover host: %w", err)
	}
	body, err := ingestion.LeagueExport(ctx, c, Export, year, leagueID, nil)
	if err != nil {
		return nil, fmt.Errorf("freeagents: %w", err)
	}
	return Parse(body)
}

// Parse reads an export body as MFL sent it, such as an archived copy. MFL's HTTP-200 error
// envelope is named, never read as an empty pool; a body with no freeAgents section at all is
// refused too, since that is how an unrecognised reply looks.
func Parse(body []byte) ([]RawFreeAgent, error) {
	if err := ingestion.CheckAPIError(body); err != nil {
		return nil, fmt.Errorf("freeagents: %w", err)
	}
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(body, &probe); err != nil {
		return nil, fmt.Errorf("freeagents: decode: %w", err)
	}
	if _, ok := probe["freeAgents"]; !ok {
		return nil, fmt.Errorf("freeagents: response has no freeAgents section")
	}
	var env freeAgentsEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("freeagents: decode: %w", err)
	}
	out := []RawFreeAgent{}
	for _, unit := range env.FreeAgents.LeagueUnit {
		for _, p := range unit.Player {
			raw := RawFreeAgent{ID: p.ID}
			if err := raw.Validate(); err != nil {
				return nil, err
			}
			out = append(out, raw)
		}
	}
	return out, nil
}

// ToPlayerIDs types the raw rows. A duplicate id is dropped, not an error: a division league
// can list a player in more than one unit.
func ToPlayerIDs(raw []RawFreeAgent) ([]playerid.PlayerID, error) {
	seen := make(map[string]bool, len(raw))
	out := make([]playerid.PlayerID, 0, len(raw))
	for _, r := range raw {
		id, err := playerid.New(r.ID)
		if err != nil {
			return nil, fmt.Errorf("freeagents: record %s: %w", r.ID, err)
		}
		if !seen[id.String()] {
			seen[id.String()] = true
			out = append(out, id)
		}
	}
	return out, nil
}
