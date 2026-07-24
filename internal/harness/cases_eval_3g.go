package harness

import (
	"fmt"
	"math"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine/l4/defense"
)

// sl021Blender is the case-3G introspection hook both DL rubrics satisfy: the SL-021 EMA blend
// rate over the pass-rush grade, by 1-based NFL year. DT returns the DYNAMIC schedule (0.50→0.10);
// DE returns its FIXED control (0.15, no mid-career switch). (Formerly a "PFF alpha" — PFF is
// retired; only the SL-021 α schedule survives, now over the pfrpassrush pressure grade.)
type sl021Blender interface{ SL021Alpha(nflYear int) float64 }

// eval3G is the SL-021 dynamic-α close gate (Testing_App_Specification Test 3G). It asserts the
// α SCHEDULE and the EMA blend math on the spec's synthetic grades (previous 0.60, observation
// 0.90): DT blends aggressively in Year 1 (α=0.50 → 0.75) and stably in Year 2+ (α=0.10 → 0.63),
// while the DE control uses a fixed α=0.15 (→ 0.645) that never switches with NFL year — proving
// the mid-career switch is DT-UNIQUE. This is the α-schedule-only wiring: no live pass-rush
// observation feeds production scoring yet (the pfrpassrush pressure composite proved largely
// redundant with the locked Madden IDP anchor at DT/DE — C-1 evidence,
// docs/data-layer/PassRush_C1_Distributions.md — so a live DT/DE weight is deferred to the
// expert-panel gate). The case sets no weight and does not touch the locked film budget.
func eval3G(reg RubricRegistry) (CaseState, string) {
	if st, why, ok := requireRubrics(reg, domain.PosDT, domain.PosDE); !ok {
		return st, why
	}
	dt, dok := reg[domain.PosDT].(sl021Blender)
	de, eok := reg[domain.PosDE].(sl021Blender)
	if !dok || !eok {
		return StateFail, "DT and DE must expose SL021Alpha (case-3G hook)"
	}
	// The DE control must not switch α mid-career (the whole point of the control).
	if de.SL021Alpha(1) != de.SL021Alpha(5) {
		return StateFail, fmt.Sprintf("DE control α switched by year (%.3f→%.3f); must be fixed",
			de.SL021Alpha(1), de.SL021Alpha(5))
	}
	const prev, obs = 0.60, 0.90
	dtY1 := defense.SL021Blend(prev, obs, dt.SL021Alpha(1))
	dtY2 := defense.SL021Blend(prev, obs, dt.SL021Alpha(2))
	deControl := defense.SL021Blend(prev, obs, de.SL021Alpha(1))
	// Spec FAIL guard: the DE control must NOT produce the DT-dynamic Year-1 value (0.75).
	if math.Abs(deControl-0.75) < 1e-9 {
		return StateFail, "DE control produced 0.75 — SL-021 dynamic α wrongly applied to DE"
	}
	for _, c := range []struct {
		label       string
		got, lo, hi float64
	}{
		{"DT Y1", dtY1, 0.74, 0.76},
		{"DT Y2+", dtY2, 0.62, 0.64},
		{"DE control", deControl, 0.64, 0.65},
	} {
		if c.got < c.lo || c.got > c.hi {
			return StateFail, fmt.Sprintf("%s blend %.4f outside [%.2f,%.2f]", c.label, c.got, c.lo, c.hi)
		}
	}
	return StatePass, fmt.Sprintf("SL-021 α: DT Y1 %.3f / Y2+ %.3f (dynamic 0.50→0.10); DE control %.3f (fixed 0.15, no switch)",
		dtY1, dtY2, deControl)
}
