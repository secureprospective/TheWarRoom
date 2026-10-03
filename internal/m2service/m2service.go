// Package m2service builds the M2 power-rankings board: it aggregates each franchise's M1
// scores from runtime state, blends them with the MFL standings through powerrankings, and
// joins the display columns. It holds read surfaces only, so m2_app.go stays a thin adapter.
package m2service

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leaguestandings"
	"github.com/secureprospective/TheWarRoom/internal/output"
	"github.com/secureprospective/TheWarRoom/internal/powerrankings"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// Aggregation modes for BuildBoard.
const (
	AggSum  = "sum"  // Σ AdjustedScore over the whole roster — rewards dynasty depth
	AggTopN = "topn" // Σ of the top-N by AdjustedScore — isolates startable talent
)

// FranchiseSource supplies the offline rulebook reads BuildBoard needs: franchise names and
// the starter count.
type FranchiseSource interface {
	FranchiseNames() map[string]string
	ActiveConfig() league.RawConfig
}

// Service builds the M2 board from read surfaces only.
type Service struct {
	state state.Reader
	rb    FranchiseSource
}

// New wires a Service. Both dependencies are required.
func New(st state.Reader, rb FranchiseSource) (*Service, error) {
	if st == nil || rb == nil {
		return nil, fmt.Errorf("m2service: nil dependency (state=%t rulebook=%t)", st != nil, rb != nil)
	}
	return &Service{state: st, rb: rb}, nil
}

// Row is one franchise's board row: the blended score, MFL's report columns and the name.
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

// Board is the rows plus the mode, starter count and weight actually applied, which the UI
// echoes back to its controls.
type Board struct {
	Rows     []Row
	Mode     string
	StarterN int
	Weight   float64
}

// BuildBoard aggregates each franchise's M1 AdjustedScores by mode (top-N falls back to sum
// when the starter count is unreadable), blends them against the MFL standings and joins the
// display columns.
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

// ResolveAggMode defaults an empty or unknown mode to sum. It is exported so the adapter echoes
// the same resolved mode on its early-error paths.
func ResolveAggMode(m string) string {
	if m == AggTopN {
		return AggTopN
	}
	return AggSum
}

// starterCount reads the league's starter count; 0 when unset or unparseable.
func (s *Service) starterCount() int {
	n, err := strconv.Atoi(strings.TrimSpace(s.rb.ActiveConfig().Starters.Count))
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// buildBlendInputs aggregates scores per franchise and parses each standings row once. The
// standings define the franchise set; a franchise with no scored players contributes 0.
func (s *Service) buildBlendInputs(
	standings []leaguestandings.RawStanding,
	scores []output.SeasonScore,
	mode string,
	starterN int,
) ([]powerrankings.Input, map[string]parsedStanding, error) {
	// Ownership comes from live runtime state, so a player traded since the M1 run counts for
	// the current team: the board reads "who is strong now".
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

// aggregateScouting reduces a franchise's scores to the full sum (depth) or the sum of the
// top N (startable talent). It sorts a copy.
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

// buildRows joins the blended scores with MFL's display columns and the rulebook's names.
func (s *Service) buildRows(blended []powerrankings.Row, parsed map[string]parsedStanding) []Row {
	names := s.rb.FranchiseNames()
	rows := make([]Row, 0, len(blended))
	for _, b := range blended {
		ps := parsed[b.FranchiseID]
		rows = append(rows, Row{
			Rank:          b.Rank,
			FranchiseID:   b.FranchiseID,
			Name:          domain.FranchiseLabel(names, b.FranchiseID),
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

// parsedStanding is a RawStanding's numbers, parsed once.
type parsedStanding struct {
	h2hW, h2hL, h2hT             int
	allPlayW, allPlayL, allPlayT int
	pf, pa, pp, pwr, altPwr      float64
	allPlayWinPct                float64
}

// parseStanding converts a RawStanding that already passed Validate, so empty means 0 and a
// parse error is a real invariant break.
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
	// All-play win% over games actually played, ties as half a win; zero games gives 0.
	if games := ps.allPlayW + ps.allPlayL + ps.allPlayT; games > 0 {
		ps.allPlayWinPct = (float64(ps.allPlayW) + 0.5*float64(ps.allPlayT)) / float64(games)
	}
	return ps, nil
}

func wrapParse(fid, field string, err error) error {
	return fmt.Errorf("power rankings: franchise %s field %s: %w", fid, field, err)
}

// atoiOrZero parses an MFL integer field with the fetcher's sanitization; empty is 0.
func atoiOrZero(s string) (int, error) {
	s = leaguestandings.SanitizeNumeric(s)
	if s == "" {
		return 0, nil
	}
	// Some MFL integer fields arrive as "421.0".
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

// clampWeight mirrors Blend's clamp so the echoed weight is the one applied.
func clampWeight(w float64) float64 {
	if math.IsNaN(w) || math.IsInf(w, 0) {
		w = powerrankings.DefaultScoutingWeight
	}
	return math.Max(0, math.Min(1, w))
}
