package main

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/leaguestandings"
	"github.com/secureprospective/TheWarRoom/internal/output"
	"github.com/secureprospective/TheWarRoom/internal/powerrankings"
)

// m2Timeout bounds the M2 IPC call. GetPowerRankings does at most one MFL
// leagueStandings fetch plus a read of the persisted M1 board — the same order of
// work as GetRankings, so it shares its budget.
const m2Timeout = 120 * time.Second

// PowerRow is one franchise's Power-Ranking row: the blended score and its two
// normalized components joined with identity and the MFL report columns that come
// free in the same standings call. Fully typed for ifaceguard; every float is a
// display value, never a stored competing truth.
type PowerRow struct {
	Rank        int    `json:"rank"`
	FranchiseID string `json:"franchiseID"`
	Name        string `json:"name"`

	PowerScore float64 `json:"powerScore"` // weighted z-blend, min-max'd to [0,1]
	ScoutingZ  float64 `json:"scoutingZ"`  // standardized scouting component (0 = league avg)
	MFLPerfZ   float64 `json:"mflPerfZ"`   // standardized all-play component

	ScoutingScore float64 `json:"scoutingScore"` // raw aggregated AdjustedScore
	AllPlayWinPct float64 `json:"allPlayWinPct"` // raw all-play win rate [0,1]

	// MFL report passthrough columns (display + sort), from leagueStandings.
	H2HW     int     `json:"h2hW"`
	H2HL     int     `json:"h2hL"`
	H2HT     int     `json:"h2hT"`
	AllPlayW int     `json:"allPlayW"`
	AllPlayL int     `json:"allPlayL"`
	AllPlayT int     `json:"allPlayT"`
	PF       float64 `json:"pf"`
	PA       float64 `json:"pa"`
	PP       float64 `json:"pp"`
	Pwr      float64 `json:"pwr"`    // MFL Power Rank (unnormalized display column)
	AltPwr   float64 `json:"altPwr"` // MFL Alternate Power Rank (display column)
}

// PowerRankingsResult is the GetPowerRankings IPC payload. Weight echoes back the
// clamped scouting weight actually applied so the UI slider and the rows never
// disagree. AggMode echoes the scouting aggregation used ("sum" or "topn") and
// StarterN the N applied for top-N (0 when not applicable). A zero-rows OK result
// means the M1 board has not been scored yet.
//
// Freshness carries the B-5 degradation contract: live when the standings fetch
// succeeded, stale when the board is built from the last-known-good cache after a failed
// fetch. Phase is the current season phase, carried alongside — NOT folded into Freshness —
// because an offseason board is fresh data about a finished season, not stale data (see
// the Freshness doc comment). An empty Phase means the phase read itself failed, which
// degrades the label only and never the board.
type PowerRankingsResult struct {
	OK        bool       `json:"ok"`
	Error     string     `json:"error"`
	Label     string     `json:"label"`
	Season    int        `json:"season"`
	Weight    float64    `json:"weight"`
	AggMode   string     `json:"aggMode"`
	StarterN  int        `json:"starterN"`
	Freshness Freshness  `json:"freshness"`
	Phase     string     `json:"phase"`
	Rows      []PowerRow `json:"rows"`
}

// Scouting aggregation modes for GetPowerRankings.
const (
	aggSum  = "sum"  // Σ AdjustedScore over the whole roster — rewards dynasty depth
	aggTopN = "topn" // Σ of the top-N by AdjustedScore — isolates startable talent
)

// GetPowerRankings composes the M2 blended board: it fetches MFL leagueStandings,
// aggregates the persisted M1 AdjustedScore per franchise, blends them with the
// caller's scouting weight (default powerrankings.DefaultScoutingWeight, clamped in
// Blend), and joins the MFL report columns + real team names for display. It is
// READ-ONLY as far as league state goes — no transactions, no staged confirm.
//
// B-5 DEGRADATION: an MFL standings failure is NO LONGER fatal. It used to be, because
// the all-play component is the whole 40% of the blend and had no persisted fallback
// (unlike M1 scores, which sit in SQLite). standings_cache now provides that fallback:
// a successful fetch is cached, and a failed fetch falls back to the last-known-good copy
// and labels the result stale. Only a failure with NOTHING cached is still fatal — that
// is the honest "no data at all" case, not a degradation. The single cache write is why
// this is no longer literally read-only; it touches no league state.
func (a *App) GetPowerRankings(weight float64, aggMode string) PowerRankingsResult {
	mode := resolveAggMode(aggMode)
	fail := func(err error) PowerRankingsResult {
		return PowerRankingsResult{
			Error: err.Error(), Label: a.proxyLabel(), Season: a.season,
			Weight: weight, AggMode: mode,
			Freshness: Freshness{State: FreshFail, Note: err.Error()},
		}
	}
	if err := a.m1Ready(); err != nil {
		return fail(err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, m2Timeout)
	defer cancel()

	ver, err := a.rulebook.ActiveVersion(ctx)
	if err != nil {
		return fail(err)
	}
	scores, err := a.output.Reader().Scores(ctx, a.season, ver)
	if err != nil {
		return fail(err)
	}

	standings, fresh, err := a.standingsOrCache(ctx)
	if err != nil {
		return fail(fmt.Errorf("power rankings: %w", err))
	}

	// N for top-N: the league's configured total starter count. On top-N with an
	// unreadable count, degrade to sum rather than an arbitrary N (fail-safe, labeled
	// by the AggMode echo).
	starterN := a.starterCount()
	if mode == aggTopN && starterN <= 0 {
		mode = aggSum
	}

	inputs, parsed, perr := a.buildBlendInputs(standings, scores, mode, starterN)
	if perr != nil {
		return fail(perr)
	}

	blended, err := powerrankings.Blend(inputs, weight)
	if err != nil {
		return fail(err)
	}

	n := 0
	if mode == aggTopN {
		n = starterN
	}
	return PowerRankingsResult{
		OK: true, Label: a.proxyLabel(),
		Season: a.season, Weight: clampWeight(weight),
		AggMode: mode, StarterN: n,
		Freshness: fresh, Phase: a.currentPhaseLabel(),
		Rows: a.buildPowerRows(blended, parsed),
	}
}

// resolveAggMode normalizes the caller's aggregation mode, defaulting to sum for an
// empty or unrecognized value so a bad param never errors the view.
func resolveAggMode(m string) string {
	if m == aggTopN {
		return aggTopN
	}
	return aggSum
}

// starterCount reads the league's total starter count from the active rulebook
// config; 0 if unset/unparseable (caller degrades top-N to sum).
func (a *App) starterCount() int {
	n, err := strconv.Atoi(strings.TrimSpace(a.rulebook.ActiveConfig().Starters.Count))
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// buildBlendInputs aggregates scouting Adjusted Score per franchise (by mode) and
// parses each standings row once, returning the blend inputs keyed off the STANDINGS
// set (the canonical franchises) plus the parsed rows for later display join. A
// franchise absent from the board (not scored / empty roster) contributes 0
// scouting, which standardizes correctly against the field.
func (a *App) buildBlendInputs(
	standings []leaguestandings.RawStanding,
	scores []output.SeasonScore,
	mode string,
	starterN int,
) ([]powerrankings.Input, map[string]parsedStanding, error) {
	// A player's owning franchise comes from runtime state (the M1 board carries no
	// franchise column); an unrostered player contributes to no team, as intended.
	// FRESHNESS: this reads LIVE runtime state, which may have moved since the M1
	// board was scored (a player traded/dropped in between accrues to their CURRENT
	// franchise, not the one they were scored under). The blend is correct for the
	// current-roster snapshot — which is the intended "who is strong now" reading.
	// Collect per-franchise score lists so top-N can select the best N.
	scoresByFranchise := make(map[string][]float64, len(standings))
	for _, s := range scores {
		if p, ok := a.state.Reader().Player(s.MFLID); ok {
			scoresByFranchise[p.FranchiseID] = append(scoresByFranchise[p.FranchiseID], s.AdjustedScore)
		}
	}

	parsed := make(map[string]parsedStanding, len(standings))
	inputs := make([]powerrankings.Input, 0, len(standings))
	for _, st := range standings {
		ps, err := parseStanding(st)
		if err != nil {
			return nil, nil, err
		}
		parsed[st.FranchiseID] = ps
		inputs = append(inputs, powerrankings.Input{
			FranchiseID:   st.FranchiseID,
			ScoutingScore: aggregateScouting(scoresByFranchise[st.FranchiseID], mode, starterN),
			AllPlayWinPct: ps.allPlayWinPct,
		})
	}
	return inputs, parsed, nil
}

// aggregateScouting reduces a franchise's per-player AdjustedScores to one number:
// the full sum (aggSum, rewards dynasty depth) or the sum of the top-N by score
// (aggTopN, isolates startable talent). Top-N sorts a COPY so the caller's slice is
// untouched; N ≥ len means the whole roster.
func aggregateScouting(scores []float64, mode string, starterN int) float64 {
	if mode == aggTopN && starterN > 0 && starterN < len(scores) {
		cp := make([]float64, len(scores))
		copy(cp, scores)
		sort.Sort(sort.Reverse(sort.Float64Slice(cp)))
		scores = cp[:starterN]
	}
	var sum float64
	for _, s := range scores {
		sum += s
	}
	return sum
}

// buildPowerRows joins the blended scores with the MFL display columns and local
// franchise names. Team names come from the LOCAL rulebook (no network) — unlike M1
// player names, there is no offline-degrade path to worry about here.
func (a *App) buildPowerRows(blended []powerrankings.Row, parsed map[string]parsedStanding) []PowerRow {
	names := a.rulebook.FranchiseNames()
	rows := make([]PowerRow, 0, len(blended))
	for _, b := range blended {
		ps := parsed[b.FranchiseID]
		rows = append(rows, PowerRow{
			Rank:          b.Rank,
			FranchiseID:   b.FranchiseID,
			Name:          franchiseDisplayName(names, b.FranchiseID),
			PowerScore:    b.PowerScore,
			ScoutingZ:     b.ScoutingZ,
			MFLPerfZ:      b.MFLPerfZ,
			ScoutingScore: b.ScoutingScore,
			AllPlayWinPct: b.AllPlayWinPct,
			H2HW:          ps.h2hW, H2HL: ps.h2hL, H2HT: ps.h2hT,
			AllPlayW: ps.allPlayW, AllPlayL: ps.allPlayL, AllPlayT: ps.allPlayT,
			PF: ps.pf, PA: ps.pa, PP: ps.pp, Pwr: ps.pwr, AltPwr: ps.altPwr,
		})
	}
	return rows
}

// parsedStanding holds a RawStanding's numeric fields after one parse pass, so the
// blend input and the display row never re-parse the same strings.
type parsedStanding struct {
	h2hW, h2hL, h2hT             int
	allPlayW, allPlayL, allPlayT int
	pf, pa, pp, pwr, altPwr      float64
	allPlayWinPct                float64
}

// parseStanding converts a shape-validated RawStanding into numbers. Every field
// already passed RawStanding.Validate (parseable or empty), so an empty string maps
// to 0 and a parse error here would be a genuine invariant break — surfaced LOUD.
func parseStanding(s leaguestandings.RawStanding) (parsedStanding, error) {
	var ps parsedStanding
	var err error
	if ps.h2hW, err = atoiOrZero(s.H2HW); err != nil {
		return ps, wrapParse(s.FranchiseID, "h2hw", err)
	}
	if ps.h2hL, err = atoiOrZero(s.H2HL); err != nil {
		return ps, wrapParse(s.FranchiseID, "h2hl", err)
	}
	if ps.h2hT, err = atoiOrZero(s.H2HT); err != nil {
		return ps, wrapParse(s.FranchiseID, "h2ht", err)
	}
	if ps.allPlayW, err = atoiOrZero(s.AllPlayW); err != nil {
		return ps, wrapParse(s.FranchiseID, "all_play_w", err)
	}
	if ps.allPlayL, err = atoiOrZero(s.AllPlayL); err != nil {
		return ps, wrapParse(s.FranchiseID, "all_play_l", err)
	}
	if ps.allPlayT, err = atoiOrZero(s.AllPlayT); err != nil {
		return ps, wrapParse(s.FranchiseID, "all_play_t", err)
	}
	if ps.pf, err = atofOrZero(s.PF); err != nil {
		return ps, wrapParse(s.FranchiseID, "pf", err)
	}
	if ps.pa, err = atofOrZero(s.PA); err != nil {
		return ps, wrapParse(s.FranchiseID, "pa", err)
	}
	if ps.pp, err = atofOrZero(s.PP); err != nil {
		return ps, wrapParse(s.FranchiseID, "pp", err)
	}
	if ps.pwr, err = atofOrZero(s.Pwr); err != nil {
		return ps, wrapParse(s.FranchiseID, "pwr", err)
	}
	if ps.altPwr, err = atofOrZero(s.AltPwr); err != nil {
		return ps, wrapParse(s.FranchiseID, "altpwr", err)
	}
	// All-play win% over ACTUAL games played (not a hardcoded 527) so a mid-season
	// or short-schedule pull is still correct; zero games → 0, never NaN. Ties count
	// as half a win — the conventional (W + 0.5T)/G rate, so a tie is neither a full
	// win nor a full loss.
	if games := ps.allPlayW + ps.allPlayL + ps.allPlayT; games > 0 {
		ps.allPlayWinPct = (float64(ps.allPlayW) + 0.5*float64(ps.allPlayT)) / float64(games)
	}
	return ps, nil
}

func wrapParse(fid, field string, err error) error {
	return fmt.Errorf("power rankings: franchise %s field %s: %w", fid, field, err)
}

// atoiOrZero parses an MFL integer field; an empty field is a legitimate 0. It
// applies the same currency/thousands sanitization as the fetcher's shape check so
// the two never disagree on what parses.
func atoiOrZero(s string) (int, error) {
	s = leaguestandings.SanitizeNumeric(s)
	if s == "" {
		return 0, nil
	}
	// Some MFL integer-ish fields arrive with a trailing ".0" — tolerate by
	// parsing as float then truncating, so "421" and "421.0" both work.
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("parse int field %q: %w", s, err)
	}
	return int(f), nil
}

// atofOrZero parses an MFL float field; an empty field is a legitimate 0.
func atofOrZero(s string) (float64, error) {
	s = leaguestandings.SanitizeNumeric(s)
	if s == "" {
		return 0, nil
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("parse float field %q: %w", s, err)
	}
	return f, nil
}

// clampWeight mirrors Blend's clamp (incl. the non-finite fallback) so the echoed
// Weight matches the weight actually applied.
func clampWeight(w float64) float64 {
	if math.IsNaN(w) || math.IsInf(w, 0) {
		w = powerrankings.DefaultScoutingWeight
	}
	return math.Max(0, math.Min(1, w))
}

// franchiseDisplayName resolves a franchise id to its league name, falling back to
// a labeled id so an unmapped franchise reads plainly rather than as a blank cell.
func franchiseDisplayName(names map[string]string, fid string) string {
	if n, ok := names[fid]; ok && n != "" {
		return n
	}
	return "(franchise " + fid + ")"
}
