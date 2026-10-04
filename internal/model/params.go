package model

import (
	"fmt"
	"math"
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// ArcCenter is the age the talent arc's terms are centred on.
const ArcCenter = 27.0

// Params are one position's fitted model.
type Params struct {
	Position domain.Position // names the prior's inputs (PriorFeatures)

	// Prior: Intercept + Σ (known ? Weight·x : Missing), on the percentile scale.
	Intercept       float64
	Weight, Missing []float64

	KNow, KDynasty float64 // games of evidence at which production and the prior weigh equally
	Exponential    bool    // Z = 1 − exp(−e/k) instead of e/(e+k)
	Recency        [2]float64

	// Talent arc: next season's percentile change at age a and experience x is
	// Arc[0] + Arc[1]·(a−27) + Arc[2]·(a−27)² + Arc[3]·[x = 1].
	Arc [4]float64

	// Survival: the log-odds of playing next season are Survival · SurvivalRow, the terms
	// SurvivalTerms names for the position.
	Survival []float64

	// Debut: the log-odds that a rookie becomes a regular are Debut · (1, draft).
	Debut [2]float64
}

// Prior is the player's expected percentile from pre-NFL facts, kept inside [0.01, 0.99].
func (p Params) Prior(pl *Player) float64 {
	x, known := pl.PriorInputs()
	v := p.Intercept
	for i := range x {
		if i >= len(p.Weight) {
			break
		}
		if known[i] {
			v += p.Weight[i] * x[i]
		} else {
			v += p.Missing[i]
		}
	}
	return min(max(v, 0.01), 0.99)
}

// Z is the weight production gets with e games of evidence against k.
func (p Params) Z(e, k float64) float64 {
	if e <= 0 || k <= 0 {
		return math.Max(0, math.Min(1, e))
	}
	if p.Exponential {
		return 1 - math.Exp(-e/k)
	}
	return e / (e + k)
}

// ArcStep is the expected percentile change from age to age+1.
func (p Params) ArcStep(age float64, experience int) float64 {
	a := age - ArcCenter
	step := p.Arc[0] + p.Arc[1]*a + p.Arc[2]*a*a
	if experience == 1 {
		step += p.Arc[3]
	}
	return step
}

// Survives is the probability of playing next season.
func (p Params) Survives(age, draftPick, games, pct float64, t Tenure) float64 {
	z := 0.0
	for i, x := range p.SurvivalRow(age, draftPick, games, pct, t) {
		if i < len(p.Survival) {
			z += p.Survival[i] * x
		}
	}
	return 1 / (1 + math.Exp(-z))
}

// SurvivalRow is the survival arc's inputs in SurvivalTerms order: 1, a−27, (a−27)², the log of
// the overall pick (260 when undrafted), games/17 and the percentile; at a defensive position
// also the tenure's cap share (×10), its years left (−1 to 5, ÷5), its guaranteed share, and 1
// when no contract is known (the other three then 0).
func (p Params) SurvivalRow(age, draftPick, games, pct float64, t Tenure) []float64 {
	a := age - ArcCenter
	row := []float64{1, a, a * a, math.Log(draftCapital(draftPick)), games / 17, pct}
	if len(SurvivalTerms(p.Position)) == len(row) {
		return row
	}
	if !t.Known {
		return append(row, 0, 0, 0, 1)
	}
	return append(row, t.CapPct*10, min(max(t.YearsLeft, -1), 5)/5, t.Guaranteed, 0)
}

// Debuts is the probability a rookie with this overall pick (0 when undrafted) becomes a
// regular in his first season.
func (p Params) Debuts(draftPick float64) float64 {
	return 1 / (1 + math.Exp(-(p.Debut[0] + p.Debut[1]*math.Log(draftCapital(draftPick)))))
}

// draftCapital is the overall pick the draft terms read: 260, past the last pick, when undrafted.
func draftCapital(pick float64) float64 {
	if pick < 1 {
		return 260
	}
	return pick
}

// Values flattens the params into named values for storage, keyed as Keys names them.
func (p Params) Values() map[string]float64 {
	out := map[string]float64{
		"model.prior.intercept": p.Intercept, "model.k_now": p.KNow, "model.k_dynasty": p.KDynasty,
		"model.z_exponential": boolValue(p.Exponential),
		"model.recency.1":     p.Recency[0], "model.recency.2": p.Recency[1],
	}
	for i, f := range PriorFeatures(p.Position) {
		out["model.prior."+f+".weight"] = p.Weight[i]
		out["model.prior."+f+".missing"] = p.Missing[i]
	}
	for i, term := range arcTerms() {
		out["model.arc."+term] = p.Arc[i]
	}
	for i, term := range SurvivalTerms(p.Position) {
		out["model.survival."+term] = p.Survival[i]
	}
	out["model.debut.level"], out["model.debut.draft"] = p.Debut[0], p.Debut[1]
	return out
}

// ParamsFrom reads a position's params back through get.
func ParamsFrom(pos domain.Position, get func(key string) (float64, error)) (Params, error) {
	n := len(PriorFeatures(pos))
	p := Params{Position: pos, Weight: make([]float64, n), Missing: make([]float64, n),
		Survival: make([]float64, len(SurvivalTerms(pos)))}
	var z float64
	targets := map[string]*float64{
		"model.prior.intercept": &p.Intercept, "model.k_now": &p.KNow, "model.k_dynasty": &p.KDynasty,
		"model.z_exponential": &z, "model.recency.1": &p.Recency[0], "model.recency.2": &p.Recency[1],
	}
	for i, f := range PriorFeatures(pos) {
		targets["model.prior."+f+".weight"] = &p.Weight[i]
		targets["model.prior."+f+".missing"] = &p.Missing[i]
	}
	for i, term := range arcTerms() {
		targets["model.arc."+term] = &p.Arc[i]
	}
	for i, term := range SurvivalTerms(pos) {
		targets["model.survival."+term] = &p.Survival[i]
	}
	targets["model.debut.level"], targets["model.debut.draft"] = &p.Debut[0], &p.Debut[1]
	for key, dst := range targets {
		v, err := get(key)
		if err != nil {
			return Params{}, fmt.Errorf("model: %w", err)
		}
		*dst = v
	}
	p.Exponential = z != 0
	return p, nil
}

func arcTerms() []string { return []string{"level", "age", "age_squared", "second_year"} }

// ReadsContract reports whether the survival arc gives the contract any weight: the terms exist
// at defensive positions, and the fit zeroes them where they did not beat the arc without them.
func (p Params) ReadsContract() bool {
	base := len(SurvivalTerms(domain.PosQB))
	return len(p.Survival) > base && slices.ContainsFunc(p.Survival[base:], func(w float64) bool { return w != 0 })
}

// SurvivalTerms names a position's survival terms. Defensive positions add the contract they are
// on; at offense it did not help (worse log loss at QB and WR on the 2024–2025 holdouts).
func SurvivalTerms(pos domain.Position) []string {
	base := []string{"level", "age", "age_squared", "draft", "games", "percentile"}
	switch pos {
	case domain.PosDT, domain.PosDE, domain.PosLB, domain.PosCB, domain.PosS:
		return append(base, "contract_cap", "contract_years_left", "contract_guaranteed", "contract_unknown")
	case domain.PosQB, domain.PosRB, domain.PosWR, domain.PosTE, domain.PosK, domain.PosFlag:
	}
	return base
}

func boolValue(b bool) float64 {
	if b {
		return 1
	}
	return 0
}
