// Package pfrcoverage is the Layer 1 fetcher for the defender COVERAGE anchor — the
// analytical anchor reserved for CB and S (scouting.NGSCoverage). It reads the
// Pro-Football-Reference advanced-defense season table (nflverse pfr_advstats,
// advstats_season_def.csv.gz) and returns RAW, gsis-keyed coverage-allowed metrics;
// it does NOT normalize and does NOT score (THE CONTRACT — the engine normalizes,
// Approach A; the scouting Profile is position-blind).
//
// SOURCE SUBSTITUTION (recon, 2026-06-26 — parallels Offense Option D): the schema
// field is named NGSCoverage after NFL Next Gen Stats, but recon found nflverse
// publishes NO defender NGS file — its nextgen_stats release is offense-only
// (passing/receiving/rushing); avg_separation there is the RECEIVER's separation. The
// clean, Go-reachable, gsis-bridgeable defender-coverage source is PFR advanced
// defense, which carries genuine coverage-ALLOWED quality: targets, completion %
// allowed, yards allowed, and passer rating allowed (rat — the headline coverage
// metric). Christopher's call: rebind the anchor onto this source. It is coverage
// quality, not tracking-based separation; the field name is retained, the source is
// documented here and in the source map.
//
// EMIT ALL DEFENDERS, RAW: this fetcher does not apply the CB/S anchor boundary — it
// emits every targeted defender keyed by gsis and leaves the "Coverage non-nil at CB/S
// only" structural boundary to the engine/normalize layer (where it is already
// enforced). The raw Position string travels with each record so the engine can select.
// A defender who was never targeted (no coverage exposure) carries no coverage signal
// and is skipped.
//
// TRADED-PLAYER DEDUP (verified live): PFR splits a traded player's season into per-team
// rows and, usually, a single [0-9]TM season-aggregate row (e.g. "2TM") that carries the
// combined full-season figure. Per (season, pfr_id) this fetcher PREFERS the aggregate
// row (PFR's own combined RAW value — the fetcher must not sum splits and recompute
// passer rating, which would be analysis, not raw passthrough) and drops the splits. A
// player with multiple rows but NO single aggregate (mid-season trade with no combined
// row — ~2/season live) is AMBIGUOUS and dropped entirely; a clean missing-signal beats
// a partial one (the ras combine-collision precedent, Christopher's call).
//
// ID RESOLUTION: the PFR table keys on pfr_id, NOT gsis_id. The scouting layer keys on
// gsis, so Fetch takes an injected pfr->gsis map (built by the assembly layer from the
// same dynastyprocess db_playerids.csv the crosswalk uses — the same pattern touchshare
// and ras use; promoting this bridge into the crosswalk package is a flagged
// module-close item). An unresolvable pfr is an ordinary skip; two DISTINCT pfr ids
// resolving to one gsis is poison and the gsis is dropped entirely (ras precedent).
//
// ZERO-LEAK (hard constraint): every emitted field is a raw coverage counting stat or a
// PFR coverage rate — targets, completions, yards/TDs allowed, completion %, yards per
// target/completion, passer rating allowed, average depth of target. None references
// fantasy points, projected volume, or MFL scoring; passer rating allowed is a football
// stat, not a fantasy one. A leak is structurally unrepresentable in this field set.
//
// It reuses the shared external-HTTP-CSV plumbing (ingestion/extcsv.go): the source is
// gzipped, so it uses FetchCSVGz (the gz sibling of FetchCSV) + byte cap + by-name
// column binding + the "NA" missing-cell sentinel.
package pfrcoverage

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
)

// SourceURL is the nflverse pfr_advstats season-defense release: all seasons in one
// gzipped file (NOT per-year). A data-source address, not a secret/host to discover;
// Fetch takes the URL as an argument so tests can point at a fixture server.
const SourceURL = "https://github.com/nflverse/nflverse-data/releases/download/pfr_advstats/advstats_season_def.csv.gz"

// The columns this fetcher depends on, located by NAME in the header.
const (
	colSeason = "season"
	colPFR    = "pfr_id"
	colTm     = "tm"
	colPos    = "pos"
	colTgt    = "tgt"         // targets thrown at this defender (counting)
	colCmp    = "cmp"         // completions allowed (counting)
	colInt    = "int"         // interceptions (counting)
	colYds    = "yds"         // yards allowed (counting)
	colTD     = "td"          // touchdowns allowed (counting)
	colCmpPct = "cmp_percent" // completion % allowed (rate)
	colYdsCmp = "yds_cmp"     // yards per completion allowed (rate)
	colYdsTgt = "yds_tgt"     // yards per target allowed (rate)
	colRat    = "rat"         // passer rating allowed — headline coverage quality (rate)
	colDadot  = "dadot"       // average depth of target (rate)
)

// errEmpty guards a fetch that resolved zero targeted defenders for the season — a
// glitch (truncated download / wrong URL / a crosswalk that resolved nothing / a
// season absent from the file) surfaced loudly rather than returning an empty map.
var errEmpty = errors.New("pfrcoverage: source resolved zero targeted defenders")

// RawCoverage is one defender's raw season coverage-allowed line, keyed by nflverse
// gsis_id. Counting stats (Targets..TDsAllowed) use the missing-is-zero rule (a defender
// not targeted for a given event recorded none). Rate stats are *float64: nil means the
// rate is ABSENT (undefined — e.g. passer rating with no targets), distinct from a real
// 0.0 (a 0 passer-rating-allowed means perfect coverage, the opposite of "no data", so
// it must never be faked from a blank). Position is the raw PFR position label; the
// engine applies the CB/S anchor boundary. No field references fantasy points or any
// position-relative score.
type RawCoverage struct {
	GSISID   string
	Position string

	Targets       int
	Completions   int
	Interceptions int
	YardsAllowed  int
	TDsAllowed    int

	CompletionPct       *float64
	YardsPerCompletion  *float64
	YardsPerTarget      *float64
	PasserRatingAllowed *float64
	AvgDepthOfTarget    *float64
}

// Fetch retrieves the gzipped PFR season-defense table from url, filters to season, and
// returns RawCoverage records keyed by gsis_id, resolving each pfr_id through pfrToGSIS.
// client, url, season, and the map are injected (not package globals) so the fetcher is
// unit-testable and source-move safe. See the package doc for the traded-player and
// pfr->gsis dedup rules.
func Fetch(ctx context.Context, client *http.Client, url, season string,
	pfrToGSIS map[string]string) (map[string]RawCoverage, error) {
	records, err := ingestion.FetchCSVGz(ctx, client, url, ingestion.DefaultMaxCSVBytes)
	if err != nil {
		return nil, fmt.Errorf("pfrcoverage: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("pfrcoverage: %q returned no rows (not even a header)", url)
	}

	cols, err := ingestion.CSVColumns(records[0], colSeason, colPFR, colTm, colPos,
		colTgt, colCmp, colInt, colYds, colTD, colCmpPct, colYdsCmp, colYdsTgt, colRat, colDadot)
	if err != nil {
		return nil, fmt.Errorf("pfrcoverage: %w", err)
	}

	// Group this season's rows by pfr_id so the traded-player aggregate/split rule can be
	// applied per player before any gsis resolution.
	byPFR := make(map[string][]int)
	for i := 1; i < len(records); i++ {
		rec := records[i]
		if strings.TrimSpace(rec[cols[colSeason]]) != season {
			continue
		}
		pfr := strings.TrimSpace(rec[cols[colPFR]])
		if ingestion.IsMissing(pfr) {
			continue // blank pfr_id — unresolvable
		}
		byPFR[pfr] = append(byPFR[pfr], i)
	}

	return resolve(records, cols, byPFR, pfrToGSIS)
}

// resolve applies the per-player season-row selection, pfr->gsis resolution, and the
// distinct-pfr->same-gsis poison drop, returning the final gsis-keyed map.
func resolve(records [][]string, cols map[string]int, byPFR map[string][]int,
	pfrToGSIS map[string]string) (map[string]RawCoverage, error) {
	out := make(map[string]RawCoverage)
	collided := make(map[string]bool) // gsis ids dropped for receiving 2+ distinct pfr

	for pfr, idxs := range byPFR {
		rec, ok := pickSeasonRow(records, idxs, cols)
		if !ok {
			continue // multiple rows, no single aggregate — ambiguous, drop
		}
		gsis, ok := pfrToGSIS[pfr]
		if !ok {
			continue // pfr not in the crosswalk — ordinary non-match
		}
		if collided[gsis] {
			continue // already known ambiguous — drop this pfr's row too
		}

		rc, hasCov, err := rowCoverage(cols, rec, gsis)
		if err != nil {
			return nil, err
		}
		if !hasCov {
			continue // defender never targeted — no coverage signal
		}
		if _, dup := out[gsis]; dup {
			delete(out, gsis) // a second distinct pfr maps here: drop both
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

// pickSeasonRow chooses the single season row for a pfr_id from its candidate rows.
// One row: use it. Multiple rows (traded player): use the sole [0-9]TM season-aggregate
// row if exactly one exists; otherwise the player has splits with no combined figure —
// ambiguous, reported as not-ok so the caller drops it.
func pickSeasonRow(records [][]string, idxs []int, cols map[string]int) ([]string, bool) {
	if len(idxs) == 1 {
		return records[idxs[0]], true
	}
	aggIdx := -1
	for _, i := range idxs {
		if isTeamAggregate(strings.TrimSpace(records[i][cols[colTm]])) {
			if aggIdx >= 0 {
				return nil, false // 2+ aggregates (not seen live) — refuse to guess
			}
			aggIdx = i
		}
	}
	if aggIdx < 0 {
		return nil, false // splits with no aggregate — ambiguous
	}
	return records[aggIdx], true
}

// isTeamAggregate reports whether a PFR team cell is a multi-team season aggregate like
// "2TM"/"3TM": a "TM" suffix preceded by one or more digits.
func isTeamAggregate(tm string) bool {
	digits, ok := strings.CutSuffix(tm, "TM")
	if !ok || digits == "" {
		return false
	}
	for _, r := range digits {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// rowCoverage parses one PFR defense record into a RawCoverage. hasCov reports whether
// the defender was targeted (Targets > 0) — an untargeted defender carries no coverage
// signal. A present-but-malformed counting or rate cell fails loud.
func rowCoverage(cols map[string]int, rec []string, gsis string) (RawCoverage, bool, error) {
	rc := RawCoverage{GSISID: gsis, Position: strings.TrimSpace(rec[cols[colPos]])}

	counts := []struct {
		name string
		dst  *int
	}{
		{colTgt, &rc.Targets},
		{colCmp, &rc.Completions},
		{colInt, &rc.Interceptions},
		{colYds, &rc.YardsAllowed},
		{colTD, &rc.TDsAllowed},
	}
	for _, c := range counts {
		n, err := ingestion.IntCell(rec, cols[c.name], c.name)
		if err != nil {
			return RawCoverage{}, false, fmt.Errorf("pfrcoverage: %w", err)
		}
		*c.dst = n
	}

	rates := []struct {
		name string
		dst  **float64
	}{
		{colCmpPct, &rc.CompletionPct},
		{colYdsCmp, &rc.YardsPerCompletion},
		{colYdsTgt, &rc.YardsPerTarget},
		{colRat, &rc.PasserRatingAllowed},
		{colDadot, &rc.AvgDepthOfTarget},
	}
	for _, rt := range rates {
		v, err := optFloat(rec[cols[rt.name]], rt.name)
		if err != nil {
			return RawCoverage{}, false, err
		}
		*rt.dst = v
	}

	return rc, rc.Targets > 0, nil
}

// optFloat parses an optional rate cell: a missing/NA cell is ABSENT (nil), a
// present-but-unparseable cell fails loud. label names the column in the error.
func optFloat(raw, label string) (*float64, error) {
	v := strings.TrimSpace(raw)
	if ingestion.IsMissing(v) {
		return nil, nil
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return nil, fmt.Errorf("pfrcoverage: column %q value %q: %w", label, v, err)
	}
	return &f, nil
}
