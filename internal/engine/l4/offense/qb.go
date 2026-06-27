package offense

import "github.com/secureprospective/TheWarRoom/internal/engine"

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
	breakoutAge   []breakpoint
	collegeShare  []breakpoint
	ageTrajectory []breakpoint
}

// NewQB builds the QB rubric with its position-specific normalization curves. The curves
// are QB_Rubric §4's normalization tables; values between anchors interpolate linearly and
// the ends are flat (≤ first / ≥ last). School tier is position-INDEPENDENT and arrives
// pre-normalized on the input, so it has no curve here.
func NewQB() *QB {
	return &QB{
		// Breakout Age (years → [0,1]): ≤20 → 1.00, 21 → 0.80, 22 → 0.50, ≥23 → 0.10.
		breakoutAge: []breakpoint{{20, 1.00}, {21, 0.80}, {22, 0.50}, {23, 0.10}},
		// College Offensive Share Index (share → [0,1]): ≤0.35 → 0.15, 0.50 → 0.55, ≥0.65 → 1.00.
		collegeShare: []breakpoint{{0.35, 0.15}, {0.50, 0.55}, {0.65, 1.00}},
		// Age Trajectory (player age → [0,1]): QB-specific slow decay around peak 32.
		ageTrajectory: []breakpoint{
			{28, 1.00}, {29, 0.90}, {30, 0.80}, {31, 0.65}, {32, 0.50},
			{33, 0.35}, {34, 0.25}, {35, 0.15}, {36, 0.10}, {37, 0.00},
		},
	}
}

// Apply computes the QB Layer-4 output: Combined = film × RAS × breakout
// (Backend_Architecture:256). RAS is forced to 1.000 (SL-020). Film is the S-curve over
// the upstream composite, or a neutral 1.000 when no film source is populated (HasFilm
// false — the Data-Parity Rule, QB_Rubric §1: a missing component returns neutral, never a
// penalty). Breakout weights the four sub-signals (breakout age, school tier, college share,
// age trajectory), then applies its S-curve.
func (q *QB) Apply(in engine.Layer4Input) engine.Layer4Output {
	sc := in.Scouting

	film := 1.0 // Data-Parity neutral default
	if sc.HasFilm {
		film = scurve(sc.FilmComposite, qbFilmInflection, qbFilmSteepness, qbFilmCap)
	}

	const rasEffective = 1.0 // SL-020: RAS forced to exactly 1.000 at QB

	// Data-Parity per sub-signal: an ABSENT signal contributes the neutral normalized value
	// (the S-curve inflection, 0.50), so it neither lifts nor penalizes the composite — never
	// the curve ceiling/floor a raw zero would hit. Age trajectory always has a value (the
	// player's age is always known).
	composite := qbWeightBreakoutAge*subSignal(sc.HasBreakoutAge, q.breakoutAge, sc.BreakoutAge) +
		qbWeightSchoolTier*present(sc.HasSchoolTier, sc.SchoolTierNorm) +
		qbWeightCollegeShare*subSignal(sc.HasCollegeShare, q.collegeShare, sc.CollegeShare) +
		qbWeightAgeTrajectory*interp(q.ageTrajectory, in.Player.Age)
	breakout := scurve(composite, qbBreakoutInflection, qbBreakoutSteepness, qbBreakoutCap)

	return engine.Layer4Output{
		FilmEffective:     film,
		RASEffective:      rasEffective,
		BreakoutEffective: breakout,
		Combined:          film * rasEffective * breakout,
	}
}
