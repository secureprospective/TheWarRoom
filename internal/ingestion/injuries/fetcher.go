// Package injuries fetches MFL's NFL injury designations: one status per player, as his NFL
// team classified him. It is a league-less export, like nflSchedule: the request carries no L,
// so the client sends it to api.myfantasyleague.com, the only host MFL serves it from — a
// league host answers with MFL's HTTP-200 error envelope. The IR and taxi rules read it.
package injuries

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// Export is the MFL export name; the archived fallback keys on it.
const Export = "injuries"

// errEmptyInjuries: the league-wide injury report is never empty in season. A zero-row body is
// a decode or upstream glitch, and reading it as "nobody is injured" would mark every player
// IR-ineligible — a wrong verdict — so it fails the fetch and the fallback serves the last
// good copy instead.
var errEmptyInjuries = errors.New("injuries: response contained zero players")

// RawInjury is one injury row as MFL sends it. Status is MFL's open vocabulary; the
// eligibility rules classify it. Details and ExpReturn are display text, never parsed.
type RawInjury struct {
	ID        string // MFL player id, leading zeros kept
	Status    string // "IR", "IR-R", "IR-PUP", "IR-NFI", "Out", "Suspended", …
	Details   string // e.g. "Knee - PCL", display only
	ExpReturn string // e.g. "Feb 15, 2027", display only
}

// Validate checks shape: a valid player id and a non-empty status, so an eligibility rule can
// never read a blank designation as a real one.
func (r RawInjury) Validate() error {
	if _, err := ingestion.ValidatePlayerID(r.ID); err != nil {
		return fmt.Errorf("injuries: %w", err)
	}
	if strings.TrimSpace(r.Status) == "" {
		return fmt.Errorf("injuries: record %s missing status", r.ID)
	}
	return nil
}

// injuriesEnvelope mirrors the MFL injuries JSON; unknown fields are tolerated. MFLList
// handles a one-row response, which arrives as a bare object.
type injuriesEnvelope struct {
	Injuries struct {
		Injury ingestion.MFLList[injuryBlock] `json:"injury"`
	} `json:"injuries"`
}

type injuryBlock struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	Details   string `json:"details"`
	ExpReturn string `json:"exp_return"`
}

// Fetch reads the league-wide injury report from the api host. The request carries no L, so
// the client routes it to api.myfantasyleague.com whatever host the league uses; no host
// discovery is needed. The body is parsed like an archived copy, so both paths name MFL's
// error envelope identically.
func Fetch(ctx context.Context, c *mfl.Client, year string) ([]RawInjury, error) {
	resp, err := c.Do(ctx, mfl.Request{Type: Export, Year: year})
	if err != nil {
		return nil, fmt.Errorf("injuries: fetch: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("injuries: unexpected status %d", resp.StatusCode)
	}
	return Parse(ctx, resp.Body)
}

// Parse reads an injuries export body as MFL sent it, such as an archived copy, with the
// checks Fetch applies. MFL's error envelope is named, not decoded into zero rows: MFL often
// returns HTTP 200 with {"error":{"$t":…}}, and reading that as "nobody is injured" would
// mark every player IR-ineligible. ctx is honored so a shutdown stops promptly.
func Parse(ctx context.Context, body []byte) ([]RawInjury, error) {
	if err := ingestion.CheckAPIError(body); err != nil {
		return nil, fmt.Errorf("injuries: %w", err)
	}
	var env injuriesEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("injuries: decode: %w", err)
	}
	return flatten(ctx, env)
}

// flatten validates every record; a malformed one fails the fetch rather than being dropped.
func flatten(ctx context.Context, env injuriesEnvelope) ([]RawInjury, error) {
	if len(env.Injuries.Injury) == 0 {
		return nil, errEmptyInjuries
	}
	out := make([]RawInjury, 0, len(env.Injuries.Injury))
	for _, ib := range env.Injuries.Injury {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("injuries: flatten cancelled: %w", ctx.Err())
		default:
		}
		ri := RawInjury(ib)
		if err := ri.Validate(); err != nil {
			return nil, err
		}
		out = append(out, ri)
	}
	return out, nil
}

// ToInjuries types the raw rows. A duplicate id is a shape error: two designations for one
// player would silently pick a winner; the fetch fails instead.
func ToInjuries(raw []RawInjury) ([]leaguefeed.Injury, error) {
	seen := make(map[string]bool, len(raw))
	out := make([]leaguefeed.Injury, 0, len(raw))
	for _, r := range raw {
		id, err := playerid.New(r.ID)
		if err != nil {
			return nil, fmt.Errorf("injuries: record %s: %w", r.ID, err)
		}
		if seen[id.String()] {
			return nil, fmt.Errorf("injuries: duplicate record %s", id.String())
		}
		seen[id.String()] = true
		out = append(out, leaguefeed.Injury{
			ID: id, Status: strings.TrimSpace(r.Status),
			Details: strings.TrimSpace(r.Details), ExpReturn: strings.TrimSpace(r.ExpReturn),
		})
	}
	return out, nil
}

// ParseToInjuries is the archived-body path Parse callers use before typing.
func ParseToInjuries(ctx context.Context, body []byte) ([]leaguefeed.Injury, error) {
	raw, err := Parse(ctx, body)
	if err != nil {
		return nil, err
	}
	return ToInjuries(raw)
}
