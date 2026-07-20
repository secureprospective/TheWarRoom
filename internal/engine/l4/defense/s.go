package defense

import (
	"github.com/secureprospective/TheWarRoom/internal/engine"
	"github.com/secureprospective/TheWarRoom/internal/engine/l4/curve"
)

// S Layer-4 mechanics (docs/scoring-engine/S_Rubric.md, locked v1.0). Rubric constants —
// never admin-exposed (Hard Constraint). S is the FOURTH SL-019 instance (after TE, DE, CB)
// and the SECOND position to elevate NFL Next Gen Stats coverage/range metrics to a dedicated
// film anchor (§2/§7). It is structurally PARALLEL to CB — same peak limit (28), same High RAS
// tier, same gating criterion, same SL-019 strengths (0.30) — and reuses curve.SL019 VERBATIM
// (M17) at the reduced 0.30 strength. Two values diverge from CB, both spec-justified (§3/§4/§7):
// RAS steepness is 10.0 (NOT CB's 11.0 — S athletic profiles are continuous across the
// box-to-deep spectrum, not bimodal like CB's boundary/slot split), and breakout steepness is
// 11.0 (NOT CB's 10.0 — S college-production data is cleaner). S carries SL-019 like TE/DE/CB
// and has NO SL-021 cushion (that is DT-specific). Per OQ-004, S scores safeties only; the
// SafetyRole free-vs-box split (SL-OQ-035/036) stays schema-only — one S rubric for all safeties
// in v1.0.
const (
	// Film component (S_Rubric §2): S-curve over the coverage film composite, STANDARD ±5% cap
	// at steepness 12.0. FILM Thread C (C-4 steps 1–2) calibrated this composite: the subjective
	// sources it originally named (PFF/TDN/Nerds/IDP/NGS-tracking) were ELIMINATED, and TWO live
	// inputs now feed it, both blended UPSTREAM (rankings.applyScouting) into the single [0,1]
	// FilmComposite this rubric still consumes unchanged: the Madden defense sub-attribute
	// composite (K1, 0.75 of the S film budget — its curated set keeps the Madden man+zone
	// coverage term) and the PFR coverage-allowed anchor (K4, 0.20). The two coverage voices are
	// statistically independent (C-1 |r|<0.30), so both are kept as additive signals. A safety with
	// neither keeps HasFilm=false → film returns neutral 1.000 (Data-Parity). The NGS dedicated anchor lives
	// INSIDE this composite; it does not change the engine's single [0,1] film input —
	// NGS-presence is asserted via the HasNGSAnchor introspection hook (case 3I), not the boundary.
	sFilmInflection = 0.50
	sFilmSteepness  = 12.0
	sFilmCap        = 0.05 // ±5% (standard) — S is rich-data via NGS, NOT SL-005 compressed

	// RAS component (S_Rubric §3): ACTIVE (no SL-020), HIGH tier. raw RAS / 10 through the
	// S-curve scaled by the position weight. Cap ±8% (High-tier) at steepness 10.0 — LOWER than
	// CB's 11.0 because S athletic profiles span a continuous range (pure centerfielder to pure
	// thumper) rather than CB's bimodal boundary/slot split, so a less aggressive transition fits.
	sRASInflection = 0.50
	sRASSteepness  = 10.0 // NOT CB's 11.0 — S profiles continuous, not bimodal (§3/§7)
	sRASCap        = 0.08 // ±8% (High-tier, SL-004)
	// RAS_position_weight follows the High-tier SL-018 schedule (rookie 1.00 → 0.50 → 0.10).
	// v1.0 ships the ROOKIE weight only — the receding schedule needs an NFL-career-stage input
	// no consumer yet supplies (the same "version the boundary" deferral as every prior position).
	sRASRookieWeight = 1.00

	// Breakout component (S_Rubric §4): standard ±5% cap, steepness 11.0 — HIGHER than CB's 10.0
	// because S college-production data is cleaner than CB's (a slightly more aggressive transition
	// is defensible).
	sBreakoutInflection = 0.50
	sBreakoutSteepness  = 11.0 // NOT CB's 10.0 — S college-production data is cleaner (§4/§7)
	sBreakoutCap        = 0.05

	// Breakout sub-signal weights (S_Rubric §4) — sum to 1.00, SAME as CB. College Production
	// Share (INT + TACKLE market share — NOT CB's PD + INT, since safeties are more tackle-
	// involved, §4) is the cleanest college-to-NFL predictor at S (0.40); School Tier 0.25;
	// breakout age 0.20; age trajectory 0.15.
	sWeightBreakoutAge   = 0.20
	sWeightSchoolTier    = 0.25
	sWeightCollegeShare  = 0.40
	sWeightAgeTrajectory = 0.15

	// sSL019Strength is the S RAS-modulator strength (S_Rubric §1/§4) — 0.30, MATCHING CB (S is
	// structurally parallel to CB: same peak, tier, and gating criterion, so the modulator
	// strengths are held at CB values for consistency, §7). Applied to BOTH the breakout-age and
	// age-trajectory sub-signals (symmetric, SL-OQ-024). Passed to the shared curve.SL019 helper —
	// never re-implemented. (The L3 SL-019 buffer is the separate 0.25 strength, a Layer-3
	// mechanic not built here — carry-forward.)
	sSL019Strength = 0.30
)

// S is the safety Layer-4 rubric. Combined = film × RAS × breakout
// (Backend_Architecture:256). Film is Data-Parity neutral in v1.0 (coverage film weights
// UNSET), so safeties differentiate on the ACTIVE High-tier RAS component and the breakout
// composite — where SL-019 (at the reduced 0.30 strength) makes the athletic profile lift the
// breakout-age and age-trajectory sub-signals (S_Rubric §4). The Hamilton/Mathieu finding (§5):
// an elite-RAS aging safety earns MORE Layer-4 age-trajectory protection than a mid-RAS aging
// peer — the eighth Lockett-pattern instance, amplified by SL-019 but scaled to the profile
// (Mathieu's RAS 4.5 earns a 0.135 age-trajectory lift vs. Peterson's 0.279 at CB with RAS 9.3).
//
// Curves are held on the struct (not package globals — gochecknoglobals) and built once.
type S struct {
	breakoutAge   []curve.Breakpoint
	collegeShare  []curve.Breakpoint
	ageTrajectory []curve.Breakpoint
}

// NewS builds the S rubric with its S_Rubric §4 normalization curves. School tier uses the
// boundary TEMPLATE (no S branch in composition.schoolTierNorm — P4 1.00 / G5 0.70 / FCS 0.40 /
// Non-FCS 0.10) and arrives pre-normalized, so it has no curve here.
func NewS() *S {
	return &S{
		// Breakout Age (years → [0,1]), half-year shaped, IDENTICAL to CB: ≤19.5 → 1.00,
		// 20.5 → 0.75, 21.5 → 0.45, ≥22.5 → 0.15.
		breakoutAge: []curve.Breakpoint{{X: 19.5, Y: 1.00}, {X: 20.5, Y: 0.75}, {X: 21.5, Y: 0.45}, {X: 22.5, Y: 0.15}},
		// College Production Share (final-year INT + TACKLE market share → [0,1]): ≤0.08 → 0.15,
		// 0.14 → 0.55, ≥0.20 → 1.00. Thresholds LOWER than CB (0.08/0.16/0.24) because safeties
		// have lower absolute market-share ceilings — they share tackle volume with LBs and INT
		// volume with CBs (§4/§7).
		collegeShare: []curve.Breakpoint{{X: 0.08, Y: 0.15}, {X: 0.14, Y: 0.55}, {X: 0.20, Y: 1.00}},
		// Age Trajectory (player age → [0,1]) relative to S peak 28 — IDENTICAL to CB. Pulls
		// begin at age 25. ≤24 → 1.00 … 28 (peak) → 0.50 … ≥32 → 0.00. SL-019 then modulates this
		// for high-RAS safeties (no SL-021 cushion).
		ageTrajectory: []curve.Breakpoint{
			{X: 24, Y: 1.00}, {X: 25, Y: 0.85}, {X: 26, Y: 0.70}, {X: 27, Y: 0.55}, {X: 28, Y: 0.50},
			{X: 29, Y: 0.35}, {X: 30, Y: 0.20}, {X: 31, Y: 0.10}, {X: 32, Y: 0.00},
		},
	}
}

// HasNGSAnchor is the case-3I introspection hook: S is the SECOND position (after CB) to elevate
// NFL Next Gen Stats coverage/range metrics to a dedicated film sub-signal anchor (S_Rubric §2 —
// 0.30 weight, §7 "NGS pattern extends to S"). Like DT.PFFAlpha and CB.HasNGSAnchor it is a
// RUBRIC INTERNAL exposed for the harness only; it deliberately does NOT live on Layer4Output
// (the production surface) — the film weights are unset in v1.0, so a weight accessor would read
// 0.00 and NGS-presence is invisible at the engine boundary. CB returns true too; offense and the
// other defensive rubrics (DT/DE/LB) do not implement this method at all, so the harness reads
// them as having no NGS anchor.
func (s *S) HasNGSAnchor() bool { return true }

// Apply computes the S Layer-4 output. Film is the S-curve over the coverage composite, or
// neutral 1.000 when no source is populated (Data-Parity Rule). RAS is ACTIVE, High-tier at
// steepness 10.0: raw RAS / 10 through the S-curve scaled by the rookie weight, or neutral
// 1.000 when absent (never forced — AD-09). Breakout weights the four sub-signals; SL-019 then
// lifts the PRESENT breakout-age and age-trajectory sub-signals by the athletic profile at the
// 0.30 strength (the shared curve.SL019, reused verbatim from CB/DE/TE) — an absent breakout age
// stays neutral (no modulation of unknown data), and an absent RAS contributes no lift (the
// modulator's Data-Parity stance).
func (s *S) Apply(in engine.Layer4Input) engine.Layer4Output {
	sc := in.Scouting
	p := in.Player

	filmRaw := curve.NeutralNorm // Data-Parity neutral default (coverage film weights unset)
	film := 1.0
	if sc.HasFilm {
		filmRaw = sc.FilmComposite
		film = curve.Scurve(sc.FilmComposite, sFilmInflection, sFilmSteepness, sFilmCap)
	}

	// RAS component (Data-Parity neutral when absent). The rookie position weight (1.00) scales
	// the full High-tier S-curve deviation from neutral; a later receding weight would pull it
	// toward 1.000 as NFL data accrues.
	rasEffective := 1.0
	if p.HasRAS {
		rasCurve := curve.Scurve(p.RAS/10.0, sRASInflection, sRASSteepness, sRASCap)
		rasEffective = 1.0 + sRASRookieWeight*(rasCurve-1.0)
	}

	// SL-019 RAS-modulator interactions (S_Rubric §4) — the FOURTH instance, reusing the shared
	// curve.SL019 helper verbatim at the 0.30 strength. RAS_normalized = raw RAS / 10; the helper
	// neutralizes an absent/out-of-range RAS. Modulation is applied ONLY to PRESENT sub-signals:
	// an absent breakout age stays neutral (modulating the neutral would manufacture a lift for
	// unknown data — a Data-Parity violation). Age trajectory is always present.
	rasNorm := p.RAS / 10.0
	breakoutAgeNorm := curve.SubSignal(sc.HasBreakoutAge, s.breakoutAge, sc.BreakoutAge)
	if sc.HasBreakoutAge {
		breakoutAgeNorm = curve.SL019(breakoutAgeNorm, rasNorm, sSL019Strength, p.HasRAS)
	}
	ageTrajectoryNorm := curve.SL019(curve.Interp(s.ageTrajectory, p.Age), rasNorm, sSL019Strength, p.HasRAS)

	composite := sWeightBreakoutAge*breakoutAgeNorm +
		sWeightSchoolTier*curve.Present(sc.HasSchoolTier, sc.SchoolTierNorm) +
		sWeightCollegeShare*curve.SubSignal(sc.HasCollegeShare, s.collegeShare, sc.CollegeShare) +
		sWeightAgeTrajectory*ageTrajectoryNorm
	breakout := curve.Scurve(composite, sBreakoutInflection, sBreakoutSteepness, sBreakoutCap)

	return engine.Layer4Output{
		FilmEffective:     film,
		FilmRaw:           filmRaw,
		RASEffective:      rasEffective,
		BreakoutEffective: breakout,
		Combined:          film * rasEffective * breakout,
	}
}
