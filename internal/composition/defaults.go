package composition

import (
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/scouting"
)

// L1 hygiene defaults composition supplies until B4/admin tables own them. These are
// documented constants, not invented: RASFallback is the engine spec's stated fallback
// (5.00 — see engine.Calibration.RASFallback). SalaryFloor has no league-minimum in the
// spec, so it defaults to 0 (no artificial floor); an admin sets it per league later.
const (
	DefaultRASFallback = 5.00
	DefaultSalaryFloor = 0.0
)

// DefaultScarcityRank is the L6 tiebreaker scarcity rank composition supplies until B5b
// ships per-position scarcity. It affects sort order only among EQUAL adjusted scores,
// so a uniform 0 is a safe, inert default (every position ties on scarcity, falling
// through to the next tiebreaker) until the real ranks land.
const DefaultScarcityRank = 0

// schoolTierNorm maps a college-competition tier to its normalized [0,1] breakout weight for
// a position. The QB/WR/TE/DT template is 1.00/0.70/0.40/0.10; RB applies a SOFTER non-P4
// penalty — 1.00/0.75/0.45/0.15 (RB_Rubric §4: "small-school RBs convert to NFL production
// more reliably than small-school WRs — workhorse usage at any FBS level is a meaningful
// signal"). So the mapping is position-DEPENDENT and lives at the boundary (not in any one
// rubric, since the engine receives a plain normalized value), following the same per-position
// switch idiom as peakLimit/cushionGuard. P4 is 1.00 at every position; SchoolUnset maps to 0
// (no positive tier signal). ok is false only for an unrecognized enum value, which the spec
// validator rejects. A function (not a map) keeps gochecknoglobals happy (M17).
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

// cushionGuard returns the SL-021 Late-Career Cushion Guard strength for a position: the
// raw-RAS threshold and the decline-velocity multiplier the engine's ApplyCushionGuard
// consumes (DT_Rubric §1/§3). It is per-position and ships WITH the rubric — only DT uses
// it in v1.0; every other position returns a zero threshold, which DISABLES the guard (a
// safe no-op). A function (not a package map) keeps gochecknoglobals happy (M17).
func cushionGuard(p domain.Position) (threshold, declineFactor float64) {
	if p == domain.PosDT {
		return 8.00, 0.90 // RAS ≥ 8.00 → late-career decline slowed 10%
	}
	return 0, 0 // disabled at every other position
}

// peakLimit returns the Layer-3 age peak limit for a position — the age past which
// decay applies (Engine_Specification:118, "Current Peak Limit Defaults"). These are
// admin-tunable per SL-017; B4 does not seed them yet, so composition supplies the spec
// defaults. A function (not a package map) keeps gochecknoglobals happy and makes the
// source-of-truth a single switch (M17). Unknown positions get the most conservative
// (latest) peak so an unclassified player is never penalized by an aggressive default.
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
