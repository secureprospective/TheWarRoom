// Package veteranfilm is the Layer 1 fetcher for the veteran Film signal — a player's
// charted, qualitative on-field traits, expressed as RAW per-player RATES. It joins
// nflverse FTN charting (the human-charted booleans: was the ball contested, was it a
// drop, was the throw interception-worthy, …) to nflverse play-by-play (which carries
// the gsis player id for each play) and aggregates per player into rates. It does NOT
// normalize and does NOT score: per Engine_Specification Layer-4 (Approach A) the
// position-specific normalization/weighting is the engine's job, and the scouting
// Profile is position-blind, so a position-blind fetcher cannot produce it.
//
// RATES, NOT COUNTS (structural zero-leak): the output is "how often" (contested-catch
// rate, drop rate, …), never "how much". A rate is leak-clean; a raw count is volume
// and risks the format-dependent-volume identity the zero-leak constraint forbids. No
// pbp fantasy/EPA/win-probability column is ever bound — a leak is unrepresentable
// because the data to leak is never read.
//
// THE JOIN: FTN identifies a play only by (nflverse_game_id, nflverse_play_id) and
// carries NO player id; pbp carries the play's gsis ids (passer/receiver) but not the
// charting. So the charted flag for a play is attributed to that play's pbp
// receiver_player_id (receiver traits) or passer_player_id (passer traits). A charted
// play with no matching pbp player for the role contributes nothing to that role.
//
// MEMORY: pbp is hundreds of MB per season and is STREAMED row-by-row (ingestion.
// StreamCSV) — never buffered whole (would OOM CT105's 2 GB). FTN is ~8 MB and is read
// with the ordinary buffered ingestion.FetchCSV. Only one season's FTN map plus one
// pbp row live in memory at once, so pooling many seasons stays bounded.
//
// SUPPRESS BELOW A FLOOR (Christopher's call): a player charted on very few plays
// yields a noisy rate (1 drop in 2 targets = 50%). A role group is emitted only when
// its charted-play count meets the floor; below it the group is nil. The denominators
// (Targets / Attempts) are emitted alongside the rates so the engine can confidence-
// gate further. Floors are PARAMETERS (provisional defaults DefaultReceiverFloor /
// DefaultPasserFloor) because the exact value is a calibration decision, not a constant.
//
// MULTI-YEAR: Fetch pools the supplied seasons by summing COUNTERS across years and
// computing each rate once at the end — rates are never averaged (that would need
// per-season weights). The season set is a parameter; the live default is one season.
package veteranfilm

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
)

// Default floors: the minimum charted plays a role needs before its rates are trusted.
// Provisional — the final values are a calibration job (see package doc).
const (
	DefaultReceiverFloor = 30  // charted targets
	DefaultPasserFloor   = 100 // charted pass attempts
)

// SeasonSource is the pair of source URLs for one season's charting (FTN) and
// play-by-play (pbp). Injecting URLs (not building them inside Fetch) keeps Fetch
// unit-testable against fixture servers and source-move safe.
type SeasonSource struct {
	FTNURL string
	PBPURL string
}

// SeasonSources builds the nflverse SeasonSource list for the given years using the
// package URL builders — the convenience the live caller uses.
func SeasonSources(years ...int) []SeasonSource {
	out := make([]SeasonSource, 0, len(years))
	for _, y := range years {
		out = append(out, SeasonSource{FTNURL: FTNSeasonURL(y), PBPURL: PBPSeasonURL(y)})
	}
	return out
}

// FTNSeasonURL is the nflverse ftn_charting release for a season (~8 MB; FetchCSV-safe).
func FTNSeasonURL(year int) string {
	y := strconv.Itoa(year)
	return "https://github.com/nflverse/nflverse-data/releases/download/ftn_charting/ftn_charting_" + y + ".csv"
}

// PBPSeasonURL is the nflverse play_by_play release for a season (hundreds of MB; must
// be STREAMED, not FetchCSV-buffered).
func PBPSeasonURL(year int) string {
	y := strconv.Itoa(year)
	return "https://github.com/nflverse/nflverse-data/releases/download/pbp/play_by_play_" + y + ".csv"
}

// PBPMaxBytes caps a single pbp stream. play_by_play_2024.csv is ~95 MiB live, so the
// shared 64 MiB DefaultMaxCSVBytes fails it by design; this ceiling is generous
// headroom for a single season while still failing loud on a runaway source. Memory
// stays flat regardless (StreamCSV holds one row) — this only bounds total bytes read.
const PBPMaxBytes = 400 << 20

// errEmpty guards a pool that resolved zero players above either floor — a glitch
// (wrong season, truncated download, an FTN/pbp mismatch) surfaced loudly rather than
// returned as "no film".
var errEmpty = errors.New("veteranfilm: sources resolved zero players above the floor")

// ReceiverFilm is one player's receiving film traits as RATES in [0,1] over the pooled
// charted seasons. Targets (charted targets) is the denominator the engine confidence-
// gates on. No field references fantasy points, projected volume, or MFL scoring.
type ReceiverFilm struct {
	Targets              int
	ContestedCatchRate   float64
	CreatedReceptionRate float64
	DropRate             float64
}

// PasserFilm is one player's passing film traits as RATES in [0,1] over the pooled
// charted seasons. Attempts (charted pass attempts) is the denominator. Zero-leak: no
// fantasy/volume/scoring field.
type PasserFilm struct {
	Attempts               int
	InterceptionWorthyRate float64
	ThrowawayRate          float64
}

// RawVeteranFilm is one player's charted film, keyed by nflverse gsis_id. Each role
// group is nil when that role's charted-play count was below the floor (or the player
// never filled that role). A player with both groups nil is not emitted.
type RawVeteranFilm struct {
	GSISID   string
	Receiver *ReceiverFilm
	Passer   *PasserFilm
}

// Fetch pools the given seasons and returns RawVeteranFilm keyed by gsis_id. For each
// season it loads FTN into a (game,play)->flags map, then STREAMS that season's pbp,
// attributing each charted play's flags to the play's receiver/passer gsis; counters
// accumulate across all seasons and rates are computed once at the end, after the
// floors are applied. client and the source URLs are injected; recvFloor/passFloor are
// the minimum charted plays a role needs to be emitted.
func Fetch(ctx context.Context, client *http.Client, sources []SeasonSource, recvFloor, passFloor int) (map[string]RawVeteranFilm, error) {
	agg := newAggregator()
	for _, s := range sources {
		flags, err := loadFTN(ctx, client, s.FTNURL)
		if err != nil {
			return nil, err
		}
		if err := agg.streamPBP(ctx, client, s.PBPURL, flags); err != nil {
			return nil, err
		}
	}

	out := agg.finalize(recvFloor, passFloor)
	if len(out) == 0 {
		return nil, errEmpty
	}
	return out, nil
}

// boolFlag parses an FTN charted boolean: "TRUE" is 1, "FALSE"/missing is 0, anything
// else fails loud (a present-but-unexpected value is corruption, not a false). Returns
// an int so callers add it straight into a counter.
func boolFlag(rec []string, idx int, label string) (int, error) {
	switch v := rec[idx]; v {
	case "TRUE", "true", "1":
		return 1, nil
	case "FALSE", "false", "0", "", ingestion.NACell:
		return 0, nil
	default:
		return 0, fmt.Errorf("veteranfilm: column %q value %q: not a boolean", label, v)
	}
}
