package model

import (
	"fmt"
	"math"
)

// ArcCenter is the age the talent arc's terms are centred on.
const ArcCenter = 27.0

// Params are one position's fitted model.
type Params struct {
	// Prior: Intercept + Σ (known ? Weight·x : Missing), on the percentile scale.
	Intercept       float64
	Weight, Missing []float64

	KNow, KDynasty float64 // games of evidence at which production and the prior weigh equally
	Exponential    bool    // Z = 1 − exp(−e/k) instead of e/(e+k)
	Recency        [2]float64

	// Talent arc: next season's percentile change at age a and experience x is
	// Arc[0] + Arc[1]·(a−27) + Arc[2]·(a−27)² + Arc[3]·[x = 1].
	Arc [4]float64

	// Survival: the log-odds of playing next season are Survival · (1, a−27, (a−27)², draft,
	// games/17, percentile), where draft is the log of the overall pick, 260 when undrafted.
	Survival [6]float64

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
func (p Params) Survives(age, draftPick, games, pct float64) float64 {
	a := age - ArcCenter
	z := p.Survival[0] + p.Survival[1]*a + p.Survival[2]*a*a + p.Survival[3]*math.Log(draftCapital(draftPick)) +
		p.Survival[4]*games/17 + p.Survival[5]*pct
	return 1 / (1 + math.Exp(-z))
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
	for i, f := range PriorFeatures() {
		out["model.prior."+f+".weight"] = p.Weight[i]
		out["model.prior."+f+".missing"] = p.Missing[i]
	}
	for i, term := range arcTerms() {
		out["model.arc."+term] = p.Arc[i]
	}
	for i, term := range survivalTerms() {
		out["model.survival."+term] = p.Survival[i]
	}
	out["model.debut.level"], out["model.debut.draft"] = p.Debut[0], p.Debut[1]
	return out
}

// ParamsFrom reads a position's params back through get.
func ParamsFrom(get func(key string) (float64, error)) (Params, error) {
	n := len(PriorFeatures())
	p := Params{Weight: make([]float64, n), Missing: make([]float64, n)}
	var z float64
	targets := map[string]*float64{
		"model.prior.intercept": &p.Intercept, "model.k_now": &p.KNow, "model.k_dynasty": &p.KDynasty,
		"model.z_exponential": &z, "model.recency.1": &p.Recency[0], "model.recency.2": &p.Recency[1],
	}
	for i, f := range PriorFeatures() {
		targets["model.prior."+f+".weight"] = &p.Weight[i]
		targets["model.prior."+f+".missing"] = &p.Missing[i]
	}
	for i, term := range arcTerms() {
		targets["model.arc."+term] = &p.Arc[i]
	}
	for i, term := range survivalTerms() {
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

func survivalTerms() []string {
	return []string{"level", "age", "age_squared", "draft", "games", "percentile"}
}

func boolValue(b bool) float64 {
	if b {
		return 1
	}
	return 0
}
