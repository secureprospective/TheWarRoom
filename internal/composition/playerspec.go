package composition

import (
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/numeric"
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

	// K film sub-signals (DECISION-011, B5b-K): the kicker rubric is the one position whose
	// film is built from these two normalized [0,1] components at a PINNED 0.60/0.40 split in
	// the engine, rather than a pre-blended FilmComposite. Non-K positions leave them zero/absent.
	MaddenFilm       float64 // normalized [0,1] Madden kick rating composite (valid only when HasMaddenFilm)
	HasMaddenFilm    bool    // false → the K rubric treats the Madden component as neutral
	NFLProduction    float64 // normalized [0,1] NFL on-field kicking production (valid only when HasNFLProduction)
	HasNFLProduction bool    // false → the K rubric treats the NFL-production component as neutral

	// Breakout sub-signals. The Has* flags distinguish ABSENT from a real zero (a zero
	// breakout age or college share would otherwise be read as an extreme, not neutral).
	// School-tier presence is inferred from the enum (SchoolUnset == absent), so it needs
	// no separate flag.
	BreakoutAge     float64             // breakout age in years; the rubric maps it to [0,1]
	HasBreakoutAge  bool                // false → breakout age is treated as neutral
	SchoolTier      scouting.SchoolTier // college-competition tier; mapped to a [0,1] norm at assemble
	CollegeShare    float64             // college production/usage share in [0,1]
	HasCollegeShare bool                // false → college share is treated as neutral (0 is a real share)

	// EDGE classification routing (OQ-004 / SL-OQ-030, Module-3 test 3J). The TRUE position
	// for a pass-rush/off-ball defender is its CONSENSUS ROLE, not its MFL tag: a pass-rush-
	// primary defender is scored as a DE regardless of tag (DE/EDGE/3-4 OLB), an off-ball LB
	// as an LB. PassRushSnapShare in [0,1] is that signal — manual entry in the harness,
	// NGS-sourced in production. Absent → the MFL position passes through unchanged.
	PassRushSnapShare    float64
	HasPassRushSnapShare bool
}

// edgePassRushThreshold is the majority share (≥50% of snaps rushing the passer) at which a
// defender is "pass-rush primary" and routes to the DE rubric (Module-3 test 3J boundary:
// 75%→DE, 25%→LB). The comparison is >= (spec: "≥ 0.50 routes to DE"), so an exact coin-flip
// (0.50) is pass-rush primary → DE. The exact cutoff is a documented boundary default pending
// SL-OQ-030 calibration.
const edgePassRushThreshold = 0.50

// ResolveRubricPosition applies the OQ-004 / SL-OQ-030 EDGE classification rule: among the
// pass-rush/off-ball ambiguous pair (DE — MFL also tags edge rushers DE per OQ-004 — and LB),
// a pass-rush-PRIMARY defender (PassRushSnapShare ≥ threshold) is scored as a DE and an off-ball
// defender as an LB, regardless of MFL tag. Every other position, and any defender without a
// pass-rush share, passes through unchanged (the MFL tag stands). This is the position-RESOLUTION
// step: it runs BEFORE Assemble so the resolved role drives BOTH the rubric AND the calibration
// (peak limit, cushion) — the role IS the position. It lives at the boundary, not in any pure
// rubric (the engine never sees the MFL tag or the snap share).
func ResolveRubricPosition(s PlayerSpec) domain.Position {
	// Only the pass-rush/off-ball ambiguous pair (DE/LB) is re-routed by role; every other
	// position, and any defender without a pass-rush share, keeps its MFL tag.
	if !s.HasPassRushSnapShare || (s.Position != domain.PosDE && s.Position != domain.PosLB) {
		return s.Position
	}
	if s.PassRushSnapShare >= edgePassRushThreshold {
		return domain.PosDE
	}
	return domain.PosLB
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
	if !numeric.Finite(s.BasePoints, s.Age, s.Salary) {
		return fmt.Errorf("composition: player %q has a non-finite numeric field (base=%v age=%v salary=%v)", s.MFLID, s.BasePoints, s.Age, s.Salary)
	}
	if s.HasRAS && !numeric.Finite(s.RAS) {
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
	if !numeric.Finite(s.BreakoutAge, s.CollegeShare) {
		return fmt.Errorf("composition: player %q has a non-finite scouting field (breakoutAge=%v collegeShare=%v)", s.MFLID, s.BreakoutAge, s.CollegeShare)
	}
	if s.BreakoutAge < 0 {
		return fmt.Errorf("composition: player %q has negative breakout age %v", s.MFLID, s.BreakoutAge)
	}
	if s.CollegeShare < 0 || s.CollegeShare > 1 {
		return fmt.Errorf("composition: player %q college share %v out of [0,1]", s.MFLID, s.CollegeShare)
	}
	// Each present-gated [0,1] sub-signal: rejected only when actively corrupt (non-finite or
	// out of range); absent (Has* false) is allowed and neutralized by the rubric's Data-Parity.
	for _, c := range []struct {
		name    string
		present bool
		v       float64
	}{
		{"film composite", s.HasFilm, s.FilmComposite},
		{"Madden film", s.HasMaddenFilm, s.MaddenFilm},
		{"NFL production", s.HasNFLProduction, s.NFLProduction},
		{"pass-rush snap share", s.HasPassRushSnapShare, s.PassRushSnapShare},
	} {
		if err := s.checkUnitRange(c.name, c.present, c.v); err != nil {
			return err
		}
	}
	if _, ok := schoolTierNorm(s.Position, s.SchoolTier); !ok {
		return fmt.Errorf("composition: player %q has unknown school tier %q", s.MFLID, s.SchoolTier)
	}
	return nil
}

// checkUnitRange fail-louds when a PRESENT sub-signal is non-finite or outside [0,1]. An
// absent signal is allowed (the rubric's Data-Parity Rule neutralizes it). Shared so each
// present-gated scouting field validates identically (M17) and validateScouting stays simple.
func (s PlayerSpec) checkUnitRange(name string, present bool, v float64) error {
	if present && (!numeric.Finite(v) || v < 0 || v > 1) {
		return fmt.Errorf("composition: player %q has a present %s %v out of [0,1]", s.MFLID, name, v)
	}
	return nil
}
