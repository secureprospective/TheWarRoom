// Package nflproduction is the Layer 1 fetcher for the NFLProduction scouting
// signal — the accumulating regular-season offensive box-score production each
// player has recorded. It reads nflverse player_stats_season and returns RAW,
// gsis-keyed records; it does NOT normalize and does NOT score. Normalization is
// deliberately the engine's job: Engine_Specification §Layer-4 applies
// position_specific_normalization within position group (Approach A), and the
// scouting Profile is position-blind — so a position-blind fetcher cannot and must
// not produce the [0,1] value. NFLProduction is also a DYNAMIC sub-signal the
// engine EMA-blends over time; this fetcher supplies the per-season observation.
//
// ZERO-LEAK (hard constraint): player_stats_season carries fantasy_points and
// fantasy_points_ppr columns. They are NEVER bound or read here — RawProduction has
// no field to hold them, so a fantasy-points leak into the scouting signal is
// unrepresentable, not merely avoided. Only clean counting stats are surfaced; the
// engine composes the production metric from them.
//
// It reuses the shared external-HTTP-CSV plumbing (ingestion/extcsv.go): fetch +
// byte cap + by-name column binding + the "NA" missing-cell sentinel. The fetcher
// owns only its column set, the RawProduction shape, and the REG-season filter.
package nflproduction

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
)

// SeasonURL builds the nflverse player_stats_season release URL for a season. The
// host/path is a data-source address (not a secret/host to discover), and year is
// a parameter so the fetcher is season-rollover safe and unit-testable.
func SeasonURL(year string) string {
	return "https://github.com/nflverse/nflverse-data/releases/download/player_stats/player_stats_season_" + year + ".csv"
}

// Bound columns. fantasy_points / fantasy_points_ppr are deliberately absent —
// binding them would be the zero-leak violation.
const (
	colGSIS        = "player_id"
	colSeasonType  = "season_type"
	colGames       = "games"
	colCompletions = "completions"
	colAttempts    = "attempts"
	colPassYds     = "passing_yards"
	colPassTDs     = "passing_tds"
	colInts        = "interceptions"
	colCarries     = "carries"
	colRushYds     = "rushing_yards"
	colRushTDs     = "rushing_tds"
	colReceptions  = "receptions"
	colTargets     = "targets"
	colRecYds      = "receiving_yards"
	colRecTDs      = "receiving_tds"
)

// regularSeason is the season_type we keep. player_stats_season carries one row
// per player per season_type; mixing in POST would bias production toward the
// minority of players whose teams made the playoffs, so the production basis is the
// regular season only.
const regularSeason = "REG"

// errEmpty guards a fetch that resolved zero REG records — the file is never
// legitimately empty for a played season, so an empty result is a glitch
// (truncated download / wrong URL) surfaced loudly rather than as "no production".
var errEmpty = errors.New("nflproduction: source resolved zero regular-season records")

// RawProduction is one player's raw regular-season offensive production, keyed by
// nflverse gsis_id. Every field is a clean counting stat; no field references
// fantasy points, projected volume, or MFL scoring (zero-leak invariant). Yards are
// float64 (the source may carry fractional aggregates); counts are int.
type RawProduction struct {
	GSISID         string
	Games          int
	Completions    int
	Attempts       int
	PassingYards   float64
	PassingTDs     int
	Interceptions  int
	Carries        int
	RushingYards   float64
	RushingTDs     int
	Receptions     int
	Targets        int
	ReceivingYards float64
	ReceivingTDs   int
}

// Fetch retrieves player_stats_season from url and returns the REG-season
// RawProduction records keyed by gsis_id. A missing/NA gsis row is skipped; a
// malformed numeric cell fails loud; two REG rows for one gsis is a source
// integrity error and fails loud (the season aggregate is one row per player).
func Fetch(ctx context.Context, client *http.Client, url string) (map[string]RawProduction, error) {
	records, err := ingestion.FetchCSV(ctx, client, url, ingestion.DefaultMaxCSVBytes)
	if err != nil {
		return nil, fmt.Errorf("nflproduction: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("nflproduction: %q returned no rows (not even a header)", url)
	}

	cols, err := ingestion.CSVColumns(records[0],
		colGSIS, colSeasonType, colGames, colCompletions, colAttempts, colPassYds,
		colPassTDs, colInts, colCarries, colRushYds, colRushTDs, colReceptions,
		colTargets, colRecYds, colRecTDs)
	if err != nil {
		return nil, fmt.Errorf("nflproduction: %w", err)
	}

	out := make(map[string]RawProduction)
	for _, rec := range records[1:] {
		gsis := strings.TrimSpace(rec[cols[colGSIS]])
		if ingestion.IsMissing(gsis) {
			continue
		}
		if strings.TrimSpace(rec[cols[colSeasonType]]) != regularSeason {
			continue
		}

		rp, err := rowProduction(cols, rec, gsis)
		if err != nil {
			return nil, err
		}
		if _, dup := out[gsis]; dup {
			return nil, fmt.Errorf("nflproduction: duplicate REG row for gsis %q", gsis)
		}
		out[gsis] = rp
	}

	if len(out) == 0 {
		return nil, errEmpty
	}
	return out, nil
}

// rowProduction parses one REG record into a RawProduction, failing loud on any
// malformed numeric cell (a present-but-unparseable value is corruption, not zero).
func rowProduction(cols map[string]int, rec []string, gsis string) (RawProduction, error) {
	ints := map[string]int{}
	for _, name := range []string{
		colGames, colCompletions, colAttempts, colPassTDs, colInts, colCarries,
		colRushTDs, colReceptions, colTargets, colRecTDs,
	} {
		v, err := ingestion.IntCell(rec, cols[name], name)
		if err != nil {
			return RawProduction{}, fmt.Errorf("nflproduction: %w", err)
		}
		ints[name] = v
	}

	floats := map[string]float64{}
	for _, name := range []string{colPassYds, colRushYds, colRecYds} {
		v, err := ingestion.FloatCell(rec, cols[name], name)
		if err != nil {
			return RawProduction{}, fmt.Errorf("nflproduction: %w", err)
		}
		floats[name] = v
	}

	return RawProduction{
		GSISID:         gsis,
		Games:          ints[colGames],
		Completions:    ints[colCompletions],
		Attempts:       ints[colAttempts],
		PassingYards:   floats[colPassYds],
		PassingTDs:     ints[colPassTDs],
		Interceptions:  ints[colInts],
		Carries:        ints[colCarries],
		RushingYards:   floats[colRushYds],
		RushingTDs:     ints[colRushTDs],
		Receptions:     ints[colReceptions],
		Targets:        ints[colTargets],
		ReceivingYards: floats[colRecYds],
		ReceivingTDs:   ints[colRecTDs],
	}, nil
}
