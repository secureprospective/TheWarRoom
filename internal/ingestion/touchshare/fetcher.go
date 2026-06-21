// Package touchshare is the Layer 1 fetcher for the RB TouchShare scouting signal —
// a player's regular-season offensive snap workload, the opportunity proxy the RB
// rubric weights at 0.20 ("Snap Counts / Touch Share"). It reads nflverse
// snap_counts and returns RAW, gsis-keyed, season-aggregated records; it does NOT
// normalize and does NOT score. Per Engine_Specification Layer-4 (Approach A) the
// [0,1] / position-leader / rolling-percentile work is the engine's POSITION-
// SPECIFIC job, and the scouting Profile is position-blind — so a position-blind
// fetcher cannot and must not produce the normalized value.
//
// ID RESOLUTION: snap_counts keys on pfr_player_id, NOT gsis_id (verified live).
// The whole scouting layer keys on gsis, so Fetch takes an injected pfr->gsis map
// (built by the assembly layer from the same dynastyprocess db_playerids.csv the
// crosswalk uses). A row whose pfr does not resolve is an ordinary skip (not every
// pfr player is in the crosswalk); two DISTINCT pfr ids resolving to one gsis is a
// source-integrity error — it would double-count one player's workload — and fails
// loud. This fetcher stays position-blind: it emits a record for every gsis-
// resolvable player and lets the engine attach TouchShare only at the RB slot,
// rather than trusting snap_counts' own position label.
//
// SL-OQ-021 (RB rubric -> Layer 1 data-hygiene spec): touch share must be computed
// per ACTIVE game, not per calendar week, so an injury-shortened season is not
// averaged down by DNP weeks. This fetcher therefore counts GamesActive (games with
// offense_snaps > 0) and sums only active-game values; the engine divides by
// GamesActive to get the per-active-game share. A 0-snap game is not an active game
// and contributes nothing.
//
// ZERO-LEAK (hard constraint): snap_counts carries no fantasy-points / projected-
// volume / MFL-scoring column, so a leak is structurally unrepresentable here.
// OffenseSnaps is a VOLUME signal — clean of fantasy points, but the engine must
// cap/watch it so raw volume does not dominate a SCOUTING rank.
//
// It reuses the shared external-HTTP-CSV plumbing (ingestion/extcsv.go): fetch +
// byte cap + by-name column binding + the "NA" missing-cell sentinel + the
// missing-is-zero / malformed-fails-loud cell parsers. The fetcher owns only its
// column set, the RawTouchShare shape, the per-active-game aggregation, and the
// pfr->gsis resolution rules.
package touchshare

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
)

// SeasonURL builds the nflverse snap_counts release URL for a season. The host/path
// is a data-source address (not a secret/host to discover), and year is a parameter
// so the fetcher is season-rollover safe and unit-testable against a fixture server.
func SeasonURL(year string) string {
	return "https://github.com/nflverse/nflverse-data/releases/download/snap_counts/snap_counts_" + year + ".csv"
}

// The columns this fetcher depends on, located by NAME in the header (snap_counts
// carries 16 columns; defense_/st_ snap fields are deliberately unbound).
const (
	colPFR      = "pfr_player_id"
	colSeason   = "season"
	colOffSnaps = "offense_snaps"
	colOffPct   = "offense_pct"
)

// errEmpty guards a fetch that resolved zero active-offense records. snap_counts is
// never legitimately empty for a played season, so an empty result is a glitch
// (truncated download / wrong URL / a crosswalk that resolved nothing) surfaced
// loudly rather than as "no workload".
var errEmpty = errors.New("touchshare: source resolved zero active-offense records")

// RawTouchShare is one player's raw regular-season offensive snap workload, keyed by
// nflverse gsis_id and aggregated across active games. No field references fantasy
// points, projected volume, or MFL scoring (zero-leak invariant). The engine derives
// the per-active-game snap share as OffensePctSum / GamesActive (SL-OQ-021).
type RawTouchShare struct {
	GSISID        string
	Season        int
	OffenseSnaps  int     // sum of offense_snaps across active games
	OffensePctSum float64 // sum of offense_pct across active games (engine / GamesActive => per-active-game share)
	GamesActive   int     // games with offense_snaps > 0
}

// Fetch retrieves snap_counts from url and returns RawTouchShare records keyed by
// gsis_id, resolving each row's pfr_player_id through pfrToGSIS. client, url, and the
// crosswalk map are injected (not package globals) so the fetcher is unit-testable
// and source-move safe. A missing/unresolvable pfr row is skipped; a 0-snap game is
// skipped (not active); a malformed numeric cell fails loud; two distinct pfr ids
// resolving to one gsis fails loud (would double-count).
func Fetch(ctx context.Context, client *http.Client, url string, pfrToGSIS map[string]string) (map[string]RawTouchShare, error) {
	records, err := ingestion.FetchCSV(ctx, client, url, ingestion.DefaultMaxCSVBytes)
	if err != nil {
		return nil, fmt.Errorf("touchshare: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("touchshare: %q returned no rows (not even a header)", url)
	}

	cols, err := ingestion.CSVColumns(records[0], colPFR, colSeason, colOffSnaps, colOffPct)
	if err != nil {
		return nil, fmt.Errorf("touchshare: %w", err)
	}

	out := make(map[string]RawTouchShare)
	owner := make(map[string]string) // gsis -> the pfr owning its accumulator (cross-pfr collision guard)
	for _, rec := range records[1:] {
		row, ok, err := parseRow(cols, rec, pfrToGSIS)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}

		if prev, seen := owner[row.gsis]; seen && prev != row.pfr {
			return nil, fmt.Errorf("touchshare: gsis %q claimed by two pfr ids %q and %q (would double-count)", row.gsis, prev, row.pfr)
		}
		owner[row.gsis] = row.pfr

		acc, exists := out[row.gsis]
		if !exists {
			acc = RawTouchShare{GSISID: row.gsis, Season: row.season}
		} else if acc.Season != row.season {
			return nil, fmt.Errorf("touchshare: gsis %q has rows from two seasons %d and %d", row.gsis, acc.Season, row.season)
		}
		acc.OffenseSnaps += row.snaps
		acc.OffensePctSum += row.pct
		acc.GamesActive++
		out[row.gsis] = acc
	}

	if len(out) == 0 {
		return nil, errEmpty
	}
	return out, nil
}

// snapRow is one resolved, parsed snap_counts game record.
type snapRow struct {
	gsis   string
	pfr    string
	snaps  int
	pct    float64
	season int
}

// parseRow resolves and parses one snap_counts record. ok=false means skip the row
// (missing pfr, a pfr absent from the crosswalk, or a 0-snap inactive game); a
// malformed numeric cell returns an error (fail loud).
func parseRow(cols map[string]int, rec []string, pfrToGSIS map[string]string) (snapRow, bool, error) {
	pfr := strings.TrimSpace(rec[cols[colPFR]])
	if ingestion.IsMissing(pfr) {
		return snapRow{}, false, nil
	}
	gsis, ok := pfrToGSIS[pfr]
	if !ok {
		return snapRow{}, false, nil // pfr not in the crosswalk — ordinary non-match
	}

	snaps, err := ingestion.IntCell(rec, cols[colOffSnaps], colOffSnaps)
	if err != nil {
		return snapRow{}, false, fmt.Errorf("touchshare: %w", err)
	}
	if snaps <= 0 {
		return snapRow{}, false, nil // not an active game — nothing to accumulate
	}
	pct, err := ingestion.FloatCell(rec, cols[colOffPct], colOffPct)
	if err != nil {
		return snapRow{}, false, fmt.Errorf("touchshare: %w", err)
	}
	season, err := ingestion.IntCell(rec, cols[colSeason], colSeason)
	if err != nil {
		return snapRow{}, false, fmt.Errorf("touchshare: %w", err)
	}
	return snapRow{gsis: gsis, pfr: pfr, snaps: snaps, pct: pct, season: season}, true, nil
}
