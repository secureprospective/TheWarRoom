package fit

import (
	"fmt"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/domain"
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
	Why      string
}

// OwnerRules are the owner's rulings of 2026-10-27, from the value board's wrong reads.
func OwnerRules() []OwnerRule {
	return []OwnerRule{
		{Position: domain.PosDE, Recency: &[2]float64{0.8, 0}, KNow: 2.0,
			Why: "reputation holds a season, 80% the next, then stops; this season's games take over as fast as at the other IDP spots"},
		{Position: domain.PosCB, Recency: &[2]float64{0.6, 0.4}, KNow: 6.0,
			Why: "a CB's points say little about his coverage, so one month counts less and years of steady scoring count more"},
		{Position: domain.PosTE, ArcShift: 3,
			Why: "tight ends play well to older ages, so they age as the fit has them three years younger"},
	}
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
	b.WriteString("| Pos | Recency, 2 and 3 seasons back | k_now | Arc later by | Why |\n|---|---|---|---|---|\n")
	for _, r := range rules {
		rec, k := "fitted", "fitted"
		if r.Recency != nil {
			rec = fmt.Sprintf("%.1f / %.1f", r.Recency[0], r.Recency[1])
		}
		if r.KNow > 0 {
			k = fmt.Sprintf("%.1f", r.KNow)
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %.0f years | %s |\n", r.Position, rec, k, r.ArcShift, r.Why)
	}
	return b.String()
}
