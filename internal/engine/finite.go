package engine

import "math"

// finite reports whether every value is a real, non-NaN, non-Inf number. The engine's
// layers multiply straight into the final score, so a single non-finite input would
// round-trip silently into rankings; the fail-loud gates use this to reject such input
// at the boundary rather than emit a poisoned score (the bug class B4's review caught).
func finite(vs ...float64) bool {
	for _, v := range vs {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return false
		}
	}
	return true
}
