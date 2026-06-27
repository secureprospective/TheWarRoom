package composition

import "github.com/secureprospective/TheWarRoom/internal/domain"

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
