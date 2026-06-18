// Package salaryadjustments is the Layer 1 fetcher for the MFL salaryAdjustments
// endpoint (WF 1B): the per-franchise dead-cap ledger. When a franchise drops a
// player who then goes unclaimed, MFL records a cap penalty here as a discrete
// entry; the franchise's total adjustment is the sum of its entries (the Cardinals'
// 0025 ledger summed to $5.495 on live data, OQ-005). Cap usage downstream is
// Σ(roster player salaries) + Σ(salaryAdjustments.amount) per franchise.
//
// It clones the reviewed rosters pattern: league-specific call, so it discovers
// the host FIRST (the ingestion layer owns "discover before any league-specific
// call"), then fetches and returns RAW records, transforming nothing (Amount stays
// a raw string; Description stays the free-text drop note — DISPLAY ONLY, never
// parsed).
//
// ONE deliberate divergence from rosters/players/schedule: there is NO empty-payload
// sentinel. An empty salaryAdjustments response is a LEGITIMATE state — a brand-new
// or disciplined franchise simply has no dead-cap. Unlike a 32-team roster or the
// player database (where empty means a glitch that would wipe/blank downstream
// state), zero adjustments is valid data, so Fetch returns an empty slice and no
// error.
package salaryadjustments

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
)

// RawSalaryAdjustment is one dead-cap ledger entry exactly as MFL returns it. Every
// field is a raw string: Amount is the penalty in millions (parsed to float at the
// cap-usage layer, not here), and Description is the human-readable drop note
// ("Dropped Kennedy, Tom DET WR (Salary: $0.58, ...)") kept for DISPLAY ONLY — it is
// never parsed for the player detail it happens to contain.
type RawSalaryAdjustment struct {
	FranchiseID string // owning franchise, "0001"–"0032"
	Amount      string // dead-cap penalty in millions, raw ("0.203"); parsed downstream
	Description string // free-text drop note; DISPLAY ONLY, never parsed
	ID          string // MFL adjustment record id
	Timestamp   string // unix timestamp of the drop, raw string
}

// Validate checks the raw record's SHAPE. It does not convert types, but it asserts
// the two fields cap math depends on: a present franchise id, and an Amount that
// actually parses as a number (a non-numeric amount is schema drift that must fail
// loud here, not silently zero out a franchise's dead-cap downstream). Timestamp is
// validated as a unix int when present (ordering/display); Description is free text
// and intentionally unchecked.
func (a RawSalaryAdjustment) Validate() error {
	if strings.TrimSpace(a.FranchiseID) == "" {
		return fmt.Errorf("salaryadjustments: record %s missing franchise id", a.ID)
	}
	// MFL franchise ids are 4-digit zero-padded numbers ("0001"–"0032"). A
	// non-numeric value ("WAIVERS", "SYS") is garbage that must not key into the
	// ledger map (Gemini B3 review). Negative/odd AMOUNTS are intentionally allowed
	// (commissioners issue negative credits) — only the franchise key is constrained.
	if _, err := strconv.Atoi(strings.TrimSpace(a.FranchiseID)); err != nil {
		return fmt.Errorf("salaryadjustments: record %s franchise id %q is not numeric: %w", a.ID, a.FranchiseID, err)
	}
	amt := strings.TrimSpace(a.Amount)
	if amt == "" {
		return fmt.Errorf("salaryadjustments: franchise %s record %s missing amount", a.FranchiseID, a.ID)
	}
	if _, err := strconv.ParseFloat(amt, 64); err != nil {
		return fmt.Errorf("salaryadjustments: franchise %s record %s non-numeric amount %q: %w", a.FranchiseID, a.ID, a.Amount, err)
	}
	if ts := strings.TrimSpace(a.Timestamp); ts != "" {
		if _, err := strconv.ParseInt(ts, 10, 64); err != nil {
			return fmt.Errorf("salaryadjustments: franchise %s record %s non-numeric timestamp %q: %w", a.FranchiseID, a.ID, a.Timestamp, err)
		}
	}
	return nil
}

// salaryAdjustmentsEnvelope mirrors the MFL salaryAdjustments JSON. Unknown fields
// are tolerated (external-API unknown-field policy). salaryAdjustment[] uses MFLList
// so the single-element collapse (a franchise with exactly one adjustment emits a
// bare object) cannot crash the decode.
type salaryAdjustmentsEnvelope struct {
	SalaryAdjustments struct {
		SalaryAdjustment ingestion.MFLList[adjustmentBlock] `json:"salaryAdjustment"`
	} `json:"salaryAdjustments"`
}

type adjustmentBlock struct {
	FranchiseID string `json:"franchise_id"`
	Amount      string `json:"amount"`
	Description string `json:"description"`
	ID          string `json:"id"`
	Timestamp   string `json:"timestamp"`
}

// Fetch retrieves the salary-adjustment ledger for the given season+league and
// returns flattened, shape-validated RawSalaryAdjustment records. year and leagueID
// are explicit arguments (ingestion.SeasonYear / ingestion.LeagueID are the
// canonical Phase-1 values). Like rosters, it discovers the league host FIRST, then
// issues one call through the transport (rate-limit, 429 backoff, host routing
// inherited). An empty ledger returns (nil, nil) — see the package doc: zero
// adjustments is valid, not a glitch.
func Fetch(ctx context.Context, c *mfl.Client, year, leagueID string) ([]RawSalaryAdjustment, error) {
	if err := c.DiscoverHost(ctx, year, leagueID); err != nil {
		return nil, fmt.Errorf("salaryadjustments: discover host: %w", err)
	}

	resp, err := c.Do(ctx, mfl.Request{
		Type:   "salaryAdjustments",
		Year:   year,
		Params: map[string]string{"L": leagueID},
	})
	if err != nil {
		return nil, fmt.Errorf("salaryadjustments: fetch: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("salaryadjustments: unexpected status %d", resp.StatusCode)
	}
	// MFL returns HTTP 200 with an {"error":{"$t":...}} body for many failures.
	// This fetcher has NO empty-payload sentinel (an empty ledger is valid), so an
	// error payload would otherwise decode to zero records and silently read as
	// "no dead-cap" — wiping every franchise's dead cap. Catch it first (Gemini B3
	// review BLOCKER).
	if err := ingestion.CheckAPIError(resp.Body); err != nil {
		return nil, fmt.Errorf("salaryadjustments: %w", err)
	}

	var env salaryAdjustmentsEnvelope
	if err := json.Unmarshal(resp.Body, &env); err != nil {
		return nil, fmt.Errorf("salaryadjustments: decode: %w", err)
	}

	return flatten(ctx, env)
}

// flatten walks the decoded envelope into RawSalaryAdjustment records, validating
// every record's shape. A malformed record fails LOUD (consistent with the other
// fetchers). An empty input yields an empty slice and no error — the legitimate
// "no dead-cap" case. ctx cancellation is honored between records.
func flatten(ctx context.Context, env salaryAdjustmentsEnvelope) ([]RawSalaryAdjustment, error) {
	out := make([]RawSalaryAdjustment, 0, len(env.SalaryAdjustments.SalaryAdjustment))
	for _, ab := range env.SalaryAdjustments.SalaryAdjustment {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("salaryadjustments: flatten cancelled: %w", ctx.Err())
		default:
		}

		ra := RawSalaryAdjustment(ab)
		if err := ra.Validate(); err != nil {
			return nil, err
		}
		out = append(out, ra)
	}
	return out, nil
}
