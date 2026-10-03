// Package numeric holds small helpers with no internal imports, so every layer, including
// the pure engine, can use them.
package numeric

import "math"

// Finite reports whether every value is neither NaN nor ±Inf.
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
