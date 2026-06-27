package defense

import (
	"github.com/secureprospective/TheWarRoom/internal/engine"
	"github.com/secureprospective/TheWarRoom/internal/engine/l4/curve"
)

// DE Layer-4 mechanics (docs/scoring-engine/DE_Rubric.md, locked v1.0). Rubric constants —
// never admin-exposed (Hard Constraint). DE is the SECOND SL-019 instance (after TE) and the
// FIRST defensive position to apply it — it reuses the shared curve.SL019 helper VERBATIM
// (M17: the modulator was extracted to curve on TE's first use precisely so DE/LB inherit it
// with no new mechanic code). DE is the standard offense-shaped template on the defense side:
// active High-tier RAS at STANDARD steepness 10.0 (NOT TE's 11.0 — DE is the same shape as
// WR/DT), standard ±5% film (NOT DT's SL-005 ±3% compression), and NO SL-021 cushion (that is
// DT-specific; DE instead carries SL-019 like TE). Per OQ-004, this rubric scores ALL
// pass-rush-primary EDGE defenders regardless of MFL tag (DE/EDGE/3-4 OLB); the DE-vs-LB
// dispatch is a position-resolution concern deferred to B5b-LB (harness case 3J).
const (
	// Film component (DE_Rubric §2): S-curve over the IDP film composite, STANDARD ±5% (SL-002
	// upper bound — NOT DT's SL-005 ±3% compression). Data-Parity neutral in v1.0 (IDP film
	// weights UNSET — DE film redesigned onto Madden defense sub-attrs, weights unset, same as DT).
	deFilmInflection = 0.50
	deFilmSteepness  = 12.0
	deFilmCap        = 0.05 // ±5% (standard) — vs DT's SL-005 ±3%

	// RAS component (DE_Rubric §3): ACTIVE (no SL-020), HIGH tier. raw RAS / 10 through the
	// S-curve scaled by the position weight. Cap ±8% (High-tier) at STANDARD steepness 10.0 —
	// NOT TE's steeper 11.0 (athletic profile is dominant at EDGE, but DE is the same S-curve
	// shape as WR/DT).
	deRASInflection = 0.50
	deRASSteepness  = 10.0
	deRASCap        = 0.08 // ±8% (High-tier, SL-004)
	// RAS_position_weight follows the High-tier SL-018 schedule (rookie 1.00 → 0.50 → 0.10).
	// v1.0 ships the ROOKIE weight only — the receding schedule needs an NFL-career-stage input
	// no consumer yet supplies (the same "version the boundary" deferral as TE/WR/DT). The
	// SL-019 modulator interactions below are INDEPENDENT of SL-018 (DE_Rubric §3): RAS keeps
	// modulating the development/career curves across career even as the RAS COMPONENT recedes.
	deRASRookieWeight = 1.00

	// Breakout component (DE_Rubric §4): standard ±5% cap, steepness 11.0.
	deBreakoutInflection = 0.50
	deBreakoutSteepness  = 11.0
	deBreakoutCap        = 0.05

	// Breakout sub-signal weights (DE_Rubric §4) — sum to 1.00. College Production Share
	// (Sack + TFL market share) elevated to 0.35 (the cleanest pre-NFL pass-rush signal);
	// breakout age held at 0.30 (EDGE players develop technique into their early 20s).
	deWeightBreakoutAge   = 0.30
	deWeightSchoolTier    = 0.20
	deWeightCollegeShare  = 0.35
	deWeightAgeTrajectory = 0.15

	// deSL019Strength is the DE RAS-modulator strength (DE_Rubric §4) — held at the TE value
	// 0.35 (CAL-022 pending), applied to BOTH the breakout-age and age-trajectory sub-signals
	// (symmetric, SL-OQ-024). Passed to the shared curve.SL019 helper — never re-implemented.
	deSL019Strength = 0.35
)

// DE is the defensive-end / EDGE Layer-4 rubric. Combined = film × RAS × breakout
// (Backend_Architecture:256). Film is Data-Parity neutral in v1.0 (IDP film weights UNSET),
// so DEs differentiate on the ACTIVE High-tier RAS component and the breakout composite —
// where SL-019 makes the athletic profile lift the breakout-age and age-trajectory sub-signals
// (DE_Rubric §4). The Garrett/Lawrence finding (§5): elite-RAS EDGE defenders earn athletic
// credit on top of their breakout, while a non-athletic late-breakout vet does not — "athletic
// + late developer" is rewarded, "non-athletic + late developer" is not.
//
// Curves are held on the struct (not package globals — gochecknoglobals) and built once.
type DE struct {
	breakoutAge   []curve.Breakpoint
	collegeShare  []curve.Breakpoint
	ageTrajectory []curve.Breakpoint
}

// NewDE builds the DE rubric with its DE_Rubric §4 normalization curves. School tier uses the
// boundary TEMPLATE (no DE branch in composition.schoolTierNorm — P4 1.00 / G5 0.70 / FCS
// 0.40 / Non-FCS 0.10) and arrives pre-normalized, so it has no curve here.
func NewDE() *DE {
	return &DE{
		// Breakout Age (years → [0,1]), RB-shaped aggressive dropoff (EDGE breakouts at 18–19
		// are rare but distinguishing): ≤19.5 → 1.00, 20.0 → 0.80, 20.5 → 0.50, ≥21.0 → 0.15.
		breakoutAge: []curve.Breakpoint{{X: 19.5, Y: 1.00}, {X: 20.0, Y: 0.80}, {X: 20.5, Y: 0.50}, {X: 21.0, Y: 0.15}},
		// College Production Share (final-year Sack + TFL market share → [0,1]): ≤0.12 → 0.15,
		// 0.20 → 0.55, ≥0.28 → 1.00.
		collegeShare: []curve.Breakpoint{{X: 0.12, Y: 0.15}, {X: 0.20, Y: 0.55}, {X: 0.28, Y: 1.00}},
		// Age Trajectory (player age → [0,1]) relative to DE peak 30. ≤26 → 1.00 … 30 (peak) →
		// 0.50 … ≥34 → 0.00. SL-019 then modulates this for high-RAS DEs (no SL-021 cushion).
		ageTrajectory: []curve.Breakpoint{
			{X: 26, Y: 1.00}, {X: 27, Y: 0.85}, {X: 28, Y: 0.70}, {X: 29, Y: 0.55}, {X: 30, Y: 0.50},
			{X: 31, Y: 0.35}, {X: 32, Y: 0.20}, {X: 33, Y: 0.10}, {X: 34, Y: 0.00},
		},
	}
}

// Apply computes the DE Layer-4 output. Film is the S-curve over the IDP composite, or neutral
// 1.000 when no source is populated (Data-Parity Rule). RAS is ACTIVE, High-tier at standard
// steepness 10.0: raw RAS / 10 through the S-curve scaled by the rookie weight, or neutral
// 1.000 when absent (never forced — AD-09). Breakout weights the four sub-signals; SL-019 then
// lifts the PRESENT breakout-age and age-trajectory sub-signals by the athletic profile (the
// shared curve.SL019, reused verbatim from TE) — an absent breakout age stays neutral (no
// modulation of unknown data), and an absent RAS contributes no lift (the modulator's
// Data-Parity stance).
func (de *DE) Apply(in engine.Layer4Input) engine.Layer4Output {
	sc := in.Scouting
	p := in.Player

	filmRaw := curve.NeutralNorm // Data-Parity neutral default (IDP film weights unset)
	film := 1.0
	if sc.HasFilm {
		filmRaw = sc.FilmComposite
		film = curve.Scurve(sc.FilmComposite, deFilmInflection, deFilmSteepness, deFilmCap)
	}

	// RAS component (Data-Parity neutral when absent). The rookie position weight (1.00) scales
	// the full High-tier S-curve deviation from neutral; a later receding weight would pull it
	// toward 1.000 as NFL data accrues.
	rasEffective := 1.0
	if p.HasRAS {
		rasCurve := curve.Scurve(p.RAS/10.0, deRASInflection, deRASSteepness, deRASCap)
		rasEffective = 1.0 + deRASRookieWeight*(rasCurve-1.0)
	}

	// SL-019 RAS-modulator interactions (DE_Rubric §4) — the SECOND instance, reusing the shared
	// curve.SL019 helper verbatim. RAS_normalized = raw RAS / 10; the helper neutralizes an
	// absent/out-of-range RAS. Modulation is applied ONLY to PRESENT sub-signals: an absent
	// breakout age stays neutral (modulating the neutral would manufacture a lift for unknown
	// data — a Data-Parity violation). Age trajectory is always present (age is known).
	rasNorm := p.RAS / 10.0
	breakoutAgeNorm := curve.SubSignal(sc.HasBreakoutAge, de.breakoutAge, sc.BreakoutAge)
	if sc.HasBreakoutAge {
		breakoutAgeNorm = curve.SL019(breakoutAgeNorm, rasNorm, deSL019Strength, p.HasRAS)
	}
	ageTrajectoryNorm := curve.SL019(curve.Interp(de.ageTrajectory, p.Age), rasNorm, deSL019Strength, p.HasRAS)

	composite := deWeightBreakoutAge*breakoutAgeNorm +
		deWeightSchoolTier*curve.Present(sc.HasSchoolTier, sc.SchoolTierNorm) +
		deWeightCollegeShare*curve.SubSignal(sc.HasCollegeShare, de.collegeShare, sc.CollegeShare) +
		deWeightAgeTrajectory*ageTrajectoryNorm
	breakout := curve.Scurve(composite, deBreakoutInflection, deBreakoutSteepness, deBreakoutCap)

	return engine.Layer4Output{
		FilmEffective:     film,
		FilmRaw:           filmRaw,
		RASEffective:      rasEffective,
		BreakoutEffective: breakout,
		Combined:          film * rasEffective * breakout,
	}
}
