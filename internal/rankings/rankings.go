// Package rankings scores the league (M1). For every rostered player it assembles engine
// inputs through composition, runs the pipeline with the position's rubric, and writes the
// batch through output.Writer stamped with the rulebook's active config version. It reads
// several stores but writes only through that one Writer.
//
// BasePoints is a labeled placeholder: a completed season's fantasy points in the league's own
// scoring, passed in as a map until a real L2 base exists.
//
// Data policy:
//   - no players-DB record, an aggregate, or a FLAG position: excluded, with a reason
//   - no birthdate: excluded, because a faked age would silently corrupt L3 decay
//   - no season score (mostly rookies): scored with BasePoints 0 and counted in the report
package rankings

import (
	"context"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/composition"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/output"
	"github.com/secureprospective/TheWarRoom/internal/scouting"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// Directory resolves a rostered mfl id to its players-DB facts. normalize.Lookup
// satisfies it.
type Directory interface {
	Facts(mflID string) (normalize.PlayerFacts, bool)
}

// ConfigSource supplies the active rulebook version the batch is stamped with.
type ConfigSource interface {
	ActiveVersion(ctx context.Context) (int, error)
}

// Registry maps a position to its Layer-4 rubric. An unregistered position falls back to
// identity L4.
type Registry map[domain.Position]engine.Layer4

// Runner runs one scoring pass. It holds read surfaces plus output.Writer and cannot change
// league state.
type Runner struct {
	state state.Reader
	dir   Directory
	scout ScoutingDirectory
	base  map[string]float64 // mflID → season fantasy points (L2 placeholder)
	cfg   ConfigSource
	out   output.Writer
	asm   *composition.Assembler
	reg   Registry
}

// New wires a Runner. Every dependency is required. An empty MapScoutingDirectory is legal
// (no scouting this pass); a nil one is a wiring error.
func New(st state.Reader, dir Directory, scout ScoutingDirectory, base map[string]float64, cfg ConfigSource,
	out output.Writer, asm *composition.Assembler, reg Registry) (*Runner, error) {
	if st == nil || dir == nil || scout == nil || cfg == nil || out == nil || asm == nil || reg == nil {
		// A nil Registry would score every position as identity L4 and freeze the wrong board, so
		// it is refused; an empty Registry{} is a deliberate choice and allowed.
		return nil, fmt.Errorf("rankings: nil dependency (state=%t dir=%t scout=%t cfg=%t out=%t asm=%t reg=%t)",
			st != nil, dir != nil, scout != nil, cfg != nil, out != nil, asm != nil, reg != nil)
	}
	if base == nil {
		return nil, fmt.Errorf("rankings: nil BasePoints map — the L2 placeholder source is required (an empty league of zeros must be deliberate, not a wiring accident)")
	}
	return &Runner{state: st, dir: dir, scout: scout, base: base, cfg: cfg, out: out, asm: asm, reg: reg}, nil
}

// Exclusion is a rostered player the pass could not score, with the reason. The UI shows
// these beside the board.
type Exclusion struct {
	MFLID       string `json:"mflID"`
	Name        string `json:"name"`
	FranchiseID string `json:"franchiseID"`
	// FranchiseName is a display label, filled in by the App adapter.
	FranchiseName string `json:"franchiseName"`
	Reason        string `json:"reason"`
}

// Report is what one pass scored, under which config, and what it skipped or excluded.
type Report struct {
	Season          int         `json:"season"`
	ConfigVersion   int         `json:"configVersion"`
	Scored          int         `json:"scored"`          // scored this pass; 0 when skipped
	SkippedExisting bool        `json:"skippedExisting"` // already scored; nothing written
	Existing        int         `json:"existing"`
	ZeroBase        int         `json:"zeroBase"`     // no season score, scored at 0
	NegativeBase    int         `json:"negativeBase"` // a negative total floored to 0
	Excluded        []Exclusion `json:"excluded"`
}

// Run scores every rostered player for season and persists the batch. asOf anchors ages.
// If this (season, config) is already scored, it reports the existing batch and writes
// nothing.
func (r *Runner) Run(ctx context.Context, season int, asOf time.Time) (Report, error) {
	ver, err := r.cfg.ActiveVersion(ctx)
	if err != nil {
		return Report{}, fmt.Errorf("rankings: read active config version: %w", err)
	}
	if ver == 0 {
		return Report{}, fmt.Errorf("rankings: no active scoring config (version 0) — initialize the rulebook before scoring")
	}
	rep := Report{Season: season, ConfigVersion: ver}

	existing, err := r.out.Scores(ctx, season, ver)
	if err != nil {
		return Report{}, fmt.Errorf("rankings: check existing scores: %w", err)
	}
	if len(existing) > 0 {
		// Scored stays 0: nothing was scored this pass.
		rep.SkippedExisting = true
		rep.Existing = len(existing)
		return rep, nil
	}

	var recs []output.ScoreRecord
	for _, fid := range r.state.Franchises() {
		roster, ok := r.state.Roster(fid)
		if !ok {
			return Report{}, fmt.Errorf("rankings: franchise %q listed but has no roster (store drift)", fid)
		}
		for _, p := range roster {
			rec, excl, origin := r.scorePlayer(fid, p, asOf)
			if excl != nil {
				rep.Excluded = append(rep.Excluded, *excl)
				continue
			}
			switch origin {
			case baseAbsent:
				rep.ZeroBase++
			case baseNegative:
				rep.NegativeBase++
			case baseReal:
			}
			recs = append(recs, rec)
		}
	}
	if len(recs) == 0 {
		return Report{}, fmt.Errorf("rankings: zero scorable players across %d franchises — refusing to persist an empty league", len(r.state.Franchises()))
	}

	if err := r.out.Write(ctx, season, ver, recs); err != nil {
		return Report{}, fmt.Errorf("rankings: persist %d scores (season %d, config %d): %w", len(recs), season, ver, err)
	}
	rep.Scored = len(recs)
	return rep, nil
}

// baseOrigin records where BasePoints came from, so the report can tell an expected rookie
// zero from a floored negative total (a data smell).
type baseOrigin int

const (
	baseReal     baseOrigin = iota // a real YTD total fed through
	baseAbsent                     // no YTD record → 0
	baseNegative                   // negative YTD total → floored to 0
)

// scorePlayer returns either a ScoreRecord or a user-facing Exclusion for one player, plus
// where its BasePoints came from.
func (r *Runner) scorePlayer(fid string, p state.PlayerState, asOf time.Time) (output.ScoreRecord, *Exclusion, baseOrigin) {
	facts, ok := r.dir.Facts(p.MFLID)
	if !ok {
		return output.ScoreRecord{}, &Exclusion{MFLID: p.MFLID, FranchiseID: fid,
			Reason: "not in the players database (aggregate or unknown id)"}, baseReal
	}
	if facts.Position == domain.PosFlag {
		return output.ScoreRecord{}, &Exclusion{MFLID: p.MFLID, Name: facts.Name, FranchiseID: fid,
			Reason: "unclassified position (FLAG) — resolve before scoring"}, baseReal
	}
	if !facts.HasBirthdate {
		return output.ScoreRecord{}, &Exclusion{MFLID: p.MFLID, Name: facts.Name, FranchiseID: fid,
			Reason: "missing birthdate — age is a required engine input; a faked age would corrupt L3 decay"}, baseReal
	}
	age := yearsBetween(time.Unix(facts.Birthdate, 0).UTC(), asOf)
	if age <= 0 {
		return output.ScoreRecord{}, &Exclusion{MFLID: p.MFLID, Name: facts.Name, FranchiseID: fid,
			Reason: fmt.Sprintf("implausible age %.1f from birthdate — players-DB data error", age)}, baseReal
	}

	basePts, hasBase := r.base[p.MFLID]
	origin := baseReal
	if !hasBase {
		origin = baseAbsent
	}
	if basePts < 0 {
		// A season can total negative, which the engine rejects. Floor it to 0 and count it apart
		// from absent.
		basePts, origin = 0, baseNegative
	}

	spec := composition.PlayerSpec{
		MFLID:      p.MFLID,
		Name:       facts.Name,
		Position:   facts.Position,
		BasePoints: basePts,
		Age:        age,
		// Money becomes float millions only here; L5 uses salary as a ratio of the cap.
		Salary:    p.CapSalary.Millions(),
		IsVeteran: !facts.IsRookie,
	}
	if profile, ok := r.scout.Profile(p.MFLID); ok {
		applyScouting(&spec, profile)
	}
	spec.Position = composition.ResolveRubricPosition(spec) // no snap share wired → passthrough today

	in, sc, cal, err := r.asm.Assemble(spec)
	if err != nil {
		return output.ScoreRecord{}, &Exclusion{MFLID: p.MFLID, Name: facts.Name, FranchiseID: fid,
			Reason: fmt.Sprintf("assemble: %v", err)}, baseReal
	}
	res, err := engine.NewPipeline(r.reg[spec.Position]).Score(in, sc, cal)
	if err != nil {
		return output.ScoreRecord{}, &Exclusion{MFLID: p.MFLID, Name: facts.Name, FranchiseID: fid,
			Reason: fmt.Sprintf("score: %v", err)}, baseReal
	}
	return output.ScoreRecord{MFLID: p.MFLID, Result: res}, nil, origin
}

// applyScouting copies each signal the Profile carries into the spec, gated per field. An
// untouched field stays absent and the rubric neutralizes it. RAS gates on HasRAS,
// SchoolTier on SchoolUnset, CollegeShare and BreakoutAge on their Has* flags (0 is a real
// value for both).
func applyScouting(spec *composition.PlayerSpec, profile scouting.Profile) {
	if profile.HasRAS {
		spec.RAS = profile.RAS
		spec.HasRAS = true
	}
	if profile.SchoolTier != scouting.SchoolUnset {
		spec.SchoolTier = profile.SchoolTier
	}
	if profile.HasCollegeProductionShare {
		spec.CollegeShare = profile.CollegeProductionShare
		spec.HasCollegeShare = true
	}
	if profile.HasBreakoutAge {
		spec.BreakoutAge = profile.BreakoutAge
		spec.HasBreakoutAge = true
	}
	// IDP film composite (DT/DE/LB/CB/S), built here because the engine takes one [0,1]
	// FilmComposite. Seats: the Madden defense composite, the PFR coverage anchor (CB/S only) and
	// NFLProduction (reserved, not wired). An absent seat contributes the neutral midpoint:
	//   - DT/DE/LB: 0.95·Madden + 0.05·neutral
	//   - CB/S:     0.20·coverage + 0.75·Madden + 0.05·neutral
	// A player with neither signal leaves HasFilm false.
	hasCoverage := profile.Coverage != nil
	hasMadden := profile.IDPFilm != nil
	if hasCoverage || hasMadden {
		coverageWeight := 0.0
		coverageTerm := 0.0
		if hasCoverage {
			coverageWeight = coverageFilmWeight // 0.20, CB/S only
			coverageTerm = coverageWeight * profile.Coverage.CoverageMetrics
		}
		maddenWeight := 1 - coverageWeight - nflProductionFilmWeight
		maddenTerm := maddenWeight * filmNeutralMidpoint // neutral when no Madden record
		if hasMadden {
			maddenTerm = maddenWeight * profile.IDPFilm.MaddenComposite
		}
		spec.FilmComposite = coverageTerm + maddenTerm +
			nflProductionFilmWeight*filmNeutralMidpoint
		spec.HasFilm = true
	} else if profile.OffenseFilm != nil {
		// Offense film (QB/RB/WR/TE): the assembler already blended Profile.OffenseFilm, so only the
		// NFLProduction seat is reserved here: 0.95·Composite + 0.05·neutral. Offense and IDP film are
		// exclusive by position; the else-if keeps one from overwriting the other.
		spec.FilmComposite = (1-nflProductionFilmWeight)*profile.OffenseFilm.Composite +
			nflProductionFilmWeight*filmNeutralMidpoint
		spec.HasFilm = true
	}
}

// Film weights (locked decisions K4 and K2). NFLProduction's seat holds the neutral midpoint
// until it is wired; the film S-curve inflects at 0.50.
const (
	coverageFilmWeight      = 0.20
	nflProductionFilmWeight = 0.05
	filmNeutralMidpoint     = 0.50
)

// yearsBetween returns fractional years. Rounding would step every player's L3 decay in
// lockstep.
func yearsBetween(birth, asOf time.Time) float64 {
	return asOf.Sub(birth).Hours() / (24 * 365.2425)
}
