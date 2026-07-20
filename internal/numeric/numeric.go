// Package numeric holds tiny, dependency-free numeric helpers shared across layers.
// It imports only the standard library math package, so every layer — including the
// pure engine (which depguard forbids from importing anything but domain) — can use it
// without breaching the three-layer law.
package numeric

import "math"

// Finite reports whether every value is a real, non-NaN, non-Inf number. The engine's
// layers multiply straight into the final score, so a single non-finite input would
// round-trip silently into rankings; the fail-loud gates use this to reject such input
// at the boundary rather than emit a poisoned score (the bug class B4's review caught).
func Finite(vs ...float64) bool {
	for _, v := range vs {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return false
		}
	}
	return true
}

// BoolToInt maps a bool to the 0/1 SQLite integer encoding.
func BoolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
