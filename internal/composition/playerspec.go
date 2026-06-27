package composition

import (
	"fmt"
	"math"

	"github.com/secureprospective/TheWarRoom/internal/domain"
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
