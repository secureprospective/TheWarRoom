// Package m2service builds the M2 power-rankings board: it sums each franchise's player values
// (a measurable from the latest model run) by current ownership, blends that with the MFL
// standings through powerrankings, and joins the display columns. It holds read surfaces only,
// so m2_app.go stays a thin adapter.
package m2service

import (
	"fmt"
	"math"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leaguestandings"
	"github.com/secureprospective/TheWarRoom/internal/powerrankings"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// Aggregation modes for BuildBoard.
const (
	AggSum    = "sum"    // Σ value over the whole roster — rewards depth
	AggTopN   = "topn"   // Σ of the top N by value, whatever their positions
	AggLineup = "lineup" // Σ of the best lineup the league's starter rules allow
)

// FranchiseSource supplies the offline rulebook reads the board needs: franchise names, the
// starter rules and the salary cap.
type FranchiseSource interface {
	FranchiseNames() map[string]string
	ActiveConfig() league.RawConfig
	GetSalaryCap() string
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
	RosterZ    float64
	MFLPerfZ   float64
	AgeZ       float64

	RosterValue float64
	Age         float64 // the counted players' value-weighted age; NaN when none is known
	Results     float64 // the result the blend read, in [0,1]: all-play win% or points for ÷ the league's best

	H2HW, H2HL, H2HT             int
	AllPlayW, AllPlayL, AllPlayT int
	PF, PA, PP, Pwr, AltPwr      float64
}

// Board is the rows plus the mode, starter count and weights actually applied, which the UI
// echoes back to its controls, and which result the blend's performance side read.
type Board struct {
	Rows        []Row
	Mode        string
	StarterN    int
	Weight      float64
	AgeWeight   float64
	Performance string // PerfAllPlay, PerfPointsFor, or PerfNone before any result
}

// The results the blend's performance side reads: all-play win% when MFL reports it, otherwise
// points for, which is as free of schedule luck; this league's standings carry no all-play.
const (
	PerfAllPlay   = "all-play"
	PerfPointsFor = "points for"
	PerfNone      = "none"
)

// PlayerValue is one player's value in the view being ranked, in league points per game, with
// the position he was valued at and his age (NaN when unknown).
type PlayerValue struct {
	MFLID    string
	Position string
	Value    float64
	Age      float64
}

// BuildBoard counts each franchise's player values by mode (top-N and the lineup fall back to
// sum when the starter rules are unreadable), blends them against the MFL standings at the
// weights and joins the display columns.
func (s *Service) BuildBoard(
	standings []leaguestandings.RawStanding,
	values []PlayerValue,
	wt powerrankings.Weights,
	aggMode string,
) (Board, error) {
	c := s.counter(ResolveAggMode(aggMode, AggLineup))

	inputs, parsed, perf, err := s.buildBlendInputs(standings, values, c)
	if err != nil {
		return Board{}, err
	}

	blended, err := powerrankings.Blend(inputs, wt)
	if err != nil {
		return Board{}, fmt.Errorf("m2service: blend: %w", err)
	}

	n := 0
	if c.mode != AggSum {
		n = c.rules.Total
	}
	return Board{
		Rows:        s.buildRows(blended, parsed),
		Mode:        c.mode,
		StarterN:    n,
		Weight:      powerrankings.ClampWeight(wt.Roster),
		AgeWeight:   wt.Age,
		Performance: perf,
	}, nil
}

// ResolveAggMode returns m when it is a known mode, otherwise fallback. It is exported so the
// adapter echoes the same resolved mode on its early-error paths.
func ResolveAggMode(m, fallback string) string {
	switch m {
	case AggSum, AggTopN, AggLineup:
		return m
	}
	return fallback
}

// counter counts a franchise's players in one mode under the league's starter rules.
type counter struct {
	mode  string
	rules powerrankings.LineupRules
}

// counter resolves mode against the league's starter rules: top-N needs the starter count and
// the lineup every position's bounds; either falls back to sum without them.
func (s *Service) counter(mode string) counter {
	rules, ok := LineupRules(s.rb.ActiveConfig().Starters)
	switch {
	case mode == AggTopN && rules.Total > 0:
	case mode == AggLineup && ok:
	default:
		mode = AggSum
	}
	return counter{mode: mode, rules: rules}
}

// LineupRules reads the league's starter rules; ok is false when any bound is unreadable, and
// Total is 0 when the starter count is.
func LineupRules(st league.Starters) (powerrankings.LineupRules, bool) {
	var r powerrankings.LineupRules
	total, err := strconv.Atoi(strings.TrimSpace(st.Count))
	if err != nil || total <= 0 {
		return r, false
	}
	r.Total = total
	def, err := strconv.Atoi(strings.TrimSpace(st.IDPStarters))
	if err != nil || def < 0 || def > total || len(st.Positions) == 0 {
		return r, false
	}
	r.Defense = def
	for _, p := range st.Positions {
		lo, hi, ok := bounds(p.Limit)
		if !ok {
			return r, false
		}
		r.Slots = append(r.Slots, powerrankings.Slot{Position: p.Name, Min: lo, Max: hi})
	}
	return r, true
}

// bounds parses MFL's "1-3" or "1".
func bounds(limit string) (lo, hi int, ok bool) {
	a, b, ranged := strings.Cut(strings.TrimSpace(limit), "-")
	lo, err := strconv.Atoi(a)
	if err != nil || lo < 0 {
		return 0, 0, false
	}
	if !ranged {
		return lo, lo, true
	}
	hi, err = strconv.Atoi(b)
	if err != nil || hi < lo {
		return 0, 0, false
	}
	return lo, hi, true
}

// buildBlendInputs aggregates values per franchise and parses each standings row once. The
// standings define the franchise set; a franchise with no valued players contributes 0.
func (s *Service) buildBlendInputs(
	standings []leaguestandings.RawStanding,
	values []PlayerValue,
	c counter,
) ([]powerrankings.Input, map[string]parsedStanding, string, error) {
	byFranchise := s.byFranchise(values)

	parsed := make(map[string]parsedStanding, len(standings))
	var allPlayGames, maxPF float64
	for _, st := range standings {
		ps, err := parseStanding(st)
		if err != nil {
			return nil, nil, "", err
		}
		parsed[st.FranchiseID] = ps
		allPlayGames += float64(ps.allPlayW + ps.allPlayL + ps.allPlayT)
		maxPF = math.Max(maxPF, ps.pf)
	}
	perf := PerfNone
	switch {
	case allPlayGames > 0:
		perf = PerfAllPlay
	case maxPF > 0:
		perf = PerfPointsFor
	}
	inputs := make([]powerrankings.Input, 0, len(standings))
	for _, st := range standings {
		ps := parsed[st.FranchiseID]
		value, age := c.count(byFranchise[st.FranchiseID])
		in := powerrankings.Input{FranchiseID: st.FranchiseID, RosterValue: value, Age: age}
		switch perf {
		case PerfAllPlay:
			in.Performance = ps.allPlayWinPct
		case PerfPointsFor:
			in.Performance = ps.pf / maxPF
		}
		inputs = append(inputs, in)
	}
	return inputs, parsed, perf, nil
}

// byFranchise groups values by current owner. Ownership comes from live runtime state, so a
// player traded since the model run counts for the current team: the board reads "who is strong
// now".
func (s *Service) byFranchise(values []PlayerValue) map[string][]PlayerValue {
	out := map[string][]PlayerValue{}
	for _, v := range values {
		if p, ok := s.state.Player(v.MFLID); ok {
			out[p.FranchiseID] = append(out[p.FranchiseID], v)
		}
	}
	return out
}

// count is a franchise's value in the counter's mode, and the value-weighted age of the players
// it counted (NaN when none of them has a known age and positive value). It sorts a copy.
func (c counter) count(vals []PlayerValue) (value, age float64) {
	counted := vals
	switch c.mode {
	case AggTopN:
		cp := slices.Clone(vals)
		sort.SliceStable(cp, func(i, j int) bool { return cp[i].Value > cp[j].Value })
		counted = cp[:min(c.rules.Total, len(cp))]
	case AggLineup:
		cands := make([]powerrankings.Candidate, len(vals))
		for i, v := range vals {
			cands[i] = powerrankings.Candidate{Position: v.Position, Value: v.Value}
		}
		_, idx := powerrankings.Lineup(cands, c.rules)
		counted = make([]PlayerValue, len(idx))
		for i, j := range idx {
			counted[i] = vals[j]
		}
	}
	var weight, aged float64
	for _, v := range counted {
		value += v.Value
		if !math.IsNaN(v.Age) && v.Value > 0 {
			weight += v.Value
			aged += v.Value * v.Age
		}
	}
	if weight == 0 {
		return value, math.NaN()
	}
	return value, aged / weight
}

// buildRows joins the blended scores with MFL's display columns and the rulebook's names.
func (s *Service) buildRows(blended []powerrankings.Row, parsed map[string]parsedStanding) []Row {
	names := s.rb.FranchiseNames()
	rows := make([]Row, 0, len(blended))
	for _, b := range blended {
		ps := parsed[b.FranchiseID]
		rows = append(rows, Row{
			Rank:        b.Rank,
			FranchiseID: b.FranchiseID,
			Name:        domain.FranchiseLabel(names, b.FranchiseID),
			PowerScore:  b.PowerScore,
			RosterZ:     b.RosterZ,
			MFLPerfZ:    b.MFLPerfZ,
			AgeZ:        b.AgeZ,
			RosterValue: b.RosterValue,
			Age:         b.Age,
			Results:     b.Performance,
			H2HW:        ps.h2hW, H2HL: ps.h2hL, H2HT: ps.h2hT,
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
