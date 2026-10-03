// Package rosters fetches every franchise's roster from MFL as raw records: salary stays a
// string and contractStatus stays dirty for normalize to clean.
package rosters

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

// errEmptyRosters guards MFL's {"rosters":{}} glitch. Read as data, it would mean every team
// dropped every player and could wipe local state.
var errEmptyRosters = errors.New("rosters: response contained zero franchises")

// RawRoster is one roster row as MFL sends it, with its franchise id.
type RawRoster struct {
	FranchiseID    string // parent franchise, "0001"–"0032"
	PlayerID       string
	Salary         string // millions, raw ("7", "1.30")
	ContractYear   string // final contract year ("2026")
	ContractStatus string // dirty ("UFA ", "YFA", "EXT (2024)")
	ContractInfo   string // free text, display only
	Status         string // "ROSTER", "TAXI_SQUAD" or "IR"
}

// Validate rejects an invalid player id or a present salary that does not parse.
func (r RawRoster) Validate() error {
	if strings.TrimSpace(r.FranchiseID) == "" {
		return fmt.Errorf("rosters: record missing franchise id")
	}
	if _, err := ingestion.ValidatePlayerID(r.PlayerID); err != nil {
		return fmt.Errorf("rosters: franchise %s: %w", r.FranchiseID, err)
	}
	if strings.TrimSpace(r.Status) == "" {
		return fmt.Errorf("rosters: record %s/%s missing status", r.FranchiseID, r.PlayerID)
	}
	if s := strings.TrimSpace(r.Salary); s != "" {
		if _, err := strconv.ParseFloat(s, 64); err != nil {
			return fmt.Errorf("rosters: record %s/%s non-numeric salary %q: %w", r.FranchiseID, r.PlayerID, r.Salary, err)
		}
	}
	return nil
}

// rostersEnvelope mirrors the MFL rosters JSON; unknown fields are tolerated. A full league
// always returns arrays here, so the single-element collapse cannot occur.
type rostersEnvelope struct {
	Rosters struct {
		Franchise ingestion.MFLList[franchiseBlock] `json:"franchise"`
	} `json:"rosters"`
}

type franchiseBlock struct {
	ID     string                         `json:"id"`
	Player ingestion.MFLList[playerBlock] `json:"player"`
}

type playerBlock struct {
	ID             string `json:"id"`
	Salary         string `json:"salary"`
	ContractYear   string `json:"contractYear"`
	ContractStatus string `json:"contractStatus"`
	ContractInfo   string `json:"contractInfo"`
	Status         string `json:"status"`
}

// Fetch discovers the league host, then returns every roster row, shape-validated.
func Fetch(ctx context.Context, c *mfl.Client, year, leagueID string) ([]RawRoster, error) {
	if err := c.DiscoverHost(ctx, year, leagueID); err != nil {
		return nil, fmt.Errorf("rosters: discover host: %w", err)
	}

	resp, err := c.Do(ctx, mfl.Request{
		Type:   "rosters",
		Year:   year,
		Params: map[string]string{"L": leagueID},
	})
	if err != nil {
		return nil, fmt.Errorf("rosters: fetch: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("rosters: unexpected status %d", resp.StatusCode)
	}

	var env rostersEnvelope
	if err := json.Unmarshal(resp.Body, &env); err != nil {
		return nil, fmt.Errorf("rosters: decode: %w", err)
	}
	if len(env.Rosters.Franchise) == 0 {
		return nil, errEmptyRosters
	}

	return flatten(ctx, env)
}

// flatten drops team-aggregate ids and validates the rest; a malformed real record fails the
// fetch rather than being dropped. It honors ctx between franchises.
func flatten(ctx context.Context, env rostersEnvelope) ([]RawRoster, error) {
	var out []RawRoster
	for _, f := range env.Rosters.Franchise {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("rosters: flatten cancelled: %w", ctx.Err())
		default:
		}
		for _, p := range f.Player {
			if ingestion.IsTeamAggregateID(p.ID) {
				continue
			}
			rr := RawRoster{
				FranchiseID:    f.ID,
				PlayerID:       p.ID,
				Salary:         p.Salary,
				ContractYear:   p.ContractYear,
				ContractStatus: p.ContractStatus,
				ContractInfo:   p.ContractInfo,
				Status:         p.Status,
			}
			if err := rr.Validate(); err != nil {
				return nil, err
			}
			out = append(out, rr)
		}
	}
	return out, nil
}
