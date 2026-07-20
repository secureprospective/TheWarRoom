// Pure RAS-equivalent normalization math, isolated from the network fetch so it
// can be unit-tested directly with fixture RawCombine maps. This is the
// PROVISIONAL v1 RAS method (Christopher-approved 2026-07-20 — S-Phase-0 brief
// §3). The real z-score calibration is a SEPARATE, deferred, decision-gated
// pass; implement it EXACTLY as written here, do not deviate.
//
// THE METHOD (pinned):
//
//  1. Cohort = resolved position. All players at a position with ≥1 measurable
//     present form the cohort for every per-measurable stat.
//  2. Per measurable, per cohort: mean μ and SAMPLE stddev σ (n−1) over PRESENT
//     values only. ABSENT IS ABSENT — a *float64 nil is "did not perform", not 0.
//  3. Sign convention (higher z = better athlete):
//     - time drills (forty, cone, shuttle): z = (μ − x) / σ  (lower time is better)
//     - everything else (height, weight, bench, vertical, broad): z = (x − μ) / σ
//  4. Per player: RAS_z = mean of that player's AVAILABLE measurable z-scores
//     (averaged only over measurables the player actually has).
//  5. Scale to 0–10: RAS = clamp(5.0 + 2.0·RAS_z, 0, 10).
//     +2.5σ → 10.0 (rail); cohort-average → 5.0 (== neutral fallback); −2.5σ → 0.0.
//  6. Guards (no NaN/Inf may ever reach Profile.RAS):
//     - σ == 0 or cohort n < 2 for a measurable → that measurable contributes
//       z = 0 (neutral) for every player who has it present.
//     - A player with zero measurables present → HasRAS = false (absent from the
//       map entirely; L1 imputes DefaultRASFallback = 5.0 downstream).
//     - If ALL of a player's available measurables resolve to z = 0 → RAS_z = 0
//       → RAS = 5.0, HasRAS = true (present-but-neutral, distinct from absent).
//
// DETERMINISM: the math is order-independent (mean, stddev, average), so map
// iteration order cannot change the output. No NaN/Inf is ever produced.

package assembly

import (
	"math"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/ras"
)

// measurables returns the fixed, ordered set of combine measurables scored.
// The order is deterministic but mathematically irrelevant (the per-player
// z-mean is order-independent); listed once so the extractor and the sign
// table share one source of truth. A func (not a package var) keeps the
// package free of mutable globals (gochecknoglobals) while still constructing
// the slice once per caller — cheap relative to the network fetch that
// produced the inputs.
func measurables() []measurableDef {
	return []measurableDef{
		{name: "height", get: func(r ras.RawCombine) *float64 { return r.HeightIn }, timeDrill: false},
		{name: "weight", get: func(r ras.RawCombine) *float64 { return r.WeightLb }, timeDrill: false},
		{name: "forty", get: func(r ras.RawCombine) *float64 { return r.Forty }, timeDrill: true},
		{name: "bench", get: func(r ras.RawCombine) *float64 { return r.Bench }, timeDrill: false},
		{name: "vertical", get: func(r ras.RawCombine) *float64 { return r.Vertical }, timeDrill: false},
		{name: "broad_jump", get: func(r ras.RawCombine) *float64 { return r.BroadJump }, timeDrill: false},
		{name: "cone", get: func(r ras.RawCombine) *float64 { return r.Cone }, timeDrill: true},
		{name: "shuttle", get: func(r ras.RawCombine) *float64 { return r.Shuttle }, timeDrill: true},
	}
}

// measurableDef pairs a measurable extractor with its sign convention. A time
// drill (lower-is-better) flips the z-score: z = (μ − x) / σ.
type measurableDef struct {
	name      string
	get       func(ras.RawCombine) *float64
	timeDrill bool
}

// stat is the running tally for one (cohort, measurable) during pass 1 of
// scoreRAS. Unexported, package-local — kept as a named type so the helper
// signatures read cleanly.
type stat struct {
	sum, sumSq float64
	n          int
}

// scoreRAS is the pure RAS-equivalent pass: given a player's RawCombine keyed
// by gsis and that gsis's resolved position, return the gsis→RAS map. A gsis is
// in the result iff it had ≥1 measurable present AND a resolved position (the
// cohort it scores against); absent players are absent from the map (HasRAS =
// false downstream). No error path: the inputs are already typed/validated;
// the math is total (guards convert every degenerate case to z=0).
func scoreRAS(rawByGSIS map[string]ras.RawCombine, posByGSIS map[string]domain.Position) map[string]float64 {
	moms := computeMoments(rawByGSIS, posByGSIS)
	return applyZ(rawByGSIS, posByGSIS, moms)
}

// computeMoments is pass 1: per (position, measurable) accumulate mean μ and
// sample stddev σ (n−1 denominator) over the PRESENT values only, then resolve
// the σ=0 / n<2 guard. The guard converts a degenerate measurable to z=0 for
// every player who has it present — never NaN/Inf.
func computeMoments(rawByGSIS map[string]ras.RawCombine, posByGSIS map[string]domain.Position) map[domain.Position]map[int]moments {
	cohort := make(map[domain.Position]map[int]*stat, len(posByGSIS))
	for gsis, rc := range rawByGSIS {
		pos, ok := posByGSIS[gsis]
		if !ok {
			continue // no resolved position → cannot cohort; ordinary miss
		}
		byMeas, ok := cohort[pos]
		if !ok {
			byMeas = make(map[int]*stat, len(measurables()))
			cohort[pos] = byMeas
		}
		accumulate(byMeas, rc)
	}
	out := make(map[domain.Position]map[int]moments, len(cohort))
	for pos, byMeas := range cohort {
		out[pos] = momentsForCohort(byMeas)
	}
	return out
}

// accumulate folds one player's RawCombine into the per-cohort per-measurable
// running tallies. ABSENT measurables (nil pointers) are skipped — the cohort
// is "players at the position who performed THIS drill".
func accumulate(byMeas map[int]*stat, rc ras.RawCombine) {
	for i, m := range measurables() {
		v := m.get(rc)
		if v == nil {
			continue
		}
		s, ok := byMeas[i]
		if !ok {
			s = &stat{}
			byMeas[i] = s
		}
		s.sum += *v
		s.sumSq += (*v) * (*v)
		s.n++
	}
}

// momentsForCohort resolves the running tallies for one cohort into per-
// measurable moments (μ, σ, guarded). Sample stddev uses the n−1 denominator;
// n<2 or σ=0 fires the guard.
func momentsForCohort(byMeas map[int]*stat) map[int]moments {
	m := make(map[int]moments, len(byMeas))
	for i, s := range byMeas {
		if s.n < 2 {
			// n<2 → sample stddev is undefined; neutralize. (n==1 means
			// exactly one player at this position performed the drill; that
			// single observation gives no comparative signal.)
			m[i] = moments{guarded: true}
			continue
		}
		mu := s.sum / float64(s.n)
		// Sample variance (n−1 denominator). sumSq/n − mu² is the population
		// variance; ×n/(n−1) converts to sample.
		popVar := s.sumSq/float64(s.n) - mu*mu
		if popVar <= 0 {
			// All present values identical (or float drift to ≤0). σ = 0
			// carries no comparative signal — neutralize.
			m[i] = moments{mu: mu, guarded: true}
			continue
		}
		sigma := math.Sqrt(popVar * float64(s.n) / float64(s.n-1))
		m[i] = moments{mu: mu, sigma: sigma}
	}
	return m
}

// applyZ is pass 2: per player, average the z-scores of AVAILABLE measurables
// (only those the player has present) and scale to 0–10. A player with zero
// measurables present is absent from the result (HasRAS false downstream).
func applyZ(rawByGSIS map[string]ras.RawCombine, posByGSIS map[string]domain.Position, moms map[domain.Position]map[int]moments) map[string]float64 {
	out := make(map[string]float64, len(rawByGSIS))
	ms := measurables()
	for gsis, rc := range rawByGSIS {
		pos, ok := posByGSIS[gsis]
		if !ok {
			continue
		}
		mom := moms[pos]
		var zSum float64
		var zCount int
		for i, m := range ms {
			v := m.get(rc)
			if v == nil {
				continue // absent for this player — not averaged
			}
			zCount++
			mo, ok := mom[i]
			if !ok || mo.guarded {
				continue // z = 0 (neutral); still counts as an available measurable
			}
			z := (*v - mo.mu) / mo.sigma
			if m.timeDrill {
				z = (mo.mu - *v) / mo.sigma
			}
			zSum += z
		}
		if zCount == 0 {
			continue // zero measurables present → absent from map (HasRAS false)
		}
		out[gsis] = clampRAS(5.0 + 2.0*(zSum/float64(zCount)))
	}
	return out
}

// moments is the per-(cohort, measurable) normalization parameters after the
// guards fire. guarded=true means every present z is forced to 0 (σ=0 or n<2).
type moments struct {
	mu, sigma float64
	guarded   bool
}

// clampRAS pins the scaled RAS to the engine's [0, 10] rail. A non-finite
// input (impossible by construction — guards killed every NaN/Inf path above)
// would still be rejected by the composition boundary; we hold the line here
// too so Profile.RAS can never carry one. The rail math: 5.0 + 2.0·(±2.5) →
// {10.0, 0.0}; anything outside is clamped to the nearer endpoint.
func clampRAS(v float64) float64 {
	if v < 0 || math.IsNaN(v) {
		return 0
	}
	if v > 10 || math.IsInf(v, 0) {
		return 10
	}
	return v
}
