// Package pfrpassrush is the Layer 1 fetcher for the defender PASS-RUSH signal — the raw
// input the IDP film redesign uses to replace the PFF grade at the DL/edge positions (DT,
// DE, and the pressure side of LB). It reads the SAME Pro-Football-Reference advanced-defense
// season table as pfrcoverage (nflverse pfr_advstats, advstats_season_def.csv.gz) and returns
// RAW, gsis-keyed pass-rush counting stats; it does NOT normalize and does NOT score (THE
// CONTRACT — the engine normalizes, Approach A; the scouting Profile is position-blind).
//
// WHY THIS SOURCE (2026-07-24, Christopher's call): PFF is OUT — its grades are TOS-restricted
// and paywalled, and TheWarRoom uses no pay-walled sources (see handoff 50). The clean, free,
// TOS-compatible, Go-reachable, gsis-bridgeable pass-rush signal is already in-repo: the same
// PFR advanced-defense drop pfrcoverage consumes carries per-player pass-rush columns
// (bltz/hrry/qbkd/sk/prss) alongside the coverage columns pfrcoverage binds. This fetcher binds
// the pass-rush half of that identical, already-trusted download — no paywall, no new source.
//
// EMIT ALL RUSHERS, RAW: this fetcher applies no position boundary — it emits every defender
// with a pass-rush signal keyed by gsis and leaves the "pass-rush feeds DL/edge film only"
// structural boundary to the engine/normalize layer. The raw Position string travels with each
// record so the engine can select. A defender with NO pass-rush production (no pressures, sacks,
// knockdowns, or hurries) carries no signal and is skipped — the mirror of pfrcoverage skipping
// an untargeted defender.
//
// RAW ONLY — no rate here: pressures/sacks are per-SEASON counts; a per-game or per-snap
// pressure RATE (the eventual film sub-signal) is normalization and belongs to the engine, not
// this fetcher. Games (g) rides along as the natural per-game denominator so the engine can build
// that rate without a second fetch, but this fetcher computes no rate itself. Pressures (prss) is
// PFR's OWN aggregate figure and is emitted verbatim — it is NOT recomputed from sk+qbkd+hrry
// (PFR's prss and that sum can differ by a point; passing PFR's raw number through is the contract,
// summing would be analysis).
//
// TRADED-PLAYER DEDUP + ID RESOLUTION: identical to pfrcoverage (the two fetchers read the same
// table shape). Per (season, pfr_id) the sole [0-9]TM season-aggregate row wins over per-team
// splits; splits with no aggregate are ambiguous and dropped; the pfr_id keys resolve through an
// injected pfr->gsis map (built by the assembly layer from the shared dynastyprocess crosswalk,
// the same pattern ras/touchshare/pfrcoverage use); an unresolvable pfr is an ordinary skip and
// two DISTINCT pfr ids resolving to one gsis is poison and the gsis is dropped (ras precedent).
//
// ZERO-LEAK (hard constraint): every emitted field is a raw pass-rush counting stat — blitzes,
// hurries, QB knockdowns, sacks, pressures — plus games played. None references fantasy points,
// projected volume, or MFL scoring; a sack and a pressure are football stats, not fantasy ones. A
// leak is structurally unrepresentable in this field set.
//
// It reuses the shared external-HTTP-CSV plumbing (ingestion/extcsv.go): the source is gzipped,
// so it uses FetchCSVGz + byte cap + by-name column binding + the missing-is-zero counting rule.
package pfrpassrush

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
)

// SourceURL is the nflverse pfr_advstats season-defense release — the SAME gzipped file
// pfrcoverage reads (all seasons in one file, NOT per-year). A data-source address, not a
// secret; Fetch takes the URL as an argument so tests can point at a fixture server.
const SourceURL = "https://github.com/nflverse/nflverse-data/releases/download/pfr_advstats/advstats_season_def.csv.gz"

// The columns this fetcher depends on, located by NAME in the header (position-independent —
// the live file has since added an `age` column, which by-name binding ignores).
const (
	colSeason = "season"
	colPFR    = "pfr_id"
	colTm     = "tm"
	colPos    = "pos"
	colGames  = "g"    // games played — the per-game denominator (raw; no rate computed here)
	colBltz   = "bltz" // times blitzed (scheme context; near-zero for base edge rushers)
	colHrry   = "hrry" // QB hurries (counting)
	colQBKd   = "qbkd" // QB knockdowns (counting)
	colSk     = "sk"   // sacks — FRACTIONAL (half-sacks), so parsed as float
	colPrss   = "prss" // total pressures — PFR's headline aggregate (emitted verbatim)
)

// errEmpty guards a fetch that resolved zero pass-rushing defenders for the season — a glitch
// (truncated download / wrong URL / a crosswalk that resolved nothing / a season absent from the
// file) surfaced loudly rather than returning an empty map.
var errEmpty = errors.New("pfrpassrush: source resolved zero pass-rushing defenders")

// RawPassRush is one defender's raw season pass-rush line, keyed by nflverse gsis_id. Every
// counting stat uses the missing-is-zero rule (a defender with none recorded carries a real 0 —
// the opposite of "no data", so no field is nilable: a blank pass-rush cell means the event did
// not happen, not that it is unknown). Sacks is a float because PFR records half-sacks. Position
// is the raw PFR position label; the engine applies the DL/edge boundary. No field references
// fantasy points or any position-relative score.
type RawPassRush struct {
	GSISID   string
	Position string
	Games    int

	Blitzes      int
	Hurries      int
	QBKnockdowns int
	Sacks        float64
	Pressures    int
}

// Fetch retrieves the gzipped PFR season-defense table from url, filters to season, and returns
// RawPassRush records keyed by gsis_id, resolving each pfr_id through pfrToGSIS. client, url,
// season, and the map are injected (not package globals) so the fetcher is unit-testable and
// source-move safe. See the package doc for the traded-player and pfr->gsis dedup rules.
func Fetch(ctx context.Context, client *http.Client, url, season string,
	pfrToGSIS map[string]string) (map[string]RawPassRush, error) {
	records, err := ingestion.FetchCSVGz(ctx, client, url, ingestion.DefaultMaxCSVBytes)
	if err != nil {
		return nil, fmt.Errorf("pfrpassrush: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("pfrpassrush: %q returned no rows (not even a header)", url)
	}

	cols, err := ingestion.CSVColumns(records[0], colSeason, colPFR, colTm, colPos,
		colGames, colBltz, colHrry, colQBKd, colSk, colPrss)
	if err != nil {
		return nil, fmt.Errorf("pfrpassrush: %w", err)
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
	pfrToGSIS map[string]string) (map[string]RawPassRush, error) {
	out := make(map[string]RawPassRush)
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

		pr, hasRush, err := rowPassRush(cols, rec, gsis)
		if err != nil {
			return nil, err
		}
		if !hasRush {
			continue // no pass-rush production — no signal
		}
		if _, dup := out[gsis]; dup {
			delete(out, gsis) // a second distinct pfr maps here: drop both
			collided[gsis] = true
			continue
		}
		out[gsis] = pr
	}

	if len(out) == 0 {
		return nil, errEmpty
	}
	return out, nil
}

// pickSeasonRow chooses the single season row for a pfr_id from its candidate rows. One row: use
// it. Multiple rows (traded player): use the sole [0-9]TM season-aggregate row if exactly one
// exists; otherwise the player has splits with no combined figure — ambiguous, reported not-ok so
// the caller drops it. (Same rule as pfrcoverage — the two fetchers read the same table shape.)
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

// rowPassRush parses one PFR defense record into a RawPassRush. hasRush reports whether the
// defender produced any pass-rush event (pressures, sacks, knockdowns, or hurries > 0) — a
// defender with none carries no signal. Blitzes alone do NOT count as production (a scheme stat:
// a base edge rusher records 0 blitzes yet real pressures). A present-but-malformed cell fails
// loud; a blank counting cell is a real 0 (missing-is-zero).
func rowPassRush(cols map[string]int, rec []string, gsis string) (RawPassRush, bool, error) {
	pr := RawPassRush{GSISID: gsis, Position: strings.TrimSpace(rec[cols[colPos]])}

	counts := []struct {
		name string
		dst  *int
	}{
		{colGames, &pr.Games},
		{colBltz, &pr.Blitzes},
		{colHrry, &pr.Hurries},
		{colQBKd, &pr.QBKnockdowns},
		{colPrss, &pr.Pressures},
	}
	for _, c := range counts {
		n, err := ingestion.IntCell(rec, cols[c.name], c.name)
		if err != nil {
			return RawPassRush{}, false, fmt.Errorf("pfrpassrush: %w", err)
		}
		*c.dst = n
	}

	// Sacks are fractional (half-sacks) — the one float pass-rush stat.
	sk, err := ingestion.FloatCell(rec, cols[colSk], colSk)
	if err != nil {
		return RawPassRush{}, false, fmt.Errorf("pfrpassrush: %w", err)
	}
	pr.Sacks = sk

	hasRush := pr.Pressures > 0 || pr.Sacks > 0 || pr.QBKnockdowns > 0 || pr.Hurries > 0
	return pr, hasRush, nil
}
