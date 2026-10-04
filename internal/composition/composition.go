package composition

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine"
	"github.com/secureprospective/TheWarRoom/internal/engine/l4"
	"github.com/secureprospective/TheWarRoom/internal/numeric"
	"github.com/secureprospective/TheWarRoom/internal/scouting"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
)

// Assembler turns a PlayerSpec into engine inputs, reading calibration from the stores
// through the ports and supplying the values not yet stored from defaults.go.
type Assembler struct {
	params ParamReader
	cap    CapReader
}

// New builds an Assembler. Both readers are required; a nil one panics on first use rather
// than scoring with zero calibration.
func New(p ParamReader, c CapReader) *Assembler {
	return &Assembler{params: p, cap: c}
}

// Calibration reads the calibration for a position; it does not depend on the player, so a
// caller ranking many players may cache it per position. It fails if any store value is
// missing, unparseable or out of range.
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
	// The params store range-checks on write; this re-checks so no other path can hand the
	// engine a poisoned value. Cold must sit at or below Hot.
	if !numeric.Finite(decay, tiers.ColdCeiling, tiers.HotFloor) {
		return engine.Calibration{}, fmt.Errorf("composition: non-finite calibration from store (decay=%v cold=%v hot=%v)", decay, tiers.ColdCeiling, tiers.HotFloor)
	}
	if decay < 0 || decay > 1 {
		return engine.Calibration{}, fmt.Errorf("composition: decay rate from store out of [0,1]: %v", decay)
	}
	if tiers.ColdCeiling < 0 || tiers.HotFloor < 0 || tiers.ColdCeiling > tiers.HotFloor {
		return engine.Calibration{}, fmt.Errorf("composition: invalid cap tiers (cold=%v hot=%v; need 0 ≤ cold ≤ hot)", tiers.ColdCeiling, tiers.HotFloor)
	}
	cushion, err := a.cushionGuard(pos)
	if err != nil {
		return engine.Calibration{}, err
	}
	return engine.Calibration{
		SalaryFloor:  DefaultSalaryFloor,
		RASFallback:  DefaultRASFallback,
		PeakLimit:    peakLimit(pos),
		DecayRate:    decay,
		Cushion:      cushion,
		LeagueCap:    leagueCap,
		ColdCeiling:  tiers.ColdCeiling,
		HotFloor:     tiers.HotFloor,
		ScarcityRank: DefaultScarcityRank,
	}, nil
}

// Assemble validates a spec and returns the engine inputs for it.
func (a *Assembler) Assemble(s PlayerSpec) (engine.PlayerInput, engine.ScoutingInput, engine.Calibration, error) {
	if err := s.Validate(); err != nil {
		return engine.PlayerInput{}, engine.ScoutingInput{}, engine.Calibration{}, err
	}
	cal, err := a.Calibration(s.Position)
	if err != nil {
		return engine.PlayerInput{}, engine.ScoutingInput{}, engine.Calibration{}, err
	}
	// An absent RAS is zeroed so a stray value cannot reach the engine; L1 imputes the fallback.
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

// scouting maps the spec's raw L4 sub-signals into ScoutingInput. Only school tier is
// normalized here; the position curves belong to the rubric. Absent film is zeroed, as RAS is.
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

// Rubrics returns each position's Layer-4 rubric: the shipped settings with every adjustable
// number read from params. Breakout weights an edit has left off 1 are rescaled to sum to 1,
// so a neutral profile stays neutral.
func (a *Assembler) Rubrics() (map[domain.Position]engine.Layer4, error) {
	table := l4.Defaults()
	for pos, s := range table {
		for _, k := range l4.Knobs() {
			if !k.AppliesTo(s) {
				continue
			}
			v, err := a.params.GetPosition(k.Key, string(pos))
			if err != nil {
				return nil, fmt.Errorf("composition: read rubric setting: %w", err)
			}
			if !numeric.Finite(v) {
				return nil, fmt.Errorf("composition: rubric setting %s at %s is not finite", k.Key, pos)
			}
			*k.Field(&s) = v
		}
		w := &s.Weights
		if sum := w.BreakoutAge + w.SchoolTier + w.CollegeShare + w.AgeTrajectory; sum > 0 && math.Abs(sum-1) > 1e-9 {
			w.BreakoutAge, w.SchoolTier, w.CollegeShare, w.AgeTrajectory =
				w.BreakoutAge/sum, w.SchoolTier/sum, w.CollegeShare/sum, w.AgeTrajectory/sum
		}
		table[pos] = s
	}
	return l4.Rubrics(table), nil
}

// leagueCap parses the rulebook's string cap amount, failing on empty or non-numeric.
func (a *Assembler) leagueCap() (float64, error) {
	raw := strings.TrimSpace(a.cap.GetSalaryCap())
	if raw == "" {
		return 0, fmt.Errorf("composition: rulebook returned an empty salary cap")
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("composition: salary cap %q is not numeric: %w", raw, err)
	}
	if v <= 0 || !numeric.Finite(v) {
		return 0, fmt.Errorf("composition: salary cap must be positive and finite, got %v", v)
	}
	return v, nil
}
