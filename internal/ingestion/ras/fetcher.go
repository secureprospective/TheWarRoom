// Package ras is the Layer 1 fetcher for the RAS (Relative Athletic Score) signal's
// raw inputs — a player's NFL combine measurables. It reads nflverse combine.csv and
// returns RAW, gsis-keyed measurables; it does NOT normalize and does NOT score.
//
// RAS-EQUIVALENT, NOT RAS: Kent Platte's RAS is a licensed per-position z-score. This
// fetcher does not reproduce it — it supplies the raw measurables, and the engine
// computes a per-position z-score RAS-EQUIVALENT (Engine_Specification Layer-4,
// Approach A: position-specific normalization is the engine's job, and the scouting
// Profile is position-blind). What ships here is a RAS-equivalent input, deliberately
// flagged as such, never Platte's exact number.
//
// MISSING IS ABSENT, NOT ZERO: combine participation is sparse — most players skip
// some drills, encoded as an empty cell. A skipped drill is ABSENT, not a 0 (a 0
// forty time or 0 vertical is nonsense, and a z-score over a phantom 0 would be
// corrupt). Each measurable is therefore a *float64: nil means "did not perform",
// distinct from a real value. (This is why the fetcher does not use the shared
// missing-is-zero FloatCell — that rule is correct for counting stats, wrong here.)
//
// ID RESOLUTION: combine.csv keys on pfr_id, NOT gsis_id (verified live). The
// scouting layer keys on gsis, so Fetch takes an injected pfr->gsis map (built by the
// assembly layer from the same db_playerids.csv the crosswalk uses); an unresolvable
// pfr is an ordinary skip. A gsis that receives 2+ combine rows is AMBIGUOUS and
// dropped entirely (see Fetch): combine.csv carries ~17 pfr_ids with multiple rows,
// some being two distinct players who share one upstream pfr_id, so keeping either
// row risks a wrong-player attribution. Dropping is the safe, lenient external-
// boundary choice — a clean missing-signal beats a silently corrupt one.
//
// ZERO-LEAK (hard constraint): combine measurables are pure athletic testing — no
// fantasy points / projected volume / MFL scoring column exists to bind, so a leak is
// structurally unrepresentable here.
//
// It reuses the shared external-HTTP-CSV plumbing (ingestion/extcsv.go): fetch + byte
// cap + by-name column binding + the "NA" missing-cell sentinel. The fetcher owns its
// column set, the RawCombine shape, the optional-measurable parsing (incl. the F-I
// height format), and the pfr->gsis resolution / dedup rules.
package ras

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
)

// SourceURL is the nflverse combine.csv release (all seasons in one file — NOT the
// per-year combine_{year}.csv). A data-source address, not a secret/host to discover;
// Fetch takes the URL as an argument so tests can point at a fixture server.
const SourceURL = "https://github.com/nflverse/nflverse-data/releases/download/combine/combine.csv"

// The columns this fetcher depends on, located by NAME in the header.
const (
	colPFR     = "pfr_id"
	colHt      = "ht"
	colWt      = "wt"
	colForty   = "forty"
	colBench   = "bench"
	colVert    = "vertical"
	colBroad   = "broad_jump"
	colCone    = "cone"
	colShuttle = "shuttle"
)

// errEmpty guards a fetch that resolved zero measured players — a glitch (truncated
// download / wrong URL / a crosswalk that resolved nothing) surfaced loudly.
var errEmpty = errors.New("ras: source resolved zero players with combine measurables")

// RawCombine is one player's raw combine measurables, keyed by nflverse gsis_id.
// Every measurable is a *float64: nil means the player did not perform that drill
// (absent != zero). Height is total inches (parsed from the source's F-I format);
// weight is pounds; bench is rep count (kept as a float for a uniform measurable
// set). No field references fantasy points or any position-relative score.
type RawCombine struct {
	GSISID    string
	HeightIn  *float64
	WeightLb  *float64
	Forty     *float64
	Bench     *float64
	Vertical  *float64
	BroadJump *float64
	Cone      *float64
	Shuttle   *float64
}

// Fetch retrieves combine.csv from url and returns RawCombine records keyed by
// gsis_id, resolving each row's pfr_id through pfrToGSIS. client, url, and the map are
// injected (not package globals) so the fetcher is unit-testable and source-move
// safe. A missing/unresolvable pfr is skipped; a resolvable player with no measurable
// at all is skipped (no RAS signal); a malformed measurable fails loud; a gsis with
// 2+ rows is dropped as ambiguous.
func Fetch(ctx context.Context, client *http.Client, url string, pfrToGSIS map[string]string) (map[string]RawCombine, error) {
	records, err := ingestion.FetchCSV(ctx, client, url, ingestion.DefaultMaxCSVBytes)
	if err != nil {
		return nil, fmt.Errorf("ras: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("ras: %q returned no rows (not even a header)", url)
	}

	cols, err := ingestion.CSVColumns(records[0],
		colPFR, colHt, colWt, colForty, colBench, colVert, colBroad, colCone, colShuttle)
	if err != nil {
		return nil, fmt.Errorf("ras: %w", err)
	}

	out := make(map[string]RawCombine)
	// A gsis that receives 2+ combine rows is AMBIGUOUS and dropped entirely (not
	// fail-loud, not first-wins). combine.csv has ~17 pfr_ids carrying multiple rows —
	// some are two distinct players sharing one pfr_id upstream (e.g. two "Derrick
	// Johnson" rows, CB + OLB) — so first-wins risks attaching the wrong player's
	// measurables to a gsis. Dropping ambiguous gsis yields a clean missing-signal the
	// engine tolerates, where a wrong RAS would be silently corrupt. This is the
	// lenient EXTERNAL-boundary policy (combine is 3rd-party data, not trusted
	// internal input); Christopher's call (matches the B3 anomaly precedent).
	collided := make(map[string]bool)
	for _, rec := range records[1:] {
		pfr := strings.TrimSpace(rec[cols[colPFR]])
		if ingestion.IsMissing(pfr) {
			continue
		}
		gsis, ok := pfrToGSIS[pfr]
		if !ok {
			continue // pfr not in the crosswalk — ordinary non-match
		}
		if collided[gsis] {
			continue // already known ambiguous — drop this row too
		}

		rc, hasData, err := rowCombine(cols, rec, gsis)
		if err != nil {
			return nil, err
		}
		if !hasData {
			continue // resolvable player but performed no drills — no RAS signal
		}
		if _, dup := out[gsis]; dup {
			delete(out, gsis) // a second row makes this gsis ambiguous: drop both
			collided[gsis] = true
			continue
		}
		out[gsis] = rc
	}

	if len(out) == 0 {
		return nil, errEmpty
	}
	return out, nil
}

// rowCombine parses one combine record into a RawCombine. hasData reports whether any
// measurable was present (a row with every drill absent carries no RAS signal). A
// present-but-malformed measurable fails loud.
func rowCombine(cols map[string]int, rec []string, gsis string) (RawCombine, bool, error) {
	rc := RawCombine{GSISID: gsis}
	hasData := false

	h, err := optHeight(rec[cols[colHt]])
	if err != nil {
		return RawCombine{}, false, err
	}
	rc.HeightIn = h
	hasData = hasData || h != nil

	vals := make(map[string]*float64, 7)
	for _, name := range []string{colWt, colForty, colBench, colVert, colBroad, colCone, colShuttle} {
		v, err := optFloat(rec[cols[name]], name)
		if err != nil {
			return RawCombine{}, false, err
		}
		vals[name] = v
		hasData = hasData || v != nil
	}
	rc.WeightLb = vals[colWt]
	rc.Forty = vals[colForty]
	rc.Bench = vals[colBench]
	rc.Vertical = vals[colVert]
	rc.BroadJump = vals[colBroad]
	rc.Cone = vals[colCone]
	rc.Shuttle = vals[colShuttle]
	return rc, hasData, nil
}

// optFloat parses an optional numeric measurable: a missing/NA cell is ABSENT (nil),
// a present-but-unparseable cell fails loud. label names the column in the error.
func optFloat(raw, label string) (*float64, error) {
	v := strings.TrimSpace(raw)
	if ingestion.IsMissing(v) {
		return nil, nil
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return nil, fmt.Errorf("ras: column %q value %q: %w", label, v, err)
	}
	return &f, nil
}

// optHeight parses the combine height format "F-I" (feet-inches, e.g. "6-4") into
// total inches. Missing is absent (nil); a present value not in F-I form fails loud.
func optHeight(raw string) (*float64, error) {
	v := strings.TrimSpace(raw)
	if ingestion.IsMissing(v) {
		return nil, nil
	}
	ft, in, ok := strings.Cut(v, "-")
	if !ok {
		return nil, fmt.Errorf("ras: malformed height %q (want feet-inches, e.g. 6-4)", v)
	}
	feet, err := strconv.Atoi(strings.TrimSpace(ft))
	if err != nil {
		return nil, fmt.Errorf("ras: height %q feet: %w", v, err)
	}
	inches, err := strconv.Atoi(strings.TrimSpace(in))
	if err != nil {
		return nil, fmt.Errorf("ras: height %q inches: %w", v, err)
	}
	total := float64(feet*12 + inches)
	return &total, nil
}
