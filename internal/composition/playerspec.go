package composition

import (
	"fmt"
	"math"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/scouting"
)

// PlayerSpec is one player's harness-supplied input: the fields a fixture or the manual
// entry form provides. It is the SOURCE the assembler maps into engine.PlayerInput. It
// deliberately holds only per-player facts; per-position calibration (peak limit,
// scarcity) and globals (decay, cap tiers, league cap) come from the stores/defaults at
// assemble time, never from the spec, so a fixture cannot silently override calibration.
//
// MFLID is a STRING and stays one end to end (Module 3 test 3L): leading zeros are
// significant ("0001" is not 1). The assembler never converts it to a number.
type PlayerSpec struct {
	MFLID      string
	Name       string
	Position   domain.Position
	BasePoints float64 // L2 output, supplied (L2 is a separate block)
	Age        float64
	RAS        float64
	HasRAS     bool // false → L1 imputes DefaultRASFallback
	Salary     float64
	IsVeteran  bool

	// --- Layer-4 scouting sub-signals (raw, position-blind) ---
	// These feed the per-position L4 rubric (B5b). Positions without a registered rubric
	// run identity L4 and ignore them; the boundary still validates them so a poisoned
	// value never rides into a rubric that DOES use them.
	FilmComposite float64 // upstream film composite in [0,1] (valid only when HasFilm)
	HasFilm       bool    // false → the rubric forces a neutral film component (Data-Parity Rule)

	// Breakout sub-signals. The Has* flags distinguish ABSENT from a real zero (a zero
	// breakout age or college share would otherwise be read as an extreme, not neutral).
	// School-tier presence is inferred from the enum (SchoolUnset == absent), so it needs
	// no separate flag.
	BreakoutAge     float64             // breakout age in years; the rubric maps it to [0,1]
	HasBreakoutAge  bool                // false → breakout age is treated as neutral
	SchoolTier      scouting.SchoolTier // college-competition tier; mapped to a [0,1] norm at assemble
	CollegeShare    float64             // college production/usage share in [0,1]
	HasCollegeShare bool                // false → college share is treated as neutral (0 is a real share)
}

// knownPositions is the engine's scorable set (domain.PosFlag is excluded — an
// unclassified player must be resolved by an admin before it can be scored).
func validPosition(p domain.Position) bool {
	switch p {
	case domain.PosQB, domain.PosRB, domain.PosWR, domain.PosTE, domain.PosK,
		domain.PosDE, domain.PosDT, domain.PosLB, domain.PosCB, domain.PosS:
		return true
	case domain.PosFlag:
		return false
	default:
		return false
	}
}

// Validate fails loud on a spec that cannot be scored. The boundary rejects bad input
// here so the pure engine never receives a poisoned value (the same fail-loud contract
// the engine enforces on its own numeric inputs).
func (s PlayerSpec) Validate() error {
	if s.MFLID == "" {
		return fmt.Errorf("composition: player spec missing MFL id")
	}
	if s.Name == "" {
		return fmt.Errorf("composition: player %q missing name", s.MFLID)
	}
	if !validPosition(s.Position) {
		return fmt.Errorf("composition: player %q has unscorable position %q (resolve FLAG before scoring)", s.MFLID, s.Position)
	}
	if !finite(s.BasePoints, s.Age, s.Salary) {
		return fmt.Errorf("composition: player %q has a non-finite numeric field (base=%v age=%v salary=%v)", s.MFLID, s.BasePoints, s.Age, s.Salary)
	}
	if s.HasRAS && !finite(s.RAS) {
		return fmt.Errorf("composition: player %q has HasRAS but a non-finite RAS %v", s.MFLID, s.RAS)
	}
	if s.Age <= 0 {
		return fmt.Errorf("composition: player %q has non-positive age %v", s.MFLID, s.Age)
	}
	if s.Salary < 0 {
		return fmt.Errorf("composition: player %q has negative salary %v", s.MFLID, s.Salary)
	}
	return s.validateScouting()
}

// validateScouting fail-louds on a poisoned L4 sub-signal (non-finite, or out of its
// documented range) so the pure rubric never normalizes garbage. It deliberately does NOT
// require the signals to be present: an absent signal (zero / SchoolUnset) is allowed and
// handled by the rubric's Data-Parity Rule — only an actively-corrupt value is rejected.
func (s PlayerSpec) validateScouting() error {
	if !finite(s.BreakoutAge, s.CollegeShare) {
		return fmt.Errorf("composition: player %q has a non-finite scouting field (breakoutAge=%v collegeShare=%v)", s.MFLID, s.BreakoutAge, s.CollegeShare)
	}
	if s.BreakoutAge < 0 {
		return fmt.Errorf("composition: player %q has negative breakout age %v", s.MFLID, s.BreakoutAge)
	}
	if s.CollegeShare < 0 || s.CollegeShare > 1 {
		return fmt.Errorf("composition: player %q college share %v out of [0,1]", s.MFLID, s.CollegeShare)
	}
	if s.HasFilm && (!finite(s.FilmComposite) || s.FilmComposite < 0 || s.FilmComposite > 1) {
		return fmt.Errorf("composition: player %q has HasFilm but film composite %v out of [0,1]", s.MFLID, s.FilmComposite)
	}
	if _, ok := schoolTierNorm(s.Position, s.SchoolTier); !ok {
		return fmt.Errorf("composition: player %q has unknown school tier %q", s.MFLID, s.SchoolTier)
	}
	return nil
}

// finite reports whether every value is a real number (no NaN, no Inf). Shared by the
// spec validator and the assembler's parsed-cap guard (M17).
func finite(vs ...float64) bool {
	for _, v := range vs {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return false
		}
	}
	return true
}
