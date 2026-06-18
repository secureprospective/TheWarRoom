// Package rosters is the Layer 1 fetcher for the MFL rosters endpoint (WF 1B).
// One exported Fetch returns RAW records: it fetches and validates the response
// SHAPE and transforms nothing (salary stays a raw string, contractStatus stays
// dirty — normalize handles both at B3). This file is the TEMPLATE every other
// fetcher clones, so its structure is deliberate: build Request → c.Do →
// schema-validate → return raw, with no domain knowledge and no upward imports.
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

// errEmptyRosters guards the MFL glitch shape {"rosters":{}} (or a null payload),
// which decodes to zero franchises. A 32-team league never legitimately returns no
// franchises, so we surface it as an error rather than an empty slice — an empty
// slice downstream would read as "every team dropped every player" and could wipe
// local state.
var errEmptyRosters = errors.New("rosters: response contained zero franchises")

// RawRoster is one player's roster row exactly as MFL returns it, flattened with
// its parent franchise ID. Every field is a raw string — MFL encodes numbers as
// strings, and a fetcher transforms nothing (WF 1B). ContractStatus is kept in
// its dirty form ("UFA ", "YFA", "EXT (2024)"); normalize cleans it at B3.
type RawRoster struct {
	FranchiseID    string // parent franchise, "0001"–"0032"
	PlayerID       string // MFL player ID, string, may carry leading zeros
	Salary         string // millions, raw ("7", "1.30", "17.70"); parsed at B3
	ContractYear   string // final contract year ("2026"); parsed at B3
	ContractStatus string // DIRTY (UFA/RFA/FT1/FT2 + variants); normalized at B3
	ContractInfo   string // free-text notes; DISPLAY ONLY, never parsed
	Status         string // "ROSTER" or "TAXI_SQUAD"
}

// Validate checks the raw record's SHAPE before anything downstream runs. It does
// not convert types (normalize's job) — it only rejects records that cannot be
// valid: the player ID must pass the ingestion boundary (RISK-003 site #1), and a
// present salary must be parseable (it is kept as a string regardless).
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

// rostersEnvelope mirrors the MFL rosters JSON. Unknown fields are tolerated (MFL
// is an external API we do not control and adds fields without versioning — the
// internal/schema unknown-field policy); correctness comes from Validate asserting
// the fields we depend on, not from rejecting fields we ignore.
//
// MFL collapses a single-element array to a bare object, but a 32-team dynasty
// league always returns franchise[] as an array of 32 and each 80-man roster's
// player[] as an array, so the array decode is safe here.
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

// Fetch retrieves all franchises' rosters for the given season+league from MFL and
// returns flattened, shape-validated RawRoster records. year and leagueID are
// explicit arguments (not package globals) so the fetcher is unit-testable and
// supports multi-league/season rollover without a recompile; ingestion.SeasonYear
// and ingestion.LeagueID are the canonical Phase-1 values a caller passes. It
// discovers the league host FIRST (the ingestion layer owns "discover before any
// league-specific call" — the Q2.1 deferral from B1), then issues one rosters call
// through the transport client: rate-limiting, 429 backoff, and host routing are
// inherited, never re-implemented.
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

// flatten walks the decoded envelope into RawRoster records, dropping team-
// aggregate IDs before they pollute downstream and validating every surviving
// record's shape. A malformed real record fails LOUD (returns an error) rather
// than being silently dropped — only the known aggregate block is filtered. It
// honors ctx cancellation between franchises so a shutdown mid-parse returns
// promptly instead of blocking CPU until the loop finishes.
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
