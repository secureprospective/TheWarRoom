package harness

// This file holds the B5b-LB close-gate evals that are CROSS-POSITION (3D — SL-005 film
// compression compared across LB/DT/WR) or POSITION-RESOLUTION (3J — EDGE classification routing
// at the composition boundary), as opposed to the single-rubric mechanic evals in cases_eval.go.
// It was split out of cases_eval.go on the B5b-LB build to keep both files under the 400-line cap
// (AD-17), following the cases.go → cases_eval.go split from B5b-DE.

import (
	"fmt"
	"math"

	"github.com/secureprospective/TheWarRoom/internal/composition"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine"
)

// eval3D is the B5b-LB close gate (SL-005 film compression, cross-position). It asserts the
// architectural effect the DT/LB rubrics share: at the two SL-005 positions (DT, LB) an ELITE
// film composite is compressed to the ±3% band and lands STRICTLY BELOW where the SAME composite
// lands at a standard ±5% position (WR is the registered control). It runs the SAME elite
// composite (1.0) through the three REGISTERED rubrics and reads FilmEffective. LB and DT must
// agree (they ship the identical SL-005 block — cap ±3%, steepness 10.0), and both must fall
// under WR's ±5%. WR's dt_test/lb_test unit checks are the per-position template; this proves the
// COMPRESSION RELATIONSHIP through the real registry. RAS is absent (film is isolated).
func eval3D(reg RubricRegistry) (CaseState, string) {
	if st, why, ok := requireRubrics(reg, domain.PosLB, domain.PosDT, domain.PosWR); !ok {
		return st, why
	}
	const elite = 1.0
	film := func(pos domain.Position) float64 {
		return reg[pos].Apply(engine.Layer4Input{
			Player:   engine.PlayerInput{Position: pos, Age: 24, HasRAS: false},
			Scouting: engine.ScoutingInput{FilmComposite: elite, HasFilm: true},
		}).FilmEffective
	}
	lb, dt, wr := film(domain.PosLB), film(domain.PosDT), film(domain.PosWR)
	for _, c := range []struct {
		name string
		v    float64
	}{{"LB", lb}, {"DT", dt}} {
		if c.v <= 1.0 || c.v > 1.03+1e-9 {
			return StateFail, fmt.Sprintf("%s film %.5f must be in the SL-005 band (1.0, 1.03]", c.name, c.v)
		}
	}
	if math.Abs(lb-dt) > 1e-9 {
		return StateFail, fmt.Sprintf("LB film %.5f and DT film %.5f must match (same SL-005 ±3%% block)", lb, dt)
	}
	if !(wr > lb) {
		return StateFail, fmt.Sprintf("WR film %.5f (±5%%) must exceed the SL-005-compressed LB/DT film %.5f (±3%%)", wr, lb)
	}
	return StatePass, fmt.Sprintf("SL-005: LB & DT film=%.5f (±3%% compressed, equal) < WR film=%.5f (±5%% standard)", lb, wr)
}

// eval3J is the B5b-LB close gate for EDGE classification routing (OQ-004 / SL-OQ-030, test 3J).
// It exercises the composition.ResolveRubricPosition resolver on the Testing_App_Specification §3J
// fixtures: an EDGE tag (normalized to DE) with high pass-rush share routes to DE, the SAME tag
// with low share routes to LB, and an LB tag with high share routes to DE (the role overrides the
// MFL tag). It gates on BOTH DE and LB being registered, and asserts every routed position lands
// on a REGISTERED rubric (a route to a missing rubric is a real FAIL). The negative case (no
// share → MFL tag stands) is checked too. This is a position-RESOLUTION concern at the boundary,
// not a rubric mechanic — so it asserts on the resolver the live RankRookies path uses.
func eval3J(reg RubricRegistry) (CaseState, string) {
	if st, why, ok := requireRubrics(reg, domain.PosDE, domain.PosLB); !ok {
		return st, why
	}
	for _, c := range []struct {
		name string
		spec composition.PlayerSpec
		want domain.Position
	}{
		{"EDGE tag, 75% pass-rush → DE", composition.PlayerSpec{Position: domain.PosDE, PassRushSnapShare: 0.75, HasPassRushSnapShare: true}, domain.PosDE},
		{"EDGE tag, 25% pass-rush → LB", composition.PlayerSpec{Position: domain.PosDE, PassRushSnapShare: 0.25, HasPassRushSnapShare: true}, domain.PosLB},
		{"LB tag, 80% pass-rush → DE", composition.PlayerSpec{Position: domain.PosLB, PassRushSnapShare: 0.80, HasPassRushSnapShare: true}, domain.PosDE},
		{"LB tag, 25% pass-rush → LB", composition.PlayerSpec{Position: domain.PosLB, PassRushSnapShare: 0.25, HasPassRushSnapShare: true}, domain.PosLB},
		// Exactly 0.50 pins the spec's ≥ boundary (a coin-flip is pass-rush primary → DE); a
		// future regression to a strict > would flip this to LB and fail here.
		{"LB tag, exactly 50% pass-rush → DE", composition.PlayerSpec{Position: domain.PosLB, PassRushSnapShare: 0.50, HasPassRushSnapShare: true}, domain.PosDE},
	} {
		got := composition.ResolveRubricPosition(c.spec)
		if got != c.want {
			return StateFail, fmt.Sprintf("%s: routed to %s, want %s", c.name, got, c.want)
		}
		if _, ok := reg[got]; !ok {
			return StateFail, fmt.Sprintf("%s: routed to unregistered rubric %s", c.name, got)
		}
	}
	// Negative contract: a defender with NO pass-rush share keeps his MFL tag (the resolver
	// must not silently re-route a player whose role is unknown).
	if got := composition.ResolveRubricPosition(composition.PlayerSpec{Position: domain.PosLB}); got != domain.PosLB {
		return StateFail, fmt.Sprintf("share-less LB must stay LB, got %s", got)
	}
	return StatePass, "EDGE routing: 75%→DE, 25%→LB, LB@80%→DE, LB@50%→DE (≥ boundary, role overrides tag); share-less tag unchanged"
}

// ngsAnchorRubric is the case-3I introspection contract: a rubric whose film composite carries a
// dedicated NFL Next Gen Stats coverage anchor (CB_Rubric §2/§7) reports it via this method. Only
// CB (and S, once built) implement it; every other rubric does not, so the type assertion fails
// and the harness reads them as having no anchor. It mirrors the DT.PFFAlpha pattern — a rubric
// INTERNAL the harness inspects, never a Layer4Output (production-surface) field.
type ngsAnchorRubric interface{ HasNGSAnchor() bool }

func reportsNGSAnchor(r engine.Layer4) bool {
	n, ok := r.(ngsAnchorRubric)
	return ok && n.HasNGSAnchor()
}

// eval3I is the B5b-CB close gate: the NGS coverage anchor is present ONLY at the NGS-coverage
// positions (CB, and S once registered) and ABSENT everywhere else. The film weights are unset in
// v1.0, so NGS-presence is invisible at the engine boundary (Layer4Output); the rubric exposes it
// via the HasNGSAnchor introspection hook instead. The case asserts the property PER-REGISTERED-
// POSITION (like eval3A asserts on both WR and QB): it gates on CB being registered, asserts every
// NGS position present reports the anchor, and asserts a non-NGS control (WR) does NOT — so CB
// alone flips it green and S strengthens it without being required (3I no longer waits on S).
func eval3I(reg RubricRegistry) (CaseState, string) {
	if st, why, ok := requireRubrics(reg, domain.PosCB); !ok {
		return st, why
	}
	for _, pos := range []domain.Position{domain.PosCB, domain.PosS} {
		r, ok := reg[pos]
		if !ok {
			continue // S may not be registered yet — assert only what is present
		}
		if !reportsNGSAnchor(r) {
			return StateFail, fmt.Sprintf("%s must report an NGS coverage anchor (CB_Rubric §2/§7)", pos)
		}
	}
	// Negative contract: a non-coverage position must NOT report an NGS anchor (NGS is CB/S-only,
	// CB_Rubric §7). WR is the registered control — if it ever grew the method this would FAIL.
	if wr, ok := reg[domain.PosWR]; ok && reportsNGSAnchor(wr) {
		return StateFail, "WR must NOT report an NGS coverage anchor (NGS is CB/S-only)"
	}
	return StatePass, "NGS coverage anchor present at CB (and S when registered), absent at WR"
}
