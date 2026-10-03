// Package salaryadjustments fetches MFL's dead-cap ledger: a cap penalty entry for each dropped
// player who went unclaimed. A franchise's cap use is its salaries plus the sum of its entries.
//
// Unlike the other fetchers, an empty response is valid (a franchise with no dead cap), so
// there is no empty guard and MFL's error envelope must be checked first.
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

// RawSalaryAdjustment is one ledger entry as MFL sends it. Description contains player
// details but is never parsed.
type RawSalaryAdjustment struct {
	FranchiseID string // owning franchise, "0001"–"0032"
	Amount      string // millions, raw ("0.203")
	Description string // display only
	ID          string // MFL adjustment record id
	Timestamp   string // unix seconds
}

// Validate requires a franchise id and an Amount that parses, so drift fails here instead of
// zeroing dead cap. A present Timestamp must be a unix int.
func (a RawSalaryAdjustment) Validate() error {
	if strings.TrimSpace(a.FranchiseID) == "" {
		return fmt.Errorf("salaryadjustments: record %s missing franchise id", a.ID)
	}
	// Franchise ids are numeric ("0001"); a value like "WAIVERS" must not key the ledger. Negative
	// amounts are allowed: commissioners issue credits.
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

// salaryAdjustmentsEnvelope mirrors the MFL JSON. MFLList handles a franchise with exactly one
// entry, which arrives as a bare object.
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

// Fetch discovers the league host and returns the ledger, validated. An empty ledger returns
// (nil, nil).
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
	// With no empty guard, an error payload would decode to zero rows and wipe every franchise's
	// dead cap.
	if err := ingestion.CheckAPIError(resp.Body); err != nil {
		return nil, fmt.Errorf("salaryadjustments: %w", err)
	}

	var env salaryAdjustmentsEnvelope
	if err := json.Unmarshal(resp.Body, &env); err != nil {
		return nil, fmt.Errorf("salaryadjustments: decode: %w", err)
	}

	return flatten(ctx, env)
}

// flatten validates every record; a malformed one fails the fetch. It honors ctx.
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
