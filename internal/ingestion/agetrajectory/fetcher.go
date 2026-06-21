// Package agetrajectory is the Layer 1 fetcher for the AgeTrajectory scouting
// signal's raw input — a player's birth date. It reads nflverse players.csv and
// returns RAW, gsis-keyed birth dates; it does NOT normalize and does NOT score.
//
// RAW BOUNDARY: the rubric's AgeTrajectory is "age vs. position peak limit" — a
// POSITION-SPECIFIC normalization that is the engine's job (Engine_Specification
// Layer-4, Approach A), and the scouting Profile is position-blind. This fetcher
// therefore emits the birth DATE, not an age: age is derived from a birth date plus
// an as-of date, and baking a reference instant into a Layer-1 fetcher would make
// the value go stale between seasons. The engine computes age as-of the scoring
// season it is grading. (players.csv ships no standalone age column to be tempted
// by — verified live; the durable raw fact is the birth date.)
//
// One row per player (gsis_id is the unique key): a row with no birth date is
// skipped (it carries no age signal — skipping is correct, a zero date would be a
// false 1970 birthday); a present-but-unparseable date fails loud; two rows for one
// gsis is a source-integrity error and fails loud.
//
// It reuses the shared external-HTTP-CSV plumbing (ingestion/extcsv.go): fetch +
// byte cap + by-name column binding + the "NA" missing-cell sentinel. The fetcher
// owns only its two columns, the RawAge shape, and the date-parse/dedup rules.
package agetrajectory

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
)

// SourceURL is the nflverse players.csv release (the full player database). It is a
// data-source address, not a secret/host to discover, so it is a documented
// constant; Fetch takes the URL as an argument so tests can point at a fixture
// server and a future config can override the source.
const SourceURL = "https://github.com/nflverse/nflverse-data/releases/download/players/players.csv"

// The two columns this fetcher depends on, located by NAME in the header
// (players.csv carries 30+ columns whose order is not guaranteed).
const (
	colGSIS  = "gsis_id"
	colBirth = "birth_date"
)

// birthLayout is the ISO date format nflverse uses for birth_date (e.g. 1987-01-25,
// verified live). A value present but not in this layout is corruption, not a zero.
const birthLayout = "2006-01-02"

// errEmpty guards a fetch that resolved zero birth dates. players.csv is never
// legitimately empty, so an empty result is a glitch (truncated download / wrong
// URL) surfaced loudly rather than as "no players".
var errEmpty = errors.New("agetrajectory: source resolved zero birth dates")

// RawAge is one player's raw birth date, keyed by nflverse gsis_id. It holds no age
// and no position-relative trajectory — those are the engine's as-of-season,
// position-specific derivations (zero-leak / Approach-A invariant).
type RawAge struct {
	GSISID    string
	BirthDate time.Time
}

// Fetch retrieves players.csv from url using client and returns the RawAge records
// keyed by gsis_id. client and url are injected (not package globals) so the fetcher
// is unit-testable against a fixture server and survives a source move. Rows missing
// either the gsis or the birth date are skipped (an ordinary absence, not an error);
// a present-but-malformed birth date fails loud; a duplicate gsis fails loud.
func Fetch(ctx context.Context, client *http.Client, url string) (map[string]RawAge, error) {
	records, err := ingestion.FetchCSV(ctx, client, url, ingestion.DefaultMaxCSVBytes)
	if err != nil {
		return nil, fmt.Errorf("agetrajectory: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("agetrajectory: %q returned no rows (not even a header)", url)
	}

	cols, err := ingestion.CSVColumns(records[0], colGSIS, colBirth)
	if err != nil {
		return nil, fmt.Errorf("agetrajectory: %w", err)
	}
	gsisIdx, birthIdx := cols[colGSIS], cols[colBirth]

	out := make(map[string]RawAge)
	for _, rec := range records[1:] {
		gsis := strings.TrimSpace(rec[gsisIdx])
		raw := strings.TrimSpace(rec[birthIdx])
		if ingestion.IsMissing(gsis) || ingestion.IsMissing(raw) {
			continue // no id or no DOB — carries no age signal
		}

		bd, err := time.Parse(birthLayout, raw)
		if err != nil {
			return nil, fmt.Errorf("agetrajectory: gsis %q birth_date %q: %w", gsis, raw, err)
		}
		if _, dup := out[gsis]; dup {
			return nil, fmt.Errorf("agetrajectory: duplicate row for gsis %q", gsis)
		}
		out[gsis] = RawAge{GSISID: gsis, BirthDate: bd}
	}

	if len(out) == 0 {
		return nil, errEmpty
	}
	return out, nil
}
