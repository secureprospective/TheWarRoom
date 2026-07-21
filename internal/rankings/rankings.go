// Package rankings is the M1 "score the league" orchestrator: the first code that
// reads the store floor as a whole. For every rostered player in B3c it assembles
// engine inputs through the composition boundary, runs the pure pipeline with the
// real B5b rubric for the player's position, and persists the batch to the B6
// output store STAMPED with the rulebook's ACTIVE config version (DECISION-010 —
// the orchestrator stamps, B6 never mints). It is composition-class code: the ONE
// layer allowed to import several stores, but it writes ONLY through output.Writer
// and duplicates no store (M1 handoff constraint).
//
// L2 NOTE (Christopher, 2026-06-28, option (b)): BasePoints is the LABELED
// MFL-fantasy-points placeholder — a completed season's YTD totals in the league's
// own scoring — until the real L2 base-scoring block ships. The proxy enters here
// as plain data (a map), so swapping in real L2 later changes the SOURCE the app
// wires, not this orchestrator and not the engine.
//
// Per-player data policy (gate-checked 2026-07-02):
//   - no players-DB record, aggregate, or FLAG position → EXCLUDED, with reason
//   - no birthdate (6 of 1232 live) → EXCLUDED — a faked age would silently corrupt
//     L3 decay; a visible exclusion beats an invisible lie
//   - no YTD score (83 of 1232 live, mostly rookies) → scored with BasePoints 0,
//     counted in the report; honest for a labeled "L2 pending" board
//
// Scouting sub-signals flow through this orchestrator as of S-Phase 0 (RAS)
// via the ScoutingDirectory port: a rostered player whose profile is present
// gets spec.RAS/HasRAS set; an absent profile falls through to L1's
// DefaultRASFallback imputation (unchanged behavior). The other scouting
// sub-signals (film, breakout) remain Data-Parity ABSENT this phase — every
// other Has* stays false, so the rubrics neutralize them.
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

// Directory resolves a rostered mfl id to its players-DB facts (name, position,
// rookie flag, birthdate). normalize.Lookup satisfies it; a fake drops in for tests.
type Directory interface {
	Facts(mflID string) (normalize.PlayerFacts, bool)
}

// ConfigSource supplies the rulebook ACTIVE config version the batch is stamped
// with. *rulebook.Store satisfies it structurally.
type ConfigSource interface {
	ActiveVersion(ctx context.Context) (int, error)
}

// Registry maps a position to its real B5b Layer-4 rubric — the same shape the
// harness registry uses (an unregistered position falls back to identity L4 via
// engine.NewPipeline(nil), though all 10 positions ship real rubrics today).
type Registry map[domain.Position]engine.Layer4

// Runner owns one league-scoring pass end to end. It holds READ surfaces plus the
// single sanctioned write path (output.Writer); it cannot mutate league state.
type Runner struct {
	state state.Reader
	dir   Directory
	scout ScoutingDirectory
	base  map[string]float64 // mflID → YTD fantasy points (the labeled L2 placeholder)
	cfg   ConfigSource
	out   output.Writer
	asm   *composition.Assembler
	reg   Registry
}

// New wires a Runner. Every dependency is required — a nil here is a programmer
// error surfaced at construction, not a silent zero-scored league at run time.
// The scouting directory follows the same nil-guard rule as the others: a nil
// directory is a wiring error, an explicitly-empty MapScoutingDirectory is a
// legal "no scouting signal this pass" condition (every player misses cleanly,
// L1 imputes the per-signal fallbacks).
func New(st state.Reader, dir Directory, scout ScoutingDirectory, base map[string]float64, cfg ConfigSource,
	out output.Writer, asm *composition.Assembler, reg Registry) (*Runner, error) {
	if st == nil || dir == nil || scout == nil || cfg == nil || out == nil || asm == nil || reg == nil {
		// reg is in the guard deliberately (GLM M1 review): a nil map reads as
		// identity-L4 for EVERY position without panicking, so a miswired app would
		// silently persist a whole wrong-rubric league that B6 then freezes. An
		// explicitly-empty Registry{} stays legal — that is a deliberate act.
		return nil, fmt.Errorf("rankings: nil dependency (state=%t dir=%t scout=%t cfg=%t out=%t asm=%t reg=%t)",
			st != nil, dir != nil, scout != nil, cfg != nil, out != nil, asm != nil, reg != nil)
	}
	if base == nil {
		return nil, fmt.Errorf("rankings: nil BasePoints map — the L2 placeholder source is required (an empty league of zeros must be deliberate, not a wiring accident)")
	}
	return &Runner{state: st, dir: dir, scout: scout, base: base, cfg: cfg, out: out, asm: asm, reg: reg}, nil
}

// Exclusion is one rostered player the pass could NOT score, with the reason. The
// UI renders these alongside the board — an invisible exclusion is a silent lie.
type Exclusion struct {
	MFLID       string `json:"mflID"`
	Name        string `json:"name"`
	FranchiseID string `json:"franchiseID"`
	Reason      string `json:"reason"`
}

// Report is the outcome of one scoring pass: what was scored under which config,
// what was skipped or excluded, and how many players ran on a zero BasePoints.
type Report struct {
	Season          int         `json:"season"`
	ConfigVersion   int         `json:"configVersion"`
	Scored          int         `json:"scored"`          // players scored THIS pass; 0 on the skip path
	SkippedExisting bool        `json:"skippedExisting"` // scores for (season, config) already persisted; nothing written
	Existing        int         `json:"existing"`        // persisted rows found on the skip path
	ZeroBase        int         `json:"zeroBase"`        // scored with BasePoints 0 — no YTD record for the proxy season
	NegativeBase    int         `json:"negativeBase"`    // scored with a NEGATIVE proxy total floored to 0 — a data smell, distinct from absent
	Excluded        []Exclusion `json:"excluded"`
}

// Run scores every rostered player for season and persists the batch stamped with
// the active config version. asOf anchors age derivation (deterministic — tests
// pin it; the app passes time.Now()). Re-running an already-scored (season,
// config) is the gate-checked skip-if-present: the pass reports the existing
// batch and writes nothing, so B6's append-only drift guard is never even hit.
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
		// Scored stays 0: nothing was scored THIS pass (GLM M1 review — a UI
		// driving a "scored N just now" indicator off Scored must read the truth).
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

// baseOrigin classifies where a player's BasePoints came from, so the report can
// distinguish the two zero populations (GLM M1 review): a rookie with no proxy
// record is expected; a flood of floored NEGATIVE totals is a data smell.
type baseOrigin int

const (
	baseReal     baseOrigin = iota // a real YTD total fed through
	baseAbsent                     // no YTD record → 0
	baseNegative                   // negative YTD total → floored to 0
)

// scorePlayer runs one rostered player through directory resolution, the
// composition boundary, and the engine. It returns exactly one of: a ScoreRecord
// (excl nil), or an Exclusion (the player cannot or must not be scored — the
// reason is user-facing). origin classifies the BasePoints source for the report.
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
		// The proxy season can legitimately total negative (turnover-heavy stretches);
		// the engine rejects a negative base as poisoned. Floor to the placeholder's
		// zero and count it SEPARATELY from absent — visible on a labeled board, but a
		// flood of these is a data smell the report must not blur into the rookie count.
		basePts, origin = 0, baseNegative
	}

	spec := composition.PlayerSpec{
		MFLID:      p.MFLID,
		Name:       facts.Name,
		Position:   facts.Position,
		BasePoints: basePts,
		Age:        age,
		// Money → float millions ONLY here: the engine's L5 cap scaling is a
		// dimensionless salary-as-%-of-cap ratio, not stored money (OQ-014 edge).
		Salary:    p.CapSalary.Millions(),
		IsVeteran: !facts.IsRookie,
		// Scouting Has* fields start false → Data-Parity neutral; applyScouting
		// copies in each signal the assembled Profile actually carries (per field).
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

// applyScouting copies each present scouting signal from the assembled Profile into
// the PlayerSpec, gated PER FIELD (see ScoutingDirectory) — a profile may carry one
// signal and not another. A field left untouched keeps its zero value, which the
// rubric reads as absent and neutralizes via Data-Parity:
//   - RAS (S-Phase 0): gate on Profile.HasRAS; absent → L1 imputes DefaultRASFallback.
//   - SchoolTier (S-Phase 1): gate on the SchoolUnset sentinel.
//   - CollegeShare (S-Phase 2): gate on HasCollegeProductionShare (0 is a real share).
//   - BreakoutAge (S-Phase 4): gate on HasBreakoutAge (a young age is a real, high-value
//     signal — 0 is not "absent"; the engine curves the RAW age per position).
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
	// IDP film composite (FILM Thread C, C-4 steps 1–2): the DT/DE/LB/CB/S film budget,
	// built here because the composite weights are "set upstream" (engine.ScoutingInput
	// doc) — the engine consumes a single [0,1] FilmComposite. Three seats, all LOCKED at
	// C-3: the Madden defense composite (K1, the primary signal), the CB/S PFR coverage
	// anchor (K4 = 0.20 — CB/S only), and NFLProduction (K2, ≤0.05 as a pure tiebreaker,
	// NOT YET wired → its seat is reserved neutral). Any seat whose signal is absent
	// contributes the neutral midpoint × its weight (Data-Parity), so:
	//   - DT/DE/LB (no coverage seat): 0.95·Madden + 0.05·neutral
	//   - CB/S with a coverage anchor:  0.20·coverage + 0.75·Madden + 0.05·neutral
	// A CB/S carrying a coverage anchor but no Madden record keeps 0.20·coverage +
	// 0.75·neutral + 0.05·neutral = the EXACT step-1 value (0.20·coverage + 0.80·neutral),
	// so step 2 is backward-compatible for coverage-only players. A player with neither
	// leaves HasFilm false and the rubric neutralizes film via Data-Parity.
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
			nflProductionFilmWeight*filmNeutralMidpoint // K2 seat reserved neutral (unwired)
		spec.HasFilm = true
	} else if profile.OffenseFilm != nil {
		// Offense film composite (FILM Thread C, C-4 step 3): the QB/RB/WR/TE film budget. The
		// assembly leaf already blended BOTH K3 seats (the Madden offense backbone + the bounded
		// FTN delta-overlay) into Profile.OffenseFilm.Composite, so the only budget arithmetic
		// here is reserving the SAME K2 NFLProduction seat the defense branch reserves (≤0.05,
		// NOT YET wired → neutral): QB/RB/WR/TE = 0.95·Composite + 0.05·neutral. This is an
		// `else if` because offense and IDP are mutually exclusive by position (OffenseFilm is
		// nil at every defensive position and K, Coverage/IDPFilm nil at every offense one) —
		// the else-if makes that invariant STRUCTURAL, so a future classification bug can never
		// silently double-write FilmComposite (GLM C-4 step-3 review, LOW-3).
		spec.FilmComposite = (1-nflProductionFilmWeight)*profile.OffenseFilm.Composite +
			nflProductionFilmWeight*filmNeutralMidpoint // K2 seat reserved neutral (unwired)
		spec.HasFilm = true
	}
}

// coverageFilmWeight is the LOCKED K4 decision (the PFR coverage anchor's share of the
// CB/S film budget). nflProductionFilmWeight is the LOCKED K2 cap (NFLProduction as a
// pure tiebreaker) — its seat is reserved here at the neutral midpoint until NFLProduction
// is wired (a later increment), so wiring it is a localized swap of neutral for the real
// value with no change to the Madden/coverage weights. filmNeutralMidpoint is the film
// composite's neutral value (the engine film S-curve inflects at 0.50); an unset seat
// contributes the midpoint, so the composite stays neutral until a real signal moves it.
// These are the single place the wiring-side film weights live.
const (
	coverageFilmWeight      = 0.20
	nflProductionFilmWeight = 0.05
	filmNeutralMidpoint     = 0.50
)

// yearsBetween is the age derivation: exact days between the instants divided by
// the mean tropical year. Fractional on purpose — the engine's age curves are
// continuous, and rounding here would quantize every player's L3 decay in lockstep.
func yearsBetween(birth, asOf time.Time) float64 {
	return asOf.Sub(birth).Hours() / (24 * 365.2425)
}
