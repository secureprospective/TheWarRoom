// Package kicker holds the B5b kicker Layer-4 rubric. K is the LAST L4 build and the one
// position that does NOT mirror an offense/defense rubric: per DECISION-011 (Scouting_Schema.md,
// authoritative — K_Rubric.md §2/§3/§4 is STALE and describes the superseded exclusion model)
// K's FILM component is ACTIVE and Madden-majority, while RAS and breakout are forced to exactly
// 1.000. So Combined = film × 1.000 × 1.000 = film. This reverses AD-10: K's combine no longer
// yields a flat 1.000 — film now moves K's valuation.
//
// PURITY (depguard engine-is-pure, **/internal/engine/**): this package imports only the engine
// (for the Layer4 contract types) and the curve package — no store, db, ingestion, or I/O. Every
// input arrives as a parameter on Layer4Input.
package kicker

import (
	"github.com/secureprospective/TheWarRoom/internal/engine"
	"github.com/secureprospective/TheWarRoom/internal/engine/l4/curve"
)

// K Layer-4 mechanics. The sub-signal split is PINNED by DECISION-011; the film S-curve
// shape (cap/steepness/inflection) was UNSPECIFIED in any doc and is pinned here by
// Christopher's B5b-K gate-check: a CONSERVATIVE ±3% cap (K is sparse-data — film may move
// the valuation only modestly), steepness 10.0, inflection 0.50. These are rubric constants,
// never admin-exposed (Hard Constraint: Layer 4 structural mechanics never appear in the UI).
const (
	// Film S-curve over the Madden/NFLProduction composite (DECISION-011 + gate-check).
	kFilmInflection = 0.50
	kFilmSteepness  = 10.0
	kFilmCap        = 0.03 // ±3% — conservative; K is sparse-data

	// Film sub-signal weights (DECISION-011, PINNED) — sum to 1.00. Madden is the majority.
	kWeightMadden        = 0.60
	kWeightNFLProduction = 0.40
)

// K is the kicker Layer-4 rubric. Film is the only active component: a Madden 0.60 /
// NFLProduction 0.40 composite passed through the film S-curve. RAS is forced to exactly
// 1.000 (SL-020 partial — like QB; RAS is unread, used only at Layer 6) and breakout is
// forced to exactly 1.000 (there is no breakout framework at K). SL-019, SL-021 and the
// NGS anchor do NOT apply. K therefore needs no normalization curves — its two sub-signals
// arrive already normalized to [0,1] from the boundary — so the struct is empty; NewK keeps
// the construction shape uniform with the other rubrics.
type K struct{}

// NewK builds the kicker rubric. It holds no curves (K's sub-signals arrive normalized), so
// there is nothing to construct; the constructor exists for registry uniformity.
func NewK() *K { return &K{} }

// Apply computes the K Layer-4 output: Combined = film × 1.000 × 1.000 = film.
//
// Film is the S-curve over the Madden 0.60 / NFLProduction 0.40 composite. Each component is
// Data-Parity-neutral when ABSENT (curve.Present substitutes the neutral 0.50). When BOTH are
// absent the film component short-circuits to exactly 1.000 — equivalently, the composite would
// be the neutral 0.50, whose S-curve is exactly 1.000 — so a missing film source is never a
// penalty (DECISION-011 / the Data-Parity Rule). RAS and breakout are
// unbypassable 1.000 consts; the active-film path cannot leak into them (this is what keeps
// Module-3 case 3C green — K's RAS stays exactly 1.000). FilmRaw carries the pre-effective
// normalized composite for the debug surface (never UI).
func (k *K) Apply(in engine.Layer4Input) engine.Layer4Output {
	sc := in.Scouting

	filmRaw := curve.NeutralNorm // Data-Parity neutral default (no film source populated)
	film := 1.0
	if sc.HasMaddenFilm || sc.HasNFLProduction {
		composite := kWeightMadden*curve.Present(sc.HasMaddenFilm, sc.MaddenFilm) +
			kWeightNFLProduction*curve.Present(sc.HasNFLProduction, sc.NFLProduction)
		filmRaw = composite
		film = curve.Scurve(composite, kFilmInflection, kFilmSteepness, kFilmCap)
	}

	const rasEffective = 1.0      // SL-020: RAS forced to exactly 1.000 at K (unread)
	const breakoutEffective = 1.0 // no breakout framework at K

	return engine.Layer4Output{
		FilmEffective:     film,
		FilmRaw:           filmRaw,
		RASEffective:      rasEffective,
		BreakoutEffective: breakoutEffective,
		Combined:          film * rasEffective * breakoutEffective,
	}
}
