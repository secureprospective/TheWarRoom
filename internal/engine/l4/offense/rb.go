package offense

import (
	"github.com/secureprospective/TheWarRoom/internal/engine"
	"github.com/secureprospective/TheWarRoom/internal/engine/l4/curve"
)

// RB Layer-4 mechanics (docs/scoring-engine/RB_Rubric.md, locked v1.0). Rubric constants —
// never admin-exposed (Hard Constraint). RB is the STANDARD offense rubric: Combined =
// film × RAS × breakout, no SL-020 (RAS is active) and no SL-021 (no cushion guard). It is
// the first non-High-tier RAS position — RAS contributes at a MEDIUM tier (SL-004): a
// narrower ±4% band and a gentler steepness than DT's High tier, reflecting RAS's reduced
// predictive weight at RB.
const (
	// Film component (RB_Rubric §2): S-curve over the upstream film composite, ±5%.
	rbFilmInflection = 0.50
	rbFilmSteepness  = 12.0
	rbFilmCap        = 0.05 // ±5%

	// RAS component (RB_Rubric §3): ACTIVE (no SL-020), MEDIUM tier. raw RAS / 10 through the
	// S-curve, scaled by the position weight. Cap ±4% and steepness 8.0 are both narrower than
	// DT's High-tier ±8%/10.0 — RAS is meaningful at year 0 but not the elite predictor it is
	// at WR/DT.
	rbRASInflection = 0.50
	rbRASSteepness  = 8.0
	rbRASCap        = 0.04 // ±4% (Medium-tier, SL-004)
	// RAS_position_weight follows the SL-018 proportional Medium-tier schedule: rookie/pre-NFL
	// 0.60, after 1 NFL season 0.30 (50% of baseline), year 2+ 0.06 (10%). v1.0 ships the
	// ROOKIE weight only — the receding schedule needs an NFL-career-stage input that no
	// consumer yet supplies (the same "version the boundary" deferral as DT's High-tier
	// schedule). It wires in when a case gates on it, not before.
	rbRASRookieWeight = 0.60

	// Breakout component (RB_Rubric §4): standard ±5% cap, steepness 11.0.
	rbBreakoutInflection = 0.50
	rbBreakoutSteepness  = 11.0
	rbBreakoutCap        = 0.05

	// Breakout sub-signal weights (RB_Rubric §4) — sum to 1.00. College Workload is elevated
	// to 0.30 (vs QB's 0.30 college share but RB's 0.10 redistributed from breakout age +
	// school tier): workhorse usage at college is the strongest dynasty signal for RB.
	rbWeightBreakoutAge   = 0.35
	rbWeightSchoolTier    = 0.20
	rbWeightCollegeShare  = 0.30
	rbWeightAgeTrajectory = 0.15
)

// RB is the running-back Layer-4 rubric. Combined = film × RAS × breakout
// (Backend_Architecture:256). Film is Data-Parity neutral in v1.0 (offense film weights
// UNSET pending the film-source redesign), so RBs differentiate on the ACTIVE (Medium-tier)
// RAS component and breakout. A genuinely-thin scouting profile (late breakout age, low
// college workload, smaller school, post-peak age) pulls breakout — and therefore Combined —
// below 1.000 (RB_Rubric §7, the Herbert pattern; harness case 3B). TouchShare (RB-only) is
// NOT consumed in v1.0 (deferred); it grows the boundary only when the film redesign that
// uses it lands.
//
// Curves are held on the struct (not package globals — gochecknoglobals) and built once.
type RB struct {
	breakoutAge   []curve.Breakpoint
	collegeShare  []curve.Breakpoint
	ageTrajectory []curve.Breakpoint
}

// NewRB builds the RB rubric with its RB_Rubric §4 normalization curves. School tier is
// position-independent and arrives pre-normalized, so it has no curve here.
func NewRB() *RB {
	return &RB{
		// Breakout Age (years → [0,1]): aggressive 20–21 dropoff (RB short career window).
		// ≤19.5 → 1.00, 20.0 → 0.80, 20.5 → 0.50, ≥21.0 → 0.20.
		breakoutAge: []curve.Breakpoint{{X: 19.5, Y: 1.00}, {X: 20.0, Y: 0.80}, {X: 20.5, Y: 0.50}, {X: 21.0, Y: 0.20}},
		// College Workload (final-year touch share of team RB touches → [0,1]): ≤0.20 → 0.15,
		// 0.30 → 0.60, ≥0.40 → 1.00. Higher thresholds than WR target share (workhorse RBs
		// claim a much higher share than alpha WRs).
		collegeShare: []curve.Breakpoint{{X: 0.20, Y: 0.15}, {X: 0.30, Y: 0.60}, {X: 0.40, Y: 1.00}},
		// Age Trajectory (player age → [0,1]) relative to RB peak 25 — the shortest peak window
		// of any skill position. ≤21 → 1.00 … 25 (peak) → 0.50 … ≥29 → 0.00.
		ageTrajectory: []curve.Breakpoint{
			{X: 21, Y: 1.00}, {X: 22, Y: 0.85}, {X: 23, Y: 0.75}, {X: 24, Y: 0.60}, {X: 25, Y: 0.50},
			{X: 26, Y: 0.30}, {X: 27, Y: 0.15}, {X: 28, Y: 0.05}, {X: 29, Y: 0.00},
		},
	}
}

// Apply computes the RB Layer-4 output. Film is the S-curve over the upstream composite, or
// neutral 1.000 when no source is populated (Data-Parity Rule — an absent component never
// penalizes). RAS is ACTIVE: raw RAS / 10 through a Medium-tier S-curve scaled by the rookie
// position weight, or neutral 1.000 when RAS is absent. Breakout weights the four sub-signals
// (breakout age, school tier, college workload, age trajectory), then applies its S-curve.
func (r *RB) Apply(in engine.Layer4Input) engine.Layer4Output {
	sc := in.Scouting
	p := in.Player

	filmRaw := curve.NeutralNorm // Data-Parity neutral default (offense film weights unset)
	film := 1.0
	if sc.HasFilm {
		filmRaw = sc.FilmComposite
		film = curve.Scurve(sc.FilmComposite, rbFilmInflection, rbFilmSteepness, rbFilmCap)
	}

	// RAS component (Data-Parity neutral when absent). The position weight scales the S-curve's
	// deviation from neutral; at the rookie weight (0.60) RAS contributes a fraction of the
	// full Medium-tier S-curve, and a later receding weight pulls it toward 1.000 as NFL data
	// accrues.
	rasEffective := 1.0
	if p.HasRAS {
		rasCurve := curve.Scurve(p.RAS/10.0, rbRASInflection, rbRASSteepness, rbRASCap)
		rasEffective = 1.0 + rbRASRookieWeight*(rasCurve-1.0)
	}

	// Breakout composite. Age trajectory always has a value (age is known); an ABSENT sub-signal
	// contributes the neutral normalized value (curve.NeutralNorm), never the curve ceiling/floor
	// a raw zero would hit.
	composite := rbWeightBreakoutAge*curve.SubSignal(sc.HasBreakoutAge, r.breakoutAge, sc.BreakoutAge) +
		rbWeightSchoolTier*curve.Present(sc.HasSchoolTier, sc.SchoolTierNorm) +
		rbWeightCollegeShare*curve.SubSignal(sc.HasCollegeShare, r.collegeShare, sc.CollegeShare) +
		rbWeightAgeTrajectory*curve.Interp(r.ageTrajectory, p.Age)
	breakout := curve.Scurve(composite, rbBreakoutInflection, rbBreakoutSteepness, rbBreakoutCap)

	return engine.Layer4Output{
		FilmEffective:     film,
		FilmRaw:           filmRaw,
		RASEffective:      rasEffective,
		BreakoutEffective: breakout,
		Combined:          film * rasEffective * breakout,
	}
}
