package defense

import (
	"github.com/secureprospective/TheWarRoom/internal/engine"
	"github.com/secureprospective/TheWarRoom/internal/engine/l4/curve"
)

// LB Layer-4 mechanics (docs/scoring-engine/LB_Rubric.md, locked v1.0). Rubric constants —
// never admin-exposed (Hard Constraint). LB is the FIRST non-SL-019 IDP rubric: it proves the
// architecture COMPOSES by reusing two already-shipped idioms with NO new engine code — DT's
// SL-005 film compression (cap ±3%, steepness 10.0) AND RB's Medium-tier RAS — while applying
// NEITHER SL-019 (the TE/DE modulator) NOR SL-021 (the DT cushion). The exclusions are
// principled (LB_Rubric §1/§7): SL-019 needs High-tier RAS + a clean athletic→longevity signal,
// and LB is Medium-tier with scheme-driven longevity (a Cover-2 LB becomes a 3-4 ILB and loses
// his RAS-aligned role); RAS is therefore NOT elevated to compensate for the thin IDP film
// signal — inflating it would manufacture false confidence in athletic-profile-strong but
// scheme-mismatched players at the most scheme-dependent dynasty position. Per OQ-004 this rubric
// scores OFF-BALL linebackers only; pass-rush-primary EDGE defenders (DE/EDGE/3-4 OLB by role)
// route through the DE rubric regardless of MFL tag (the DE-vs-LB dispatch is composition's
// resolveRubric, harness case 3J).
const (
	// Film component (LB_Rubric §2): S-curve over the IDP film composite, SL-005-compressed.
	// IDENTICAL to DT's film block — compression is expressed via the cap (±3%) and steepness
	// (10.0), NOT a position_weight (held at 1.00, reserved for the SL-018 schedule). FILM
	// Thread C (C-4 step 2) wired the composite onto the Madden defense sub-attribute composite
	// (K1) — LB's curated set includes zone coverage (its rubric names no man coverage), blended
	// UPSTREAM at 0.95 of the LB film budget (rankings.applyScouting) into the single [0,1]
	// FilmComposite this rubric consumes unchanged; an unresolved Madden record → neutral 1.000.
	lbFilmInflection = 0.50
	lbFilmSteepness  = 10.0 // SL-005 compression — vs standard 12.0 (matches DT)
	lbFilmCap        = 0.03 // ±3% (SL-005) — vs standard ±5% (matches DT)

	// RAS component (LB_Rubric §3): ACTIVE (no SL-020), MEDIUM tier — NOT elevated by SL-005.
	// raw RAS / 10 through the S-curve scaled by the position weight. Cap ±4% (Medium standard,
	// same band as RB) but at steepness 11.0 (NOT RB's 8.0 — LB_Rubric §3).
	lbRASInflection = 0.50
	lbRASSteepness  = 11.0
	lbRASCap        = 0.04 // ±4% (Medium-tier, SL-004) — NOT elevated under SL-005
	// RAS_position_weight follows the Medium-tier SL-018 schedule (rookie 0.60 → 0.30 → 0.06).
	// v1.0 ships the ROOKIE weight only — the receding schedule needs an NFL-career-stage input
	// no consumer yet supplies (the same "version the boundary" deferral as every prior position;
	// RB was the first 0.60-rookie position).
	lbRASRookieWeight = 0.60

	// Breakout component (LB_Rubric §4): standard ±5% cap (NOT compressed), steepness 11.0.
	lbBreakoutInflection = 0.50
	lbBreakoutSteepness  = 11.0
	lbBreakoutCap        = 0.05

	// Breakout sub-signal weights (LB_Rubric §4) — sum to 1.00. College Production Share
	// (Tackle + Sack + TFL market share) elevated to 0.40 — the HIGHEST of any position; its
	// richness compensates for the position's thin pre-draft scouting depth. Breakout age dropped
	// to 0.25 (LBs break out later — junior-year emergence is normative, not exceptional).
	lbWeightBreakoutAge   = 0.25
	lbWeightSchoolTier    = 0.20
	lbWeightCollegeShare  = 0.40
	lbWeightAgeTrajectory = 0.15
)

// LB is the off-ball linebacker Layer-4 rubric. Combined = film × RAS × breakout
// (Backend_Architecture:256). Film is Data-Parity neutral in v1.0 (IDP film weights UNSET) and,
// even when populated, SL-005-compressed so it cannot drive Layer 4 hard. RAS is ACTIVE but
// Medium-tier and residual at veteran stages; LBs therefore differentiate primarily on the
// breakout composite — where College Production Share (0.40) is dominant. NO SL-019 modulators
// (Medium tier + scheme-driven longevity) and NO SL-021 cushion (DT-specific). The Warner/David
// finding (§5): at LB, Layer 4 alone cannot meaningfully pull aging veterans below 1.0 — strong
// static breakout signals hold and SL-005 mutes the film decline, so the deterministic Layer 3
// age decay carries the aging work (the Lockett pattern, amplified by SL-005).
//
// Curves are held on the struct (not package globals — gochecknoglobals) and built once.
type LB struct {
	breakoutAge   []curve.Breakpoint
	collegeShare  []curve.Breakpoint
	ageTrajectory []curve.Breakpoint
}

// NewLB builds the LB rubric with its LB_Rubric §4 normalization curves. School tier uses the
// boundary TEMPLATE (no LB branch in composition.schoolTierNorm — P4 1.00 / G5 0.70 / FCS
// 0.40 / Non-FCS 0.10) and arrives pre-normalized, so it has no curve here.
func NewLB() *LB {
	return &LB{
		// Breakout Age (years → [0,1]), DT-shaped but recognizing the later-emergence norm:
		// ≤20.0 → 1.00, 21.0 → 0.75, 22.0 → 0.45, ≥23.0 → 0.15.
		breakoutAge: []curve.Breakpoint{{X: 20, Y: 1.00}, {X: 21, Y: 0.75}, {X: 22, Y: 0.45}, {X: 23, Y: 0.15}},
		// College Production Share (final-year Tackle+Sack+TFL market share → [0,1]): ≤0.10 →
		// 0.15, 0.18 → 0.55, ≥0.25 → 1.00.
		collegeShare: []curve.Breakpoint{{X: 0.10, Y: 0.15}, {X: 0.18, Y: 0.55}, {X: 0.25, Y: 1.00}},
		// Age Trajectory (player age → [0,1]) relative to LB peak 29. ≤25 → 1.00 … 29 (peak) →
		// 0.50 … ≥33 → 0.00. NO SL-019 modulation and NO SL-021 cushion — the curve declines
		// unmodulated (the L3 age_pull carries the buffered aging work, not L4).
		ageTrajectory: []curve.Breakpoint{
			{X: 25, Y: 1.00}, {X: 26, Y: 0.85}, {X: 27, Y: 0.70}, {X: 28, Y: 0.55}, {X: 29, Y: 0.50},
			{X: 30, Y: 0.35}, {X: 31, Y: 0.20}, {X: 32, Y: 0.10}, {X: 33, Y: 0.00},
		},
	}
}

// Apply computes the LB Layer-4 output. Film is the SL-005-compressed S-curve over the IDP
// composite, or neutral 1.000 when no source is populated (Data-Parity Rule). RAS is ACTIVE,
// Medium-tier: raw RAS / 10 through the S-curve scaled by the rookie weight (0.60), or neutral
// 1.000 when absent (never forced — AD-09). Breakout weights the four sub-signals with NO SL-019
// modulation (each sub-signal uses its BASE curve) — LB is excluded from the modulator by the
// SL-OQ-027 gating rule (Medium tier + scheme-driven longevity).
func (lb *LB) Apply(in engine.Layer4Input) engine.Layer4Output {
	sc := in.Scouting
	p := in.Player

	filmRaw := curve.NeutralNorm // Data-Parity neutral default (IDP film weights unset)
	film := 1.0
	if sc.HasFilm {
		filmRaw = sc.FilmComposite
		film = curve.Scurve(sc.FilmComposite, lbFilmInflection, lbFilmSteepness, lbFilmCap)
	}

	// RAS component (Data-Parity neutral when absent). The rookie position weight (0.60) scales
	// the Medium-tier S-curve deviation from neutral; a later receding weight would pull it
	// further toward 1.000 as NFL data accrues. RAS is NOT elevated under SL-005 (§3).
	rasEffective := 1.0
	if p.HasRAS {
		rasCurve := curve.Scurve(p.RAS/10.0, lbRASInflection, lbRASSteepness, lbRASCap)
		rasEffective = 1.0 + lbRASRookieWeight*(rasCurve-1.0)
	}

	// Breakout composite — BASE curves, no SL-019 (excluded at LB). Age trajectory always has a
	// value (age is known) and declines unmodulated past peak (no SL-021 cushion at LB).
	composite := lbWeightBreakoutAge*curve.SubSignal(sc.HasBreakoutAge, lb.breakoutAge, sc.BreakoutAge) +
		lbWeightSchoolTier*curve.Present(sc.HasSchoolTier, sc.SchoolTierNorm) +
		lbWeightCollegeShare*curve.SubSignal(sc.HasCollegeShare, lb.collegeShare, sc.CollegeShare) +
		lbWeightAgeTrajectory*curve.Interp(lb.ageTrajectory, p.Age)
	breakout := curve.Scurve(composite, lbBreakoutInflection, lbBreakoutSteepness, lbBreakoutCap)

	return engine.Layer4Output{
		FilmEffective:     film,
		FilmRaw:           filmRaw,
		RASEffective:      rasEffective,
		BreakoutEffective: breakout,
		Combined:          film * rasEffective * breakout,
	}
}
