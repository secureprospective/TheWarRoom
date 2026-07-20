package defense

import (
	"github.com/secureprospective/TheWarRoom/internal/engine"
	"github.com/secureprospective/TheWarRoom/internal/engine/l4/curve"
)

// CB Layer-4 mechanics (docs/scoring-engine/CB_Rubric.md, locked v1.0). Rubric constants —
// never admin-exposed (Hard Constraint). CB is the THIRD SL-019 instance (after TE and DE),
// the first applied at REDUCED modulator strength (0.30 vs TE/DE's 0.35), and the FIRST
// position to elevate NFL Next Gen Stats coverage metrics to a dedicated film anchor (§2).
// It reuses the shared curve.SL019 helper VERBATIM (M17) at the reduced 0.30 strength. CB is
// a standard-shaped defense rubric: active High-tier RAS (steepness 11.0 — CB athletic
// profiles are bimodal, sharper transition), STANDARD ±5% film (NOT DT/LB's SL-005 ±3%
// compression — CB is rich-data via NGS, not a thin-data IDP), and NO SL-021 cushion (that
// is DT-specific; CB carries SL-019 like TE/DE). Per OQ-004, CB scores cornerbacks only;
// hybrid safety-corner roles route through the S rubric.
const (
	// Film component (CB_Rubric §2): S-curve over the coverage film composite, STANDARD ±5% cap
	// at steepness 12.0. FILM Thread C (C-4 step 1) calibrated this composite: the subjective
	// sources it originally named (PFF/TDN/Nerds/IDP/NGS-tracking) were ELIMINATED, and the ONE
	// surviving live input is the PFR coverage-allowed anchor, blended UPSTREAM at the locked
	// K4 weight (0.20 of the film budget, in rankings.applyScouting) into the single [0,1]
	// FilmComposite this rubric still consumes unchanged. A CB with no coverage anchor keeps
	// HasFilm=false → film returns neutral 1.000 (Data-Parity). The NGS dedicated anchor lives
	// INSIDE this composite; it does not change the engine's single [0,1] film input —
	// NGS-presence is asserted via the HasNGSAnchor introspection hook (case 3I), not the boundary.
	cbFilmInflection = 0.50
	cbFilmSteepness  = 12.0
	cbFilmCap        = 0.05 // ±5% (standard) — CB is rich-data, NOT SL-005 compressed

	// RAS component (CB_Rubric §3): ACTIVE (no SL-020), HIGH tier. raw RAS / 10 through the
	// S-curve scaled by the position weight. Cap ±8% (High-tier) at steepness 11.0 (CB athletic
	// profiles are bimodal — a sharper transition than DE/WR's 10.0 is defensible, §3).
	cbRASInflection = 0.50
	cbRASSteepness  = 11.0
	cbRASCap        = 0.08 // ±8% (High-tier, SL-004)
	// RAS_position_weight follows the High-tier SL-018 schedule (rookie 1.00 → 0.50 → 0.10).
	// v1.0 ships the ROOKIE weight only — the receding schedule needs an NFL-career-stage input
	// no consumer yet supplies (the same "version the boundary" deferral as every prior position).
	cbRASRookieWeight = 1.00

	// Breakout component (CB_Rubric §4): standard ±5% cap, steepness 10.0 (slightly less
	// aggressive than DE/LB's 11.0 — CB college-production data is noisier).
	cbBreakoutInflection = 0.50
	cbBreakoutSteepness  = 10.0
	cbBreakoutCap        = 0.05

	// Breakout sub-signal weights (CB_Rubric §4) — sum to 1.00. College Production Share (PD +
	// INT market share) is the cleanest college-to-NFL predictor at CB (0.40); School Tier is
	// ELEVATED to 0.25 (NFL-caliber CBs come from P4 reliably); breakout age dropped to 0.20
	// (CB breakout timing is less predictive — many CBs ascend through coverage repetition).
	cbWeightBreakoutAge   = 0.20
	cbWeightSchoolTier    = 0.25
	cbWeightCollegeShare  = 0.40
	cbWeightAgeTrajectory = 0.15

	// cbSL019Strength is the CB RAS-modulator strength (CB_Rubric §1/§4) — REDUCED to 0.30 from
	// TE/DE's 0.35 (CB longevity is more variable within the position — boundary vs slot, press
	// vs zone, scheme-dependency on safety help adds noise to the RAS-longevity signal). Applied
	// to BOTH the breakout-age and age-trajectory sub-signals (symmetric, SL-OQ-024). Passed to
	// the shared curve.SL019 helper — never re-implemented. (The L3 SL-019 buffer is the separate
	// 0.25 strength, a Layer-3 mechanic not built here — carry-forward.)
	cbSL019Strength = 0.30
)

// CB is the cornerback Layer-4 rubric. Combined = film × RAS × breakout
// (Backend_Architecture:256). Film is Data-Parity neutral in v1.0 (coverage film weights
// UNSET), so CBs differentiate on the ACTIVE High-tier RAS component and the breakout
// composite — where SL-019 (at the reduced 0.30 strength) makes the athletic profile lift the
// breakout-age and age-trajectory sub-signals (CB_Rubric §4). The Surtain/Peterson finding
// (§5): an elite-RAS aging CB earns MORE Layer-4 age-trajectory protection than a mid-RAS
// aging peer — the Lockett pattern amplified by SL-019, but less aggressively than at TE/DE.
//
// Curves are held on the struct (not package globals — gochecknoglobals) and built once.
type CB struct {
	breakoutAge   []curve.Breakpoint
	collegeShare  []curve.Breakpoint
	ageTrajectory []curve.Breakpoint
}

// NewCB builds the CB rubric with its CB_Rubric §4 normalization curves. School tier uses the
// boundary TEMPLATE (no CB branch in composition.schoolTierNorm — P4 1.00 / G5 0.70 / FCS
// 0.40 / Non-FCS 0.10) and arrives pre-normalized, so it has no curve here.
func NewCB() *CB {
	return &CB{
		// Breakout Age (years → [0,1]), half-year DE-shaped: ≤19.5 → 1.00, 20.5 → 0.75,
		// 21.5 → 0.45, ≥22.5 → 0.15.
		breakoutAge: []curve.Breakpoint{{X: 19.5, Y: 1.00}, {X: 20.5, Y: 0.75}, {X: 21.5, Y: 0.45}, {X: 22.5, Y: 0.15}},
		// College Production Share (final-year PD + INT market share → [0,1]): ≤0.08 → 0.15,
		// 0.16 → 0.55, ≥0.24 → 1.00.
		collegeShare: []curve.Breakpoint{{X: 0.08, Y: 0.15}, {X: 0.16, Y: 0.55}, {X: 0.24, Y: 1.00}},
		// Age Trajectory (player age → [0,1]) relative to CB peak 28 — one year earlier than
		// WR/TE/LB (peak 29): pulls begin at age 25. ≤24 → 1.00 … 28 (peak) → 0.50 … ≥32 → 0.00.
		// SL-019 then modulates this for high-RAS CBs (no SL-021 cushion).
		ageTrajectory: []curve.Breakpoint{
			{X: 24, Y: 1.00}, {X: 25, Y: 0.85}, {X: 26, Y: 0.70}, {X: 27, Y: 0.55}, {X: 28, Y: 0.50},
			{X: 29, Y: 0.35}, {X: 30, Y: 0.20}, {X: 31, Y: 0.10}, {X: 32, Y: 0.00},
		},
	}
}

// HasNGSAnchor is the case-3I introspection hook: CB is the first position to elevate NFL Next
// Gen Stats coverage metrics to a dedicated film sub-signal anchor (CB_Rubric §2 — 0.30 weight,
// §7 "NGS as dedicated anchor"). Like DT.PFFAlpha it is a RUBRIC INTERNAL exposed for the
// harness only; it deliberately does NOT live on Layer4Output (the production surface) — the
// film weights are unset in v1.0, so a weight accessor would read 0.00 and NGS-presence is
// invisible at the engine boundary. S returns true too; offense and the other defensive rubrics
// do not implement this method at all, so the harness reads them as having no NGS anchor.
func (cb *CB) HasNGSAnchor() bool { return true }

// Apply computes the CB Layer-4 output. Film is the S-curve over the coverage composite, or
// neutral 1.000 when no source is populated (Data-Parity Rule). RAS is ACTIVE, High-tier at
// steepness 11.0: raw RAS / 10 through the S-curve scaled by the rookie weight, or neutral
// 1.000 when absent (never forced — AD-09). Breakout weights the four sub-signals; SL-019 then
// lifts the PRESENT breakout-age and age-trajectory sub-signals by the athletic profile at the
// REDUCED 0.30 strength (the shared curve.SL019, reused verbatim from DE/TE) — an absent
// breakout age stays neutral (no modulation of unknown data), and an absent RAS contributes no
// lift (the modulator's Data-Parity stance).
func (cb *CB) Apply(in engine.Layer4Input) engine.Layer4Output {
	sc := in.Scouting
	p := in.Player

	filmRaw := curve.NeutralNorm // Data-Parity neutral default (coverage film weights unset)
	film := 1.0
	if sc.HasFilm {
		filmRaw = sc.FilmComposite
		film = curve.Scurve(sc.FilmComposite, cbFilmInflection, cbFilmSteepness, cbFilmCap)
	}

	// RAS component (Data-Parity neutral when absent). The rookie position weight (1.00) scales
	// the full High-tier S-curve deviation from neutral; a later receding weight would pull it
	// toward 1.000 as NFL data accrues.
	rasEffective := 1.0
	if p.HasRAS {
		rasCurve := curve.Scurve(p.RAS/10.0, cbRASInflection, cbRASSteepness, cbRASCap)
		rasEffective = 1.0 + cbRASRookieWeight*(rasCurve-1.0)
	}

	// SL-019 RAS-modulator interactions (CB_Rubric §4) — the THIRD instance, reusing the shared
	// curve.SL019 helper verbatim at the REDUCED 0.30 strength. RAS_normalized = raw RAS / 10;
	// the helper neutralizes an absent/out-of-range RAS. Modulation is applied ONLY to PRESENT
	// sub-signals: an absent breakout age stays neutral (modulating the neutral would manufacture
	// a lift for unknown data — a Data-Parity violation). Age trajectory is always present.
	rasNorm := p.RAS / 10.0
	breakoutAgeNorm := curve.SubSignal(sc.HasBreakoutAge, cb.breakoutAge, sc.BreakoutAge)
	if sc.HasBreakoutAge {
		breakoutAgeNorm = curve.SL019(breakoutAgeNorm, rasNorm, cbSL019Strength, p.HasRAS)
	}
	ageTrajectoryNorm := curve.SL019(curve.Interp(cb.ageTrajectory, p.Age), rasNorm, cbSL019Strength, p.HasRAS)

	composite := cbWeightBreakoutAge*breakoutAgeNorm +
		cbWeightSchoolTier*curve.Present(sc.HasSchoolTier, sc.SchoolTierNorm) +
		cbWeightCollegeShare*curve.SubSignal(sc.HasCollegeShare, cb.collegeShare, sc.CollegeShare) +
		cbWeightAgeTrajectory*ageTrajectoryNorm
	breakout := curve.Scurve(composite, cbBreakoutInflection, cbBreakoutSteepness, cbBreakoutCap)

	return engine.Layer4Output{
		FilmEffective:     film,
		FilmRaw:           filmRaw,
		RASEffective:      rasEffective,
		BreakoutEffective: breakout,
		Combined:          film * rasEffective * breakout,
	}
}
