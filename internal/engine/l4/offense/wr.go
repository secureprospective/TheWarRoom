package offense

import (
	"github.com/secureprospective/TheWarRoom/internal/engine"
	"github.com/secureprospective/TheWarRoom/internal/engine/l4/curve"
)

// WR Layer-4 mechanics (docs/scoring-engine/WR_Rubric.md, locked v1.0). Rubric constants —
// never admin-exposed (Hard Constraint). WR is the canonical STANDARD offense rubric
// (WR_Rubric §7 — "the cleanest template for offensive skill positions"): Combined =
// film × RAS × breakout, no SL-020 (RAS is active) and no SL-021 (no cushion guard). RAS
// contributes at the HIGH tier (SL-004) — a wide ±8% band and steepness 10.0, identical in
// shape to DT's active RAS, reflecting RAS's elite predictive weight at WR (vs RB's Medium
// tier). SL-019 is NOT applied (SL-022): Layer 3 carries WR aging, so the breakout
// sub-signals use base curve values with no RAS modulation (like QB, unlike DT's cushion).
const (
	// Film component (WR_Rubric §2): standard S-curve over the upstream film composite, ±5%.
	wrFilmInflection = 0.50
	wrFilmSteepness  = 12.0
	wrFilmCap        = 0.05 // ±5%

	// RAS component (WR_Rubric §3): ACTIVE (no SL-020), HIGH tier. raw RAS / 10 through the
	// S-curve, scaled by the position weight. Cap ±8% and steepness 10.0 match DT's High tier —
	// RAS is the primary athletic predictor at WR at year 0.
	wrRASInflection = 0.50
	wrRASSteepness  = 10.0
	wrRASCap        = 0.08 // ±8% (High-tier, SL-004)
	// RAS_position_weight follows the High-tier SL-018 schedule: rookie/pre-NFL 1.00, after 1
	// NFL season 0.50, year 2+ 0.10. v1.0 ships the ROOKIE weight only — the receding schedule
	// needs an NFL-career-stage input that no consumer yet supplies (the same "version the
	// boundary" deferral as DT/RB). AD-09: RAS is NOT forced to 1.000 (that is SL-020/QB only).
	wrRASRookieWeight = 1.00

	// Breakout component (WR_Rubric §4): standard ±5% cap, steepness 11.0.
	wrBreakoutInflection = 0.50
	wrBreakoutSteepness  = 11.0
	wrBreakoutCap        = 0.05

	// Breakout sub-signal weights (WR_Rubric §4) — sum to 1.00. Breakout Age is the gold-standard
	// WR predictor at the highest weight (0.40) of any breakout sub-signal across positions.
	wrWeightBreakoutAge   = 0.40
	wrWeightSchoolTier    = 0.25
	wrWeightCollegeShare  = 0.20
	wrWeightAgeTrajectory = 0.15
)

// WR is the wide-receiver Layer-4 rubric. Combined = film × RAS × breakout
// (Backend_Architecture:256). Film is Data-Parity neutral in v1.0 (offense film weights
// UNSET — the four manual WR film sources were eliminated, redesign pending per the offense
// source map §5), so WRs differentiate on the ACTIVE (High-tier) RAS component and breakout.
// A declining ELITE-draft-profile WR stays at/above 1.000 because the static breakout
// sub-signals (school tier, college usage, early breakout age) reflect draft-era development
// and offset the age-trajectory crater — the Lockett pattern (WR_Rubric §5 / harness case 3A),
// the deliberate contrast to RB's Herbert (3B), where weak static signals DO pull below 1.000.
//
// Curves are held on the struct (not package globals — gochecknoglobals) and built once.
type WR struct {
	breakoutAge   []curve.Breakpoint
	collegeShare  []curve.Breakpoint
	ageTrajectory []curve.Breakpoint
}

// NewWR builds the WR rubric with its WR_Rubric §4 normalization curves. School tier is
// position-independent for WR (it uses the boundary's TEMPLATE values — no WR branch in
// composition.schoolTierNorm) and arrives pre-normalized, so it has no curve here.
func NewWR() *WR {
	return &WR{
		// Breakout Age (years → [0,1]): age at first college season with ≥20% team receiving
		// production. ≤19.0 → 1.00, 20.0 → 0.75, 21.0 → 0.40, ≥22.0 → 0.10.
		breakoutAge: []curve.Breakpoint{{X: 19.0, Y: 1.00}, {X: 20.0, Y: 0.75}, {X: 21.0, Y: 0.40}, {X: 22.0, Y: 0.10}},
		// College Usage Rate (final-year reception/yardage share of team → [0,1]): ≤0.15 → 0.10,
		// 0.25 → 0.50, ≥0.35 → 1.00. Lower thresholds than RB's workhorse touch share (alpha WRs
		// claim a smaller share of a passing offense than workhorse RBs claim of the run game).
		collegeShare: []curve.Breakpoint{{X: 0.15, Y: 0.10}, {X: 0.25, Y: 0.50}, {X: 0.35, Y: 1.00}},
		// Age Trajectory (player age → [0,1]) relative to WR peak 29 — the longest peak window of
		// the skill positions. ≤25 → 1.00 … 29 (peak) → 0.50 … ≥33 → 0.00.
		ageTrajectory: []curve.Breakpoint{
			{X: 25, Y: 1.00}, {X: 26, Y: 0.85}, {X: 27, Y: 0.70}, {X: 28, Y: 0.55}, {X: 29, Y: 0.50},
			{X: 30, Y: 0.35}, {X: 31, Y: 0.20}, {X: 32, Y: 0.10}, {X: 33, Y: 0.00},
		},
	}
}

// Apply computes the WR Layer-4 output. Film is the S-curve over the upstream composite, or
// neutral 1.000 when no source is populated (Data-Parity Rule — an absent component never
// penalizes). RAS is ACTIVE and High-tier: raw RAS / 10 through the S-curve scaled by the
// rookie position weight, or neutral 1.000 when RAS is absent (AD-09 — never forced). Breakout
// weights the four sub-signals (breakout age, school tier, college usage, age trajectory), then
// applies its S-curve; SL-019 is NOT applied (SL-022 — no RAS modulation of breakout at WR).
func (w *WR) Apply(in engine.Layer4Input) engine.Layer4Output {
	sc := in.Scouting
	p := in.Player

	filmRaw := curve.NeutralNorm // Data-Parity neutral default (offense film weights unset)
	film := 1.0
	if sc.HasFilm {
		filmRaw = sc.FilmComposite
		film = curve.Scurve(sc.FilmComposite, wrFilmInflection, wrFilmSteepness, wrFilmCap)
	}

	// RAS component (Data-Parity neutral when absent). The position weight scales the S-curve's
	// deviation from neutral; at the rookie weight (1.00) the effect is the full High-tier
	// S-curve, and a later receding weight pulls it toward 1.000 as NFL data accrues.
	rasEffective := 1.0
	if p.HasRAS {
		rasCurve := curve.Scurve(p.RAS/10.0, wrRASInflection, wrRASSteepness, wrRASCap)
		rasEffective = 1.0 + wrRASRookieWeight*(rasCurve-1.0)
	}

	// Breakout composite. Age trajectory always has a value (age is known); an ABSENT sub-signal
	// contributes the neutral normalized value (curve.NeutralNorm), never the curve ceiling/floor
	// a raw zero would hit (Data-Parity Rule).
	composite := wrWeightBreakoutAge*curve.SubSignal(sc.HasBreakoutAge, w.breakoutAge, sc.BreakoutAge) +
		wrWeightSchoolTier*curve.Present(sc.HasSchoolTier, sc.SchoolTierNorm) +
		wrWeightCollegeShare*curve.SubSignal(sc.HasCollegeShare, w.collegeShare, sc.CollegeShare) +
		wrWeightAgeTrajectory*curve.Interp(w.ageTrajectory, p.Age)
	breakout := curve.Scurve(composite, wrBreakoutInflection, wrBreakoutSteepness, wrBreakoutCap)

	return engine.Layer4Output{
		FilmEffective:     film,
		FilmRaw:           filmRaw,
		RASEffective:      rasEffective,
		BreakoutEffective: breakout,
		Combined:          film * rasEffective * breakout,
	}
}
