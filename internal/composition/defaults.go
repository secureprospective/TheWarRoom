package composition

import (
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/scouting"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
)

// L1 defaults until the params store holds them. RASFallback is the engine spec's 5.00; the
// spec sets no salary floor, so it is 0.
const (
	DefaultRASFallback = 5.00
	DefaultSalaryFloor = 0.0
)

// DefaultScarcityRank is the L6 scarcity rank until per-position ranks exist. It only orders
// exact score ties, so a uniform 0 is inert.
const DefaultScarcityRank = 0

// schoolTierNorm maps a college tier to its [0,1] breakout weight for a position: P4/G5/FCS/
// lower = 1.00/0.70/0.40/0.10, except RB at 1.00/0.75/0.45/0.15, because small-school RBs
// convert to NFL production more reliably (RB_Rubric §4). SchoolUnset maps to 0. ok is false
// only for an unknown enum value.
func schoolTierNorm(p domain.Position, t scouting.SchoolTier) (float64, bool) {
	rb := p == domain.PosRB
	switch t {
	case scouting.SchoolPowerFour:
		return 1.00, true
	case scouting.SchoolGroupOfFive:
		if rb {
			return 0.75, true
		}
		return 0.70, true
	case scouting.SchoolFCS:
		if rb {
			return 0.45, true
		}
		return 0.40, true
	case scouting.SchoolNonFCS:
		if rb {
			return 0.15, true
		}
		return 0.10, true
	case scouting.SchoolUnset:
		return 0.0, true
	default:
		return 0, false
	}
}

// cushionGuard returns the DT late-career cushion (SL-021) from the admin params: the RAS
// threshold and the decline multiplier (1 − reduction). Other positions get a zero threshold,
// which disables it.
func (a *Assembler) cushionGuard(p domain.Position) (threshold, declineFactor float64, err error) {
	if p != domain.PosDT {
		return 0, 0, nil
	}
	threshold, err = a.params.GetGlobal(params.KeyCushionGuardRAS)
	if err != nil {
		return 0, 0, fmt.Errorf("composition: read cushion threshold: %w", err)
	}
	reduction, err := a.params.GetGlobal(params.KeyCushionGuardReduct)
	if err != nil {
		return 0, 0, fmt.Errorf("composition: read cushion reduction: %w", err)
	}
	return threshold, 1 - reduction, nil
}

// peakLimit is the Layer-3 age past which decay applies (Engine_Specification, "Current Peak
// Limit Defaults"). These are meant to be admin-tunable (SL-017) but are not in the params
// store yet. An unknown position gets the latest peak, so it is never penalized early.
func peakLimit(p domain.Position) float64 {
	switch p {
	case domain.PosQB:
		return 32
	case domain.PosRB:
		return 25
	case domain.PosWR:
		return 29
	case domain.PosTE:
		return 29
	case domain.PosDE:
		return 30
	case domain.PosDT:
		return 30
	case domain.PosLB:
		return 29
	case domain.PosCB:
		return 28
	case domain.PosS:
		return 28
	case domain.PosK:
		return 30
	case domain.PosFlag:
		return 32 // unclassified: most conservative (latest) peak
	default:
		return 32
	}
}
