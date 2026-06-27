// Package offense holds the B5b offense Layer-4 rubrics (QB first; RB/WR/TE reuse this
// shape). Each rubric implements engine.Layer4 as a PURE function: it reads the player's
// scouting sub-signals from the Layer4Input, normalizes the position-specific ones with
// its own curves (via the shared curve package), and returns the three component
// multipliers plus their product.
//
// PURITY (depguard engine-is-pure, **/internal/engine/**): this package imports only the
// engine (for the Layer4 contract types) and the curve package — no store, db, ingestion,
// or I/O. Every input arrives as a parameter on Layer4Input.
package offense

import (
	"github.com/secureprospective/TheWarRoom/internal/engine"
	"github.com/secureprospective/TheWarRoom/internal/engine/l4/curve"
)

// QB Layer-4 mechanics (docs/scoring-engine/QB_Rubric.md, locked v1.0). All values are
// rubric constants — the structural mechanics are NEVER admin-exposed (Hard Constraint:
// Layer 4 structural mechanics never appear in the Admin UI). A future calibration pass
// may move tunable weights into B4; until then they ship WITH this layer.
const (
	// Film component (QB_Rubric §2): S-curve over the upstream film composite.
	qbFilmInflection = 0.50
	qbFilmSteepness  = 12.0
	qbFilmCap        = 0.05 // ±5%

	// Breakout component (QB_Rubric §4): S-curve over the weighted sub-signal composite.
	qbBreakoutInflection = 0.50
	qbBreakoutSteepness  = 11.0
	qbBreakoutCap        = 0.05 // ±5%

	// Breakout sub-signal weights (QB_Rubric §4) — sum to 1.00.
	qbWeightBreakoutAge   = 0.30
	qbWeightSchoolTier    = 0.25
	qbWeightCollegeShare  = 0.30
	qbWeightAgeTrajectory = 0.15
)

// QB is the quarterback Layer-4 rubric. Per SL-020 (Low-tier RAS) it forces the RAS
// component to exactly 1.000 — RAS contributes nothing to Layer 4 at QB and is used only
// at Layer 6 (tiebreaker). SL-019 does NOT apply at QB, so the breakout sub-signals use
// base curve values with no RAS modulation. Film and breakout are the active components.
//
// It is constructed with its normalization curves (held on the struct so they are not
// package-level globals — gochecknoglobals — and are not rebuilt per Apply call).
type QB struct {
	breakoutAge   []curve.Breakpoint
	collegeShare  []curve.Breakpoint
	ageTrajectory []curve.Breakpoint
}

// NewQB builds the QB rubric with its position-specific normalization curves. The curves
// are QB_Rubric §4's normalization tables; values between anchors interpolate linearly and
// the ends are flat (≤ first / ≥ last). School tier is position-INDEPENDENT and arrives
// pre-normalized on the input, so it has no curve here.
func NewQB() *QB {
	return &QB{
		// Breakout Age (years → [0,1]): ≤20 → 1.00, 21 → 0.80, 22 → 0.50, ≥23 → 0.10.
		breakoutAge: []curve.Breakpoint{{X: 20, Y: 1.00}, {X: 21, Y: 0.80}, {X: 22, Y: 0.50}, {X: 23, Y: 0.10}},
		// College Offensive Share Index (share → [0,1]): ≤0.35 → 0.15, 0.50 → 0.55, ≥0.65 → 1.00.
		collegeShare: []curve.Breakpoint{{X: 0.35, Y: 0.15}, {X: 0.50, Y: 0.55}, {X: 0.65, Y: 1.00}},
		// Age Trajectory (player age → [0,1]): QB-specific slow decay around peak 32.
		ageTrajectory: []curve.Breakpoint{
			{X: 28, Y: 1.00}, {X: 29, Y: 0.90}, {X: 30, Y: 0.80}, {X: 31, Y: 0.65}, {X: 32, Y: 0.50},
			{X: 33, Y: 0.35}, {X: 34, Y: 0.25}, {X: 35, Y: 0.15}, {X: 36, Y: 0.10}, {X: 37, Y: 0.00},
		},
	}
}

// Apply computes the QB Layer-4 output: Combined = film × RAS × breakout
// (Backend_Architecture:256). RAS is forced to 1.000 (SL-020). Film is the S-curve over
// the upstream composite, or a neutral 1.000 when no film source is populated (HasFilm
// false — the Data-Parity Rule, QB_Rubric §1: a missing component returns neutral, never a
// penalty). Breakout weights the four sub-signals (breakout age, school tier, college share,
// age trajectory), then applies its S-curve. FilmRaw carries the pre-effective normalized
// film input for the debug surface (never UI).
func (q *QB) Apply(in engine.Layer4Input) engine.Layer4Output {
	sc := in.Scouting

	filmRaw := curve.NeutralNorm // Data-Parity neutral default (no source populated)
	film := 1.0
	if sc.HasFilm {
		filmRaw = sc.FilmComposite
		film = curve.Scurve(sc.FilmComposite, qbFilmInflection, qbFilmSteepness, qbFilmCap)
	}

	const rasEffective = 1.0 // SL-020: RAS forced to exactly 1.000 at QB

	// Data-Parity per sub-signal: an ABSENT signal contributes the neutral normalized value
	// (the S-curve inflection, 0.50), so it neither lifts nor penalizes the composite — never
	// the curve ceiling/floor a raw zero would hit. Age trajectory always has a value (the
	// player's age is always known).
	composite := qbWeightBreakoutAge*curve.SubSignal(sc.HasBreakoutAge, q.breakoutAge, sc.BreakoutAge) +
		qbWeightSchoolTier*curve.Present(sc.HasSchoolTier, sc.SchoolTierNorm) +
		qbWeightCollegeShare*curve.SubSignal(sc.HasCollegeShare, q.collegeShare, sc.CollegeShare) +
		qbWeightAgeTrajectory*curve.Interp(q.ageTrajectory, in.Player.Age)
	breakout := curve.Scurve(composite, qbBreakoutInflection, qbBreakoutSteepness, qbBreakoutCap)

	return engine.Layer4Output{
		FilmEffective:     film,
		FilmRaw:           filmRaw,
		RASEffective:      rasEffective,
		BreakoutEffective: breakout,
		Combined:          film * rasEffective * breakout,
	}
}
