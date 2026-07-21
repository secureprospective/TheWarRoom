package offense

import (
	"github.com/secureprospective/TheWarRoom/internal/engine"
	"github.com/secureprospective/TheWarRoom/internal/engine/l4/curve"
)

// TE Layer-4 mechanics (docs/scoring-engine/TE_Rubric.md, locked v1.0). Rubric constants —
// never admin-exposed (Hard Constraint). TE is the standard offense template (film × RAS ×
// breakout, no SL-020/SL-021) PLUS the first instance of SL-019 RAS-modulator interactions
// (TE_Rubric §4) — the generalizable mechanic DE/LB inherit (the shared curve.SL019 helper).
// RAS is ACTIVE at the HIGH tier (SL-004 amended Medium→High for TE: athletic profile is the
// dominant explanatory variable here), but with a STEEPER steepness (11.0 vs WR's 10.0) — at
// TE small athletic differences separate outcomes more sharply (TE_Rubric §3).
const (
	// Film component (TE_Rubric §2): S-curve over the upstream composite, ±5%. The composite
	// is WIRED (FILM Thread C, C-4 step 3, same as QB/RB/WR), Data-Parity neutral when absent.
	teFilmInflection = 0.50
	teFilmSteepness  = 12.0
	teFilmCap        = 0.05 // ±5%

	// RAS component (TE_Rubric §3): ACTIVE (no SL-020), HIGH tier. raw RAS / 10 through the
	// S-curve scaled by the position weight. Cap ±8% (High-tier) but steepness 11.0 — STEEPER
	// than WR's 10.0: the 8.5-vs-9.5 RAS gap is more decisive at TE.
	teRASInflection = 0.50
	teRASSteepness  = 11.0
	teRASCap        = 0.08 // ±8% (High-tier, SL-004 amended)
	// RAS_position_weight follows the High-tier SL-018 schedule (rookie 1.00 → 0.50 → 0.10).
	// v1.0 ships the ROOKIE weight only — the receding schedule needs an NFL-career-stage
	// input no consumer yet supplies (the same "version the boundary" deferral as WR/DT). The
	// SL-019 modulator interactions below are INDEPENDENT of SL-018 (TE_Rubric §3): RAS keeps
	// modulating the development/career curves across career even as the RAS COMPONENT recedes.
	teRASRookieWeight = 1.00

	// Breakout component (TE_Rubric §4): ±5% cap, steepness 11.0.
	teBreakoutInflection = 0.50
	teBreakoutSteepness  = 11.0
	teBreakoutCap        = 0.05

	// Breakout sub-signal weights (TE_Rubric §4) — sum to 1.00. Same allocation as RB; college
	// usage elevated to 0.30 (TE breakout age is naturally later, so college production carries
	// more of the signal).
	teWeightBreakoutAge   = 0.35
	teWeightSchoolTier    = 0.20
	teWeightCollegeShare  = 0.30
	teWeightAgeTrajectory = 0.15

	// teSL019Strength is the TE-specific RAS-modulator strength (TE_Rubric §4) — the SAME 0.35
	// applied to BOTH the breakout-age and age-trajectory sub-signals (symmetric, SL-OQ-024).
	teSL019Strength = 0.35
)

// TE is the tight-end Layer-4 rubric. Combined = film × RAS × breakout
// (Backend_Architecture:256). Film is the WIRED offense composite (FILM Thread C, C-4
// step 3), or Data-Parity neutral when absent; TEs also differentiate on the ACTIVE
// High-tier RAS component and the breakout
// composite — where SL-019 makes the athletic profile lift the breakout-age and
// age-trajectory sub-signals (TE_Rubric §4). The Higbee/Henry finding (§5): two aging TEs
// with similar RAS but different draft-era profile strength land apart, and the modulator
// adds athletic credit on top — "athletic + late developer" is rewarded, "non-athletic +
// late developer" is not.
//
// Curves are held on the struct (not package globals — gochecknoglobals) and built once.
type TE struct {
	breakoutAge   []curve.Breakpoint
	collegeShare  []curve.Breakpoint
	ageTrajectory []curve.Breakpoint
}

// NewTE builds the TE rubric with its TE_Rubric §4 normalization curves. School tier uses the
// boundary TEMPLATE (no TE branch in composition.schoolTierNorm) and arrives pre-normalized,
// so it has no curve here.
func NewTE() *TE {
	return &TE{
		// Breakout Age (years → [0,1]), later than WR (TE breakouts at 18–19 are vanishingly
		// rare): ≤20 → 1.00, 21 → 0.80, 22 → 0.50, ≥23 → 0.15.
		breakoutAge: []curve.Breakpoint{{X: 20.0, Y: 1.00}, {X: 21.0, Y: 0.80}, {X: 22.0, Y: 0.50}, {X: 23.0, Y: 0.15}},
		// College Usage Rate (final-year target share → [0,1]), lower thresholds than WR (TEs
		// share targets): ≤0.08 → 0.10, 0.15 → 0.50, ≥0.22 → 1.00.
		collegeShare: []curve.Breakpoint{{X: 0.08, Y: 0.10}, {X: 0.15, Y: 0.50}, {X: 0.22, Y: 1.00}},
		// Age Trajectory (player age → [0,1]) relative to TE peak 29 — same shape as WR.
		// ≤25 → 1.00 … 29 (peak) → 0.50 … ≥33 → 0.00.
		ageTrajectory: []curve.Breakpoint{
			{X: 25, Y: 1.00}, {X: 26, Y: 0.85}, {X: 27, Y: 0.70}, {X: 28, Y: 0.55}, {X: 29, Y: 0.50},
			{X: 30, Y: 0.35}, {X: 31, Y: 0.20}, {X: 32, Y: 0.10}, {X: 33, Y: 0.00},
		},
	}
}

// Apply computes the TE Layer-4 output. Film is the S-curve over the upstream composite, or
// neutral 1.000 when no source is populated (Data-Parity Rule). RAS is ACTIVE, High-tier and
// steeper than WR: raw RAS / 10 through the S-curve scaled by the rookie weight, or neutral
// 1.000 when absent (never forced — AD-09). Breakout weights the four sub-signals; SL-019
// then lifts the PRESENT breakout-age and age-trajectory sub-signals by the athletic profile
// (curve.SL019) — an absent breakout age stays neutral (no modulation of unknown data), and an
// absent RAS contributes no lift (the modulator's Data-Parity stance).
func (te *TE) Apply(in engine.Layer4Input) engine.Layer4Output {
	sc := in.Scouting
	p := in.Player

	filmRaw := curve.NeutralNorm // Data-Parity neutral default (no film composite resolved)
	film := 1.0
	if sc.HasFilm {
		filmRaw = sc.FilmComposite
		film = curve.Scurve(sc.FilmComposite, teFilmInflection, teFilmSteepness, teFilmCap)
	}

	// RAS component (Data-Parity neutral when absent). The rookie position weight (1.00) scales
	// the full High-tier S-curve deviation from neutral; a later receding weight would pull it
	// toward 1.000 as NFL data accrues.
	rasEffective := 1.0
	if p.HasRAS {
		rasCurve := curve.Scurve(p.RAS/10.0, teRASInflection, teRASSteepness, teRASCap)
		rasEffective = 1.0 + teRASRookieWeight*(rasCurve-1.0)
	}

	// SL-019 RAS-modulator interactions (TE_Rubric §4). RAS_normalized = raw RAS / 10; the helper
	// neutralizes an absent/out-of-range RAS. Modulation is applied ONLY to PRESENT sub-signals:
	// an absent breakout age stays neutral (modulating the neutral would manufacture a lift for
	// unknown data — a Data-Parity violation). Age trajectory is always present (age is known).
	rasNorm := p.RAS / 10.0
	breakoutAgeNorm := curve.SubSignal(sc.HasBreakoutAge, te.breakoutAge, sc.BreakoutAge)
	if sc.HasBreakoutAge {
		breakoutAgeNorm = curve.SL019(breakoutAgeNorm, rasNorm, teSL019Strength, p.HasRAS)
	}
	ageTrajectoryNorm := curve.SL019(curve.Interp(te.ageTrajectory, p.Age), rasNorm, teSL019Strength, p.HasRAS)

	composite := teWeightBreakoutAge*breakoutAgeNorm +
		teWeightSchoolTier*curve.Present(sc.HasSchoolTier, sc.SchoolTierNorm) +
		teWeightCollegeShare*curve.SubSignal(sc.HasCollegeShare, te.collegeShare, sc.CollegeShare) +
		teWeightAgeTrajectory*ageTrajectoryNorm
	breakout := curve.Scurve(composite, teBreakoutInflection, teBreakoutSteepness, teBreakoutCap)

	return engine.Layer4Output{
		FilmEffective:     film,
		FilmRaw:           filmRaw,
		RASEffective:      rasEffective,
		BreakoutEffective: breakout,
		Combined:          film * rasEffective * breakout,
	}
}
