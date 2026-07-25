// Package m2service is the M2 "power rankings board" orchestrator — the mirror of
// internal/rankings' relationship to m1_app.go (Session 43, SLIM_MAP §6.1). It is
// composition-class code: the ONE layer allowed to hold a state.Reader and a
// rulebook read surface for M2, so m2_app.go can stay a thin adapter (validate →
// fetch IO → route → format), same shape as ScoreLeague delegating to
// rankings.Runner. internal/powerrankings stays the pure blend-math leaf beneath
// this package, untouched (no I/O, no store — its own doc comment's contract).
//
// This is a MECHANICAL extraction (Session 43): every function here is a verbatim
// move from m2_app.go, byte-identical math and byte-identical output. No blend
// formula, no franchise-name fallback, no parse tolerance changed.
package m2service

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leaguestandings"
	"github.com/secureprospective/TheWarRoom/internal/output"
	"github.com/secureprospective/TheWarRoom/internal/powerrankings"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// Aggregation modes for BuildBoard's mode param.
const (
	AggSum  = "sum"  // Σ AdjustedScore over the whole roster — rewards dynasty depth
	AggTopN = "topn" // Σ of the top-N by AdjustedScore — isolates startable talent
)

// FranchiseSource supplies the local (no-network) rulebook reads BuildBoard needs:
// display names and the starter count used to resolve top-N. *rulebook.Store
// satisfies it structurally.
type FranchiseSource interface {
	FranchiseNames() map[string]string
	ActiveConfig() league.RawConfig
}

// Service owns the M2 composition: aggregate + blend + join. It holds READ
// surfaces only — like rankings.Runner, it cannot mutate league state.
type Service struct {
	state state.Reader
	rb    FranchiseSource
}

// New wires a Service. Both dependencies are required — a nil here is a
// programmer error surfaced at construction, mirroring rankings.New.
func New(st state.Reader, rb FranchiseSource) (*Service, error) {
	if st == nil || rb == nil {
		return nil, fmt.Errorf("m2service: nil dependency (state=%t rulebook=%t)", st != nil, rb != nil)
	}
	return &Service{state: st, rb: rb}, nil
}

// Row is one franchise's fully joined M2 board row: the blended score plus the
// MFL report passthrough columns and the display name. m2_app.go copies this
// field-for-field into the Wails-bound PowerRow DTO (the adapter's FORMAT step).
type Row struct {
	Rank        int
	FranchiseID string
	Name        string

	PowerScore float64
	ScoutingZ  float64
	MFLPerfZ   float64

	ScoutingScore float64
	AllPlayWinPct float64

	H2HW, H2HL, H2HT             int
	AllPlayW, AllPlayL, AllPlayT int
	PF, PA, PP, Pwr, AltPwr      float64
}

// Board is the BuildBoard result: the joined rows plus the echoed mode/starterN/
// weight so the IPC layer's echo fields (which the UI slider trusts) never
// disagree with what was actually applied.
type Board struct {
	Rows     []Row
	Mode     string
	StarterN int
	Weight   float64
}

// BuildBoard runs the full M2 composition: resolves the aggregation mode,
// resolves the starter count for top-N (degrading to sum if unreadable),
// aggregates each franchise's scouting AdjustedScore from the M1 board via the
// injected state.Reader, blends against the MFL standings, and joins the
// display columns. Byte-identical to the pre-extraction m2_app.go pipeline.
func (s *Service) BuildBoard(
	standings []leaguestandings.RawStanding,
	scores []output.SeasonScore,
	weight float64,
	aggMode string,
) (Board, error) {
	mode := ResolveAggMode(aggMode)

	starterN := s.starterCount()
	if mode == AggTopN && starterN <= 0 {
		mode = AggSum
	}

	inputs, parsed, err := s.buildBlendInputs(standings, scores, mode, starterN)
	if err != nil {
		return Board{}, err
	}

	blended, err := powerrankings.Blend(inputs, weight)
	if err != nil {
		return Board{}, fmt.Errorf("m2service: blend: %w", err)
	}

	n := 0
	if mode == AggTopN {
		n = starterN
	}
	return Board{
		Rows:     s.buildRows(blended, parsed),
		Mode:     mode,
		StarterN: n,
		Weight:   clampWeight(weight),
	}, nil
}

// ResolveAggMode normalizes the caller's aggregation mode, defaulting to sum for an
// empty or unrecognized value so a bad param never errors the view. Exported so the
// app-layer adapter can echo the SAME resolved value on its early-fail paths (before
// BuildBoard ever runs) that it echoes on success — GLM 5.2 review lead 1 (Session
// 43): the pre-fix fail closure echoed the raw, unresolved caller argument.
func ResolveAggMode(m string) string {
	if m == AggTopN {
		return AggTopN
	}
	return AggSum
}

// starterCount reads the league's total starter count from the active rulebook
// config; 0 if unset/unparseable (caller degrades top-N to sum).
func (s *Service) starterCount() int {
	n, err := strconv.Atoi(strings.TrimSpace(s.rb.ActiveConfig().Starters.Count))
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
func (s *Service) buildBlendInputs(
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
	for _, sc := range scores {
		if p, ok := s.state.Player(sc.MFLID); ok {
			scoresByFranchise[p.FranchiseID] = append(scoresByFranchise[p.FranchiseID], sc.AdjustedScore)
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
// the full sum (AggSum, rewards dynasty depth) or the sum of the top-N by score
// (AggTopN, isolates startable talent). Top-N sorts a COPY so the caller's slice is
// untouched; N ≥ len means the whole roster.
func aggregateScouting(scores []float64, mode string, starterN int) float64 {
	if mode == AggTopN && starterN > 0 && starterN < len(scores) {
		cp := make([]float64, len(scores))
		copy(cp, scores)
		sort.Sort(sort.Reverse(sort.Float64Slice(cp)))
		scores = cp[:starterN]
	}
	var sum float64
	for _, sc := range scores {
		sum += sc
	}
	return sum
}

// buildRows joins the blended scores with the MFL display columns and local
// franchise names. Team names come from the LOCAL rulebook (no network) — unlike M1
// player names, there is no offline-degrade path to worry about here.
func (s *Service) buildRows(blended []powerrankings.Row, parsed map[string]parsedStanding) []Row {
	names := s.rb.FranchiseNames()
	rows := make([]Row, 0, len(blended))
	for _, b := range blended {
		ps := parsed[b.FranchiseID]
		rows = append(rows, Row{
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
