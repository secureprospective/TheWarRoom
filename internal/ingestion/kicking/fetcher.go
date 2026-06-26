// Package kicking is the Layer 1 fetcher for the K NFLProduction scouting signal —
// the accumulating regular-season field-goal and extra-point production each kicker
// has recorded. It is the K-specific sibling of nflproduction: nflverse's
// player_stats_season is offense-only (no kicking columns at all), so kicker
// production comes from a SEPARATE nflverse release file, player_stats_kicking_season.
// Under DECISION-011 this is K's Layer-4 NFLProduction sub-signal (0.40, with Madden
// kick power/accuracy the 0.60 majority); see docs/scoring-engine/K_Rubric.md and
// docs/scoring-engine/Scouting_Schema.md.
//
// Like every scouting fetcher it returns RAW, gsis-keyed records; it does NOT
// normalize and does NOT score. Normalization is the engine's job (Approach A,
// position-blind Profile). It emits clean COUNTING stats only — made/attempted/
// missed/blocked, the made and missed field-goal-by-distance buckets, longest make,
// and PAT counts. It deliberately does NOT bind the source's fg_pct / pat_pct rate
// columns: a rate is analysis (made ÷ attempted), which the engine derives from the
// raw counts — emitting it here would pre-bake an engine decision into the fetcher.
//
// TRADED KICKERS (source shape): unlike nflverse's offense season file, this file is
// one row per (player, team, season_type) — a kicker traded mid-season has a REG row
// per team and there is NO source-provided cross-team aggregate row (no "2TM"-style
// combined row, unlike PFR). To emit one gsis-keyed season record this fetcher SUMS a
// kicker's per-team REG rows: counts and the distance buckets add, fg_long takes the
// max (the season's longest make). This is legitimate season aggregation, not the
// "sum + recompute a rate" anti-pattern barred for pfrcoverage — this fetcher emits
// only raw counts, and no aggregate row exists to prefer. (Christopher's call,
// 2026-06-26; diverges in mechanism from the pfrcoverage prefer-aggregate dedup
// because the source shape and the rate-vs-count distinction differ.)
//
// ZERO-LEAK (hard constraint): no field references fantasy points, projected volume,
// or MFL scoring. The guarantee is structural — RawKicking has no field to hold such
// a value, so a leak is unrepresentable, not merely avoided.
//
// It reuses the shared external-HTTP-CSV plumbing (ingestion/extcsv.go): fetch +
// byte cap + by-name column binding + the "NA"/empty missing-cell sentinel. The
// fetcher owns only its column set, the RawKicking shape, and the REG-season filter.
package kicking

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
)

// SeasonURL builds the nflverse player_stats_kicking_season release URL for a season.
// The host/path is a data-source address (not a secret/host to discover), and year is
// a parameter so the fetcher is season-rollover safe and unit-testable.
func SeasonURL(year string) string {
	return "https://github.com/nflverse/nflverse-data/releases/download/player_stats/player_stats_kicking_season_" + year + ".csv"
}

// Bound scalar columns. The source's fg_pct / pat_pct rate columns and the
// fg_*_list / fg_*_distance string columns are deliberately NOT bound — rates are
// engine-derived analysis and the list/distance columns are play-level detail this
// season-level signal does not use.
const (
	colGSIS       = "player_id"
	colSeasonType = "season_type"
	colGames      = "games"
	colFGMade     = "fg_made"
	colFGAtt      = "fg_att"
	colFGMissed   = "fg_missed"
	colFGBlocked  = "fg_blocked"
	colFGLong     = "fg_long"
	colPATMade    = "pat_made"
	colPATAtt     = "pat_att"
	colPATMissed  = "pat_missed"
	colPATBlocked = "pat_blocked"
)

// numBuckets is the count of field-goal distance buckets nflverse publishes.
const numBuckets = 6

// fgMadeBuckets and fgMissedBuckets name the made/missed distance-bucket columns in
// the fixed order used by RawKicking's arrays: [0-19, 20-29, 30-39, 40-49, 50-59,
// 60+]. The 60+ column name ends in a trailing underscore in the source.
func fgMadeBuckets() []string {
	return []string{"fg_made_0_19", "fg_made_20_29", "fg_made_30_39", "fg_made_40_49", "fg_made_50_59", "fg_made_60_"}
}

func fgMissedBuckets() []string {
	return []string{"fg_missed_0_19", "fg_missed_20_29", "fg_missed_30_39", "fg_missed_40_49", "fg_missed_50_59", "fg_missed_60_"}
}

// scalarCols are the non-bucket integer columns parsed into RawKicking.
func scalarCols() []string {
	return []string{
		colGames, colFGMade, colFGAtt, colFGMissed, colFGBlocked, colFGLong,
		colPATMade, colPATAtt, colPATMissed, colPATBlocked,
	}
}

// regularSeason is the season_type we keep. player_stats_kicking_season carries one
// row per kicker per season_type; mixing in POST would bias production toward the
// minority of kickers whose teams made the playoffs, so the production basis is the
// regular season only (parallel to nflproduction).
const regularSeason = "REG"

// errEmpty guards a fetch that resolved zero REG records — the file is never
// legitimately empty for a played season, so an empty result is a glitch (truncated
// download / wrong URL) surfaced loudly rather than as "no kicking production".
var errEmpty = errors.New("kicking: source resolved zero regular-season records")

// RawKicking is one kicker's raw regular-season kicking production, keyed by nflverse
// gsis_id. Every field is a clean counting stat; no field references fantasy points,
// projected volume, or MFL scoring (zero-leak invariant). FGMadeByDistance and
// FGMissedByDistance are indexed [0-19, 20-29, 30-39, 40-49, 50-59, 60+]. FGLong is
// the longest made field goal in yards (a raw leg-strength observation, not a rate).
type RawKicking struct {
	GSISID             string
	Games              int
	FGMade             int
	FGAtt              int
	FGMissed           int
	FGBlocked          int
	FGLong             int
	FGMadeByDistance   [numBuckets]int
	FGMissedByDistance [numBuckets]int
	PATMade            int
	PATAtt             int
	PATMissed          int
	PATBlocked         int
}

// Fetch retrieves player_stats_kicking_season from url and returns the REG-season
// RawKicking records keyed by gsis_id. A missing/NA gsis row is skipped; a malformed
// numeric cell fails loud; multiple REG rows for one gsis (a traded kicker's per-team
// splits) are SUMMED into one season record (see the package doc).
func Fetch(ctx context.Context, client *http.Client, url string) (map[string]RawKicking, error) {
	records, err := ingestion.FetchCSV(ctx, client, url, ingestion.DefaultMaxCSVBytes)
	if err != nil {
		return nil, fmt.Errorf("kicking: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("kicking: %q returned no rows (not even a header)", url)
	}

	made, missed := fgMadeBuckets(), fgMissedBuckets()
	required := append([]string{colGSIS, colSeasonType}, scalarCols()...)
	required = append(required, made...)
	required = append(required, missed...)
	cols, err := ingestion.CSVColumns(records[0], required...)
	if err != nil {
		return nil, fmt.Errorf("kicking: %w", err)
	}

	out := make(map[string]RawKicking)
	for _, rec := range records[1:] {
		gsis := strings.TrimSpace(rec[cols[colGSIS]])
		if ingestion.IsMissing(gsis) {
			continue
		}
		if strings.TrimSpace(rec[cols[colSeasonType]]) != regularSeason {
			continue
		}

		rk, err := rowKicking(cols, rec, gsis, made, missed)
		if err != nil {
			return nil, err
		}
		if prev, ok := out[gsis]; ok {
			out[gsis] = mergeKicking(prev, rk)
		} else {
			out[gsis] = rk
		}
	}

	if len(out) == 0 {
		return nil, errEmpty
	}
	return out, nil
}

// rowKicking parses one REG record into a RawKicking, failing loud on any malformed
// numeric cell (a present-but-unparseable value is corruption, not zero).
func rowKicking(cols map[string]int, rec []string, gsis string, made, missed []string) (RawKicking, error) {
	ints := map[string]int{}
	for _, name := range scalarCols() {
		v, err := ingestion.IntCell(rec, cols[name], name)
		if err != nil {
			return RawKicking{}, fmt.Errorf("kicking: %w", err)
		}
		ints[name] = v
	}

	rk := RawKicking{
		GSISID:     gsis,
		Games:      ints[colGames],
		FGMade:     ints[colFGMade],
		FGAtt:      ints[colFGAtt],
		FGMissed:   ints[colFGMissed],
		FGBlocked:  ints[colFGBlocked],
		FGLong:     ints[colFGLong],
		PATMade:    ints[colPATMade],
		PATAtt:     ints[colPATAtt],
		PATMissed:  ints[colPATMissed],
		PATBlocked: ints[colPATBlocked],
	}

	for i := 0; i < numBuckets; i++ {
		m, err := ingestion.IntCell(rec, cols[made[i]], made[i])
		if err != nil {
			return RawKicking{}, fmt.Errorf("kicking: %w", err)
		}
		x, err := ingestion.IntCell(rec, cols[missed[i]], missed[i])
		if err != nil {
			return RawKicking{}, fmt.Errorf("kicking: %w", err)
		}
		rk.FGMadeByDistance[i] = m
		rk.FGMissedByDistance[i] = x
	}

	return rk, nil
}

// mergeKicking combines a traded kicker's per-team REG rows into one season record:
// counting stats and the distance buckets add; FGLong takes the season's longest make
// (max). a and b share a gsis (the caller guarantees it); a.GSISID is preserved.
func mergeKicking(a, b RawKicking) RawKicking {
	a.Games += b.Games
	a.FGMade += b.FGMade
	a.FGAtt += b.FGAtt
	a.FGMissed += b.FGMissed
	a.FGBlocked += b.FGBlocked
	a.PATMade += b.PATMade
	a.PATAtt += b.PATAtt
	a.PATMissed += b.PATMissed
	a.PATBlocked += b.PATBlocked
	if b.FGLong > a.FGLong {
		a.FGLong = b.FGLong
	}
	for i := 0; i < numBuckets; i++ {
		a.FGMadeByDistance[i] += b.FGMadeByDistance[i]
		a.FGMissedByDistance[i] += b.FGMissedByDistance[i]
	}
	return a
}
