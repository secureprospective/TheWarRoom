package harness

import (
	"fmt"
	"math"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine"
	"github.com/secureprospective/TheWarRoom/internal/schema"
)

// eval3C is the fully-wired exemplar that auto-flips when a QB/K rubric registers. SL-020
// forces Layer-4 RAS to exactly 1.000 for QB and K regardless of the athlete's RAS, so a
// very-low (0.10) and an elite (9.99) QB RAS must both yield RASEffective == 1.0000. For K
// (B5b-K, whose film is now ACTIVE per DECISION-011) it asserts both halves: Combined is
// exactly 1.0000 when no film is present (Data-Parity), AND with film populated the active
// film does not leak into RAS/breakout (both stay exactly 1.000, Combined == film). RAS rides
// in on PlayerInput, so this needs no new Layer4Input field — only the rubric.
func eval3C(reg RubricRegistry) (CaseState, string) {
	if st, why, ok := requireRubrics(reg, domain.PosQB, domain.PosK); !ok {
		return st, why
	}
	qb := reg[domain.PosQB]
	for _, ras := range []float64{0.10, 9.99} {
		out := qb.Apply(engine.Layer4Input{Player: engine.PlayerInput{
			Position: domain.PosQB, RAS: ras, HasRAS: true,
		}})
		if out.RASEffective != 1.0 {
			return StateFail, fmt.Sprintf("QB RAS %.2f → RASEffective %.4f, want exactly 1.0000", ras, out.RASEffective)
		}
	}
	// Empty input: K's film is Data-Parity neutral (no Madden/NFLProduction), so all three
	// components are 1.000 and Combined is exactly 1.000.
	kout := reg[domain.PosK].Apply(engine.Layer4Input{Player: engine.PlayerInput{Position: domain.PosK}})
	if kout.Combined != 1.0 {
		return StateFail, fmt.Sprintf("K Combined %.4f, want exactly 1.0000", kout.Combined)
	}
	// Populated film: the real B5b-K rubric's active film must NOT leak into RAS or breakout —
	// both stay forced to exactly 1.000 (SL-020 partial / no breakout framework), and Combined
	// equals the film component alone. This keeps 3C strong now that K has an active film path
	// (the placeholder identity rubric could not exercise it).
	kfilm := reg[domain.PosK].Apply(engine.Layer4Input{
		Player:   engine.PlayerInput{Position: domain.PosK},
		Scouting: engine.ScoutingInput{MaddenFilm: 0.90, HasMaddenFilm: true, NFLProduction: 0.80, HasNFLProduction: true},
	})
	if kfilm.RASEffective != 1.0 {
		return StateFail, fmt.Sprintf("K RASEffective %.4f with film populated, want exactly 1.0000", kfilm.RASEffective)
	}
	if kfilm.BreakoutEffective != 1.0 {
		return StateFail, fmt.Sprintf("K BreakoutEffective %.4f with film populated, want exactly 1.0000", kfilm.BreakoutEffective)
	}
	if kfilm.Combined != kfilm.FilmEffective {
		return StateFail, fmt.Sprintf("K Combined %.6f != FilmEffective %.6f (RAS/breakout must not move it)", kfilm.Combined, kfilm.FilmEffective)
	}
	return StatePass, "QB RASEffective=1.0000 at RAS 0.10/9.99; K RAS+breakout forced 1.0000 with active film; K Combined=1.0000 when film absent"
}

// eval3B is the B5b-RB close gate (the Herbert pattern): Layer 4 PULLS BELOW 1.000 for a
// genuinely-thin-profile vet, while staying above 1.000 for a strong-profile RB. With film
// Data-Parity neutral this session (no offense film source), the sub-1.000 pull is driven by
// the BREAKOUT component — a late breakout age, low college workload, smaller school, and a
// post-peak age push the breakout composite below the 0.50 inflection (RB_Rubric §7). The
// spec's literal Khalil Herbert is a borderline profile whose sub-1.000 result in §5 is
// film-driven; that half reproduces when the film source lands. RAS rides on PlayerInput, so
// this needs no new Layer4Input field — only the registered RB rubric.
func eval3B(reg RubricRegistry) (CaseState, string) {
	if st, why, ok := requireRubrics(reg, domain.PosRB); !ok {
		return st, why
	}
	rb := reg[domain.PosRB]
	// Genuinely-thin vet: breakout age 22 (≥21 → 0.20), Group of Five (0.70), college workload
	// 18% (≤20% → 0.15), age 28 (trajectory 0.05) → breakout composite well below 0.50.
	thin := rb.Apply(engine.Layer4Input{
		Player: engine.PlayerInput{Position: domain.PosRB, Age: 28, RAS: 6.0, HasRAS: true},
		Scouting: engine.ScoutingInput{
			BreakoutAge: 22, HasBreakoutAge: true,
			SchoolTierNorm: 0.75, HasSchoolTier: true, // RB G5 norm (RB_Rubric §4 softer non-P4)
			CollegeShare: 0.18, HasCollegeShare: true,
		},
	})
	// Strong profile (Bijan pattern): true-freshman breakout, Power Four, dominant workload.
	strong := rb.Apply(engine.Layer4Input{
		Player: engine.PlayerInput{Position: domain.PosRB, Age: 24, RAS: 9.55, HasRAS: true},
		Scouting: engine.ScoutingInput{
			BreakoutAge: 19, HasBreakoutAge: true,
			SchoolTierNorm: 1.00, HasSchoolTier: true,
			CollegeShare: 0.53, HasCollegeShare: true,
		},
	})
	if !(thin.Combined < 1.0) {
		return StateFail, fmt.Sprintf("thin-profile RB Combined %.4f, want < 1.0000 (the Herbert pull)", thin.Combined)
	}
	if !(strong.Combined > 1.0) {
		return StateFail, fmt.Sprintf("strong-profile RB Combined %.4f, want > 1.0000", strong.Combined)
	}
	if !(strong.Combined > thin.Combined) {
		return StateFail, fmt.Sprintf("strong RB %.4f should out-score thin RB %.4f", strong.Combined, thin.Combined)
	}
	if thin.FilmEffective != 1.0 {
		return StateFail, fmt.Sprintf("film must be Data-Parity neutral (no source), got %.4f", thin.FilmEffective)
	}
	return StatePass, fmt.Sprintf("thin RB Combined=%.4f (<1, breakout-driven) vs strong RB %.4f (>1); film neutral", thin.Combined, strong.Combined)
}

// eval3F is the B5b-DT close gate (SL-021 Late-Career Cushion Guard). The guard has TWO
// halves and this case closes BOTH (GLM review M4): (1) the L3 decay modulator at the
// engine level — past peak, a DT with raw RAS ≥ 8.00 decays SLOWER than an identical DT
// below threshold, and a sub-threshold RAS leaves the raw pull unchanged; (2) the L4
// breakout Age-Trajectory cushion inside the REGISTERED DT rubric — past peak a qualifying
// DT's breakout component is lifted vs an identical sub-threshold DT. The cushion strength
// is the per-position value the DT rubric ships via Calibration (8.00 / 0.90).
func eval3F(reg RubricRegistry) (CaseState, string) {
	if st, why, ok := requireRubrics(reg, domain.PosDT); !ok {
		return st, why
	}
	// --- Half 1: L3 decay modulator ---
	const peak, rate, age = 30.0, 0.03, 33.0 // three years past the DT peak
	const threshold, decline = 8.00, 0.90
	raw, err := engine.ApplyDecay(age, peak, rate)
	if err != nil {
		return StateFail, fmt.Sprintf("raw decay errored: %v", err)
	}
	cushioned := engine.ApplyCushionGuard(raw, 8.00, true, threshold, decline) // qualifying RAS
	below := engine.ApplyCushionGuard(raw, 7.99, true, threshold, decline)     // just under threshold
	if !(cushioned > raw) {
		return StateFail, fmt.Sprintf("L3: cushion did not slow decay: cushioned %.4f not > raw %.4f", cushioned, raw)
	}
	if below != raw {
		return StateFail, fmt.Sprintf("L3: sub-threshold RAS must not be cushioned: got %.4f, want raw %.4f", below, raw)
	}
	// --- Half 2: L4 breakout Age-Trajectory cushion through the registered DT rubric ---
	dt := reg[domain.PosDT]
	bk := engine.ScoutingInput{BreakoutAge: 22, HasBreakoutAge: true, SchoolTierNorm: 0.70, HasSchoolTier: true, CollegeShare: 0.15, HasCollegeShare: true}
	hi := dt.Apply(engine.Layer4Input{Player: engine.PlayerInput{Position: domain.PosDT, Age: 32, RAS: 9.0, HasRAS: true}, Scouting: bk})
	lo := dt.Apply(engine.Layer4Input{Player: engine.PlayerInput{Position: domain.PosDT, Age: 32, RAS: 7.0, HasRAS: true}, Scouting: bk})
	if !(hi.BreakoutEffective > lo.BreakoutEffective) {
		return StateFail, fmt.Sprintf("L4: age-32 cushioned breakout %.5f not > un-cushioned %.5f", hi.BreakoutEffective, lo.BreakoutEffective)
	}
	return StatePass, fmt.Sprintf("L3 age33 pull %.4f→%.4f at RAS 8.00 (RAS 7.99 stays %.4f); L4 age32 breakout %.5f (RAS 9) > %.5f (RAS 7)",
		raw, cushioned, below, hi.BreakoutEffective, lo.BreakoutEffective)
}

// eval3A is the B5b-WR close gate (the Lockett pattern, WR_Rubric §5 / SL-OQ-017): Layer 4
// does NOT pull a declining elite-draft-profile vet below 1.000 — the deliberate contrast to
// RB's Herbert (3B), which DOES. The mechanism is the breakout component: a declining WR's
// STATIC draft-era sub-signals (early breakout age, Power-Four school, strong college usage)
// reflect what he was at entry and offset the age-trajectory crater (0.00 past age 32), so his
// breakout stays ≥ 1.000 even at age 33; a genuinely-thin declining WR (late breakout, smaller
// school, low usage) has no such offset and his breakout pulls below 1.000. Film is Data-Parity
// neutral this session, and RAS ships rookie-weight-only — so the architectural claim is asserted
// on the BREAKOUT component, the half WR_Rubric §5 attributes the structural finding to (the same
// principled substitution as 3B). 3A co-gates WR AND QB: the structural finding (SL-OQ-017 —
// veteran static breakout signals offset the age crater) is not WR-specific, so the case
// asserts it on BOTH registered offense rubrics, and a QB that loses the property is a real
// FAIL (the QB gate is exercised, not dead weight — GLM review MED2).
func eval3A(reg RubricRegistry) (CaseState, string) {
	if st, why, ok := requireRubrics(reg, domain.PosWR, domain.PosQB); !ok {
		return st, why
	}
	wr := reg[domain.PosWR]
	// Declining ELITE-draft profile: age 33 (trajectory 0.00), breakout age 19 (1.00), Power
	// Four (1.00), strong college usage 0.32 (≈0.85) → composite ≈ 0.82 → breakout > 1.000.
	elite := wr.Apply(engine.Layer4Input{
		Player: engine.PlayerInput{Position: domain.PosWR, Age: 33, RAS: 8.5, HasRAS: true},
		Scouting: engine.ScoutingInput{
			BreakoutAge: 19, HasBreakoutAge: true,
			SchoolTierNorm: 1.00, HasSchoolTier: true, // Power Four (template)
			CollegeShare: 0.32, HasCollegeShare: true,
		},
	})
	// Declining THIN profile: same age-33 crater but weak static signals — late breakout age 22
	// (0.10), FCS (0.40), low usage 0.16 (≈0.14) → composite ≈ 0.17 → breakout < 1.000.
	thin := wr.Apply(engine.Layer4Input{
		Player: engine.PlayerInput{Position: domain.PosWR, Age: 33, RAS: 8.5, HasRAS: true},
		Scouting: engine.ScoutingInput{
			BreakoutAge: 22, HasBreakoutAge: true,
			SchoolTierNorm: 0.40, HasSchoolTier: true, // FCS (template)
			CollegeShare: 0.16, HasCollegeShare: true,
		},
	})
	if !(elite.BreakoutEffective >= 1.0) {
		return StateFail, fmt.Sprintf("declining elite-profile WR breakout %.4f, want ≥ 1.0000 (static signals offset age)", elite.BreakoutEffective)
	}
	if !(thin.BreakoutEffective < 1.0) {
		return StateFail, fmt.Sprintf("declining thin-profile WR breakout %.4f, want < 1.0000 (no static offset)", thin.BreakoutEffective)
	}
	if elite.FilmEffective != 1.0 {
		return StateFail, fmt.Sprintf("film must be Data-Parity neutral (no source), got %.4f", elite.FilmEffective)
	}
	// The same non-negative property must hold at QB (the co-gate): a declining elite-draft QB
	// — early breakout, Power Four, dominant college share — at age 36 (past the QB peak 32)
	// keeps his breakout ≥ 1.000 because the static signals offset the age crater (SL-OQ-017).
	qbElite := reg[domain.PosQB].Apply(engine.Layer4Input{
		Player: engine.PlayerInput{Position: domain.PosQB, Age: 36, RAS: 8.0, HasRAS: true},
		Scouting: engine.ScoutingInput{
			BreakoutAge: 20, HasBreakoutAge: true, // ≤20 → 1.00
			SchoolTierNorm: 1.00, HasSchoolTier: true, // Power Four (template)
			CollegeShare: 0.65, HasCollegeShare: true, // ≥0.65 → 1.00
		},
	})
	if !(qbElite.BreakoutEffective >= 1.0) {
		return StateFail, fmt.Sprintf("declining elite-profile QB breakout %.4f, want ≥ 1.0000 (static offset holds at QB too)", qbElite.BreakoutEffective)
	}
	return StatePass, fmt.Sprintf("declining elite WR breakout=%.4f & QB %.4f (both ≥1, static offset) vs thin WR %.4f (<1); film neutral", elite.BreakoutEffective, qbElite.BreakoutEffective, thin.BreakoutEffective)
}

// eval3K is the B5b-WR close gate for S-curve boundary safety: pathological inputs (overflow,
// NaN, +Inf) driven through the registered WR rubric must never push a component past its
// documented asymptote (film ±5%, RAS ±8%, breakout ±5%) and must never escape as non-finite
// (the curve.Scurve NaN guard + clamp, M3). WR is 3K's sole gate. The rb_test/qb_test
// M3 clamp test is the unit-level template; this asserts it through the real registered rubric.
func eval3K(reg RubricRegistry) (CaseState, string) {
	if st, why, ok := requireRubrics(reg, domain.PosWR); !ok {
		return st, why
	}
	wr := reg[domain.PosWR]
	for _, bad := range []float64{1e9, math.NaN(), math.Inf(1)} {
		out := wr.Apply(engine.Layer4Input{
			Player: engine.PlayerInput{Position: domain.PosWR, Age: 24, RAS: bad, HasRAS: true},
			Scouting: engine.ScoutingInput{
				FilmComposite: bad, HasFilm: true,
				BreakoutAge: bad, HasBreakoutAge: true,
				SchoolTierNorm: bad, HasSchoolTier: true,
				CollegeShare: bad, HasCollegeShare: true,
			},
		})
		for _, c := range []struct {
			name    string
			v, band float64
		}{
			{"film", out.FilmEffective, 0.05},
			{"RAS", out.RASEffective, 0.08},
			{"breakout", out.BreakoutEffective, 0.05},
		} {
			if math.IsNaN(c.v) || math.IsInf(c.v, 0) {
				return StateFail, fmt.Sprintf("%s effect %v non-finite — an input escaped the NaN guard (bad=%v)", c.name, c.v, bad)
			}
			if c.v < 1-c.band-1e-9 || c.v > 1+c.band+1e-9 {
				return StateFail, fmt.Sprintf("%s effect %v escaped band ±%v (bad=%v)", c.name, c.v, c.band, bad)
			}
		}
	}
	return StatePass, "extreme/NaN/Inf inputs stay clamped to [1-cap,1+cap] for film(±5%) RAS(±8%) breakout(±5%)"
}

// eval3M is the B5b-TE close gate (SL-019, TE_Rubric §4): the RAS-modulator lifts the breakout
// component by the player's athletic profile. For the SAME aging-late TE (age 32 past peak,
// breakout age 22 — both sub-signals carry headroom), a higher RAS yields a higher breakout
// component (more athletic → more credit on breakout-age + age-trajectory), and an ABSENT RAS
// gives the LEAST (the modulator's Data-Parity stance — no athletic credit, modulated = base).
// TE is the first SL-019 instance; 3E stays DE's canonical instance. RAS rides on PlayerInput,
// so this needs no new Layer4Input field — only the registered TE rubric. Film stays neutral.
func eval3M(reg RubricRegistry) (CaseState, string) {
	if st, why, ok := requireRubrics(reg, domain.PosTE); !ok {
		return st, why
	}
	te := reg[domain.PosTE]
	// Aging-late TE: age 32 (trajectory base 0.10) + breakout age 22 (base 0.50) → both
	// modulated sub-signals have headroom. Group of Five + 15% usage anchor the static half.
	bk := engine.ScoutingInput{
		BreakoutAge: 22, HasBreakoutAge: true,
		SchoolTierNorm: 0.70, HasSchoolTier: true,
		CollegeShare: 0.15, HasCollegeShare: true,
	}
	at := func(age, ras float64, hasRAS bool, s engine.ScoutingInput) engine.Layer4Output {
		return te.Apply(engine.Layer4Input{Player: engine.PlayerInput{Position: domain.PosTE, Age: age, RAS: ras, HasRAS: hasRAS}, Scouting: s})
	}
	hi := at(32, 9.5, true, bk)
	lo := at(32, 4.0, true, bk)
	absent := at(32, 0, false, bk)
	if !(hi.BreakoutEffective > lo.BreakoutEffective && lo.BreakoutEffective > absent.BreakoutEffective) {
		return StateFail, fmt.Sprintf("SL-019: breakout must rise with RAS: hi %.5f > lo %.5f > absent %.5f", hi.BreakoutEffective, lo.BreakoutEffective, absent.BreakoutEffective)
	}
	// Isolate (a) breakout-age: a YOUNG TE (age 25 → trajectory base 1.00, modulator inert) with
	// a late breakout age present — only the breakout-age sub-signal can move with RAS.
	baHi := at(25, 9.5, true, bk)
	baLo := at(25, 4.0, true, bk)
	if !(baHi.BreakoutEffective > baLo.BreakoutEffective) {
		return StateFail, fmt.Sprintf("SL-019(a): breakout-age not RAS-modulated in isolation: %.5f not > %.5f", baHi.BreakoutEffective, baLo.BreakoutEffective)
	}
	// Isolate (b) age-trajectory: breakout age ABSENT (neutral, un-modulated) at age 32 — only the
	// age-trajectory sub-signal can move with RAS.
	noBA := engine.ScoutingInput{SchoolTierNorm: 0.70, HasSchoolTier: true, CollegeShare: 0.15, HasCollegeShare: true}
	atHi := at(32, 9.5, true, noBA)
	atLo := at(32, 4.0, true, noBA)
	if !(atHi.BreakoutEffective > atLo.BreakoutEffective) {
		return StateFail, fmt.Sprintf("SL-019(b): age-trajectory not RAS-modulated in isolation: %.5f not > %.5f", atHi.BreakoutEffective, atLo.BreakoutEffective)
	}
	// Negative contract: with BOTH modulated sub-signals inert (breakout age absent + young age
	// → trajectory base 1.00), school tier and college usage must NOT move with RAS — the breakout
	// component is flat across the RAS sweep (SL-019 touches only breakout-age + age-trajectory).
	flatHi := at(25, 9.5, true, noBA)
	flatLo := at(25, 0.5, true, noBA)
	if math.Abs(flatHi.BreakoutEffective-flatLo.BreakoutEffective) > 1e-9 {
		return StateFail, fmt.Sprintf("SL-019: school/college must NOT be RAS-modulated, but breakout moved %.6f→%.6f", flatLo.BreakoutEffective, flatHi.BreakoutEffective)
	}
	if hi.FilmEffective != 1.0 {
		return StateFail, fmt.Sprintf("film must be Data-Parity neutral (no source), got %.4f", hi.FilmEffective)
	}
	return StatePass, fmt.Sprintf("SL-019 lifts breakout-age AND age-trajectory with RAS (hi=%.5f>lo=%.5f>absent=%.5f); school/college un-modulated; film neutral", hi.BreakoutEffective, lo.BreakoutEffective, absent.BreakoutEffective)
}

// eval3E is the B5b-DE close gate (SL-019, DE_Rubric §4) — the SECOND SL-019 instance and the
// FIRST defensive one, proving the shared curve.SL019 helper transfers verbatim from TE (M17).
// It mirrors eval3M's structure: for the SAME aging-late DE (age 32 past peak, breakout age 21
// — both sub-signals carry headroom), a higher RAS yields a higher breakout component, an
// ABSENT RAS gives the LEAST (the modulator's Data-Parity stance), each modulated sub-signal is
// proven to move with RAS IN ISOLATION, and the negative contract holds (school/college are NOT
// RAS-modulated). RAS rides on PlayerInput, so this needs no new Layer4Input field — only the
// registered DE rubric. Film stays Data-Parity neutral (IDP film weights unset).
func eval3E(reg RubricRegistry) (CaseState, string) {
	if st, why, ok := requireRubrics(reg, domain.PosDE); !ok {
		return st, why
	}
	de := reg[domain.PosDE]
	// Aging-late DE: age 32 (trajectory base 0.20) + breakout age 21 (base 0.15) → both
	// modulated sub-signals have headroom. Group of Five + 20% share anchor the static half.
	bk := engine.ScoutingInput{
		BreakoutAge: 21, HasBreakoutAge: true,
		SchoolTierNorm: 0.70, HasSchoolTier: true,
		CollegeShare: 0.20, HasCollegeShare: true,
	}
	at := func(age, ras float64, hasRAS bool, s engine.ScoutingInput) engine.Layer4Output {
		return de.Apply(engine.Layer4Input{Player: engine.PlayerInput{Position: domain.PosDE, Age: age, RAS: ras, HasRAS: hasRAS}, Scouting: s})
	}
	hi := at(32, 9.99, true, bk)
	lo := at(32, 4.18, true, bk)
	absent := at(32, 0, false, bk)
	if !(hi.BreakoutEffective > lo.BreakoutEffective && lo.BreakoutEffective > absent.BreakoutEffective) {
		return StateFail, fmt.Sprintf("SL-019: breakout must rise with RAS: hi %.5f > lo %.5f > absent %.5f", hi.BreakoutEffective, lo.BreakoutEffective, absent.BreakoutEffective)
	}
	// Isolate (a) breakout-age: a YOUNG DE (age 26 → trajectory base 1.00, modulator inert) with
	// a late breakout age present — only the breakout-age sub-signal can move with RAS.
	baHi := at(26, 9.99, true, bk)
	baLo := at(26, 4.18, true, bk)
	if !(baHi.BreakoutEffective > baLo.BreakoutEffective) {
		return StateFail, fmt.Sprintf("SL-019(a): breakout-age not RAS-modulated in isolation: %.5f not > %.5f", baHi.BreakoutEffective, baLo.BreakoutEffective)
	}
	// Isolate (b) age-trajectory: breakout age ABSENT (neutral, un-modulated) at age 32 — only the
	// age-trajectory sub-signal can move with RAS.
	noBA := engine.ScoutingInput{SchoolTierNorm: 0.70, HasSchoolTier: true, CollegeShare: 0.20, HasCollegeShare: true}
	atHi := at(32, 9.99, true, noBA)
	atLo := at(32, 4.18, true, noBA)
	if !(atHi.BreakoutEffective > atLo.BreakoutEffective) {
		return StateFail, fmt.Sprintf("SL-019(b): age-trajectory not RAS-modulated in isolation: %.5f not > %.5f", atHi.BreakoutEffective, atLo.BreakoutEffective)
	}
	// Negative contract: with BOTH modulated sub-signals inert (breakout age absent + young age
	// → trajectory base 1.00), school tier and college share must NOT move with RAS — the breakout
	// component is flat across the RAS sweep (SL-019 touches only breakout-age + age-trajectory).
	flatHi := at(26, 9.99, true, noBA)
	flatLo := at(26, 0.5, true, noBA)
	if math.Abs(flatHi.BreakoutEffective-flatLo.BreakoutEffective) > 1e-9 {
		return StateFail, fmt.Sprintf("SL-019: school/college must NOT be RAS-modulated, but breakout moved %.6f→%.6f", flatLo.BreakoutEffective, flatHi.BreakoutEffective)
	}
	if hi.FilmEffective != 1.0 {
		return StateFail, fmt.Sprintf("film must be Data-Parity neutral (no source), got %.4f", hi.FilmEffective)
	}
	return StatePass, fmt.Sprintf("SL-019 (DE, 2nd instance) lifts breakout-age AND age-trajectory with RAS (hi=%.5f>lo=%.5f>absent=%.5f); school/college un-modulated; film neutral", hi.BreakoutEffective, lo.BreakoutEffective, absent.BreakoutEffective)
}

// eval3L runs today: MFL player IDs are strings and leading zeros are significant. It
// drives the real schema decode path (the boundary that would convert an ID to an int if
// it were going to) and asserts each ID survives byte-for-byte.
func eval3L(_ RubricRegistry) (CaseState, string) {
	for _, id := range []string{"0001", "0999", "14263"} {
		js := fmt.Sprintf(`{"id":%q,"name":"Test","position":"WR","salary":"1"}`, id)
		rec, err := schema.DecodePlayerRecord(strings.NewReader(js))
		if err != nil {
			return StateFail, fmt.Sprintf("id %q failed decode: %v", id, err)
		}
		if rec.ID != id {
			return StateFail, fmt.Sprintf("id %q mutated to %q (leading zero stripped or int-coerced)", id, rec.ID)
		}
	}
	return StatePass, `IDs "0001","0999","14263" preserved as strings with leading zeros`
}
