package fit

import (
	"fmt"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/model"
)

// OwnerRule is a valuation choice the league owner made over the fit, applied to the stored
// params after fitting so a refit cannot quietly undo it. The holdout cannot see these: the fit
// scores ranks, and the rules change how much this season and older seasons count, and when a
// position starts to age.
type OwnerRule struct {
	Position domain.Position
	Recency  *[2]float64 // weights of the seasons two and three before the one projected
	KNow     float64     // games of this season that count as much as the projection; 0 keeps the fit's
	ArcShift float64     // years later the talent arc runs: the step at age a is the fitted step at a-ArcShift
	Arc      *[3]float64 // the arc's level, age and age-squared terms, where the owner sets the curve
	Cliff    [2]float64  // from this age on, this extra step each season; 0 is no cliff
	Unproven float64     // shrink toward the middle on the prior of a player with no NFL games; 0 is none
	Why      string
}

// OwnerRules are the owner's rulings of 2026-10-27, from the value board's wrong reads.
func OwnerRules() []OwnerRule {
	rules := []OwnerRule{
		{Position: domain.PosDE, Recency: &[2]float64{0.8, 0}, KNow: 2.0,
			Arc: &[3]float64{0, -0.01, -0.006}, Cliff: [2]float64{35, -1},
			Why: "reputation holds a season, 80% the next, then stops; this season's games take over as fast as at the other IDP spots; " +
				"an end ages gently from his late 20s and falls off a cliff at 35"},
		{Position: domain.PosCB, Recency: &[2]float64{0.6, 0.4}, KNow: 6.0,
			Why: "a CB's points say little about his coverage, so one month counts less and years of steady scoring count more"},
		{Position: domain.PosTE, ArcShift: 3,
			Why: "tight ends play well to older ages, so they age as the fit has them three years younger"},
	}
	for _, pos := range model.Positions() {
		rules = append(rules, OwnerRule{Position: pos, Unproven: 0.3,
			Why: "a player with no NFL games is boom or bust, high picks too; his draft profile counts 30% nearer the middle until he plays"})
	}
	return rules
}

// ApplyOwnerRules sets each rule's weights on its position's stored params.
func ApplyOwnerRules(results []Result, rules []OwnerRule) {
	for i := range results {
		for _, r := range rules {
			if results[i].Position != r.Position {
				continue
			}
			p := &results[i].Params
			if r.Recency != nil {
				p.Recency = *r.Recency
			}
			if r.KNow > 0 {
				p.KNow = r.KNow
			}
			if r.Arc != nil {
				p.Arc[0], p.Arc[1], p.Arc[2] = r.Arc[0], r.Arc[1], r.Arc[2]
			}
			if r.Cliff[0] > 0 {
				p.Cliff = r.Cliff
			}
			if r.Unproven > 0 {
				p.Unproven = r.Unproven
			}
			// The arc is a quadratic in age-ArcCenter; moving it s years later rewrites its terms.
			s := r.ArcShift
			p.Arc[0], p.Arc[1] = p.Arc[0]-p.Arc[1]*s+p.Arc[2]*s*s, p.Arc[1]-2*p.Arc[2]*s
		}
	}
}

// ownerRulesMarkdown is the report's record of the rules over the fit.
func ownerRulesMarkdown(rules []OwnerRule) string {
	var b strings.Builder
	b.WriteString("\n## Owner rules over the fit\n\nApplied to the stored params after fitting; the holdout above scores the fitted values.\n\n")
	b.WriteString("| Pos | Recency, 2 and 3 seasons back | k_now | Arc | Cliff | Why |\n|---|---|---|---|---|---|\n")
	for _, r := range rules {
		rec, k := "fitted", "fitted"
		if r.Recency != nil {
			rec = fmt.Sprintf("%.1f / %.1f", r.Recency[0], r.Recency[1])
		}
		if r.KNow > 0 {
			k = fmt.Sprintf("%.1f", r.KNow)
		}
		arc, cliff := "fitted", "none"
		switch {
		case r.Arc != nil:
			arc = fmt.Sprintf("set: %+.3f %+.3f·(a−27) %+.4f·(a−27)²", r.Arc[0], r.Arc[1], r.Arc[2])
		case r.ArcShift != 0:
			arc = fmt.Sprintf("fitted, %.0f years later", r.ArcShift)
		}
		if r.Cliff[0] > 0 {
			cliff = fmt.Sprintf("%+.1f a season from %.0f", r.Cliff[1], r.Cliff[0])
		}
		if r.Unproven > 0 {
			cliff += fmt.Sprintf("; unplayed prior %.0f%% nearer the middle", 100*r.Unproven)
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s |\n", r.Position, rec, k, arc, cliff, r.Why)
	}
	return b.String()
}
