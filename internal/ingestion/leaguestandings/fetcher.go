// Package leaguestandings fetches MFL's standings for the M2 board as raw records.
package leaguestandings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
)

// errEmptyStandings guards MFL's {"leagueStandings":{}} glitch, which would blank the view.
var errEmptyStandings = errors.New("leaguestandings: response contained zero franchises")

// RawStanding is one franchise's standings row, numbers as raw strings. Fields a league does
// not use (all-play) arrive empty. MFL sends all-play as one "W-L-T" field; Parse splits it.
type RawStanding struct {
	FranchiseID string // "0001"–"0032"
	H2HW        string // head-to-head wins
	H2HL        string // head-to-head losses
	H2HT        string // head-to-head ties
	AllPlayW    string // all-play wins (blank when disabled)
	AllPlayL    string // all-play losses
	AllPlayT    string // all-play ties
	PF          string // points for
	PA          string // points against
	AvgPF       string // average points for
	AvgPA       string // average points against
	PP          string // potential points (optimal-lineup sum)
	Pwr         string // MFL Power Rank, display only
	AltPwr      string // display only
	Salary      string // dynasty cap salary
}

// Validate requires a franchise id and that every present number parses, so garbage fails
// here instead of reading as zero later.
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
			// ParseFloat accepts "NaN" and "Inf", which would break the table sort.
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return fmt.Errorf("leaguestandings: franchise %s non-finite %s %q", r.FranchiseID, f.name, f.val)
			}
		}
	}
	return nil
}

// SanitizeNumeric strips the "$" and "," MFL puts in salary ("$120.72") and large totals
// ("1,850.50") so the value parses. The record keeps the original string. Exported so m2service
// parses exactly what was validated.
func SanitizeNumeric(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "$", "")
	s = strings.ReplaceAll(s, ",", "")
	return s
}

// standingsEnvelope mirrors the MFL leagueStandings JSON; unknown fields are tolerated.
type standingsEnvelope struct {
	LeagueStandings struct {
		Franchise ingestion.MFLList[franchiseStanding] `json:"franchise"`
	} `json:"leagueStandings"`
}

type franchiseStanding struct {
	ID      string `json:"id"`
	H2HW    string `json:"h2hw"`
	H2HL    string `json:"h2hl"`
	H2HT    string `json:"h2ht"`
	AllPlay string `json:"all_play_wlt"` // "89-4-0"; blank when the league has all-play off
	PF      string `json:"pf"`
	PA      string `json:"pa"`
	AvgPF   string `json:"avgpf"`
	AvgPA   string `json:"avgpa"`
	PP      string `json:"pp"`
	Pwr     string `json:"pwr"`
	AltPwr  string `json:"altpwr"`
	Salary  string `json:"salary"`
}

// Export is the MFL export this package reads.
const Export = "leagueStandings"

// Fetch discovers the league host, then returns the export, validated.
func Fetch(ctx context.Context, c *mfl.Client, year, leagueID string) ([]RawStanding, error) {
	if err := c.DiscoverHost(ctx, year, leagueID); err != nil {
		return nil, fmt.Errorf("leaguestandings: discover host: %w", err)
	}
	body, err := ingestion.LeagueExport(ctx, c, Export, year, leagueID, nil)
	if err != nil {
		return nil, fmt.Errorf("leaguestandings: %w", err)
	}
	return Parse(body)
}

// Parse decodes and validates an export body, live or archived.
func Parse(body []byte) ([]RawStanding, error) {
	var env standingsEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("leaguestandings: decode: %w", err)
	}
	if len(env.LeagueStandings.Franchise) == 0 {
		return nil, errEmptyStandings
	}
	return flatten(env)
}

// flatten validates every record; a malformed one fails the fetch.
func flatten(env standingsEnvelope) ([]RawStanding, error) {
	out := make([]RawStanding, 0, len(env.LeagueStandings.Franchise))
	for _, f := range env.LeagueStandings.Franchise {
		w, l, t, err := splitWLT(f.AllPlay)
		if err != nil {
			return nil, fmt.Errorf("leaguestandings: franchise %s: %w", f.ID, err)
		}
		rs := RawStanding{
			FranchiseID: f.ID,
			H2HW:        f.H2HW,
			H2HL:        f.H2HL,
			H2HT:        f.H2HT,
			AllPlayW:    w,
			AllPlayL:    l,
			AllPlayT:    t,
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

// splitWLT splits MFL's "W-L-T" record into its three counts; blank stays blank.
func splitWLT(s string) (w, l, t string, err error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", "", "", nil
	}
	parts := strings.Split(s, "-")
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("all-play record %q is not W-L-T", s)
	}
	return parts[0], parts[1], parts[2], nil
}
