package composition

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine"
	"github.com/secureprospective/TheWarRoom/internal/scouting"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
)

// Assembler turns a PlayerSpec into the engine's inputs by reading calibration from the
// stores (through the ports) and supplying the not-yet-stored per-position/L1 values
// from documented defaults. It holds the readers, not concrete stores, so it is
// testable with fakes.
type Assembler struct {
	params ParamReader
	cap    CapReader
}

// New constructs an Assembler over the param and rulebook readers. Both are required; a
// nil reader is a programmer error. The live harness path guards a.params != nil before
// constructing, so a nil reader here surfaces as a nil-dereference panic at first use
// rather than a silent zero calibration — callers must supply real readers.
func New(p ParamReader, c CapReader) *Assembler {
	return &Assembler{params: p, cap: c}
}

// Calibration reads the position-and-global calibration: it does not depend on a specific
// player, only on the position, so a caller may compute it once per position and reuse it
// across that group's players. The store reads are cheap and internally synchronized, so
// Assemble currently calls it per player; a caller that ranks large sets can cache by
// position. It fails loud if any store value is missing, unparseable, or out of range —
// a half-built or poisoned calibration must never reach the engine.
func (a *Assembler) Calibration(pos domain.Position) (engine.Calibration, error) {
	tiers, err := a.params.GetCapTiers()
	if err != nil {
		return engine.Calibration{}, fmt.Errorf("composition: read cap tiers: %w", err)
	}
	decay, err := a.params.GetGlobal(params.KeyLayer3DecayRate)
	if err != nil {
		return engine.Calibration{}, fmt.Errorf("composition: read decay rate: %w", err)
	}
	leagueCap, err := a.leagueCap()
	if err != nil {
		return engine.Calibration{}, err
	}
	// Defense-in-depth (GLM review B1): the boundary is the engine's fail-loud gate, so it
	// must not pass through a poisoned STORE value any more than a poisoned player value.
	// B4 range-gates these on write, but a bug or future un-gated path must still not reach
	// the pure engine. decay ∈ [0,1]; cap tiers non-negative with the Cold threshold at or
	// below the Hot threshold (Neutral is the band between them).
	if !finite(decay, tiers.ColdCeiling, tiers.HotFloor) {
		return engine.Calibration{}, fmt.Errorf("composition: non-finite calibration from store (decay=%v cold=%v hot=%v)", decay, tiers.ColdCeiling, tiers.HotFloor)
	}
	if decay < 0 || decay > 1 {
		return engine.Calibration{}, fmt.Errorf("composition: decay rate from store out of [0,1]: %v", decay)
	}
	if tiers.ColdCeiling < 0 || tiers.HotFloor < 0 || tiers.ColdCeiling > tiers.HotFloor {
		return engine.Calibration{}, fmt.Errorf("composition: invalid cap tiers (cold=%v hot=%v; need 0 ≤ cold ≤ hot)", tiers.ColdCeiling, tiers.HotFloor)
	}
	cushionRAS, cushionDecline := cushionGuard(pos)
	return engine.Calibration{
		SalaryFloor:          DefaultSalaryFloor,
		RASFallback:          DefaultRASFallback,
		PeakLimit:            peakLimit(pos),
		DecayRate:            decay,
		CushionRASThreshold:  cushionRAS,
		CushionDeclineFactor: cushionDecline,
		LeagueCap:            leagueCap,
		ColdCeiling:          tiers.ColdCeiling,
		HotFloor:             tiers.HotFloor,
		ScarcityRank:         DefaultScarcityRank,
	}, nil
}

// Assemble validates a spec and returns the engine inputs for it: the per-player
// PlayerInput, the L4 ScoutingInput, and the position's Calibration. This is the single
// call the harness IPC layer makes per player before invoking engine.Pipeline.Score.
func (a *Assembler) Assemble(s PlayerSpec) (engine.PlayerInput, engine.ScoutingInput, engine.Calibration, error) {
	if err := s.Validate(); err != nil {
		return engine.PlayerInput{}, engine.ScoutingInput{}, engine.Calibration{}, err
	}
	cal, err := a.Calibration(s.Position)
	if err != nil {
		return engine.PlayerInput{}, engine.ScoutingInput{}, engine.Calibration{}, err
	}
	// When RAS is absent the raw value is meaningless: zero it so a partially-populated
	// spec cannot ride a stray (even non-finite) RAS into the engine (GLM review l1). L1
	// hygiene imputes RASFallback in this case regardless.
	ras := s.RAS
	if !s.HasRAS {
		ras = 0
	}
	in := engine.PlayerInput{
		Position:   s.Position,
		BasePoints: s.BasePoints,
		Age:        s.Age,
		RAS:        ras,
		HasRAS:     s.HasRAS,
		Salary:     s.Salary,
		IsVeteran:  s.IsVeteran,
	}
	return in, a.scouting(s), cal, nil
}

// scouting maps the spec's raw L4 sub-signals into the engine's ScoutingInput. The school
// tier is normalized here (position-independent template); the position-specific curves
// (breakout age, college share, age trajectory) are the rubric's job, so those values pass
// through raw. Validate has already rejected a poisoned value and an unknown tier, so the
// schoolTierNorm lookup here is total. Film rides through only when HasFilm; otherwise the
// composite is zeroed so a stray value cannot reach the rubric (mirrors the RAS handling).
func (a *Assembler) scouting(s PlayerSpec) engine.ScoutingInput {
	film := s.FilmComposite
	if !s.HasFilm {
		film = 0
	}
	tierNorm, _ := schoolTierNorm(s.Position, s.SchoolTier)
	return engine.ScoutingInput{
		FilmComposite: film,
		HasFilm:       s.HasFilm,

		BreakoutAge:     s.BreakoutAge,
		HasBreakoutAge:  s.HasBreakoutAge,
		SchoolTierNorm:  tierNorm,
		HasSchoolTier:   s.SchoolTier != scouting.SchoolUnset,
		CollegeShare:    s.CollegeShare,
		HasCollegeShare: s.HasCollegeShare,
	}
}

// leagueCap parses the rulebook's string cap amount into the float the engine reads,
// failing loud on an empty or non-numeric value (MFL encodes the cap as a string;
// parsing it is the boundary's job, per the schema package's transform/validate split).
func (a *Assembler) leagueCap() (float64, error) {
	raw := strings.TrimSpace(a.cap.GetSalaryCap())
	if raw == "" {
		return 0, fmt.Errorf("composition: rulebook returned an empty salary cap")
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("composition: salary cap %q is not numeric: %w", raw, err)
	}
	if v <= 0 || !finite(v) {
		return 0, fmt.Errorf("composition: salary cap must be positive and finite, got %v", v)
	}
	return v, nil
}
