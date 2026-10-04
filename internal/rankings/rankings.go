// Package rankings scores the league (M1). For every rostered player it assembles engine
// inputs through composition, runs the pipeline with the position's rubric, and writes the
// result as one scoring run in history. A run reads params from one frozen snapshot and facts
// from the history features, so it records exactly what it scored with.
//
// BasePoints is a labeled placeholder: the last completed season's fantasy points in the
// league's own scoring, read from the outcome.fantasy_points measure until a real L2 base exists.
//
// Data policy:
//   - no players-DB record, an aggregate, or a FLAG position: excluded, with a reason
//   - no birthdate: excluded, because a faked age would silently corrupt L3 decay
//   - no base points (mostly rookies): scored with BasePoints 0 and counted in the report
package rankings

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/composition"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/scouting"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// BaseMeasure is where the board's base points come from.
const BaseMeasure = "outcome.fantasy_points"

// BoardMeasures are the measures the M1 model reads from history.
func BoardMeasures() []string { return []string{BaseMeasure} }

// Directory resolves a rostered mfl id to its players-DB facts. normalize.Lookup
// satisfies it.
type Directory interface {
	Facts(mflID string) (normalize.PlayerFacts, bool)
}

// History is what a scoring pass needs from the history store.
type History interface {
	Features(ctx context.Context, q history.FeatureQuery) ([]history.Feature, error)
	WriteRun(ctx context.Context, nr history.NewRun) (history.Run, bool, error)
}

// Runner runs scoring passes. It reads league state and writes only scoring runs.
type Runner struct {
	state  state.Reader
	dir    Directory
	scout  ScoutingDirectory
	caps   composition.CapReader
	hist   History
	engine string
}

// New wires a Runner. engine is the build label recorded on every run. An empty
// MapScoutingDirectory is legal (no scouting this pass); a nil one is a wiring error.
func New(st state.Reader, dir Directory, scout ScoutingDirectory, caps composition.CapReader,
	hist History, engine string) (*Runner, error) {
	if st == nil || dir == nil || scout == nil || caps == nil || hist == nil || engine == "" {
		return nil, fmt.Errorf("rankings: missing dependency (state=%t dir=%t scout=%t caps=%t history=%t engine=%t)",
			st != nil, dir != nil, scout != nil, caps != nil, hist != nil, engine != "")
	}
	return &Runner{state: st, dir: dir, scout: scout, caps: caps, hist: hist, engine: engine}, nil
}

// RunSpec is one pass. AsOf bounds the facts read; ages are taken at the start of AsOf's UTC
// day, so passes on the same day with the same facts score the same board. Measures are what
// the model reads: BoardMeasures for the board, fewer for a rebalance proposal.
type RunSpec struct {
	Kind     history.RunKind
	Season   int
	AsOf     time.Time
	Params   params.Set
	Measures []string
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

// Report is what one pass scored and which run holds it.
type Report struct {
	Season int   `json:"season"`
	RunID  int64 `json:"runID"`
	// Unchanged means the pass matched the latest run exactly, so nothing was written.
	Unchanged    bool `json:"unchanged"`
	Scored       int  `json:"scored"`
	ZeroBase     int  `json:"zeroBase"`     // no base points, scored at 0
	NegativeBase int  `json:"negativeBase"` // a negative total floored to 0
	// MissingMeasures are measures the model reads that no active source feeds: the board is
	// running on a reduced set.
	MissingMeasures []string    `json:"missingMeasures"`
	Excluded        []Exclusion `json:"excluded"`
}

// playerInput is everything the engine read for one player; the run's inputs hash covers it.
type playerInput struct {
	MFLID string
	In    engine.PlayerInput
	Sc    engine.ScoutingInput
	Cal   engine.Calibration
}

// Run scores every rostered player and writes the run.
func (r *Runner) Run(ctx context.Context, spec RunSpec) (Report, error) {
	base, err := r.basePoints(ctx, spec)
	if err != nil {
		return Report{}, err
	}
	asm := composition.New(spec.Params, r.caps)
	rubrics, err := asm.Rubrics()
	if err != nil {
		return Report{}, fmt.Errorf("rankings: %w", err)
	}
	ps := pass{asm: asm, rubrics: rubrics, base: base, ageDate: spec.AsOf.UTC().Truncate(24 * time.Hour)}
	rep := Report{Season: spec.Season}
	var scores []history.Score
	var inputs []playerInput
	for _, fid := range r.state.Franchises() {
		roster, ok := r.state.Roster(fid)
		if !ok {
			return Report{}, fmt.Errorf("rankings: franchise %q listed but has no roster (store drift)", fid)
		}
		for _, p := range roster {
			sc, in, excl, origin := r.scorePlayer(ps, fid, p)
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
			scores = append(scores, sc)
			inputs = append(inputs, in)
		}
	}
	if len(scores) == 0 {
		return Report{}, fmt.Errorf("rankings: zero scorable players across %d franchises — refusing to write an empty board", len(r.state.Franchises()))
	}
	hash, err := inputsHash(inputs, rep.Excluded)
	if err != nil {
		return Report{}, err
	}
	run, written, err := r.hist.WriteRun(ctx, history.NewRun{
		Kind: spec.Kind, Season: spec.Season, AsOf: spec.AsOf, Engine: r.engine, InputsHash: hash, Scores: scores,
		Params: history.ParamSet{Params: spec.Params.Board(), Measures: spec.Measures},
	})
	if err != nil {
		return Report{}, fmt.Errorf("rankings: write %d scores (season %d): %w", len(scores), spec.Season, err)
	}
	rep.RunID, rep.Unchanged, rep.Scored, rep.MissingMeasures = run.ID, !written, len(scores), run.MissingMeasures
	return rep, nil
}

// basePoints reads the last completed season's fantasy points when the model reads them.
func (r *Runner) basePoints(ctx context.Context, spec RunSpec) (map[string]float64, error) {
	base := map[string]float64{}
	if !slices.Contains(spec.Measures, BaseMeasure) {
		return base, nil
	}
	feats, err := r.hist.Features(ctx, history.FeatureQuery{
		AsOf: spec.AsOf, Season: spec.Season - 1, Measures: []string{BaseMeasure}})
	if err != nil {
		return nil, fmt.Errorf("rankings: read base points: %w", err)
	}
	for _, f := range feats {
		base[f.PlayerID] = f.Value
	}
	return base, nil
}

// inputsHash is the sha256 of every player's engine input, in roster order, plus who was
// excluded and why.
func inputsHash(inputs []playerInput, excluded []Exclusion) (string, error) {
	enc, err := json.Marshal(struct {
		Inputs   []playerInput
		Excluded []Exclusion
	}{inputs, excluded})
	if err != nil {
		return "", fmt.Errorf("rankings: encode inputs: %w", err)
	}
	sum := sha256.Sum256(enc)
	return hex.EncodeToString(sum[:]), nil
}

// baseOrigin records where BasePoints came from, so the report can tell an expected rookie
// zero from a floored negative total (a data smell).
type baseOrigin int

const (
	baseReal     baseOrigin = iota // a real season total fed through
	baseAbsent                     // no base points → 0
	baseNegative                   // negative season total → floored to 0
)

// pass is what every player in one run is scored with: the run's params, as an assembler and as
// rubrics, its base points and the day ages are taken at.
type pass struct {
	asm     *composition.Assembler
	rubrics map[domain.Position]engine.Layer4
	base    map[string]float64
	ageDate time.Time
}

// scorePlayer returns either a score and the inputs it came from, or a user-facing Exclusion,
// plus where its BasePoints came from.
func (r *Runner) scorePlayer(ps pass, fid string, p state.PlayerState) (history.Score, playerInput, *Exclusion, baseOrigin) {
	exclude := func(name, reason string) (history.Score, playerInput, *Exclusion, baseOrigin) {
		return history.Score{}, playerInput{}, &Exclusion{MFLID: p.MFLID, Name: name, FranchiseID: fid, Reason: reason}, baseReal
	}
	facts, ok := r.dir.Facts(p.MFLID)
	if !ok {
		return exclude("", "not in the players database (aggregate or unknown id)")
	}
	if facts.Position == domain.PosFlag {
		return exclude(facts.Name, "unclassified position (FLAG) — resolve before scoring")
	}
	if !facts.HasBirthdate {
		return exclude(facts.Name, "missing birthdate — age is a required engine input; a faked age would corrupt L3 decay")
	}
	age := yearsBetween(time.Unix(facts.Birthdate, 0).UTC(), ps.ageDate)
	if age <= 0 {
		return exclude(facts.Name, fmt.Sprintf("implausible age %.1f from birthdate — players-DB data error", age))
	}

	basePts, hasBase := ps.base[p.MFLID]
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

	in, sc, cal, err := ps.asm.Assemble(spec)
	if err != nil {
		return exclude(facts.Name, fmt.Sprintf("assemble: %v", err))
	}
	res, err := engine.NewPipeline(ps.rubrics[spec.Position]).Score(in, sc, cal)
	if err != nil {
		return exclude(facts.Name, fmt.Sprintf("score: %v", err))
	}
	return history.Score{MFLID: p.MFLID, Result: res}, playerInput{MFLID: p.MFLID, In: in, Sc: sc, Cal: cal}, nil, origin
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
	// Film is the coverage anchor, CB and S only: 0.20 of the film budget, the rest at the
	// neutral midpoint. No other position has a film source.
	if profile.Coverage != nil {
		spec.FilmComposite = coverageFilmWeight*profile.Coverage.CoverageMetrics +
			(1-coverageFilmWeight)*filmNeutralMidpoint
		spec.HasFilm = true
	}
}

// The coverage anchor's share of the film budget; the film S-curve inflects at 0.50.
const (
	coverageFilmWeight  = 0.20
	filmNeutralMidpoint = 0.50
)

// yearsBetween returns fractional years. Rounding would step every player's L3 decay in
// lockstep.
func yearsBetween(birth, asOf time.Time) float64 {
	return asOf.Sub(birth).Hours() / (24 * 365.2425)
}
