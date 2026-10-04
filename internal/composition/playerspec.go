package composition

import (
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/numeric"
	"github.com/secureprospective/TheWarRoom/internal/scouting"
)

// PlayerSpec is one player's facts. Calibration never comes from the spec, so a fixture cannot
// override it. MFLID stays a string end to end: "0001" is not 1.
type PlayerSpec struct {
	MFLID      string
	Name       string
	Position   domain.Position
	BasePoints float64 // L2 output
	Age        float64
	RAS        float64
	HasRAS     bool // false → L1 imputes DefaultRASFallback
	Salary     float64
	IsVeteran  bool

	// Layer-4 scouting sub-signals, raw and position-blind. Each Has* flag separates absent
	// (neutral in the rubric) from a real zero. They are validated even where a rubric ignores them.
	FilmComposite float64 // [0,1]
	HasFilm       bool

	// K film: composition blends these two into a kicker's FilmComposite. Other positions leave
	// them unset.
	MaddenFilm       float64 // [0,1] Madden kick-rating composite
	HasMaddenFilm    bool
	NFLProduction    float64 // [0,1] NFL kicking production
	HasNFLProduction bool

	// Breakout sub-signals. SchoolUnset means absent, so school tier needs no flag.
	BreakoutAge     float64 // years
	HasBreakoutAge  bool
	SchoolTier      scouting.SchoolTier
	CollegeShare    float64 // [0,1]
	HasCollegeShare bool

	// PassRushSnapShare routes an EDGE defender by role, not MFL tag (OQ-004): pass-rush primary
	// scores as DE, off-ball as LB. Absent leaves the MFL position unchanged.
	PassRushSnapShare    float64
	HasPassRushSnapShare bool
}

// edgePassRushThreshold: a share at or above it is pass-rush primary and routes to DE, so an
// exact 0.50 goes to DE. Pending SL-OQ-030 calibration.
const edgePassRushThreshold = 0.50

// ResolveRubricPosition re-routes a DE or LB by PassRushSnapShare; every other position, and
// any defender without a share, keeps its MFL tag. It runs before Assemble so the resolved
// role drives both the rubric and the calibration.
func ResolveRubricPosition(s PlayerSpec) domain.Position {
	if !s.HasPassRushSnapShare || (s.Position != domain.PosDE && s.Position != domain.PosLB) {
		return s.Position
	}
	if s.PassRushSnapShare >= edgePassRushThreshold {
		return domain.PosDE
	}
	return domain.PosLB
}

// validPosition reports whether the engine can score p. PosFlag is excluded: an unclassified
// player needs an admin to resolve it first.
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

// Validate rejects a spec that cannot be scored, so the engine never receives a poisoned
// value.
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

// validateScouting rejects a corrupt sub-signal (non-finite or out of range). An absent one
// is fine: the rubric's Data-Parity Rule neutralizes it.
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

// checkUnitRange rejects a present sub-signal that is non-finite or outside [0,1].
func (s PlayerSpec) checkUnitRange(name string, present bool, v float64) error {
	if present && (!numeric.Finite(v) || v < 0 || v > 1) {
		return fmt.Errorf("composition: player %q has a present %s %v out of [0,1]", s.MFLID, name, v)
	}
	return nil
}
