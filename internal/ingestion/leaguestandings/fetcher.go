// Package leaguestandings is the Layer 1 fetcher for the MFL leagueStandings
// endpoint (M2 Power Rankings, slice-1). One exported Fetch returns RAW records: it
// fetches and validates the response SHAPE and transforms nothing — every MFL number
// stays a raw string, normalization happens downstream. It clones the rosters
// fetcher template (WF 1B): build Request → DiscoverHost → c.Do → schema-validate →
// return raw, with no domain knowledge and no upward imports.
package leaguestandings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
)

// errEmptyStandings guards the MFL glitch shape {"leagueStandings":{}} (or a null
// payload), which decodes to zero franchises. A 32-team league never legitimately
// returns no standings rows, so we surface it as an error rather than an empty slice
// — an empty slice downstream would read as "no teams" and blank the whole view.
var errEmptyStandings = errors.New("leaguestandings: response contained zero franchises")

// RawStanding is one franchise's standings row exactly as MFL returns it. Every
// numeric field is a raw string — MFL encodes numbers as strings, and a fetcher
// transforms nothing (WF 1B). Fields absent for a league (e.g. all-play disabled)
// arrive empty; Validate requires only the identity + the always-present H2H record.
type RawStanding struct {
	FranchiseID string // "0001"–"0032"
	H2HW        string // head-to-head wins
	H2HL        string // head-to-head losses
	H2HT        string // head-to-head ties
	AllPlayW    string // all-play wins (blank if league disabled all-play)
	AllPlayL    string // all-play losses
	AllPlayT    string // all-play ties
	PF          string // points for
	PA          string // points against
	AvgPF       string // average points for
	AvgPA       string // average points against
	PP          string // potential points (optimal-lineup sum)
	Pwr         string // MFL Power Rank (unnormalized; DISPLAY column)
	AltPwr      string // MFL Alternate Power Rank (DISPLAY column)
	Salary      string // dynasty cap salary
}

// Validate checks the raw record's SHAPE before anything downstream runs. It does
// not convert types (normalize's job) — it rejects only records that cannot be
// valid: the franchise ID must be present, and any non-empty numeric field must be
// parseable so a garbage payload fails LOUD here rather than as a silent zero later.
func (r RawStanding) Validate() error {
	if strings.TrimSpace(r.FranchiseID) == "" {
		return fmt.Errorf("leaguestandings: record missing franchise id")
	}
	for _, f := range []struct {
		name, val string
	}{
		{"h2hw", r.H2HW}, {"h2hl", r.H2HL}, {"h2ht", r.H2HT},
		{"all_play_w", r.AllPlayW}, {"all_play_l", r.AllPlayL}, {"all_play_t", r.AllPlayT},
		{"pf", r.PF}, {"pa", r.PA}, {"avgpf", r.AvgPF}, {"avgpa", r.AvgPA},
		{"pp", r.PP}, {"pwr", r.Pwr}, {"altpwr", r.AltPwr}, {"salary", r.Salary},
	} {
		if s := SanitizeNumeric(f.val); s != "" {
			v, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return fmt.Errorf("leaguestandings: franchise %s non-numeric %s %q: %w", r.FranchiseID, f.name, f.val, err)
			}
			// ParseFloat accepts "NaN"/"Inf"/"+Inf" — a garbage payload that would
			// later render as "NaN" in the UI and poison the table's sort comparator
			// (NaN compares false both ways). Reject non-finite at the boundary.
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return fmt.Errorf("leaguestandings: franchise %s non-finite %s %q", r.FranchiseID, f.name, f.val)
			}
		}
	}
	return nil
}

// SanitizeNumeric strips MFL's display formatting from a numeric field so the raw
// value can be parsed: the leagueStandings `salary` arrives currency-formatted
// ("$120.72") and large point totals can carry thousands separators ("1,850.50"),
// unlike the bare numbers on other endpoints. Trimming "$" and "," leaves the plain
// number (or "" for an absent field). The RawStanding still keeps the ORIGINAL
// string — a fetcher transforms nothing; this only feeds the shape check. Exported so
// the app-layer parse of the same rows stays consistent with this validation.
func SanitizeNumeric(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "$", "")
	s = strings.ReplaceAll(s, ",", "")
	return s
}

// standingsEnvelope mirrors the MFL leagueStandings JSON. Unknown fields are
// tolerated (MFL is an external API we do not control and adds fields without
// versioning — the internal/schema unknown-field policy); correctness comes from
// Validate asserting the fields we depend on, not from rejecting fields we ignore.
type standingsEnvelope struct {
	LeagueStandings struct {
		Franchise ingestion.MFLList[franchiseStanding] `json:"franchise"`
	} `json:"leagueStandings"`
}

type franchiseStanding struct {
	ID       string `json:"id"`
	H2HW     string `json:"h2hw"`
	H2HL     string `json:"h2hl"`
	H2HT     string `json:"h2ht"`
	AllPlayW string `json:"all_play_w"`
	AllPlayL string `json:"all_play_l"`
	AllPlayT string `json:"all_play_t"`
	PF       string `json:"pf"`
	PA       string `json:"pa"`
	AvgPF    string `json:"avgpf"`
	AvgPA    string `json:"avgpa"`
	PP       string `json:"pp"`
	Pwr      string `json:"pwr"`
	AltPwr   string `json:"altpwr"`
	Salary   string `json:"salary"`
}

// Fetch retrieves the league standings for the given season+league from MFL and
// returns flattened, shape-validated RawStanding records. year and leagueID are
// explicit arguments (not package globals) so the fetcher is unit-testable and
// supports multi-league/season rollover; ingestion.SeasonYear and ingestion.LeagueID
// are the canonical Phase-1 values a caller passes. It discovers the league host
// FIRST, then issues one leagueStandings call through the transport client:
// rate-limiting, 429 backoff, and host routing are inherited, never re-implemented.
func Fetch(ctx context.Context, c *mfl.Client, year, leagueID string) ([]RawStanding, error) {
	if err := c.DiscoverHost(ctx, year, leagueID); err != nil {
		return nil, fmt.Errorf("leaguestandings: discover host: %w", err)
	}

	resp, err := c.Do(ctx, mfl.Request{
		Type:   "leagueStandings",
		Year:   year,
		Params: map[string]string{"L": leagueID},
	})
	if err != nil {
		return nil, fmt.Errorf("leaguestandings: fetch: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("leaguestandings: unexpected status %d", resp.StatusCode)
	}

	// MFL returns HTTP 200 with an {"error":{"$t":...}} envelope for invalid league
	// ids / maintenance; check before decode so an error payload never reads as
	// "no standings".
	if err := ingestion.CheckAPIError(resp.Body); err != nil {
		return nil, fmt.Errorf("leaguestandings: %w", err)
	}

	var env standingsEnvelope
	if err := json.Unmarshal(resp.Body, &env); err != nil {
		return nil, fmt.Errorf("leaguestandings: decode: %w", err)
	}
	if len(env.LeagueStandings.Franchise) == 0 {
		return nil, errEmptyStandings
	}

	return flatten(ctx, env)
}

// flatten walks the decoded envelope into RawStanding records, validating every
// record's shape. A malformed real record fails LOUD (returns an error) rather than
// being silently dropped. It honors ctx cancellation between franchises so a
// shutdown mid-parse returns promptly instead of blocking until the loop finishes.
func flatten(ctx context.Context, env standingsEnvelope) ([]RawStanding, error) {
	out := make([]RawStanding, 0, len(env.LeagueStandings.Franchise))
	for _, f := range env.LeagueStandings.Franchise {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("leaguestandings: flatten cancelled: %w", ctx.Err())
		default:
		}
		rs := RawStanding{
			FranchiseID: f.ID,
			H2HW:        f.H2HW,
			H2HL:        f.H2HL,
			H2HT:        f.H2HT,
			AllPlayW:    f.AllPlayW,
			AllPlayL:    f.AllPlayL,
			AllPlayT:    f.AllPlayT,
			PF:          f.PF,
			PA:          f.PA,
			AvgPF:       f.AvgPF,
			AvgPA:       f.AvgPA,
			PP:          f.PP,
			Pwr:         f.Pwr,
			AltPwr:      f.AltPwr,
			Salary:      f.Salary,
		}
		if err := rs.Validate(); err != nil {
			return nil, err
		}
		out = append(out, rs)
	}
	return out, nil
}
